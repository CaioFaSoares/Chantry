package discord

import (
	"github.com/bwmarrin/discordgo"
)

// DiscordService encapsulates the session client for the Discord API
type DiscordService struct {
	Session *discordgo.Session
}

// NewDiscordService instantiates a new Discord REST service session.
// This does not open a WebSocket gateway connection, keeping interactions lightweight.
func NewDiscordService(botToken string) (*DiscordService, error) {
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, err
	}

	return &DiscordService{
		Session: session,
	}, nil
}
