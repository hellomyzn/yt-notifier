package controller

import (
	"errors"
	"testing"
	"time"

	"github.com/hellomyzn/yt-notifier/internal/model"
	"github.com/hellomyzn/yt-notifier/internal/service"
)

// ---- mocks ----

type mockChannelRepo struct {
	channels []model.ChannelDTO
	err      error
}

func (m *mockChannelRepo) ListEnabled() ([]model.ChannelDTO, error) {
	return m.channels, m.err
}

type mockFeedService struct {
	videos map[string][]model.VideoDTO // channelID → videos
	err    error
	calls  int
}

func (m *mockFeedService) ListNewVideos(ch model.ChannelDTO) ([]model.VideoDTO, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.videos[ch.ChannelID], nil
}

func (m *mockFeedService) Stats() service.FeedStats { return service.FeedStats{} }

type mockNotifyService struct {
	notified []string
	err      error
}

func (m *mockNotifyService) Notify(category string, v model.VideoDTO) error {
	if m.err != nil {
		return m.err
	}
	m.notified = append(m.notified, v.VideoID)
	return nil
}

func (m *mockNotifyService) Stats() service.NotificationStats { return service.NotificationStats{} }

// ---- helpers ----

func makeChannel(id string) model.ChannelDTO {
	return model.ChannelDTO{ChannelID: id, Category: "test"}
}

func makeVideo(id string) model.VideoDTO {
	return model.VideoDTO{VideoID: id, PublishedAt: time.Now()}
}

// ---- RunOnce ----

func TestJobController_RunOnce_NotifiesNewVideos(t *testing.T) {
	chRepo := &mockChannelRepo{
		channels: []model.ChannelDTO{makeChannel("UC1"), makeChannel("UC2")},
	}
	feedSvc := &mockFeedService{
		videos: map[string][]model.VideoDTO{
			"UC1": {makeVideo("v1"), makeVideo("v2")},
			"UC2": {makeVideo("v3")},
		},
	}
	notifySvc := &mockNotifyService{}

	c := NewJobController(chRepo, feedSvc, notifySvc, 0, nil, "")
	if err := c.RunOnce(); err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}

	if feedSvc.calls != 2 {
		t.Errorf("FeedService calls = %d; want 2", feedSvc.calls)
	}
	if len(notifySvc.notified) != 3 {
		t.Errorf("notified videos = %v; want 3", notifySvc.notified)
	}
}

func TestJobController_RunOnce_SkipsChannelOnFetchError(t *testing.T) {
	chRepo := &mockChannelRepo{
		channels: []model.ChannelDTO{makeChannel("UC1"), makeChannel("UC2")},
	}
	feedSvc := &mockFeedService{err: errors.New("fetch failed")}
	notifySvc := &mockNotifyService{}

	c := NewJobController(chRepo, feedSvc, notifySvc, 0, nil, "")
	if err := c.RunOnce(); err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}

	// フェッチエラーでも RunOnce 自体はエラーを返さない
	if len(notifySvc.notified) != 0 {
		t.Errorf("notified = %v; want empty (all fetches failed)", notifySvc.notified)
	}
}

func TestJobController_RunOnce_NoNewVideos(t *testing.T) {
	chRepo := &mockChannelRepo{
		channels: []model.ChannelDTO{makeChannel("UC1")},
	}
	feedSvc := &mockFeedService{videos: map[string][]model.VideoDTO{"UC1": {}}}
	notifySvc := &mockNotifyService{}

	c := NewJobController(chRepo, feedSvc, notifySvc, 0, nil, "")
	if err := c.RunOnce(); err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}

	if len(notifySvc.notified) != 0 {
		t.Errorf("notified = %v; want empty", notifySvc.notified)
	}
}

func TestJobController_RunOnce_ChannelRepoError(t *testing.T) {
	chRepo := &mockChannelRepo{err: errors.New("csv read error")}
	feedSvc := &mockFeedService{}
	notifySvc := &mockNotifyService{}

	c := NewJobController(chRepo, feedSvc, notifySvc, 0, nil, "")
	err := c.RunOnce()
	if err == nil {
		t.Error("want error when channel repo fails, got nil")
	}
}

func TestJobController_RunOnce_NotifyErrorContinues(t *testing.T) {
	chRepo := &mockChannelRepo{
		channels: []model.ChannelDTO{makeChannel("UC1"), makeChannel("UC2")},
	}
	feedSvc := &mockFeedService{
		videos: map[string][]model.VideoDTO{
			"UC1": {makeVideo("v1")},
			"UC2": {makeVideo("v2")},
		},
	}
	notifySvc := &mockNotifyService{err: errors.New("webhook failed")}

	c := NewJobController(chRepo, feedSvc, notifySvc, 0, nil, "")
	// 通知エラーでも RunOnce 自体はエラーを返さない
	if err := c.RunOnce(); err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
}

// ---- fmtDuration ----

func TestFmtDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{45 * time.Second, "45s"},
		{60 * time.Second, "1m"},
		{90 * time.Second, "1m30s"},
		{3*time.Minute + 12*time.Second, "3m12s"},
		{time.Hour + 2*time.Minute, "1h2m"},
		{2*time.Hour + 30*time.Minute, "2h30m"},
	}
	for _, tt := range tests {
		got := fmtDuration(tt.d)
		if got != tt.want {
			t.Errorf("fmtDuration(%v) = %q; want %q", tt.d, got, tt.want)
		}
	}
}

// ---- aggregateStats ----

func TestAggregateStats(t *testing.T) {
	stats := []channelStat{
		{newCount: 3, notified: 3, failed: 0},
		{newCount: 2, notified: 1, failed: 1},
		{newCount: 0, notified: 0, failed: 0},
		{fetchErr: true},
	}
	totalNew, totalNotified, totalFailed, channelsWithNew, channelsWithError := aggregateStats(stats)
	if totalNew != 5 {
		t.Errorf("totalNew = %d; want 5", totalNew)
	}
	if totalNotified != 4 {
		t.Errorf("totalNotified = %d; want 4", totalNotified)
	}
	if totalFailed != 1 {
		t.Errorf("totalFailed = %d; want 1", totalFailed)
	}
	if channelsWithNew != 2 {
		t.Errorf("channelsWithNew = %d; want 2", channelsWithNew)
	}
	if channelsWithError != 1 {
		t.Errorf("channelsWithError = %d; want 1", channelsWithError)
	}
}
