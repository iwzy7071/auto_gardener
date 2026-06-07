package compat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

type Proxy struct {
	baseURL string
	server  *http.Server
	client  *http.Client
}

const maxCompatFunctionOutputBytes = 256 * 1024

type providerSpec struct {
	Name                 string
	BaseURL              string
	SupportsStreamUsage  bool
	SupportsParallelTool bool
}

var providers = map[string]providerSpec{
	"minimax": {Name: "MiniMax", BaseURL: "https://api.minimaxi.com/v1"},
	"kimi":    {Name: "Kimi Coding", BaseURL: "https://api.kimi.com/coding/v1", SupportsStreamUsage: true, SupportsParallelTool: true},
}

func Start() (*Proxy, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	p := &Proxy{
		baseURL: "http://" + ln.Addr().String(),
		client:  &http.Client{Timeout: 0},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handle)
	p.server = &http.Server{Handler: mux}
	go func() {
		if err := p.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("compat proxy stopped: %v", err)
		}
	}()
	return p, nil
}

func (p *Proxy) BaseURL() string {
	if p == nil {
		return ""
	}
	return p.baseURL
}

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[1] != "v1" || parts[2] != "responses" {
		http.NotFound(w, r)
		return
	}
	spec, ok := providers[parts[0]]
	if !ok {
		http.NotFound(w, r)
		return
	}
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeProxyError(w, http.StatusUnauthorized, "missing provider token")
		return
	}
	var req responseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProxyError(w, http.StatusBadRequest, "invalid responses request")
		return
	}
	if err := validateFunctionOutputSize(req.Input); err != nil {
		writeProxyError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := p.forwardChatStream(w, r, spec, token, req); err != nil {
		log.Printf("compat proxy %s error: %v", spec.Name, err)
	}
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return header
}

type responseRequest struct {
	Model             string          `json:"model"`
	Instructions      string          `json:"instructions"`
	Input             json.RawMessage `json:"input"`
	Tools             []responseTool  `json:"tools,omitempty"`
	ToolChoice        any             `json:"tool_choice,omitempty"`
	ParallelToolCalls bool            `json:"parallel_tool_calls,omitempty"`
	Temperature       *float64        `json:"temperature,omitempty"`
	TopP              *float64        `json:"top_p,omitempty"`
	MaxOutputTokens   *int            `json:"max_output_tokens,omitempty"`
	Stream            bool            `json:"stream"`
}

type responseTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type chatRequest struct {
	Model             string          `json:"model"`
	Messages          []chatMessage   `json:"messages"`
	Tools             []chatTool      `json:"tools,omitempty"`
	ToolChoice        any             `json:"tool_choice,omitempty"`
	ParallelToolCalls *bool           `json:"parallel_tool_calls,omitempty"`
	Temperature       *float64        `json:"temperature,omitempty"`
	TopP              *float64        `json:"top_p,omitempty"`
	MaxTokens         *int            `json:"max_tokens,omitempty"`
	Stream            bool            `json:"stream"`
	StreamOptions     map[string]bool `json:"stream_options,omitempty"`
}

type chatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
}

type chatTool struct {
	Type     string       `json:"type"`
	Function chatFunction `json:"function"`
}

type chatFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type chatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function chatCallFunction `json:"function"`
}

type chatCallFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments"`
}

func (p *Proxy) forwardChatStream(w http.ResponseWriter, r *http.Request, spec providerSpec, token string, req responseRequest) error {
	chatReq := chatRequest{
		Model:       strings.TrimSpace(req.Model),
		Messages:    responseMessages(req),
		Tools:       responseTools(req.Tools),
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxOutputTokens,
		Stream:      true,
	}
	if spec.SupportsStreamUsage {
		chatReq.StreamOptions = map[string]bool{"include_usage": true}
	}
	if req.ToolChoice != nil {
		chatReq.ToolChoice = req.ToolChoice
	}
	if len(req.Tools) > 0 && spec.SupportsParallelTool {
		v := req.ParallelToolCalls
		chatReq.ParallelToolCalls = &v
	}
	body, err := json.Marshal(chatReq)
	if err != nil {
		writeProxyError(w, http.StatusInternalServerError, "failed to build chat request")
		return err
	}
	upstreamURL := strings.TrimRight(spec.BaseURL, "/") + "/chat/completions"
	upReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		writeProxyError(w, http.StatusInternalServerError, "failed to build upstream request")
		return err
	}
	upReq.Header.Set("Authorization", "Bearer "+token)
	upReq.Header.Set("Content-Type", "application/json")
	upReq.Header.Set("Accept", "text/event-stream")
	resp, err := p.client.Do(upReq)
	if err != nil {
		writeProxyError(w, http.StatusBadGateway, err.Error())
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		writeProxyError(w, resp.StatusCode, string(data))
		return fmt.Errorf("%s upstream status %s", spec.Name, resp.Status)
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return streamChatAsResponses(w, resp.Body, req.Model)
}

func responseMessages(req responseRequest) []chatMessage {
	var messages []chatMessage
	if strings.TrimSpace(req.Instructions) != "" {
		messages = append(messages, chatMessage{Role: "system", Content: req.Instructions})
	}
	var items []responseInputItem
	if len(req.Input) > 0 && req.Input[0] == '[' {
		_ = json.Unmarshal(req.Input, &items)
	}
	if len(items) == 0 {
		var text string
		if err := json.Unmarshal(req.Input, &text); err == nil && strings.TrimSpace(text) != "" {
			messages = append(messages, chatMessage{Role: "user", Content: text})
		}
		return messages
	}
	for _, item := range items {
		switch item.Type {
		case "message":
			role := item.Role
			if role == "developer" {
				role = "system"
			}
			if role == "" {
				role = "user"
			}
			messages = append(messages, chatMessage{Role: role, Content: contentText(item.Content)})
		case "function_call":
			callID := firstNonEmpty(item.CallID, item.ID)
			messages = append(messages, chatMessage{Role: "assistant", ToolCalls: []chatToolCall{{
				ID:       callID,
				Type:     "function",
				Function: chatCallFunction{Name: item.Name, Arguments: firstNonEmpty(item.Arguments, "{}")},
			}}})
		case "function_call_output":
			messages = append(messages, chatMessage{Role: "tool", ToolCallID: firstNonEmpty(item.CallID, item.ID), Content: item.Output})
		}
	}
	return normalizeChatMessages(messages)
}

type responseInputItem struct {
	Type      string          `json:"type"`
	ID        string          `json:"id,omitempty"`
	Role      string          `json:"role,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments string          `json:"arguments,omitempty"`
	Output    string          `json:"output,omitempty"`
}

func validateFunctionOutputSize(raw json.RawMessage) error {
	if len(raw) == 0 || raw[0] != '[' {
		return nil
	}
	var items []responseInputItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	for _, item := range items {
		if item.Type == "function_call_output" && len(item.Output) > maxCompatFunctionOutputBytes {
			return fmt.Errorf("function call output too large; maximum is %d bytes", maxCompatFunctionOutputBytes)
		}
	}
	return nil
}

func contentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		var out []string
		for _, p := range parts {
			if p.Text != "" {
				out = append(out, p.Text)
			}
		}
		return strings.Join(out, "\n")
	}
	return string(raw)
}

func responseTools(tools []responseTool) []chatTool {
	out := make([]chatTool, 0, len(tools))
	for _, tool := range tools {
		if tool.Type != "function" || tool.Name == "" {
			continue
		}
		out = append(out, chatTool{
			Type:     "function",
			Function: chatFunction{Name: tool.Name, Description: tool.Description, Parameters: tool.Parameters},
		})
	}
	return out
}

func normalizeChatMessages(messages []chatMessage) []chatMessage {
	var systemParts []string
	var out []chatMessage
	for _, msg := range messages {
		if msg.Role == "system" && msg.Content != "" && len(msg.ToolCalls) == 0 {
			systemParts = append(systemParts, msg.Content)
			continue
		}
		out = append(out, msg)
	}
	if len(systemParts) == 0 {
		return out
	}
	system := chatMessage{Role: "system", Content: strings.Join(systemParts, "\n\n")}
	return append([]chatMessage{system}, out...)
}

type chatStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content,omitempty"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id,omitempty"`
				Type     string `json:"type,omitempty"`
				Function struct {
					Name      string `json:"name,omitempty"`
					Arguments string `json:"arguments,omitempty"`
				} `json:"function,omitempty"`
			} `json:"tool_calls,omitempty"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens,omitempty"`
		CompletionTokens int `json:"completion_tokens,omitempty"`
		TotalTokens      int `json:"total_tokens,omitempty"`
	} `json:"usage,omitempty"`
}

type pendingToolCall struct {
	ID        string
	ItemID    string
	Name      string
	Arguments strings.Builder
	Started   bool
}

func streamChatAsResponses(w http.ResponseWriter, body io.Reader, model string) error {
	flusher, _ := w.(http.Flusher)
	flush := func() {
		if flusher != nil {
			flusher.Flush()
		}
	}
	respID := fmt.Sprintf("resp_%d", time.Now().UnixNano())
	msgID := "msg_" + respID
	createdAt := time.Now().Unix()
	var text strings.Builder
	outputIndex := 0
	textStarted := false
	toolCalls := map[int]*pendingToolCall{}
	var usage any
	writeSSE(w, "response.created", map[string]any{"type": "response.created", "response": responseEnvelope(respID, model, "in_progress", nil, nil, createdAt)})
	flush()
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk chatStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Usage != nil {
			usage = map[string]any{"input_tokens": chunk.Usage.PromptTokens, "output_tokens": chunk.Usage.CompletionTokens, "total_tokens": chunk.Usage.TotalTokens}
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				if !textStarted {
					writeSSE(w, "response.output_item.added", map[string]any{"type": "response.output_item.added", "output_index": outputIndex, "item": messageItem(msgID, "in_progress", "")})
					writeSSE(w, "response.content_part.added", map[string]any{"type": "response.content_part.added", "item_id": msgID, "output_index": outputIndex, "content_index": 0, "part": map[string]any{"type": "output_text", "text": ""}})
					textStarted = true
				}
				text.WriteString(choice.Delta.Content)
				writeSSE(w, "response.output_text.delta", map[string]any{"type": "response.output_text.delta", "item_id": msgID, "output_index": outputIndex, "content_index": 0, "delta": choice.Delta.Content})
				flush()
			}
			for _, tc := range choice.Delta.ToolCalls {
				call := toolCalls[tc.Index]
				if call == nil {
					call = &pendingToolCall{ID: tc.ID, ItemID: firstNonEmpty(tc.ID, fmt.Sprintf("call_%d_%d", time.Now().UnixNano(), tc.Index)), Name: tc.Function.Name}
					toolCalls[tc.Index] = call
				}
				if tc.ID != "" {
					call.ID = tc.ID
					call.ItemID = tc.ID
				}
				if tc.Function.Name != "" {
					call.Name = tc.Function.Name
				}
				if !call.Started && call.Name != "" {
					writeSSE(w, "response.output_item.added", map[string]any{"type": "response.output_item.added", "output_index": outputIndex, "item": functionCallItem(call, "in_progress")})
					call.Started = true
				}
				if tc.Function.Arguments != "" {
					call.Arguments.WriteString(tc.Function.Arguments)
					writeSSE(w, "response.function_call_arguments.delta", map[string]any{"type": "response.function_call_arguments.delta", "item_id": call.ItemID, "output_index": outputIndex, "delta": tc.Function.Arguments})
					flush()
				}
			}
		}
	}
	output := []any{}
	if textStarted {
		finalText := text.String()
		writeSSE(w, "response.output_text.done", map[string]any{"type": "response.output_text.done", "item_id": msgID, "output_index": 0, "content_index": 0, "text": finalText})
		writeSSE(w, "response.content_part.done", map[string]any{"type": "response.content_part.done", "item_id": msgID, "output_index": 0, "content_index": 0, "part": map[string]any{"type": "output_text", "text": finalText}})
		msg := messageItem(msgID, "completed", finalText)
		writeSSE(w, "response.output_item.done", map[string]any{"type": "response.output_item.done", "output_index": 0, "item": msg})
		output = append(output, msg)
		outputIndex++
	}
	for _, call := range orderedToolCalls(toolCalls) {
		if !call.Started {
			writeSSE(w, "response.output_item.added", map[string]any{"type": "response.output_item.added", "output_index": outputIndex, "item": functionCallItem(call, "in_progress")})
		}
		writeSSE(w, "response.function_call_arguments.done", map[string]any{"type": "response.function_call_arguments.done", "item_id": call.ItemID, "output_index": outputIndex, "arguments": call.Arguments.String()})
		item := functionCallItem(call, "completed")
		writeSSE(w, "response.output_item.done", map[string]any{"type": "response.output_item.done", "output_index": outputIndex, "item": item})
		output = append(output, item)
		outputIndex++
	}
	writeSSE(w, "response.completed", map[string]any{"type": "response.completed", "response": responseEnvelope(respID, model, "completed", output, usage, createdAt)})
	_, _ = io.WriteString(w, "data: [DONE]\n\n")
	flush()
	return scanner.Err()
}

func orderedToolCalls(calls map[int]*pendingToolCall) []*pendingToolCall {
	out := make([]*pendingToolCall, 0, len(calls))
	for i := 0; i < len(calls); i++ {
		if call := calls[i]; call != nil {
			out = append(out, call)
		}
	}
	return out
}

func messageItem(id, status, text string) map[string]any {
	content := []any{}
	if text != "" || status == "completed" {
		content = append(content, map[string]any{"type": "output_text", "text": text})
	}
	return map[string]any{"id": id, "type": "message", "status": status, "role": "assistant", "content": content}
}

func functionCallItem(call *pendingToolCall, status string) map[string]any {
	return map[string]any{
		"id":        call.ItemID,
		"type":      "function_call",
		"status":    status,
		"call_id":   firstNonEmpty(call.ID, call.ItemID),
		"name":      call.Name,
		"arguments": call.Arguments.String(),
	}
}

func responseEnvelope(id, model, status string, output []any, usage any, createdAt int64) map[string]any {
	if output == nil {
		output = []any{}
	}
	resp := map[string]any{"id": id, "object": "response", "created_at": createdAt, "status": status, "model": model, "output": output}
	if usage != nil {
		resp["usage"] = usage
	}
	return resp
}

func writeSSE(w io.Writer, event string, data any) {
	b, _ := json.Marshal(data)
	_, _ = fmt.Fprintf(w, "event: %s\n", event)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
}

func writeProxyError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": strings.TrimSpace(message)}})
}

func firstNonEmpty(items ...string) string {
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			return item
		}
	}
	return ""
}
