package service

import (
	"errors"
	"log"
	"sync"

	"github.com/hellomyzn/yt-notifier/internal/model"
	"github.com/hellomyzn/yt-notifier/internal/repository"
)

const rssMaxWindow = 15

type FeedService interface {
	ListNewVideos(ch model.ChannelDTO) ([]model.VideoDTO, error)
	Stats() FeedStats
}

type FeedStats struct {
	RSSFetches         int
	APIFetches         int
	APIFallbacks       int
	RSSFallbacks       int
	SaturationTriggers int
}

type feedService struct {
	rssRepo          repository.FeedRepository
	ytRepo           repository.YouTubeRepository
	notifiedRepo     repository.NotifiedRepository
	includeLive      bool
	includePremieres bool
	includeShorts    bool

	mu    sync.Mutex
	stats FeedStats
}

func NewFeedService(rss repository.FeedRepository, yt repository.YouTubeRepository, notified repository.NotifiedRepository,
	includeLive, includePremieres, includeShorts bool) FeedService {
	return &feedService{rssRepo: rss, ytRepo: yt, notifiedRepo: notified,
		includeLive: includeLive, includePremieres: includePremieres, includeShorts: includeShorts}
}

func (s *feedService) ListNewVideos(ch model.ChannelDTO) ([]model.VideoDTO, error) {
	var (
		videos []model.VideoDTO
		err    error
	)
	useYouTube := ch.FetchLimit >= rssMaxWindow && s.ytRepo != nil
	if useYouTube {
		videos, err = s.ytRepo.FetchUploads(ch.ChannelID, ch.FetchLimit)
		if err == nil {
			s.recordAPIFetch()
		}
	} else {
		videos, err = s.rssRepo.Fetch(ch.ChannelID)
		s.recordRSSFetch()
	}
	if err != nil && useYouTube {
		if errors.Is(err, repository.ErrYouTubeRateLimited) {
			log.Printf("youtube api quota exceeded for channel=%s, falling back to rss", ch.ChannelID)
		} else {
			log.Printf("youtube api fetch failed for channel=%s: %v; falling back to rss", ch.ChannelID, err)
		}
		s.recordAPIFallback()
		s.recordRSSFetch()
		videos, err = s.rssRepo.Fetch(ch.ChannelID)
		s.recordRSSFallback()
	}
	if err != nil {
		return nil, err
	}

	// If the RSS feed is saturated (15 items) we might miss uploads, so escalate to API when available.
	if !useYouTube && s.ytRepo != nil && len(videos) >= rssMaxWindow {
		log.Printf("rss feed saturated for channel=%s; attempting youtube api", ch.ChannelID)
		s.recordSaturationTrigger()
		apiVideos, apiErr := s.ytRepo.FetchUploads(ch.ChannelID, ch.FetchLimit)
		if apiErr == nil {
			videos = apiVideos
			s.recordAPIFetch()
		} else {
			log.Printf("failed to load youtube api feed after rss saturation for channel=%s: %v", ch.ChannelID, apiErr)
			if errors.Is(apiErr, repository.ErrYouTubeRateLimited) {
				s.recordAPIFallback()
			}
		}
	}

	var out []model.VideoDTO
	for _, v := range videos {
		seen, err := s.notifiedRepo.Has(v.VideoID)
		if err != nil {
			return nil, err
		}
		if seen {
			continue
		}
		// TODO: includeLive/includePremieres/includeShorts に応じたフィルタを追加
		out = append(out, v)
	}
	return out, nil
}

func (s *feedService) Stats() FeedStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats
}

func (s *feedService) recordAPIFetch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.APIFetches++
}

func (s *feedService) recordRSSFetch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.RSSFetches++
}

func (s *feedService) recordAPIFallback() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.APIFallbacks++
}

func (s *feedService) recordRSSFallback() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.RSSFallbacks++
}

func (s *feedService) recordSaturationTrigger() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.SaturationTriggers++
}
