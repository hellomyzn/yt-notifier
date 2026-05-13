package controller

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hellomyzn/yt-notifier/internal/model"
	"github.com/hellomyzn/yt-notifier/internal/notifier"
	"github.com/hellomyzn/yt-notifier/internal/repository"
	"github.com/hellomyzn/yt-notifier/internal/service"
)

const fetchWorkers = 5

type JobController interface {
	RunOnce() error
}

type jobController struct {
	chRepo         repository.ChannelRepository
	feedSvc        service.FeedService
	notifySvc      service.NotifyService
	fetchSleep     time.Duration
	ytRepo         *repository.YouTubeAPIRepository
	summaryWebhook string
}

func NewJobController(
	chRepo repository.ChannelRepository,
	fs service.FeedService,
	ns service.NotifyService,
	fetchSleep time.Duration,
	ytRepo *repository.YouTubeAPIRepository,
	summaryWebhook string,
) JobController {
	return &jobController{
		chRepo:         chRepo,
		feedSvc:        fs,
		notifySvc:      ns,
		fetchSleep:     fetchSleep,
		ytRepo:         ytRepo,
		summaryWebhook: summaryWebhook,
	}
}

type fetchResult struct {
	ch     model.ChannelDTO
	videos []model.VideoDTO
	err    error
}

type channelStat struct {
	ch       model.ChannelDTO
	newCount int
	notified int
	failed   int
	fetchErr bool
}

func (c *jobController) RunOnce() error {
	jobStart := time.Now()

	channels, err := c.chRepo.ListEnabled()
	if err != nil {
		return err
	}

	// Phase 1: fetch all channels concurrently (read-only operations).
	fetchStart := time.Now()
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
	fetchElapsed := time.Since(fetchStart)

	// Phase 2: notify concurrently per channel.
	// Channels sharing the same webhook URL auto-serialize via webhookDispatcher's mutex.
	// Channels with distinct webhooks notify in parallel.
	notifyStart := time.Now()
	stats := make([]channelStat, len(results))
	var wg2 sync.WaitGroup
	for i, r := range results {
		stats[i] = channelStat{ch: r.ch}
		if r.err != nil {
			log.Printf("failed to list new videos for channel=%s: %v", r.ch.ChannelID, r.err)
			stats[i].fetchErr = true
			continue
		}
		stats[i].newCount = len(r.videos)
		if len(r.videos) == 0 {
			continue
		}
		wg2.Add(1)
		go func(idx int, r fetchResult) {
			defer wg2.Done()
			for _, v := range r.videos {
				if err := c.notifySvc.Notify(r.ch.Category, v); err != nil {
					log.Printf("failed to notify channel=%s video=%s: %v", r.ch.ChannelID, v.VideoID, err)
					stats[idx].failed++
				} else {
					stats[idx].notified++
				}
			}
		}(i, r)
	}
	wg2.Wait()
	notifyElapsed := time.Since(notifyStart)
	jobElapsed := time.Since(jobStart)

	feedStats := c.feedSvc.Stats()
	notifyStats := c.notifySvc.Stats()

	log.Printf("feed stats: rss=%d api=%d rss_fallbacks=%d api_fallbacks=%d saturation_triggers=%d",
		feedStats.RSSFetches, feedStats.APIFetches, feedStats.RSSFallbacks, feedStats.APIFallbacks, feedStats.SaturationTriggers)
	log.Printf("notification stats: sent=%d retried_messages=%d retry_attempts=%d failed=%d",
		notifyStats.Sent, notifyStats.RetriedMessages, notifyStats.RetryAttempts, notifyStats.Failed)

	if c.ytRepo != nil {
		if m := c.ytRepo.Metrics(); m.Requests > 0 || m.QuotaUnits > 0 {
			log.Printf("youtube api usage: requests=%d quota_units=%d", m.Requests, m.QuotaUnits)
		}
	}

	c.printSummary(stats, feedStats, notifyStats, fetchElapsed, notifyElapsed, jobElapsed)

	if c.summaryWebhook != "" {
		if err := c.sendDiscordSummary(stats, feedStats, notifyStats, fetchElapsed, notifyElapsed, jobElapsed); err != nil {
			log.Printf("failed to send summary to discord: %v", err)
		}
	}

	return nil
}

func (c *jobController) printSummary(stats []channelStat, feedStats service.FeedStats, notifyStats service.NotificationStats, fetchElapsed, notifyElapsed, jobElapsed time.Duration) {
	totalNew, totalNotified, totalFailed, channelsWithNew, channelsWithError := aggregateStats(stats)

	log.Printf("=== Job Summary ===")
	log.Printf("time    : total=%s  fetch=%s  notify=%s", fmtDuration(jobElapsed), fmtDuration(fetchElapsed), fmtDuration(notifyElapsed))
	log.Printf("channels: total=%d  with_new=%d  errors=%d", len(stats), channelsWithNew, channelsWithError)
	log.Printf("videos  : new=%d  notified=%d  failed=%d", totalNew, totalNotified, totalFailed)

	for _, s := range stats {
		if s.fetchErr {
			log.Printf("  [%s] %s: fetch error", s.ch.Category, s.ch.Name)
		} else if s.newCount > 0 {
			log.Printf("  [%s] %s: new=%d  notified=%d  failed=%d", s.ch.Category, s.ch.Name, s.newCount, s.notified, s.failed)
		}
	}
}

func (c *jobController) sendDiscordSummary(stats []channelStat, feedStats service.FeedStats, notifyStats service.NotificationStats, fetchElapsed, notifyElapsed, jobElapsed time.Duration) error {
	totalNew, totalNotified, totalFailed, channelsWithNew, _ := aggregateStats(stats)

	var sb strings.Builder
	fmt.Fprintf(&sb, "⏱ **実行時間**: %s (フェッチ: %s | 通知: %s)\n\n", fmtDuration(jobElapsed), fmtDuration(fetchElapsed), fmtDuration(notifyElapsed))
	fmt.Fprintf(&sb, "📺 **チャンネル**: %d件 (新着あり: %d件)\n", len(stats), channelsWithNew)
	fmt.Fprintf(&sb, "📼 **動画**: 新着%d件 | 通知%d件", totalNew, totalNotified)
	if totalFailed > 0 {
		fmt.Fprintf(&sb, " | 失敗%d件", totalFailed)
	}
	fmt.Fprintf(&sb, "\n\n")

	fmt.Fprintf(&sb, "**フィード**: RSS=%d API=%d", feedStats.RSSFetches, feedStats.APIFetches)
	if feedStats.SaturationTriggers > 0 || feedStats.APIFallbacks > 0 || feedStats.RSSFallbacks > 0 {
		fmt.Fprintf(&sb, " (飽和=%d API→RSS=%d RSS→API=%d)", feedStats.SaturationTriggers, feedStats.APIFallbacks, feedStats.RSSFallbacks)
	}
	fmt.Fprintf(&sb, "\n")

	if notifyStats.RetriedMessages > 0 {
		fmt.Fprintf(&sb, "**リトライ**: %d件 (%d回)\n", notifyStats.RetriedMessages, notifyStats.RetryAttempts)
	}

	if c.ytRepo != nil {
		if m := c.ytRepo.Metrics(); m.Requests > 0 {
			fmt.Fprintf(&sb, "**YouTube API**: リクエスト=%d クォータ=%d\n", m.Requests, m.QuotaUnits)
		}
	}

	if channelsWithNew > 0 {
		fmt.Fprintf(&sb, "\n**チャンネル別 (新着あり)**\n")
		for _, s := range stats {
			if s.newCount == 0 && !s.fetchErr {
				continue
			}
			var line string
			if s.fetchErr {
				line = fmt.Sprintf("• [%s] %s: ⚠ フェッチエラー\n", s.ch.Category, s.ch.Name)
			} else if s.failed > 0 {
				line = fmt.Sprintf("• [%s] %s: %d件通知 (%d件失敗)\n", s.ch.Category, s.ch.Name, s.notified, s.failed)
			} else {
				line = fmt.Sprintf("• [%s] %s: %d件通知\n", s.ch.Category, s.ch.Name, s.notified)
			}
			if sb.Len()+len(line)+20 > 4000 {
				fmt.Fprintf(&sb, "• ...(以下省略)\n")
				break
			}
			fmt.Fprintf(&sb, "%s", line)
		}
	}

	n := &notifier.DiscordNotifier{Webhook: c.summaryWebhook}
	return n.Send(notifier.NotificationContent{
		Title:   "📊 YT Notifier 実行結果",
		Message: sb.String(),
	})
}

func aggregateStats(stats []channelStat) (totalNew, totalNotified, totalFailed, channelsWithNew, channelsWithError int) {
	for _, s := range stats {
		totalNew += s.newCount
		totalNotified += s.notified
		totalFailed += s.failed
		if s.newCount > 0 {
			channelsWithNew++
		}
		if s.fetchErr {
			channelsWithError++
		}
	}
	return
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm%ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}
