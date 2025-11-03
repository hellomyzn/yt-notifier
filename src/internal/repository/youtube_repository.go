package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hellomyzn/yt-notifier/internal/model"
)

type YouTubeRepository interface {
	FetchUploads(channelID string, maxResults int) ([]model.VideoDTO, error)
}

type YouTubeAPIRepository struct {
	APIKey        string
	Client        *http.Client
	playlistCache map[string]string
	cacheMu       sync.RWMutex
	metrics       *YouTubeAPIMetrics
}

func NewYouTubeAPIRepository(apiKey string) *YouTubeAPIRepository {
	if apiKey == "" {
		return nil
	}
	return &YouTubeAPIRepository{
		APIKey:        apiKey,
		Client:        http.DefaultClient,
		playlistCache: map[string]string{},
		metrics:       &YouTubeAPIMetrics{},
	}
}

func (r *YouTubeAPIRepository) FetchUploads(channelID string, maxResults int) ([]model.VideoDTO, error) {
	if r == nil {
		return nil, fmt.Errorf("youtube api repository is nil")
	}
	if r.APIKey == "" {
		return nil, fmt.Errorf("youtube api key is empty")
	}
	playlistID := r.cachedPlaylistID(channelID)
	if playlistID == "" {
		return nil, fmt.Errorf("invalid channel id %s", channelID)
	}
	totalRequested := maxResults
	if totalRequested <= 0 {
		return nil, nil
	}

	endpoint := "https://www.googleapis.com/youtube/v3/playlistItems"
	client := r.Client
	if client == nil {
		client = http.DefaultClient
	}

	var (
		out           []model.VideoDTO
		nextPageToken string
	)

	for len(out) < totalRequested {
		remaining := totalRequested - len(out)
		pageSize := clamp(remaining, 1, 50)

		params := url.Values{}
		params.Set("part", "snippet,contentDetails")
		params.Set("playlistId", playlistID)
		params.Set("maxResults", strconv.Itoa(pageSize))
		params.Set("key", r.APIKey)
		if nextPageToken != "" {
			params.Set("pageToken", nextPageToken)
		}

		requestURL := endpoint + "?" + params.Encode()

		resp, err := client.Get(requestURL)
		if err != nil {
			return nil, err
		}
		func() {
			defer resp.Body.Close()

			if resp.StatusCode >= 300 {
				snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
				ytErr := &YouTubeAPIError{
					StatusCode: resp.StatusCode,
					Message:    strings.TrimSpace(string(snippet)),
				}
				if retryAfter := parseRetryAfter(resp.Header.Get("Retry-After")); retryAfter > 0 {
					ytErr.RetryAfter = retryAfter
				}
				if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden {
					ytErr.Err = ErrYouTubeRateLimited
				}
				err = ytErr
				return
			}

			var payload youtubePlaylistItemsResponse
			if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr != nil {
				err = decodeErr
				return
			}

			for _, item := range payload.Items {
				videoID := item.ContentDetails.VideoID
				if videoID == "" {
					videoID = item.Snippet.ResourceID.VideoID
				}
				if videoID == "" {
					continue
				}
				published := firstTime(item.ContentDetails.VideoPublishedAt, item.Snippet.PublishedAt)
				out = append(out, model.VideoDTO{
					VideoID:     videoID,
					Title:       item.Snippet.Title,
					Link:        fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID),
					ChannelID:   channelID,
					ChannelName: item.Snippet.ChannelTitle,
					PublishedAt: published,
				})
				if len(out) >= totalRequested {
					break
				}
			}

			nextPageToken = payload.NextPageToken
			if nextPageToken == "" || len(payload.Items) == 0 {
				totalRequested = len(out)
			}
		}()

		if err != nil {
			return nil, err
		}

		r.metrics.IncrementRequests()

		if len(out) >= totalRequested {
			break
		}
		if nextPageToken == "" {
			break
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].PublishedAt.After(out[j].PublishedAt)
	})

	if len(out) > maxResults {
		out = out[:maxResults]
	}

	return out, nil
}

func (r *YouTubeAPIRepository) cachedPlaylistID(channelID string) string {
	r.cacheMu.RLock()
	cached := r.playlistCache[channelID]
	r.cacheMu.RUnlock()
	if cached != "" {
		return cached
	}
	computed := uploadsPlaylistID(channelID)
	if computed == "" {
		return ""
	}
	r.cacheMu.Lock()
	r.playlistCache[channelID] = computed
	r.cacheMu.Unlock()
	return computed
}

// Metrics returns a snapshot of the quota usage observed by the repository.
func (r *YouTubeAPIRepository) Metrics() YouTubeAPIMetricsSnapshot {
	if r == nil || r.metrics == nil {
		return YouTubeAPIMetricsSnapshot{}
	}
	return r.metrics.Snapshot()
}

type YouTubeAPIMetrics struct {
	mu             sync.Mutex
	requestCount   int
	quotaUnitCount int
}

func (m *YouTubeAPIMetrics) IncrementRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestCount++
	m.quotaUnitCount++ // playlistItems.list = 1 unit per call
}

func (m *YouTubeAPIMetrics) Snapshot() YouTubeAPIMetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return YouTubeAPIMetricsSnapshot{
		Requests:   m.requestCount,
		QuotaUnits: m.quotaUnitCount,
	}
}

type YouTubeAPIMetricsSnapshot struct {
	Requests   int
	QuotaUnits int
}

var ErrYouTubeRateLimited = errors.New("youtube api rate limited")

type YouTubeAPIError struct {
	Err        error
	StatusCode int
	RetryAfter time.Duration
	Message    string
}

func (e *YouTubeAPIError) Error() string {
	if e == nil {
		return ""
	}
	msg := fmt.Sprintf("youtube api status %d", e.StatusCode)
	if e.Message != "" {
		msg = fmt.Sprintf("%s: %s", msg, e.Message)
	}
	if e.RetryAfter > 0 {
		msg = fmt.Sprintf("%s (retry after %s)", msg, e.RetryAfter)
	}
	return msg
}

func (e *YouTubeAPIError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if secs, err := strconv.Atoi(value); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil {
		return time.Until(t)
	}
	return 0
}

type youtubePlaylistItemsResponse struct {
	Items []struct {
		Snippet struct {
			Title        string `json:"title"`
			ChannelTitle string `json:"channelTitle"`
			PublishedAt  string `json:"publishedAt"`
			ResourceID   struct {
				VideoID string `json:"videoId"`
			} `json:"resourceId"`
		} `json:"snippet"`
		ContentDetails struct {
			VideoID          string `json:"videoId"`
			VideoPublishedAt string `json:"videoPublishedAt"`
		} `json:"contentDetails"`
	} `json:"items"`
	NextPageToken string `json:"nextPageToken"`
}

func uploadsPlaylistID(channelID string) string {
	trimmed := strings.TrimSpace(channelID)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "UC") && len(trimmed) > 2 {
		return "UU" + trimmed[2:]
	}
	return trimmed
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func firstTime(values ...string) time.Time {
	for _, raw := range values {
		if raw == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t
		}
	}
	return time.Now()
}
