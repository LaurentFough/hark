package openai_compatible

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hark/internal/ai"
)

func TestAskStreamsTextDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		if got := r.Header.Get("HTTP-Referer"); got != "" {
			t.Fatalf("unexpected referer header: %q", got)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"hel"},"finish_reason":null}]}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"lo"},"finish_reason":null}]}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}` + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := Client{
		Label:      "Test GW",
		APIKey:     "test-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	events, err := client.Ask(context.Background(), ai.Request{Prompt: "hello", Model: "test-model"})
	if err != nil {
		t.Fatalf("Ask returned error: %v", err)
	}

	var text string
	var done bool
	for event := range events {
		if event.Type == ai.EventDelta {
			text += event.Text
		}
		if event.Type == ai.EventDone {
			done = true
		}
	}

	if text != "hello" {
		t.Fatalf("unexpected streamed text: %q", text)
	}
	if !done {
		t.Fatal("expected done event")
	}
}

func TestAskSendsReasoningEffort(t *testing.T) {
	var body chatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := Client{APIKey: "test-key", BaseURL: server.URL, HTTPClient: server.Client()}
	events, err := client.Ask(context.Background(), ai.Request{Prompt: "hello", Model: "test-model", ReasoningEffort: "high"})
	if err != nil {
		t.Fatalf("Ask returned error: %v", err)
	}
	for range events {
	}

	if body.Reasoning == nil || body.Reasoning.Effort != "high" {
		t.Fatalf("unexpected reasoning config: %#v", body.Reasoning)
	}
}

func TestAskOmitsReasoningEffortForAuto(t *testing.T) {
	var body chatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := Client{APIKey: "test-key", BaseURL: server.URL, HTTPClient: server.Client()}
	events, err := client.Ask(context.Background(), ai.Request{Prompt: "hello", Model: "test-model", ReasoningEffort: "auto"})
	if err != nil {
		t.Fatalf("Ask returned error: %v", err)
	}
	for range events {
	}

	if body.Reasoning != nil {
		t.Fatalf("expected no reasoning config, got %#v", body.Reasoning)
	}
}

func TestAskRequiresAPIKeyForRemoteHosts(t *testing.T) {
	client := Client{BaseURL: "https://llm.example.com/v1"}
	_, err := client.Ask(context.Background(), ai.Request{Prompt: "hello", Model: "test-model"})
	if err == nil || !strings.Contains(err.Error(), "API key is not set") {
		t.Fatalf("Ask error = %v, want missing key error", err)
	}
}

func TestAskAllowsMissingAPIKeyOnLoopback(t *testing.T) {
	var authHeader string
	var authPresent bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, authPresent = r.Header["Authorization"]
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	// httptest binds to 127.0.0.1, so this exercises the loopback exemption.
	client := Client{BaseURL: server.URL, HTTPClient: server.Client()}
	events, err := client.Ask(context.Background(), ai.Request{Prompt: "hello", Model: "test-model"})
	if err != nil {
		t.Fatalf("Ask returned error: %v", err)
	}
	for range events {
	}

	if authPresent || authHeader != "" {
		t.Fatalf("expected no Authorization header without a key, got %q", authHeader)
	}
}

func TestIsLoopbackBaseURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"http://localhost:8080/v1", true},
		{"http://LOCALHOST/v1", true},
		{"http://127.0.0.1:1234/v1", true},
		{"http://127.1.2.3/v1", true},
		{"http://[::1]:11434/v1", true},
		{"https://llm.example.com/v1", false},
		{"http://192.168.1.20:8000/v1", false},
		{"http://0.0.0.0:8000/v1", false},
		{"http://localhost.example.com/v1", false},
		{"not a url", false},
		{"", false},
	}
	for _, test := range tests {
		if got := isLoopbackBaseURL(test.url); got != test.want {
			t.Errorf("isLoopbackBaseURL(%q) = %v, want %v", test.url, got, test.want)
		}
	}
}

func TestAskRequiresBaseURL(t *testing.T) {
	client := Client{Label: "Test GW", APIKey: "test-key"}
	_, err := client.Ask(context.Background(), ai.Request{Prompt: "hello", Model: "test-model"})
	if err == nil || !strings.Contains(err.Error(), "base URL is not set") {
		t.Fatalf("Ask error = %v, want missing base URL error", err)
	}
}
