package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hellomyzn/yt-notifier/config"
	"github.com/hellomyzn/yt-notifier/internal/controller"
	"github.com/hellomyzn/yt-notifier/internal/repository"
	"github.com/hellomyzn/yt-notifier/internal/service"
)

func main() {
	root, err := repoRoot()
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := config.Load(filepath.Join(root, "config", "app.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	webhookFile := cfg.WebhookFile
	if webhookFile == "" {
		webhookFile = filepath.Join("config", "webhooks.env")
	}
	if !filepath.IsAbs(webhookFile) {
		webhookFile = filepath.Join(root, webhookFile)
	}
	webhookSecrets, err := config.LoadWebhookFile(webhookFile)
	if err != nil {
		log.Fatal(err)
	}

	categoryToWebhook := map[string]string{}
	for category, envName := range cfg.CategoryToEnv {
		if envName == "" {
			continue
		}
		webhook, ok := webhookSecrets[envName]
		if !ok || webhook == "" {
			log.Fatalf("webhook secret not found for %s", envName)
		}
		categoryToWebhook[category] = webhook
	}

	csvDir := filepath.Join(root, "src", "csv")
	chRepo := &repository.CSVChannelRepository{Path: filepath.Join(csvDir, "channels.csv")}
	notiRepo := &repository.CSVNotifiedRepository{Path: filepath.Join(csvDir, "notified.csv")}
	feedRepo := &repository.RSSFeedRepository{}

	feedSvc := service.NewFeedService(
		feedRepo, notiRepo,
		cfg.Filters.IncludeLive, cfg.Filters.IncludePremieres, cfg.Filters.IncludeShorts,
	)

	notifySvc := service.NewNotifyService(
		notiRepo,
		categoryToWebhook,
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

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(wd, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			return wd, nil
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}

	return "", fmt.Errorf("go.mod not found from %s", wd)
}
