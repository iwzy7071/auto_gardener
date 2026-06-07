package app

import (
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"auto_gardener/internal/codex"
)

func TestDingTalkSign(t *testing.T) {
	got := dingTalkSign("1700000000000", "secret")
	if got == "" {
		t.Fatal("empty sign")
	}
	if got != dingTalkSign("1700000000000", "secret") {
		t.Fatal("sign should be deterministic")
	}
	if got == dingTalkSign("1700000000001", "secret") {
		t.Fatal("different timestamp should change sign")
	}
}

func TestVerifyDingTalkIncomingSignature(t *testing.T) {
	t.Setenv("AUTO_GARDENER_DINGTALK_INCOMING_SECRET", "secret")
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	req := httptest.NewRequest("POST", "/api/dingtalk/robot", nil)
	req.Header.Set("timestamp", ts)
	req.Header.Set("sign", dingTalkSign(ts, "secret"))
	if err := verifyDingTalkIncomingSignature(req); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	req.Header.Set("sign", "bad")
	if err := verifyDingTalkIncomingSignature(req); err == nil {
		t.Fatal("bad signature accepted")
	}
}

func TestDingTalkSignedWebhook(t *testing.T) {
	url := dingTalkSignedWebhook("https://example.invalid/hook?access_token=abc", "secret")
	if url == "" || url == "https://example.invalid/hook?access_token=abc" {
		t.Fatalf("webhook not signed: %s", url)
	}
	if !(strings.Contains(url, "timestamp=") && strings.Contains(url, "sign=")) {
		t.Fatalf("signed webhook missing params: %s", url)
	}
}

func TestNoDingTalkSecretSkipsVerify(t *testing.T) {
	_ = os.Unsetenv("AUTO_GARDENER_DINGTALK_INCOMING_SECRET")
	_ = os.Unsetenv("AUTO_GARDENER_DINGTALK_APP_SECRET")
	req := httptest.NewRequest("POST", "/api/dingtalk/robot", nil)
	if err := verifyDingTalkIncomingSignature(req); err != nil {
		t.Fatalf("verification should be skipped without secret: %v", err)
	}
}

func TestDingTalkCommandRejectsOversizedContent(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, codex.MockRunner{}, store.DataDir(), "")
	server := NewServer(store, orch, "", events)

	reply := server.handleDingTalkCommand(dingTalkIncomingMessage{ConversationID: "c", SenderID: "s"}, strings.Repeat("长", maxDingTalkCommandRunes+1))
	if !strings.Contains(reply, "消息过长") {
		t.Fatalf("expected oversized DingTalk message to be rejected, got %q", reply)
	}
	if tasks := store.ListTasks(); len(tasks) != 0 {
		t.Fatalf("oversized DingTalk message should not create task: %+v", tasks)
	}
}
