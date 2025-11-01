package service

import (
	"fmt"
	"os"
	"time"

	"github.com/hellomyzn/yt-notifier/src/internal/model"
	"github.com/hellomyzn/yt-notifier/src/internal/notifier"
	"github.com/hellomyzn/yt-notifier/src/internal/repository"
)

type NotifyService interface {
	Notify(category string, v model.VideoDTO) error
}

type notifyService struct {
	notifiedRepo  repository.NotifiedRepository
	categoryToEnv map[string]string
	postSleep     time.Duration
}

func NewNotifyService(notified repository.NotifiedRepository, categoryToEnv map[string]string, postSleep time.Duration) NotifyService {
	return &notifyService{
		notifiedRepo:  notified,
		categoryToEnv: categoryToEnv,
		postSleep:     postSleep,
	}
}

func (s *notifyService) Notify(category string, v model.VideoDTO) error {
	envName, ok := s.categoryToEnv[category]
	if !ok || envName == "" {
		return fmt.Errorf("webhook env not mapped for category=%s", category)
	}
	webhook := os.Getenv(envName)
	if webhook == "" {
		return fmt.Errorf("webhook env is empty: %s", envName)
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
