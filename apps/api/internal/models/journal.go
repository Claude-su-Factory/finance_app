package models

import (
	"encoding/json"
	"time"
)

type JournalEntry struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	EntryType         string    `json:"entry_type"` // "auto" | "manual"
	Action            *string   `json:"action,omitempty"`
	RelatedHoldingID  *string   `json:"related_holding_id,omitempty"`
	RelatedHolding    *RelatedHoldingBrief `json:"related_holding,omitempty"`
	RelatedSymbols    []string  `json:"related_symbols"`
	Title             *string   `json:"title,omitempty"`
	Content           string    `json:"content"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// RelatedHoldingBrief — entry 노출 시 종목 정보 (read 시 JOIN으로 채움)
type RelatedHoldingBrief struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

type AnalysisRun struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	RunType       string    `json:"run_type"` // "auto_monthly" | "on_demand"
	PeriodStart   string    `json:"period_start"` // YYYY-MM-DD
	PeriodEnd     string    `json:"period_end"`
	EntriesCount  int       `json:"entries_count"`
	ContentMD     string    `json:"content_md"`
	Model         string    `json:"model"`
	CreatedAt     time.Time `json:"created_at"`
}

// JournalEntryCreate — POST body
type JournalEntryCreate struct {
	Action          *string  `json:"action"`
	RelatedSymbols  []string `json:"related_symbols"`
	Title           *string  `json:"title"`
	Content         string   `json:"content"`
}

// JournalEntryPatch — PATCH body
type JournalEntryPatch struct {
	Action          *string  `json:"action"`
	RelatedSymbols  *[]string `json:"related_symbols"`
	Title           *string  `json:"title"`
	Content         *string  `json:"content"`
}

// json.RawMessage import 가드
var _ json.RawMessage
