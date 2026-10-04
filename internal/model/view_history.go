package model

import "time"

// ViewHistory is one file a user opened in the web UI. Each (user, path)
// pair has a single row: reopening a file moves it to the top and bumps Count.
type ViewHistory struct {
	ID       uint      `json:"-" gorm:"primaryKey"`
	UserID   uint      `json:"-" gorm:"index:idx_view_history_user_time,priority:1;not null"`
	Path     string    `json:"path" gorm:"type:text;not null"`
	ViewedAt time.Time `json:"viewed_at" gorm:"index:idx_view_history_user_time,priority:2"`
	Count    int       `json:"count"`
}
