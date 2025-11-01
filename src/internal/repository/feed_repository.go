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
	XMLName xml.Name  `xml:"feed"`
	Title   string    `xml:"title"`
	Entries []ytEntry `xml:"entry"`
}

type ytEntry struct {
	ID        string
	AltID     string
	Title     string
	Links     []ytLink
	Published string
}

func (e *ytEntry) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	const (
		atomNS = "http://www.w3.org/2005/Atom"
		ytNS   = "http://www.youtube.com/xml/schemas/2015"
	)

	*e = ytEntry{}
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case (t.Name.Space == atomNS || t.Name.Space == "") && t.Name.Local == "id":
				var id string
				if err := dec.DecodeElement(&id, &t); err != nil {
					return err
				}
				e.AltID = id
				if e.ID == "" {
					e.ID = id
				}
			case (t.Name.Space == atomNS || t.Name.Space == "") && t.Name.Local == "title":
				var title string
				if err := dec.DecodeElement(&title, &t); err != nil {
					return err
				}
				e.Title = title
			case (t.Name.Space == atomNS || t.Name.Space == "") && t.Name.Local == "published":
				var published string
				if err := dec.DecodeElement(&published, &t); err != nil {
					return err
				}
				e.Published = published
			case (t.Name.Space == atomNS || t.Name.Space == "") && t.Name.Local == "link":
				link := ytLink{}
				for _, attr := range t.Attr {
					switch attr.Name.Local {
					case "rel":
						link.Rel = attr.Value
					case "href":
						link.Href = attr.Value
					}
				}
				e.Links = append(e.Links, link)
				if err := dec.Skip(); err != nil {
					return err
				}
			case t.Name.Space == ytNS && t.Name.Local == "videoId":
				var videoID string
				if err := dec.DecodeElement(&videoID, &t); err != nil {
					return err
				}
				if videoID != "" {
					e.ID = videoID
				}
			default:
				if err := dec.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if t.Name == start.Name {
				return nil
			}
		}
	}
	return nil
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
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

func decodeFeed(r io.Reader) (*ytFeed, error) {
	dec := xml.NewDecoder(r)
	dec.DefaultSpace = "http://www.w3.org/2005/Atom"
	var feed ytFeed
	if err := dec.Decode(&feed); err != nil {
		return nil, err
	}
	return &feed, nil
}
