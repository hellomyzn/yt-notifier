package notifier

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---- parseRetryAfterHeader ----

func TestParseRetryAfterHeader(t *testing.T) {
	tests := []struct {
		raw  string
		want time.Duration
	}{
		{"60", 60 * time.Second},
		{"1", time.Second},
		{"0", 0},
		{"-1", 0},
		{"", 0},
		{"abc", 0},
		{"1.5", 1500 * time.Millisecond},
	}
	for _, tt := range tests {
		got := parseRetryAfterHeader(tt.raw)
		if got != tt.want {
			t.Errorf("parseRetryAfterHeader(%q) = %v; want %v", tt.raw, got, tt.want)
		}
	}
}

// ---- parseRetryAfterBody ----

func TestParseRetryAfterBody(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want time.Duration
	}{
		{"retry_after フィールドあり", []byte(`{"retry_after":2.5}`), 2500 * time.Millisecond},
		{"retry_after = 0", []byte(`{"retry_after":0}`), 0},
		{"フィールドなし", []byte(`{"message":"rate limited"}`), 0},
		{"空ボディ", []byte{}, 0},
		{"不正JSON", []byte(`{broken`), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRetryAfterBody(tt.body)
			if got != tt.want {
				t.Errorf("parseRetryAfterBody() = %v; want %v", got, tt.want)
			}
		})
	}
}

// ---- DiscordNotifier.Send ----

func TestDiscordNotifier_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s; want POST", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	n := &DiscordNotifier{Webhook: srv.URL, Client: srv.Client()}
	err := n.Send(NotificationContent{Title: "Test", Message: "msg", URL: "http://example.com"})
	if err != nil {
		t.Errorf("Send() error: %v", err)
	}
}

func TestDiscordNotifier_Send_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	n := &DiscordNotifier{Webhook: srv.URL, Client: srv.Client()}
	err := n.Send(NotificationContent{Title: "Test"})
	if err == nil {
		t.Error("want error for 5xx, got nil")
	}
}

func TestDiscordNotifier_Send_429WithRetryAfterHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	n := &DiscordNotifier{Webhook: srv.URL, Client: srv.Client()}
	err := n.Send(NotificationContent{Title: "Test"})

	var httpErr *DiscordHTTPError
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if e, ok := err.(*DiscordHTTPError); ok {
		httpErr = e
	} else {
		t.Fatalf("want *DiscordHTTPError, got %T", err)
	}
	if httpErr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("StatusCode = %d; want 429", httpErr.StatusCode)
	}
	if httpErr.RetryAfter != 30*time.Second {
		t.Errorf("RetryAfter = %v; want 30s", httpErr.RetryAfter)
	}
}

func TestDiscordNotifier_Send_429WithBodyRetryAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"retry_after":5.0,"message":"rate limited"}`))
	}))
	defer srv.Close()

	n := &DiscordNotifier{Webhook: srv.URL, Client: srv.Client()}
	err := n.Send(NotificationContent{Title: "Test"})

	e, ok := err.(*DiscordHTTPError)
	if !ok {
		t.Fatalf("want *DiscordHTTPError, got %T", err)
	}
	if e.RetryAfter != 5*time.Second {
		t.Errorf("RetryAfter = %v; want 5s", e.RetryAfter)
	}
}

func TestDiscordNotifier_Send_EmptyURLOmitted(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	n := &DiscordNotifier{Webhook: srv.URL, Client: srv.Client()}
	_ = n.Send(NotificationContent{Title: "T", Message: "M"})

	var payload struct {
		Embeds []map[string]any `json:"embeds"`
	}
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	if len(payload.Embeds) == 0 {
		t.Fatal("no embeds in payload")
	}
	if _, ok := payload.Embeds[0]["url"]; ok {
		t.Errorf("embed should not contain 'url' when URL is empty, got: %s", gotBody)
	}
}

func TestDiscordNotifier_Send_WithThumbnail(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	n := &DiscordNotifier{Webhook: srv.URL, Client: srv.Client()}
	n.Send(NotificationContent{Title: "T", Message: "M", URL: "http://x.com", ThumbURL: "http://img.com/thumb.jpg"})

	if gotBody == "" {
		t.Error("want non-empty request body")
	}
}
