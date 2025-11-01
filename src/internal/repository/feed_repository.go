package repository

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hellomyzn/yt-notifier/src/internal/model"
	"github.com/mmcdole/gofeed"
)

type FeedRepository interface {
	Fetch(channelID string) ([]model.VideoDTO, error)
}

type RSSFeedRepository struct{}

func (r *RSSFeedRepository) Fetch(channelID string) ([]model.VideoDTO, error) {
	u := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelID)
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(u)
	if err != nil {
		return nil, err
	}
	var out []model.VideoDTO
	for _, it := range feed.Items {
		vid := normalizeVideoID(it.GUID, it.Link)
		t := time.Now()
		if it.PublishedParsed != nil {
			t = *it.PublishedParsed
		}
		out = append(out, model.VideoDTO{
			VideoID:     vid,
			Title:       it.Title,
			Link:        it.Link,
			ChannelID:   channelID,
			ChannelName: feed.Title,
			PublishedAt: t,
		})
	}
	return out, nil
}

func normalizeVideoID(guid, link string) string {
	// 1) GUIDが "yt:video:VIDEOID" 形式のことが多い
	if strings.Contains(guid, ":") {
		parts := strings.Split(guid, ":")
		return parts[len(parts)-1]
	}
	// 2) link ?v=VIDEOID を優先抽出
	if u, err := url.Parse(link); err == nil {
		if v := u.Query().Get("v"); v != "" {
			return v
		}
	}
	// 3) 最後の手段としてGUIDを返す
	return guid
}
