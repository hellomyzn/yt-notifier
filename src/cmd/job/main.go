package main

import (
	"log"
	"path/filepath"
	"time"

	"github.com/hellomyzn/yt-notifier/src/internal/config"
	"github.com/hellomyzn/yt-notifier/src/internal/controller"
	"github.com/hellomyzn/yt-notifier/src/internal/repository"
	"github.com/hellomyzn/yt-notifier/src/internal/service"
)

func main() {
	cfg, err := config.Load("config/app.yaml")
	if err != nil {
		log.Fatal(err)
	}

	chRepo := &repository.CSVChannelRepository{Path: filepath.Clean("src/src/csv/channels.csv")}
	notiRepo := &repository.CSVNotifiedRepository{Path: filepath.Clean("src/src/csv/notified.csv")}
	feedRepo := &repository.RSSFeedRepository{}

	feedSvc := service.NewFeedService(
		feedRepo, notiRepo,
		cfg.Filters.IncludeLive, cfg.Filters.IncludePremieres, cfg.Filters.IncludeShorts,
	)

	notifySvc := service.NewNotifyService(
		notiRepo,
		cfg.CategoryToEnv,
		time.Duration(cfg.RateLimit.PostSleepMS)*time.Millisecond,
	)

	job := controller.NewJobController(
		chRepo,
		feedSvc,
		notifySvc,
		time.Duration(cfg.RateLimit.FetchSleepMS)*time.Millisecond,
	)

	if err := job.RunOnce(); err != nil {
		log.Fatal(err)
	}
}
