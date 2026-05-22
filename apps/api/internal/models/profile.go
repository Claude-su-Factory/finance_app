package models

import "time"

type Profile struct {
	ID                   string    `json:"id"`
	DisplayName          *string   `json:"display_name"`
	BaseCurrency         string    `json:"base_currency"`
	UIIntensity          string    `json:"ui_intensity"`
	OnboardingCompleted  bool      `json:"onboarding_completed"`
	DailyBriefingEnabled bool      `json:"daily_briefing_enabled"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}
