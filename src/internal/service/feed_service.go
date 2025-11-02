package service

import (
	"github.com/hellomyzn/yt-notifier/internal/model"
	"github.com/hellomyzn/yt-notifier/internal/repository"
)

type FeedService interface {
	ListNewVideos(ch model.ChannelDTO) ([]model.VideoDTO, error)
}

type feedService struct {
	rssRepo          repository.FeedRepository
	ytRepo           repository.YouTubeRepository
	notifiedRepo     repository.NotifiedRepository
	includeLive      bool
	includePremieres bool
	includeShorts    bool
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
	if ch.FetchLimit >= 15 && s.ytRepo != nil {
		videos, err = s.ytRepo.FetchUploads(ch.ChannelID, ch.FetchLimit)
	} else {
		videos, err = s.rssRepo.Fetch(ch.ChannelID)
	}
	if err != nil {
		return nil, err
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
