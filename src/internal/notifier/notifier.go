package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

type DiscordNotifier struct{ Webhook string }

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
	resp, err := http.Post(n.Webhook, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook status %d", resp.StatusCode)
	}
	return nil
}
