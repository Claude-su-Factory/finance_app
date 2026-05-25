package config

import "github.com/caarlos0/env/v10"

type Config struct {
	Port              int    `env:"API_PORT" envDefault:"8080"`
	Env               string `env:"API_ENV" envDefault:"development"`
	DatabaseURL       string `env:"DATABASE_URL,required"`
	SupabaseJWTSecret string `env:"SUPABASE_JWT_SECRET,required"`
	CORSOrigin        string `env:"CORS_ORIGIN" envDefault:"http://localhost:3000"`
	SentryDSN         string `env:"SENTRY_DSN_API"`
	FREDAPIKey        string `env:"FRED_API_KEY"`
	ECOSAPIKey        string `env:"ECOS_API_KEY"`
	AnthropicAPIKey   string `env:"ANTHROPIC_API_KEY"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
