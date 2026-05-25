package models

import "time"

type AIBriefing struct {
	UserID    string    `json:"user_id"`
	Date      string    `json:"date"`
	ContentMD string    `json:"content_md"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
}
