package compat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestForwardChatStreamAsResponses(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected upstream path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"READY"}}]}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2},"choices":[{"delta":{}}]}` + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	req := responseRequest{
		Model:        "test-model",
		Instructions: "system",
		Input:        []byte(`[{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}]`),
		Stream:       true,
	}
	p := &Proxy{client: upstream.Client()}
	rr := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/minimax/v1/responses", nil)
	err := p.forwardChatStream(rr, httpReq, providerSpec{Name: "test", BaseURL: upstream.URL + "/v1", SupportsStreamUsage: true}, "token", req)
	if err != nil {
		t.Fatalf("forwardChatStream error: %v", err)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "response.output_text.delta") || !strings.Contains(body, "READY") || !strings.Contains(body, "response.completed") {
		t.Fatalf("unexpected responses stream: %s", body)
	}
}

func TestNormalizeChatMessagesMergesSystem(t *testing.T) {
	got := normalizeChatMessages([]chatMessage{
		{Role: "system", Content: "a"},
		{Role: "user", Content: "u"},
		{Role: "system", Content: "b"},
	})
	if len(got) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got))
	}
	if got[0].Role != "system" || got[0].Content != "a\n\nb" {
		t.Fatalf("unexpected merged system message: %#v", got[0])
	}
	if got[1].Role != "user" {
		t.Fatalf("unexpected second message: %#v", got[1])
	}
}

func TestStartSetsReadHeaderTimeout(t *testing.T) {
	p, err := Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer p.server.Shutdown(context.Background())
	if p.server.ReadHeaderTimeout <= 0 {
		t.Fatal("ReadHeaderTimeout is not configured")
	}
}
