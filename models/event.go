package models

import "time"

type Event struct {
	Title     string
	Time      time.Time
	ChannelID string
}
