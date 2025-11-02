package repository

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hellomyzn/yt-notifier/internal/model"
)

type YouTubeRepository interface {
	FetchUploads(channelID string, maxResults int) ([]model.VideoDTO, error)
}

type YouTubeAPIRepository struct {
	APIKey string
	Client *http.Client
}

func NewYouTubeAPIRepository(apiKey string) *YouTubeAPIRepository {
	if apiKey == "" {
		return nil
	}
	return &YouTubeAPIRepository{APIKey: apiKey, Client: http.DefaultClient}
}

func (r *YouTubeAPIRepository) FetchUploads(channelID string, maxResults int) ([]model.VideoDTO, error) {
	if r == nil {
		return nil, fmt.Errorf("youtube api repository is nil")
	}
	if r.APIKey == "" {
		return nil, fmt.Errorf("youtube api key is empty")
	}
	playlistID := uploadsPlaylistID(channelID)
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
				err = fmt.Errorf("youtube api status %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
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
