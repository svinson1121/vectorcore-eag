package models

import "time"

// XMPPPeer represents an inbound XMPP client that connects to the EAG XMPP server.
// Authentication is SASL PLAIN (username + password). Filter fields control which
// alerts are pushed to this peer — empty = match all.
type XMPPPeer struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name           string    `json:"name"`                    // human-readable label
	Username       string    `gorm:"uniqueIndex" json:"username"` // SASL auth username
	Password       string    `json:"password"`                // stored plaintext — TODO: hash in future
	Enabled        bool      `gorm:"default:true" json:"enabled"`
	FilterSeverity string    `gorm:"type:text" json:"filter_severity"` // JSON []string
	FilterEvent    string    `gorm:"type:text" json:"filter_event"`    // JSON []string
	FilterArea     string    `gorm:"type:text" json:"filter_area"`     // JSON []string
	FilterStatus   string    `gorm:"type:text" json:"filter_status"`   // JSON []string
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
