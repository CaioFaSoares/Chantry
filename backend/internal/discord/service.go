package discord

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// SimpleEntity represents a minimized output structure (DTO) to prevent
// internal data leaks and leak only public-facing structural properties.
type SimpleEntity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SimpleMember represents a minimized member output structure (DTO) to prevent
// leaking sensitive fields like email, avatar hashes, or authorization permissions.
type SimpleMember struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Nickname string   `json:"nickname"`
	Roles    []string `json:"roles"`
}

// Helpers seguros para logging
func getChannelID(ch *discordgo.Channel) string {
	if ch == nil {
		return "nil"
	}
	return ch.ID
}

func getMessageID(m *discordgo.Message) string {
	if m == nil {
		return "nil"
	}
	return m.ID
}

// GetGuilds queries the Discord REST API to fetch all servers (Guilds)
// where the bot is currently added and authorized.
func (s *DiscordService) GetGuilds() ([]SimpleEntity, error) {
	log.Printf("[DISCORD REST] Enviando GET /users/@me/guilds...")
	res, err := s.Session.UserGuilds(100, "", "", false)
	log.Printf("[DISCORD REST] GET /users/@me/guilds retornou %d guildas | Erro: %v", len(res), err)
	if err != nil {
		return nil, err
	}

	guilds := make([]SimpleEntity, 0, len(res))
	for _, g := range res {
		guilds = append(guilds, SimpleEntity{
			ID:   g.ID,
			Name: g.Name,
		})
	}

	return guilds, nil
}

// GetGuildRoles queries the Discord REST API to fetch all roles declared in a
// specific server, filtering out the default implicit '@everyone' role.
func (s *DiscordService) GetGuildRoles(guildID string) ([]SimpleEntity, error) {
	log.Printf("[DISCORD REST] Enviando GET /guilds/%s/roles...", guildID)
	res, err := s.Session.GuildRoles(guildID)
	log.Printf("[DISCORD REST] GET /guilds/%s/roles retornou %d cargos | Erro: %v", guildID, len(res), err)
	if err != nil {
		return nil, err
	}

	roles := make([]SimpleEntity, 0)
	for _, r := range res {
		if r.Name == "@everyone" {
			continue
		}
		roles = append(roles, SimpleEntity{
			ID:   r.ID,
			Name: r.Name,
		})
	}

	return roles, nil
}

// GetGuildMembersByRole fetches all server (Guild) members using cursor pagination,
// returning only the users holding the specified roleID. If roleID is empty, it returns all members.
func (s *DiscordService) GetGuildMembersByRole(guildID, roleID string) ([]SimpleMember, error) {
	filteredMembers := make([]SimpleMember, 0)
	after := ""

	log.Printf("[DISCORD REST] Iniciando busca paginada de membros da guilda: %s, filtro de cargo: %q", guildID, roleID)

	for {
		log.Printf("[DISCORD REST] Enviando GET /guilds/%s/members (after=%q, limit=1000)...", guildID, after)
		members, err := s.Session.GuildMembers(guildID, after, 1000)
		log.Printf("[DISCORD REST] GET /guilds/%s/members retornou %d membros | Erro: %v", guildID, len(members), err)
		if err != nil {
			return nil, err
		}

		if len(members) == 0 {
			break
		}

		for _, m := range members {
			if m.User == nil {
				continue
			}

			// Se roleID for vazio, aceitamos o membro. Caso contrário, checamos a presença do cargo.
			hasRole := false
			if roleID == "" {
				hasRole = true
			} else {
				for _, r := range m.Roles {
					if r == roleID {
						hasRole = true
						break
					}
				}
			}

			if hasRole {
				nickname := m.Nick
				if nickname == "" {
					nickname = m.User.GlobalName
				}
				if nickname == "" {
					nickname = m.User.Username
				}

				log.Printf("[DISCORD REST MEMBER MATCH] Membro localizado: %s (%s) | Roles: %v", m.User.Username, m.User.ID, m.Roles)

				filteredMembers = append(filteredMembers, SimpleMember{
					ID:       m.User.ID,
					Username: m.User.Username,
					Nickname: nickname,
					Roles:    m.Roles,
				})
			}
		}

		if len(members) < 1000 {
			break
		}
		after = members[len(members)-1].User.ID
	}

	log.Printf("[DISCORD REST] Finalizada busca de membros. Total filtrado/retornado: %d", len(filteredMembers))
	return filteredMembers, nil
}

// GetGuildCategories queries the Discord REST API to fetch all channels in a server,
// filtering out any channels that are not categories and sorting the remaining list by Position in ascending order.
func (s *DiscordService) GetGuildCategories(guildID string) ([]DiscordCategory, error) {
	log.Printf("[DISCORD REST] Enviando GET /guilds/%s/channels (Categories)...", guildID)
	channels, err := s.Session.GuildChannels(guildID)
	log.Printf("[DISCORD REST] GET /guilds/%s/channels retornou %d canais | Erro: %v", guildID, len(channels), err)
	if err != nil {
		return nil, err
	}

	categories := make([]DiscordCategory, 0)
	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildCategory {
			categories = append(categories, DiscordCategory{
				ID:       ch.ID,
				Name:     ch.Name,
				Position: ch.Position,
			})
		}
	}

	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Position < categories[j].Position
	})

	return categories, nil
}

// CreateCategory creates a new channel category in the specified guild.
func (s *DiscordService) CreateCategory(guildID string, name string, position int) (*DiscordCategory, error) {
	data := discordgo.GuildChannelCreateData{
		Name:     name,
		Type:     discordgo.ChannelTypeGuildCategory,
		Position: position,
	}

	log.Printf("[DISCORD REST] Enviando POST /guilds/%s/channels (Category name=%q)...", guildID, name)
	ch, err := s.Session.GuildChannelCreateComplex(guildID, data)
	log.Printf("[DISCORD REST] POST /guilds/%s/channels retornou ID: %s | Erro: %v", guildID, getChannelID(ch), err)
	if err != nil {
		return nil, err
	}

	return &DiscordCategory{
		ID:       ch.ID,
		Name:     ch.Name,
		Position: ch.Position,
	}, nil
}

// sanitizeChannelName cleans the student name by converting to lowercase, replacing spaces
// with hyphens, and removing any characters not allowed in Discord text channel names.
func sanitizeChannelName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")

	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		}
	}

	res := result.String()
	for strings.Contains(res, "--") {
		res = strings.ReplaceAll(res, "--", "-")
	}
	return strings.Trim(res, "-")
}

// CreatePrivateChannel constructs a private text channel in Discord under the parent category,
// restricting access exclusively to the target student and the list of authorized manager IDs
// while denying access to the rest of the server members (@everyone).
func (s *DiscordService) CreatePrivateChannel(
	guildID string,
	categoryID string,
	studentDiscordID string,
	studentName string,
	managerDiscordIDs []string,
) (*DiscordChannel, error) {
	channelName := "1-on-1-" + sanitizeChannelName(studentName)
	overwrites := make([]*discordgo.PermissionOverwrite, 0)

	overwrites = append(overwrites, &discordgo.PermissionOverwrite{
		ID:    guildID,
		Type:  discordgo.PermissionOverwriteTypeRole,
		Allow: 0,
		Deny:  discordgo.PermissionViewChannel,
	})

	overwrites = append(overwrites, &discordgo.PermissionOverwrite{
		ID:   studentDiscordID,
		Type: discordgo.PermissionOverwriteTypeMember,
		Allow: discordgo.PermissionViewChannel |
			discordgo.PermissionSendMessages |
			discordgo.PermissionReadMessageHistory,
		Deny: 0,
	})

	for _, managerID := range managerDiscordIDs {
		if managerID == "" {
			continue
		}
		overwrites = append(overwrites, &discordgo.PermissionOverwrite{
			ID:   managerID,
			Type: discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel |
				discordgo.PermissionSendMessages |
				discordgo.PermissionReadMessageHistory |
				discordgo.PermissionManageMessages,
			Deny: 0,
		})
	}

	data := discordgo.GuildChannelCreateData{
		Name:                 channelName,
		Type:                 discordgo.ChannelTypeGuildText,
		ParentID:             categoryID,
		PermissionOverwrites: overwrites,
	}

	log.Printf("[DISCORD REST] Enviando POST /guilds/%s/channels (Private Channel name=%q)...", guildID, channelName)
	ch, err := s.Session.GuildChannelCreateComplex(guildID, data)
	log.Printf("[DISCORD REST] POST /guilds/%s/channels retornou ID: %s | Erro: %v", guildID, getChannelID(ch), err)
	if err != nil {
		return nil, fmt.Errorf("failed to create private channel on Discord: %w", err)
	}

	return &DiscordChannel{
		ID:       ch.ID,
		Name:     ch.Name,
		ParentID: ch.ParentID,
	}, nil
}

// SendAttendanceButtons sends the interactive Check-In / Check-Out button prompt to a specific Discord channel.
func (s *DiscordService) SendAttendanceButtons(channelID string) error {
	log.Printf("[DISCORD REST] Enviando POST /channels/%s/messages (Attendance Buttons)...", channelID)
	resp, err := s.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: "☀️ **Bom dia!** Está na hora de registrar a sua presença hoje.\nUse os botões abaixo para bater o seu ponto de entrada e de saída:",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Entrada (Check-In)",
						Style:    discordgo.SuccessButton,
						CustomID: "btn_clock_in",
						Emoji: &discordgo.ComponentEmoji{
							Name: "🟢",
						},
					},
					discordgo.Button{
						Label:    "Saída (Check-Out)",
						Style:    discordgo.DangerButton,
						CustomID: "btn_clock_out",
						Emoji: &discordgo.ComponentEmoji{
							Name: "🔴",
						},
					},
				},
			},
		},
	})
	log.Printf("[DISCORD REST] POST /channels/%s/messages retornou mensagem ID: %s | Erro: %v", channelID, getMessageID(resp), err)
	return err
}

// SendCheckoutPrompt sends the interactive Clock-Out button prompt to a specific Discord channel.
func (s *DiscordService) SendCheckoutPrompt(channelID string) error {
	log.Printf("[DISCORD REST] Enviando POST /channels/%s/messages (Checkout Prompt)...", channelID)
	resp, err := s.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: "⏰ **O teu turno terminou!**\nPor favor, clica no botão abaixo para registar a saída:",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Saída (Check-Out)",
						Style:    discordgo.DangerButton,
						CustomID: "btn_clock_out",
						Emoji: &discordgo.ComponentEmoji{
							Name: "🔴",
						},
					},
				},
			},
		},
	})
	log.Printf("[DISCORD REST] POST /channels/%s/messages retornou mensagem ID: %s | Erro: %v", channelID, getMessageID(resp), err)
	return err
}

// GetGuildTextChannels queries the Discord REST API to fetch all text channels in a server.
func (s *DiscordService) GetGuildTextChannels(guildID string) ([]DiscordChannel, error) {
	log.Printf("[DISCORD REST] Enviando GET /guilds/%s/channels (Text Channels)...", guildID)
	channels, err := s.Session.GuildChannels(guildID)
	log.Printf("[DISCORD REST] GET /guilds/%s/channels (Text) retornou %d canais | Erro: %v", guildID, len(channels), err)
	if err != nil {
		return nil, err
	}

	textChannels := make([]DiscordChannel, 0)
	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildText {
			textChannels = append(textChannels, DiscordChannel{
				ID:       ch.ID,
				Name:     ch.Name,
				ParentID: ch.ParentID,
			})
		}
	}

	return textChannels, nil
}


