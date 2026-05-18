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
}

// LoadConfig loads variables from a physical .env file if available,
// falling back to system OS/Docker injected variables, and validates credentials.
func LoadConfig() (*Config, error) {
	// Attempt to load .env. If it doesn't exist, ignore as variables
	// are expected to be injected via OS environment or Docker Compose.
	_ = godotenv.Load()

	discordAppID := os.Getenv("DISCORD_APP_ID")
	discordPublicKey := os.Getenv("DISCORD_PUBLIC_KEY")
	discordBotToken := os.Getenv("DISCORD_BOT_TOKEN")

	// Critical validation: Go daemon cannot boot or authenticate with Discord API without a Token
	if discordBotToken == "" {
		return nil, errors.New("DISCORD_BOT_TOKEN is missing or empty in the environment configuration")
	}

	return &Config{
		DiscordAppID:     discordAppID,
		DiscordPublicKey: discordPublicKey,
		DiscordBotToken:  discordBotToken,
	}, nil
}
