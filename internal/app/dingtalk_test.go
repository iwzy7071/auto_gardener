package app

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
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

type failingRoundTripper struct{}

func (failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("transport failed with access_token=secret")
}

func TestDingTalkReplyErrorDoesNotExposeTransportDetails(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/dingtalk/robot", strings.NewReader(`{
		"conversationId":"c1",
		"senderId":"s1",
		"msgtype":"text",
		"sessionWebhook":"https://oapi.dingtalk.com/robot/send?access_token=secret",
		"text":{"content":"帮助"}
	}`))
	server := &Server{httpClient: &http.Client{Transport: failingRoundTripper{}}}
	rr := httptest.NewRecorder()

	server.handleDingTalkRobot(rr, req)

	body := rr.Body.String()
	if strings.Contains(body, "access_token=secret") || strings.Contains(body, "transport failed") {
		t.Fatalf("response exposed transport error details: %s", body)
	}
	if !strings.Contains(body, "发送钉钉回复失败") {
		t.Fatalf("response missing generic reply failure: %s", body)
	}
}
