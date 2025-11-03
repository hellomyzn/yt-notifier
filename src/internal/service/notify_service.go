package service

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hellomyzn/yt-notifier/internal/model"
	"github.com/hellomyzn/yt-notifier/internal/notifier"
	"github.com/hellomyzn/yt-notifier/internal/repository"
)

type NotifyService interface {
	Notify(category string, v model.VideoDTO) error
	Stats() NotificationStats
}

type NotificationStats struct {
	Sent            int
	Failed          int
	RetriedMessages int
	RetryAttempts   int
}

type notifyService struct {
	notifiedRepo      repository.NotifiedRepository
	categoryToWebhook map[string]string
	postSleep         time.Duration

	mu          sync.Mutex
	dispatchers map[string]*webhookDispatcher
	stats       NotificationStats
}

func NewNotifyService(notified repository.NotifiedRepository, categoryToWebhook map[string]string, postSleep time.Duration) NotifyService {
	return &notifyService{
		notifiedRepo:      notified,
		categoryToWebhook: categoryToWebhook,
		postSleep:         postSleep,
		dispatchers:       map[string]*webhookDispatcher{},
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

	dispatcher := s.dispatcherFor(webhook)
	thumb := fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", v.VideoID)
	content := notifier.NotificationContent{
		Title:    v.Title,
		Message:  fmt.Sprintf("%s | %s", v.ChannelName, v.PublishedAt.Format(time.RFC3339)),
		URL:      v.Link,
		ThumbURL: thumb,
	}

	retries, err := dispatcher.send(content)
	if err != nil {
		s.recordFailure()
		return err
	}

	s.recordSuccess(retries)
	_ = s.notifiedRepo.Append(v.VideoID, v.ChannelID, v.PublishedAt, time.Now())
	return nil
}

func (s *notifyService) Stats() NotificationStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats
}

func (s *notifyService) dispatcherFor(webhook string) *webhookDispatcher {
	s.mu.Lock()
	defer s.mu.Unlock()
	dispatcher, ok := s.dispatchers[webhook]
	if ok {
		return dispatcher
	}
	minInterval := time.Second
	if s.postSleep > minInterval {
		minInterval = s.postSleep
	}
	dispatcher = &webhookDispatcher{
		notifier:    &notifier.DiscordNotifier{Webhook: webhook},
		minInterval: minInterval,
		maxRetries:  5,
		baseBackoff: 2 * time.Second,
	}
	s.dispatchers[webhook] = dispatcher
	return dispatcher
}

func (s *notifyService) recordSuccess(retries int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.Sent++
	if retries > 0 {
		s.stats.RetriedMessages++
		s.stats.RetryAttempts += retries
	}
}

func (s *notifyService) recordFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.Failed++
}

type webhookDispatcher struct {
	notifier    notifier.Notifier
	minInterval time.Duration
	maxRetries  int
	baseBackoff time.Duration

	mu            sync.Mutex
	nextAvailable time.Time
}

func (d *webhookDispatcher) send(content notifier.NotificationContent) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if wait := time.Until(d.nextAvailable); wait > 0 {
		time.Sleep(wait)
	}

	backoff := d.baseBackoff
	if backoff <= 0 {
		backoff = time.Second
	}
	var lastErr error
	retries := 0
	for attempt := 0; attempt < d.maxRetries; attempt++ {
		lastErr = d.notifier.Send(content)
		if lastErr == nil {
			d.nextAvailable = time.Now().Add(d.minInterval)
			return retries, nil
		}

		retries++

		if httpErr := asDiscordHTTPError(lastErr); httpErr != nil {
			wait := httpErr.RetryAfter
			if wait <= 0 {
				wait = backoff
			}
			time.Sleep(wait)
			if wait == backoff {
				backoff = minDuration(backoff*2, 30*time.Second)
			}
			continue
		}

		time.Sleep(backoff)
		backoff = minDuration(backoff*2, 30*time.Second)
	}
	return retries, fmt.Errorf("failed to send discord notification after %d attempts: %w", d.maxRetries, lastErr)
}

func asDiscordHTTPError(err error) *notifier.DiscordHTTPError {
	var httpErr *notifier.DiscordHTTPError
	if errors.As(err, &httpErr) {
		return httpErr
	}
	return nil
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
