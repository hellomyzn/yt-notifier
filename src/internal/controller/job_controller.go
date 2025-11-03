package controller

import (
	"log"
	"time"

	"github.com/hellomyzn/yt-notifier/internal/repository"
	"github.com/hellomyzn/yt-notifier/internal/service"
)

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

func (c *jobController) RunOnce() error {
	channels, err := c.chRepo.ListEnabled()
	if err != nil {
		return err
	}
	for _, ch := range channels {
		videos, err := c.feedSvc.ListNewVideos(ch)
		if err != nil {
			log.Printf("failed to list new videos for channel=%s: %v", ch.ChannelID, err)
			time.Sleep(c.fetchSleep)
			continue
		}
		for _, v := range videos {
			if err := c.notifySvc.Notify(ch.Category, v); err != nil {
				log.Printf("failed to notify channel=%s video=%s: %v", ch.ChannelID, v.VideoID, err)
			}
		}
		time.Sleep(c.fetchSleep)
	}
	feedStats := c.feedSvc.Stats()
	notifyStats := c.notifySvc.Stats()
	log.Printf("feed stats: rss=%d api=%d rss_fallbacks=%d api_fallbacks=%d saturation_triggers=%d", feedStats.RSSFetches, feedStats.APIFetches, feedStats.RSSFallbacks, feedStats.APIFallbacks, feedStats.SaturationTriggers)
	log.Printf("notification stats: sent=%d retried_messages=%d retry_attempts=%d failed=%d", notifyStats.Sent, notifyStats.RetriedMessages, notifyStats.RetryAttempts, notifyStats.Failed)
	return nil
}
