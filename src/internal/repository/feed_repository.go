package repository

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hellomyzn/yt-notifier/internal/model"
)

type FeedRepository interface {
	Fetch(channelID string) ([]model.VideoDTO, error)
}

type RSSFeedRepository struct{}

func (r *RSSFeedRepository) Fetch(channelID string) ([]model.VideoDTO, error) {
	u := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelID)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("youtube feed status %d", resp.StatusCode)
	}

	feed, err := decodeFeed(resp.Body)
	if err != nil {
		return nil, err
	}

	var out []model.VideoDTO
	for _, entry := range feed.Entries {
		vid := normalizeVideoID(entry.ID, entry.Link())
		published := time.Now()
		if entry.Published != "" {
			if t, err := time.Parse(time.RFC3339, entry.Published); err == nil {
				published = t
			}
		}
		out = append(out, model.VideoDTO{
			VideoID:     vid,
			Title:       entry.Title,
			Link:        entry.Link(),
			ChannelID:   channelID,
			ChannelName: feed.Title,
			PublishedAt: published,
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

type ytFeed struct {
	Title   string
	Entries []ytEntry
}

type ytEntry struct {
	ID        string
	Title     string
	Links     []ytLink
	Published string
}

func (e ytEntry) Link() string {
	for _, l := range e.Links {
		if l.Rel == "alternate" && l.Href != "" {
			return l.Href
		}
	}
	if len(e.Links) > 0 {
		return e.Links[0].Href
	}
	return ""
}

type ytLink struct {
	Rel  string
	Href string
}

func decodeFeed(r io.Reader) (*ytFeed, error) {
	dec := xml.NewDecoder(r)
	const (
		atomNS = "http://www.w3.org/2005/Atom"
		ytNS   = "http://www.youtube.com/xml/schemas/2015"
	)

	feed := &ytFeed{}
	var current *ytEntry
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Space {
			case atomNS, "":
				switch t.Name.Local {
				case "title":
					var text string
					if err := dec.DecodeElement(&text, &t); err != nil {
						return nil, err
					}
					if current == nil {
						feed.Title = text
					} else {
						current.Title = text
					}
				case "entry":
					feed.Entries = append(feed.Entries, ytEntry{})
					current = &feed.Entries[len(feed.Entries)-1]
				case "id":
					if current == nil {
						if err := dec.Skip(); err != nil {
							return nil, err
						}
					} else {
						var id string
						if err := dec.DecodeElement(&id, &t); err != nil {
							return nil, err
						}
						if current.ID == "" {
							current.ID = id
						}
					}
				case "published":
					if current == nil {
						if err := dec.Skip(); err != nil {
							return nil, err
						}
					} else {
						var published string
						if err := dec.DecodeElement(&published, &t); err != nil {
							return nil, err
						}
						current.Published = published
					}
				case "link":
					if current == nil {
						if err := dec.Skip(); err != nil {
							return nil, err
						}
					} else {
						link := ytLink{}
						for _, attr := range t.Attr {
							switch attr.Name.Local {
							case "rel":
								link.Rel = attr.Value
							case "href":
								link.Href = attr.Value
							}
						}
						current.Links = append(current.Links, link)
						if err := dec.Skip(); err != nil {
							return nil, err
						}
					}
				default:
					if err := dec.Skip(); err != nil {
						return nil, err
					}
				}
			case ytNS:
				if current == nil {
					if err := dec.Skip(); err != nil {
						return nil, err
					}
					continue
				}
				switch t.Name.Local {
				case "videoId":
					var videoID string
					if err := dec.DecodeElement(&videoID, &t); err != nil {
						return nil, err
					}
					current.ID = videoID
				default:
					if err := dec.Skip(); err != nil {
						return nil, err
					}
				}
			default:
				if err := dec.Skip(); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if t.Name.Space == atomNS && t.Name.Local == "entry" {
				current = nil
			}
		}
	}
	return feed, nil
}
