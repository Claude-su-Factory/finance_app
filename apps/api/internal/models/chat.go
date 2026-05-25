package models

import (
	"encoding/json"
	"time"
)

type ChatSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ChatMessage struct {
	ID           string          `json:"id"`
	SessionID    string          `json:"session_id"`
	Role         string          `json:"role"`
	Content      string          `json:"content"`
	ToolCalls    json.RawMessage `json:"tool_calls,omitempty"`
	InputTokens  int             `json:"input_tokens"`
	OutputTokens int             `json:"output_tokens"`
	Model        *string         `json:"model,omitempty"`
	FinishedAt   *time.Time      `json:"finished_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type ChatUsageMonthly struct {
	UserID       string `json:"user_id"`
	YearMonth    string `json:"year_month"`
	ChatCount    int    `json:"chat_count"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	OpusCount    int    `json:"opus_count"`
}
