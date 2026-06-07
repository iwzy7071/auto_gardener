package app

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type dingTalkIncomingMessage struct {
	ConversationID string `json:"conversationId"`
	SenderID       string `json:"senderId"`
	SenderNick     string `json:"senderNick"`
	MsgType        string `json:"msgtype"`
	SessionWebhook string `json:"sessionWebhook"`
	Text           struct {
		Content string `json:"content"`
	} `json:"text"`
	Content struct {
		Recognition string `json:"recognition"`
	} `json:"content"`
}

type dingTalkTextResponse struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

func (s *Server) handleDingTalkRobot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := verifyDingTalkIncomingSignature(r); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	var msg dingTalkIncomingMessage
	if !decodeLimitedJSON(w, r, &msg, maxDingTalkJSONBodyBytes, "钉钉消息不是合法 JSON") {
		return
	}
	content := extractDingTalkContent(msg)
	if strings.TrimSpace(content) == "" {
		content = "帮助"
	}
	reply := s.handleDingTalkCommand(msg, content)
	if strings.TrimSpace(reply) == "" {
		reply = "已收到。"
	}
	if err := s.sendDingTalkReply(msg.SessionWebhook, reply); err != nil {
		// Even if async/session reply fails, return the text payload so webhook tests
		// and DingTalk compatible callers can still see the response.
		writeJSON(w, http.StatusOK, dingTalkTextPayload("发送钉钉回复失败："+err.Error()+"\n\n"+reply))
		return
	}
	writeJSON(w, http.StatusOK, dingTalkTextPayload(reply))
}

func extractDingTalkContent(msg dingTalkIncomingMessage) string {
	switch strings.ToLower(strings.TrimSpace(msg.MsgType)) {
	case "text", "":
		return strings.TrimSpace(msg.Text.Content)
	case "audio":
		return strings.TrimSpace(msg.Content.Recognition)
	default:
		return strings.TrimSpace(firstNonEmpty(msg.Text.Content, msg.Content.Recognition))
	}
}

func (s *Server) handleDingTalkCommand(msg dingTalkIncomingMessage, content string) string {
	content = strings.TrimSpace(strings.TrimPrefix(content, "@Gardener"))
	content = strings.TrimSpace(strings.TrimPrefix(content, "Gardener"))
	key := dingTalkSessionKey(msg)
	cmd, arg := splitDingTalkCommand(content)
	switch cmd {
	case "帮助", "help", "?":
		return dingTalkHelpText()
	case "任务列表", "列表", "list", "tasks":
		return s.dingTalkListTasks()
	case "新任务", "创建任务", "create", "new":
		if strings.TrimSpace(arg) == "" {
			return "请在“新任务”后面写清楚目标，例如：新任务 帮我整理这个项目的 README。"
		}
		task, err := s.orchestrator.CreateTask(arg, "")
		if err != nil {
			return "创建任务失败：" + err.Error()
		}
		s.setDingTalkSessionTask(key, task.ID)
		return fmt.Sprintf("已创建任务：%s\n任务 ID：%s\n你可以发送“状态”“继续”“停止”，或直接补充要求。", task.Title, task.ID)
	case "状态", "进度", "status", "progress":
		taskID := strings.TrimSpace(arg)
		if taskID == "" {
			taskID = s.getDingTalkSessionTask(key)
		}
		return s.dingTalkTaskStatus(taskID)
	case "继续", "继续任务", "resume", "continue":
		taskID := strings.TrimSpace(arg)
		if taskID == "" {
			taskID = s.getDingTalkSessionTask(key)
		}
		if taskID == "" {
			return "还没有绑定任务。请先发送“新任务 你的目标”，或“继续 任务ID”。"
		}
		task, err := s.orchestrator.ResumeTask(taskID)
		if err != nil {
			return "继续任务失败：" + err.Error()
		}
		s.setDingTalkSessionTask(key, task.ID)
		return "已继续任务：" + task.Title + "\n任务 ID：" + task.ID
	case "停止", "stop":
		taskID := strings.TrimSpace(arg)
		if taskID == "" {
			taskID = s.getDingTalkSessionTask(key)
		}
		if taskID == "" {
			return "还没有绑定任务。"
		}
		task, err := s.orchestrator.StopTask(taskID)
		if err != nil {
			return "停止任务失败：" + err.Error()
		}
		return "已停止任务：" + task.Title
	}
	if taskID := s.getDingTalkSessionTask(key); taskID != "" {
		task, err := s.orchestrator.SendMessage(taskID, content)
		if err != nil {
			return "发送给 Gardener 失败：" + err.Error()
		}
		return fmt.Sprintf("已发送给任务：%s\n任务 ID：%s", task.Title, task.ID)
	}
	task, err := s.orchestrator.CreateTask(content, "")
	if err != nil {
		return "创建任务失败：" + err.Error()
	}
	s.setDingTalkSessionTask(key, task.ID)
	return fmt.Sprintf("已按你的消息创建新任务：%s\n任务 ID：%s\n之后可发送“状态”“继续”“停止”。", task.Title, task.ID)
}

func splitDingTalkCommand(content string) (string, string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "帮助", ""
	}
	fields := strings.Fields(content)
	if len(fields) == 0 {
		return "帮助", ""
	}
	cmd := strings.ToLower(fields[0])
	arg := strings.TrimSpace(strings.TrimPrefix(content, fields[0]))
	return cmd, arg
}

func dingTalkHelpText() string {
	return `Gardener 钉钉远程控制命令：
- 新任务 <目标>：创建并启动任务
- 状态 [任务ID]：查看任务进度，不会中断任务
- 继续 [任务ID]：从当前进度继续
- 停止 [任务ID]：停止任务
- 任务列表：查看最近任务
- 其他消息：发送给当前绑定任务；若未绑定，则创建新任务`
}

func (s *Server) dingTalkListTasks() string {
	tasks := s.store.ListTasks()
	if len(tasks) == 0 {
		return "暂无任务。"
	}
	if len(tasks) > 8 {
		tasks = tasks[:8]
	}
	var b strings.Builder
	b.WriteString("最近任务：")
	for _, task := range tasks {
		b.WriteString(fmt.Sprintf("\n- %s｜%s｜阶段 %d｜%s", task.ID, task.Status, task.Forest, task.Title))
	}
	return b.String()
}

func (s *Server) dingTalkTaskStatus(taskID string) string {
	if strings.TrimSpace(taskID) == "" {
		return "还没有绑定任务。请先发送“新任务 你的目标”，或指定任务 ID：状态 forest_xxx。"
	}
	task, ok := s.store.GetTask(taskID)
	if !ok {
		return "任务不存在：" + taskID
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("任务：%s\nID：%s\n状态：%s\n阶段：%d", task.Title, task.ID, task.Status, task.Forest))
	if task.LastProgressAt != nil {
		b.WriteString("\n最近进展时间：" + task.LastProgressAt.Format("2006-01-02 15:04:05"))
	}
	progress := append([]string(nil), task.GardenerProgress...)
	if len(progress) > 4 {
		progress = progress[len(progress)-4:]
	}
	if len(progress) > 0 {
		b.WriteString("\n最近进展：")
		for _, line := range progress {
			b.WriteString("\n- " + line)
		}
	}
	if task.Status == StatusFinished {
		b.WriteString("\n如未完成，可发送：继续")
	} else {
		b.WriteString("\n你可以继续询问状态，不会中断任务。")
	}
	return b.String()
}

func dingTalkSessionKey(msg dingTalkIncomingMessage) string {
	conversation := strings.TrimSpace(msg.ConversationID)
	if conversation == "" {
		conversation = "direct"
	}
	sender := strings.TrimSpace(msg.SenderID)
	if sender == "" {
		sender = strings.TrimSpace(msg.SenderNick)
	}
	if sender == "" {
		sender = "anonymous"
	}
	return conversation + ":" + sender
}

func (s *Server) setDingTalkSessionTask(key, taskID string) {
	s.dingTalkMu.Lock()
	defer s.dingTalkMu.Unlock()
	s.dingTalkSessions[key] = taskID
}

func (s *Server) getDingTalkSessionTask(key string) string {
	s.dingTalkMu.Lock()
	defer s.dingTalkMu.Unlock()
	return s.dingTalkSessions[key]
}

func verifyDingTalkIncomingSignature(r *http.Request) error {
	secret := strings.TrimSpace(firstNonEmpty(os.Getenv("AUTO_GARDENER_DINGTALK_INCOMING_SECRET"), os.Getenv("AUTO_GARDENER_DINGTALK_APP_SECRET")))
	if secret == "" {
		return nil
	}
	timestamp := strings.TrimSpace(firstNonEmpty(r.Header.Get("timestamp"), r.Header.Get("Timestamp"), r.URL.Query().Get("timestamp")))
	signature := strings.TrimSpace(firstNonEmpty(r.Header.Get("sign"), r.Header.Get("Sign"), r.URL.Query().Get("sign")))
	if timestamp == "" || signature == "" {
		return errors.New("缺少钉钉签名 header：timestamp/sign")
	}
	if ms, err := strconv.ParseInt(timestamp, 10, 64); err == nil && ms > 0 {
		if age := time.Since(time.UnixMilli(ms)); age > time.Hour || age < -time.Hour {
			return errors.New("钉钉签名 timestamp 已过期")
		}
	}
	decoded, _ := url.QueryUnescape(signature)
	expected := dingTalkSign(timestamp, secret)
	if !hmac.Equal([]byte(decoded), []byte(expected)) && !hmac.Equal([]byte(signature), []byte(expected)) {
		return errors.New("钉钉签名校验失败")
	}
	return nil
}

func dingTalkSign(timestamp, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "\n" + secret))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func dingTalkSignedWebhook(rawURL, secret string) string {
	if strings.TrimSpace(secret) == "" || strings.TrimSpace(rawURL) == "" {
		return rawURL
	}
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	sign := url.QueryEscape(dingTalkSign(timestamp, secret))
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + "timestamp=" + timestamp + "&sign=" + sign
}

func (s *Server) sendDingTalkReply(sessionWebhook, content string) error {
	webhook := strings.TrimSpace(sessionWebhook)
	if webhook == "" {
		webhook = strings.TrimSpace(os.Getenv("AUTO_GARDENER_DINGTALK_WEBHOOK"))
		webhook = dingTalkSignedWebhook(webhook, firstNonEmpty(os.Getenv("AUTO_GARDENER_DINGTALK_OUTGOING_SECRET"), os.Getenv("AUTO_GARDENER_DINGTALK_ROBOT_SECRET")))
	}
	if webhook == "" {
		return nil
	}
	body, _ := json.Marshal(dingTalkTextPayload(content))
	resp, err := s.httpClient.Post(webhook, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("钉钉 webhook HTTP %d", resp.StatusCode)
	}
	return nil
}

func dingTalkTextPayload(content string) dingTalkTextResponse {
	var payload dingTalkTextResponse
	payload.MsgType = "text"
	payload.Text.Content = content
	return payload
}

// sortedDingTalkCommands is intentionally unused by runtime but documents the
// stable command vocabulary for tests and future UI hints.
func sortedDingTalkCommands() []string {
	items := []string{"新任务", "状态", "继续", "停止", "任务列表", "帮助"}
	sort.Strings(items)
	return items
}
