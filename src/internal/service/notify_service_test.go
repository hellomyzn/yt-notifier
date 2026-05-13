package service

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hellomyzn/yt-notifier/internal/model"
	"github.com/hellomyzn/yt-notifier/internal/notifier"
)

// ---- mock Notifier（webhookDispatcher 単体テスト用）----

type mockNotifier struct {
	sendErr  error
	calls    int32
	sendFunc func() error
}

func (m *mockNotifier) Send(_ notifier.NotificationContent) error {
	atomic.AddInt32(&m.calls, 1)
	if m.sendFunc != nil {
		return m.sendFunc()
	}
	return m.sendErr
}

// ---- webhookDispatcher ----

func newTestDispatcher(n notifier.Notifier, minInterval time.Duration) *webhookDispatcher {
	return &webhookDispatcher{
		notifier:    n,
		minInterval: minInterval,
		maxRetries:  5,
		baseBackoff: time.Millisecond, // テストでは最小バックオフ
	}
}

func TestWebhookDispatcher_Send_Success(t *testing.T) {
	n := &mockNotifier{}
	d := newTestDispatcher(n, 0)
	_, err := d.send(notifier.NotificationContent{Title: "test"})
	if err != nil {
		t.Errorf("send() error: %v", err)
	}
	if n.calls != 1 {
		t.Errorf("Send() calls = %d; want 1", n.calls)
	}
}

func TestWebhookDispatcher_Send_429RetryUntilSuccess(t *testing.T) {
	attempt := int32(0)
	n := &mockNotifier{
		sendFunc: func() error {
			if atomic.AddInt32(&attempt, 1) < 3 {
				return &notifier.DiscordHTTPError{
					StatusCode: http.StatusTooManyRequests,
					RetryAfter: time.Millisecond,
				}
			}
			return nil
		},
	}
	d := newTestDispatcher(n, 0)
	_, err := d.send(notifier.NotificationContent{})
	if err != nil {
		t.Errorf("send() error after retries: %v", err)
	}
	if attempt < 3 {
		t.Errorf("attempt = %d; want >= 3", attempt)
	}
}

func TestWebhookDispatcher_Send_NonRateLimitErrorRetriesWithLimit(t *testing.T) {
	n := &mockNotifier{sendErr: &notifier.DiscordHTTPError{StatusCode: http.StatusInternalServerError}}
	d := newTestDispatcher(n, 0)
	_, err := d.send(notifier.NotificationContent{})
	if err == nil {
		t.Error("want error after exhausting retries, got nil")
	}
	if n.calls < 1 {
		t.Error("Send() never called")
	}
}

// ---- notifyService.Notify ----

func newTestNotifyService(webhookURL string, postSleep time.Duration) *notifyService {
	return &notifyService{
		notifiedRepo:      &mockNotifiedRepo{seen: map[string]bool{}},
		categoryToWebhook: map[string]string{"test_cat": webhookURL},
		postSleep:         postSleep,
		dispatchers:       map[string]*webhookDispatcher{},
	}
}

func TestNotifyService_Notify_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := newTestNotifyService(srv.URL, 0)
	video := model.VideoDTO{VideoID: "v1", Title: "T", Link: "http://example.com", PublishedAt: time.Now()}

	err := svc.Notify("test_cat", video)
	if err != nil {
		t.Errorf("Notify() error: %v", err)
	}

	// 通知済みに記録されている
	repo := svc.notifiedRepo.(*mockNotifiedRepo)
	if len(repo.appended) != 1 || repo.appended[0] != "v1" {
		t.Errorf("appended = %v; want [v1]", repo.appended)
	}
}

func TestNotifyService_Notify_UnknownCategory(t *testing.T) {
	svc := newTestNotifyService("http://example.com", 0)
	err := svc.Notify("unknown_cat", model.VideoDTO{VideoID: "v1"})
	if err == nil {
		t.Error("want error for unknown category, got nil")
	}
}

func TestNotifyService_Notify_EmptyWebhook(t *testing.T) {
	svc := &notifyService{
		notifiedRepo:      &mockNotifiedRepo{seen: map[string]bool{}},
		categoryToWebhook: map[string]string{"test_cat": ""},
		postSleep:         0,
		dispatchers:       map[string]*webhookDispatcher{},
	}
	err := svc.Notify("test_cat", model.VideoDTO{VideoID: "v1"})
	if err == nil {
		t.Error("want error for empty webhook, got nil")
	}
}

func TestNotifyService_Stats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := newTestNotifyService(srv.URL, 0)
	video := model.VideoDTO{VideoID: "v1", PublishedAt: time.Now()}
	svc.Notify("test_cat", video)

	stats := svc.Stats()
	if stats.Sent != 1 {
		t.Errorf("Stats().Sent = %d; want 1", stats.Sent)
	}
	if stats.Failed != 0 {
		t.Errorf("Stats().Failed = %d; want 0", stats.Failed)
	}
}
