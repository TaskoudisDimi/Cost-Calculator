package config

import "os"

type Config struct {
	Port             string
	FirebaseProject  string
	CredentialsFile  string
	AnthropicAPIKey  string
}

func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "8080"),
		FirebaseProject: getEnv("FIREBASE_PROJECT_ID", "billcalculator-fb2dd"),
		CredentialsFile: getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
