package models

import "time"

type FeedSource struct {
	ID           uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string     `json:"name"`
	URL          string     `gorm:"uniqueIndex" json:"url"`
	Type         string     `json:"type"`
	Enabled      bool       `gorm:"default:true" json:"enabled"`
	PollInterval int        `gorm:"default:60" json:"poll_interval"`
	// JSON-encoded map[string]string of extra query params.
	Params       string     `gorm:"type:text" json:"params"`
	LastPolled   *time.Time `json:"last_polled"`
	LastStatus   string     `json:"last_status"`
	AlertCount   int        `gorm:"default:0" json:"alert_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
