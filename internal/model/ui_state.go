package model

import "time"

// UIState is a small per-user value the web UI keeps on the server, such as
// the tab layout, so it survives browser restarts and other devices.
type UIState struct {
	ID        uint      `json:"-" gorm:"primaryKey"`
	UserID    uint      `json:"-" gorm:"uniqueIndex:idx_ui_state_user_key,priority:1;not null"`
	Key       string    `json:"key" gorm:"size:64;uniqueIndex:idx_ui_state_user_key,priority:2;not null"`
	Value     string    `json:"value" gorm:"type:text"`
	UpdatedAt time.Time `json:"updated_at"`
}
