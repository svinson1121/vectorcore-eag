package models

import (
	"time"

	"gorm.io/gorm"
)

type Alert struct {
	// Primary key is the CAP <identifier> field — globally unique per CAP spec.
	ID          string         `gorm:"primaryKey" json:"id"`
	Sender      string         `gorm:"index" json:"sender"`
	Sent        time.Time      `gorm:"index" json:"sent"`
	Status      string         `gorm:"index" json:"status"`
	MsgType     string         `gorm:"index" json:"msg_type"`
	Scope       string         `json:"scope"`
	References  string         `gorm:"type:text" json:"references"`
	Event       string         `gorm:"index" json:"event"`
	Headline    string         `json:"headline"`
	Description string         `gorm:"type:text" json:"description"`
	Severity    string         `gorm:"index" json:"severity"`
	Urgency     string         `gorm:"index" json:"urgency"`
	Certainty   string         `json:"certainty"`
	Effective   time.Time      `json:"effective"`
	Onset       time.Time      `json:"onset"`
	Expires     time.Time      `gorm:"index" json:"expires"`
	AreaDesc    string         `gorm:"index" json:"area_desc"`
	FeedSource  string         `gorm:"index" json:"feed_source"`
	Geometry    string         `gorm:"type:text" json:"geometry,omitempty"` // GeoJSON geometry object
	RawCAP      string         `gorm:"type:text" json:"raw_cap,omitempty"`
	Forwarded   bool           `gorm:"default:false;index" json:"forwarded"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
