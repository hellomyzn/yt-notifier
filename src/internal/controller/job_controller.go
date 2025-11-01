package controller

import (
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
		if err == nil {
			for _, v := range videos {
				_ = c.notifySvc.Notify(ch.Category, v)
			}
		}
		time.Sleep(c.fetchSleep)
	}
	return nil
}
