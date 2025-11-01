package model

import "time"

type ChannelDTO struct {
	ChannelID string
	Category  string
	Name      string
	Enabled   bool
}

type VideoDTO struct {
	VideoID     string
	Title       string
	Link        string
	ChannelID   string
	ChannelName string
	PublishedAt time.Time
}
