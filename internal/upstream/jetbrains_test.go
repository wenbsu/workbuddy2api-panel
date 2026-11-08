package upstream

import (
	"compress/gzip"
	"encoding/json"
	"github.com/wenbsu/workbuddy2api-panel/internal/auth"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJetBrainsProfileIsolation(t *testing.T) {
	c := &Client{ChatAdapter: &JetBrainsAdapter{IDEType: "JetBrains", Platform: "JetBrainsIntelliJIdea", PlatformVersion: "2026.2", ProductName: "CodeBuddy", ProductVersion: "4.12.42888899", DeploymentType: "SaaS"}}
	a := &auth.Auth{UID: "test-user", EnterpriseID: "test-enterprise"}
	req := httptest.NewRequest("POST", "https://example.test/v2/chat/completions", nil)
	c.ChatHeaders(req, a, "192.0.2.1", ChatMeta{ConversationID: "conversation"})
	for k, want := range map[string]string{"X-IDE-Type": "JetBrains", "X-IDE-Name": "JetBrainsIntelliJIdea", "X-IDE-Version": "2026.2", "X-Product-Version": "4.12.42888899", "X-Tenant-Id": "test-enterprise", "Accept": "*/*", "User-Agent": "JetBrainsIntelliJIdea/2026.2 CodeBuddy/4.12.42888899", "X-Conversation-ID": "conversation"} {
		if got := req.Header.Get(k); got != want {
			t.Errorf("%s=%q, want %q", k, got, want)
		}
	}
	for _, k := range []string{"X-Machine-ID", "X-Refresh-Token", "X-Forwarded-For", "Origin", "X-No-Authorization"} {
		if req.Header.Get(k) != "" {
			t.Errorf("unexpected inherited header %s", k)
		}
	}
	if req.Header.Get("X-Request-ID") != req.Header.Get("X-Conversation-Request-ID") {
		t.Error("request ID must match conversation request ID")
	}
	if len(req.Header.Get("X-Request-Trace-Id")) != 36 {
		t.Error("missing independent HTTP trace UUID")
	}
	if req.Header.Get("X-Request-ID") == "" {
		t.Error("missing request id")
	}
}

func TestJetBrainsCapturedTransport(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) {
		for k, want := range map[string]string{"Content-Encoding": "gzip", "Accept": "*/*", "Content-Type": "application/json;charset=UTF-8", "X-Model-ID": "auto", "X-Agent-Intent": "craft", "X-Agent-Type": "main"} {
			if got := r.Header.Get(k); got != want {
				t.Errorf("%s=%q want %q", k, got, want)
			}
		}
		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer zr.Close()
		var body map[string]json.RawMessage
		if err := json.NewDecoder(zr).Decode(&body); err != nil {
			t.Fatal(err)
		}
		for _, k := range []string{"previous_response_id", "reasoning_summary", "messages", "tools"} {
			if _, ok := body[k]; !ok {
				t.Errorf("lost %s", k)
			}
		}
		if r.Header.Get("X-Root-Request-ID") != "" {
			t.Error("unexpected root request header")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: [DONE]\n\n"))}, nil
	})
	c.ChatAdapter = &JetBrainsAdapter{AgentIntent: "craft", AgentType: "main"}
	rc, status, _, err := c.ChatStream(&auth.Auth{UID: "test-user"}, []byte(`{"model":"auto","previous_response_id":"test-response","reasoning_summary":"auto","messages":[{"role":"user","content":"OK"}],"tools":[]}`), "", ChatMeta{})
	if err != nil || status != 200 {
		t.Fatalf("status=%d err=%v", status, err)
	}
	rc.Close()
}
