package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// Config represents all backend environment configurations
type Config struct {
	DiscordAppID     string
	DiscordPublicKey string
	DiscordBotToken  string
	PocketBaseURL    string
	PBAdminEmail     string
	PBAdminPassword  string
	Timezone         string
}

// LoadConfig loads variables from a physical .env file if available,
// falling back to system OS/Docker injected variables, and validates credentials.
func LoadConfig() (*Config, error) {
	// Attempt to load .env. If it doesn't exist, ignore as variables
	// are expected to be injected via OS environment or Docker Compose.
	_ = godotenv.Load()

	timezone := os.Getenv("TIMEZONE")
	if timezone == "" {
		timezone = "America/Sao_Paulo"
	}

	discordAppID := os.Getenv("DISCORD_APP_ID")
	discordPublicKey := os.Getenv("DISCORD_PUBLIC_KEY")
	discordBotToken := os.Getenv("DISCORD_BOT_TOKEN")

	// Critical validation: Go daemon cannot boot or authenticate with Discord API without a Token
	if discordBotToken == "" {
		return nil, errors.New("DISCORD_BOT_TOKEN is missing or empty in the environment configuration")
	}

	pocketBaseURL := os.Getenv("POCKETBASE_URL")
	if pocketBaseURL == "" {
		pocketBaseURL = "http://pocketbase:8090"
	}
	if len(pocketBaseURL) > 0 && pocketBaseURL[len(pocketBaseURL)-1] != '/' {
		pocketBaseURL += "/"
	}

	pbAdminEmail := os.Getenv("PB_ADMIN_EMAIL")
	pbAdminPassword := os.Getenv("PB_ADMIN_PASSWORD")

	if pbAdminEmail == "" {
		return nil, errors.New("PB_ADMIN_EMAIL is missing or empty in the environment configuration")
	}
	if pbAdminPassword == "" {
		return nil, errors.New("PB_ADMIN_PASSWORD is missing or empty in the environment configuration")
	}

	return &Config{
		DiscordAppID:     discordAppID,
		DiscordPublicKey: discordPublicKey,
		DiscordBotToken:  discordBotToken,
		PocketBaseURL:    pocketBaseURL,
		PBAdminEmail:     pbAdminEmail,
		PBAdminPassword:  pbAdminPassword,
		Timezone:         timezone,
	}, nil
}
