package discord

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// DiscordService encapsulates the session client for the Discord API
type DiscordService struct {
	Session *discordgo.Session
}

// NewDiscordService instantiates a new Discord session and opens the WebSocket Gateway connection.
func NewDiscordService(botToken string) (*DiscordService, error) {
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, err
	}

	// Enable gateway intents so the bot can fetch guilds, server members list and receive interactions
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers

	// Open the WebSocket gateway connection to receive interactions in real-time
	if err := session.Open(); err != nil {
		return nil, fmt.Errorf("falha ao abrir a conexão WebSocket do Discord: %w", err)
	}
	log.Println("✅ Conexão WebSocket do Discord aberta com sucesso")

	return &DiscordService{
		Session: session,
	}, nil
}

// Close gracefully terminates the Discord Gateway connection.
func (s *DiscordService) Close() error {
	log.Println("🔌 Fechando a conexão WebSocket do Discord...")
	return s.Session.Close()
}
