package config

import "os"

type Config struct {
	Port                string
	FirebaseProject     string
	CredentialsFile     string
	AnthropicAPIKey     string
	SchedulerSecret     string
	NordigenSecretID    string
	NordigenSecretKey   string
	NordigenRedirectURL string
}

func Load() *Config {
	return &Config{
		Port:                getEnv("PORT", "8080"),
		FirebaseProject:     getEnv("FIREBASE_PROJECT_ID", "billcalculator-fb2dd"),
		CredentialsFile:     getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
		AnthropicAPIKey:     getEnv("ANTHROPIC_API_KEY", ""),
		SchedulerSecret:     getEnv("SCHEDULER_SECRET", ""),
		NordigenSecretID:    getEnv("NORDIGEN_SECRET_ID", ""),
		NordigenSecretKey:   getEnv("NORDIGEN_SECRET_KEY", ""),
		NordigenRedirectURL: getEnv("NORDIGEN_REDIRECT_URL", "https://billcalculator-fb2dd.web.app/bank/callback"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
