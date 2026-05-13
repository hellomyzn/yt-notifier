package controller

import (
	"log"
	"sync"
	"time"

	"github.com/hellomyzn/yt-notifier/internal/model"
	"github.com/hellomyzn/yt-notifier/internal/repository"
	"github.com/hellomyzn/yt-notifier/internal/service"
)

const fetchWorkers = 5

type JobController interface {
	RunOnce() error
}

type jobController struct {
	chRepo     repository.ChannelRepository
	feedSvc    service.FeedService
	notifySvc  service.NotifyService
	fetchSleep time.Duration
}

func NewJobController(chRepo repository.ChannelRepository, fs service.FeedService, ns service.NotifyService, fetchSleep time.Duration) JobController {
	return &jobController{chRepo: chRepo, feedSvc: fs, notifySvc: ns, fetchSleep: fetchSleep}
}

type fetchResult struct {
	ch     model.ChannelDTO
	videos []model.VideoDTO
	err    error
}

func (c *jobController) RunOnce() error {
	channels, err := c.chRepo.ListEnabled()
	if err != nil {
		return err
	}

	// Phase 1: fetch all channels concurrently (read-only operations).
	results := make([]fetchResult, len(channels))
	sem := make(chan struct{}, fetchWorkers)
	var wg sync.WaitGroup
	for i, ch := range channels {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, ch model.ChannelDTO) {
			defer wg.Done()
			defer func() { <-sem }()
			videos, err := c.feedSvc.ListNewVideos(ch)
			results[i] = fetchResult{ch, videos, err}
			time.Sleep(c.fetchSleep)
		}(i, ch)
	}
	wg.Wait()

	// Phase 2: notify concurrently per channel.
	// Channels sharing the same webhook URL auto-serialize via webhookDispatcher's mutex.
	// Channels with distinct webhooks notify in parallel.
	var wg2 sync.WaitGroup
	for _, r := range results {
		if r.err != nil {
			log.Printf("failed to list new videos for channel=%s: %v", r.ch.ChannelID, r.err)
			continue
		}
		if len(r.videos) == 0 {
			continue
		}
		wg2.Add(1)
		go func(r fetchResult) {
			defer wg2.Done()
			for _, v := range r.videos {
				if err := c.notifySvc.Notify(r.ch.Category, v); err != nil {
					log.Printf("failed to notify channel=%s video=%s: %v", r.ch.ChannelID, v.VideoID, err)
				}
			}
		}(r)
	}
	wg2.Wait()

	feedStats := c.feedSvc.Stats()
	notifyStats := c.notifySvc.Stats()
	log.Printf("feed stats: rss=%d api=%d rss_fallbacks=%d api_fallbacks=%d saturation_triggers=%d", feedStats.RSSFetches, feedStats.APIFetches, feedStats.RSSFallbacks, feedStats.APIFallbacks, feedStats.SaturationTriggers)
	log.Printf("notification stats: sent=%d retried_messages=%d retry_attempts=%d failed=%d", notifyStats.Sent, notifyStats.RetriedMessages, notifyStats.RetryAttempts, notifyStats.Failed)
	return nil
}
