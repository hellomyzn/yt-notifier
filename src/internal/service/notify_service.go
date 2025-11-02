package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/hellomyzn/yt-notifier/internal/model"
	"github.com/hellomyzn/yt-notifier/internal/notifier"
	"github.com/hellomyzn/yt-notifier/internal/repository"
)

type NotifyService interface {
	Notify(category string, v model.VideoDTO) error
}

type notifyService struct {
	notifiedRepo      repository.NotifiedRepository
	categoryToWebhook map[string]string
	postSleep         time.Duration
}

func NewNotifyService(notified repository.NotifiedRepository, categoryToWebhook map[string]string, postSleep time.Duration) NotifyService {
	return &notifyService{
		notifiedRepo:      notified,
		categoryToWebhook: categoryToWebhook,
		postSleep:         postSleep,
	}
}

func (s *notifyService) Notify(category string, v model.VideoDTO) error {
	webhook, ok := s.categoryToWebhook[strings.ToLower(category)]
	if webhook == "" {
		if ok {
			return fmt.Errorf("webhook is empty for category=%s", category)
		}
		return fmt.Errorf("webhook not mapped for category=%s", category)
	}

	cli := &notifier.DiscordNotifier{Webhook: webhook}
	thumb := fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", v.VideoID)
	content := notifier.NotificationContent{
		Title:    v.Title,
		Message:  fmt.Sprintf("%s | %s", v.ChannelName, v.PublishedAt.Format(time.RFC3339)),
		URL:      v.Link,
		ThumbURL: thumb,
	}
	if err := cli.Send(content); err != nil {
		return err
	}
	_ = s.notifiedRepo.Append(v.VideoID, v.ChannelID, v.PublishedAt, time.Now())
	time.Sleep(s.postSleep)
	return nil
}
