package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type NotificationContent struct {
	Title    string
	Message  string
	URL      string
	ThumbURL string
}

type Notifier interface {
	Send(NotificationContent) error
}

type DiscordNotifier struct {
	Webhook string
	Client  *http.Client
}

func (n *DiscordNotifier) Send(c NotificationContent) error {
	embed := map[string]any{
		"title":       c.Title,
		"description": c.Message,
		"url":         c.URL,
	}
	if c.ThumbURL != "" {
		embed["image"] = map[string]string{"url": c.ThumbURL}
	}

	payload := map[string]any{
		"embeds": []map[string]any{embed},
	}
	b, _ := json.Marshal(payload)
	cli := n.Client
	if cli == nil {
		cli = http.DefaultClient
	}
	resp, err := cli.Post(n.Webhook, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		retryAfter := parseDiscordRetryAfter(resp.Header.Get("Retry-After"), snippet)
		message := strings.TrimSpace(string(snippet))
		if message == "" {
			if derr := parseDiscordErrorMessage(snippet); derr != "" {
				message = derr
			}
		}
		return &DiscordHTTPError{
			StatusCode: resp.StatusCode,
			RetryAfter: retryAfter,
			Message:    message,
		}
	}
	return nil
}

type DiscordHTTPError struct {
	StatusCode int
	RetryAfter time.Duration
	Message    string
}

func (e *DiscordHTTPError) Error() string {
	if e == nil {
		return ""
	}
	msg := fmt.Sprintf("discord webhook status %d", e.StatusCode)
	if e.Message != "" {
		msg = fmt.Sprintf("%s: %s", msg, e.Message)
	}
	if e.RetryAfter > 0 {
		msg = fmt.Sprintf("%s (retry after %s)", msg, e.RetryAfter)
	}
	return msg
}

func parseDiscordRetryAfter(raw string, body []byte) time.Duration {
	if d := parseRetryAfterHeader(raw); d > 0 {
		return d
	}
	if d := parseRetryAfterBody(body); d > 0 {
		return d
	}
	return 0
}

func parseRetryAfterHeader(raw string) time.Duration {
	if raw == "" {
		return 0
	}
	if secs, err := strconv.ParseFloat(raw, 64); err == nil {
		if secs <= 0 {
			return 0
		}
		return time.Duration(secs * float64(time.Second))
	}
	if t, err := http.ParseTime(raw); err == nil {
		return time.Until(t)
	}
	return 0
}

func parseRetryAfterBody(body []byte) time.Duration {
	if len(body) == 0 {
		return 0
	}
	var payload struct {
		RetryAfter float64 `json:"retry_after"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0
	}
	if payload.RetryAfter <= 0 {
		return 0
	}
	return time.Duration(payload.RetryAfter * float64(time.Second))
}

func parseDiscordErrorMessage(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var payload struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Message)
}
