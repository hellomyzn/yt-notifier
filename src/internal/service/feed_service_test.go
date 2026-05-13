package service

import (
	"errors"
	"testing"
	"time"

	"github.com/hellomyzn/yt-notifier/internal/model"
	"github.com/hellomyzn/yt-notifier/internal/repository"
)

// ---- mocks ----

type mockFeedRepo struct {
	videos []model.VideoDTO
	err    error
	calls  int
}

func (m *mockFeedRepo) Fetch(channelID string) ([]model.VideoDTO, error) {
	m.calls++
	return m.videos, m.err
}

type mockYTRepo struct {
	videos []model.VideoDTO
	err    error
	calls  int
}

func (m *mockYTRepo) FetchUploads(channelID string, maxResults int) ([]model.VideoDTO, error) {
	m.calls++
	return m.videos, m.err
}

type mockNotifiedRepo struct {
	seen     map[string]bool
	appended []string
}

func (m *mockNotifiedRepo) Has(id string) (bool, error) { return m.seen[id], nil }
func (m *mockNotifiedRepo) Append(id, chID string, pub, notAt time.Time) error {
	m.appended = append(m.appended, id)
	return nil
}

// ---- helpers ----

func makeVideos(ids ...string) []model.VideoDTO {
	var out []model.VideoDTO
	for _, id := range ids {
		out = append(out, model.VideoDTO{VideoID: id, Title: "Title " + id})
	}
	return out
}

func makeCh(fetchLimit int) model.ChannelDTO {
	return model.ChannelDTO{ChannelID: "UCtest", Category: "test", FetchLimit: fetchLimit}
}

// ---- ListNewVideos ----

func TestFeedService_ListNewVideos_RSSPath(t *testing.T) {
	rss := &mockFeedRepo{videos: makeVideos("v1", "v2")}
	yt := &mockYTRepo{}
	notified := &mockNotifiedRepo{seen: map[string]bool{}}

	svc := NewFeedService(rss, yt, notified, false, false, true)
	got, err := svc.ListNewVideos(makeCh(10))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len(result) = %d; want 2", len(got))
	}
	if rss.calls != 1 {
		t.Errorf("RSS calls = %d; want 1", rss.calls)
	}
	if yt.calls != 0 {
		t.Errorf("YouTube API calls = %d; want 0 (fetch_limit < 15)", yt.calls)
	}
}

func TestFeedService_ListNewVideos_APIPath(t *testing.T) {
	rss := &mockFeedRepo{}
	yt := &mockYTRepo{videos: makeVideos("v1", "v2", "v3")}
	notified := &mockNotifiedRepo{seen: map[string]bool{}}

	svc := NewFeedService(rss, yt, notified, false, false, true)
	got, err := svc.ListNewVideos(makeCh(15))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("len(result) = %d; want 3", len(got))
	}
	if yt.calls != 1 {
		t.Errorf("YouTube API calls = %d; want 1 (fetch_limit >= 15)", yt.calls)
	}
	if rss.calls != 0 {
		t.Errorf("RSS calls = %d; want 0", rss.calls)
	}
}

func TestFeedService_ListNewVideos_Deduplication(t *testing.T) {
	rss := &mockFeedRepo{videos: makeVideos("v1", "v2", "v3")}
	notified := &mockNotifiedRepo{seen: map[string]bool{"v1": true, "v3": true}}

	svc := NewFeedService(rss, nil, notified, false, false, true)
	got, err := svc.ListNewVideos(makeCh(10))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(result) = %d; want 1 (v2 のみ未通知)", len(got))
	}
	if got[0].VideoID != "v2" {
		t.Errorf("result[0].VideoID = %q; want v2", got[0].VideoID)
	}
}

func TestFeedService_ListNewVideos_AllAlreadyNotified(t *testing.T) {
	rss := &mockFeedRepo{videos: makeVideos("v1", "v2")}
	notified := &mockNotifiedRepo{seen: map[string]bool{"v1": true, "v2": true}}

	svc := NewFeedService(rss, nil, notified, false, false, true)
	got, err := svc.ListNewVideos(makeCh(10))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(result) = %d; want 0", len(got))
	}
}

func TestFeedService_ListNewVideos_RSSFallbackWhenAPIFails(t *testing.T) {
	rss := &mockFeedRepo{videos: makeVideos("v1")}
	yt := &mockYTRepo{err: errors.New("api error")}
	notified := &mockNotifiedRepo{seen: map[string]bool{}}

	svc := NewFeedService(rss, yt, notified, false, false, true)
	got, err := svc.ListNewVideos(makeCh(15))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("len(result) = %d; want 1 (RSS フォールバック)", len(got))
	}
	if rss.calls != 1 {
		t.Errorf("RSS calls = %d; want 1 (フォールバック)", rss.calls)
	}
}

func TestFeedService_ListNewVideos_RSSFallbackWhenRateLimited(t *testing.T) {
	rss := &mockFeedRepo{videos: makeVideos("v1")}
	yt := &mockYTRepo{err: &ytRateLimitErr{}}
	notified := &mockNotifiedRepo{seen: map[string]bool{}}

	svc := NewFeedService(rss, yt, notified, false, false, true)
	got, err := svc.ListNewVideos(makeCh(15))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("len(result) = %d; want 1 (rate limit フォールバック)", len(got))
	}
}

func TestFeedService_ListNewVideos_RSSError(t *testing.T) {
	rss := &mockFeedRepo{err: errors.New("network error")}
	notified := &mockNotifiedRepo{seen: map[string]bool{}}

	svc := NewFeedService(rss, nil, notified, false, false, true)
	_, err := svc.ListNewVideos(makeCh(10))

	if err == nil {
		t.Error("want error when RSS fails, got nil")
	}
}

func TestFeedService_ListNewVideos_SaturationTriggersAPIEscalation(t *testing.T) {
	// RSS が 15 件返す（飽和） → API に昇格
	rssVideos := makeVideos(
		"v1", "v2", "v3", "v4", "v5",
		"v6", "v7", "v8", "v9", "v10",
		"v11", "v12", "v13", "v14", "v15",
	)
	rss := &mockFeedRepo{videos: rssVideos}
	yt := &mockYTRepo{videos: makeVideos("v1", "v2", "v3", "v4", "v5", "v16", "v17")}
	notified := &mockNotifiedRepo{seen: map[string]bool{}}

	svc := NewFeedService(rss, yt, notified, false, false, true)
	got, err := svc.ListNewVideos(makeCh(10)) // fetch_limit < 15 なので RSS 優先

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// API に昇格されたので API の結果を使う
	if yt.calls != 1 {
		t.Errorf("YouTube API calls = %d; want 1 (飽和昇格)", yt.calls)
	}
	_ = got
}

// ---- ytRateLimitErr: repository.ErrYouTubeRateLimited をラップ ----

type ytRateLimitErr struct{}

func (e *ytRateLimitErr) Error() string  { return "rate limited" }
func (e *ytRateLimitErr) Unwrap() error  { return repository.ErrYouTubeRateLimited }
