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

// GetGuilds queries the Discord REST API to fetch all servers (Guilds)
// where the bot is currently added and authorized.
func (s *DiscordService) GetGuilds() ([]SimpleEntity, error) {
	// Query user guilds, fetching up to 100 entries, using false for member/local count inclusion.
	res, err := s.Session.UserGuilds(100, "", "", false)
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
	res, err := s.Session.GuildRoles(guildID)
	if err != nil {
		return nil, err
	}

	roles := make([]SimpleEntity, 0)
	for _, r := range res {
		// Ignore the default everyone group role which represents the entire server membership base
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
// returning only the users holding the specified roleID mapped to our SimpleMember DTO.
func (s *DiscordService) GetGuildMembersByRole(guildID, roleID string) ([]SimpleMember, error) {
	filteredMembers := make([]SimpleMember, 0)
	after := ""

	for {
		// Query members list page, max 1000 items starting after the given user ID.
		members, err := s.Session.GuildMembers(guildID, after, 1000)
		if err != nil {
			return nil, err
		}

		// Empty response means we've successfully iterated through the entire membership list.
		if len(members) == 0 {
			break
		}

		for _, m := range members {
			// Skip users who have null profiles or empty credentials (defensive checks)
			if m.User == nil {
				continue
			}

			// Verify if the user possesses the target role ID
			hasRole := false
			for _, r := range m.Roles {
				if r == roleID {
					hasRole = true
					break
				}
			}

			if hasRole {
				// Hierarchical nickname resolution
				nickname := m.Nick
				if nickname == "" {
					nickname = m.User.GlobalName
				}
				if nickname == "" {
					nickname = m.User.Username
				}

				log.Printf("[DEBUG-NICK] User: %s (%s) | Nick: %q | GlobalName: %q | ResolvedNickname: %q", 
					m.User.Username, m.User.ID, m.Nick, m.User.GlobalName, nickname)

				filteredMembers = append(filteredMembers, SimpleMember{
					ID:       m.User.ID,
					Username: m.User.Username,
					Nickname: nickname,
					Roles:    m.Roles,
				})
			}
		}

		// If we fetched fewer items than the max limit of 1000, we've naturally reached the end
		if len(members) < 1000 {
			break
		}

		// Update the pagination cursor to start after the last processed user's ID
		after = members[len(members)-1].User.ID
	}

	return filteredMembers, nil
}

// GetGuildCategories queries the Discord REST API to fetch all channels in a server,
// filtering out any channels that are not categories and sorting the remaining list by Position in ascending order.
func (s *DiscordService) GetGuildCategories(guildID string) ([]DiscordCategory, error) {
	channels, err := s.Session.GuildChannels(guildID)
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

	// Sort ascending by Position (0 is at the top)
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

	ch, err := s.Session.GuildChannelCreateComplex(guildID, data)
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
	// Remove multiple consecutive hyphens
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
	// 1. Sanitize the channel name (e.g., 1-on-1-joaozinho)
	channelName := "1-on-1-" + sanitizeChannelName(studentName)

	// 2. Build permission overwrites slice
	overwrites := make([]*discordgo.PermissionOverwrite, 0)

	// A. Deny ViewChannel permission to @everyone role (Role ID is always identical to Guild ID)
	overwrites = append(overwrites, &discordgo.PermissionOverwrite{
		ID:    guildID,
		Type:  discordgo.PermissionOverwriteTypeRole,
		Allow: 0,
		Deny:  discordgo.PermissionViewChannel,
	})

	// B. Allow ViewChannel, SendMessages and ReadMessageHistory to the target Student
	overwrites = append(overwrites, &discordgo.PermissionOverwrite{
		ID:   studentDiscordID,
		Type: discordgo.PermissionOverwriteTypeMember,
		Allow: discordgo.PermissionViewChannel |
			discordgo.PermissionSendMessages |
			discordgo.PermissionReadMessageHistory,
		Deny: 0,
	})

	// C. Allow ViewChannel, SendMessages, ReadMessageHistory and ManageMessages to all Guild Managers
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

	// 3. Assemble native discordgo structure
	data := discordgo.GuildChannelCreateData{
		Name:                 channelName,
		Type:                 discordgo.ChannelTypeGuildText,
		ParentID:             categoryID,
		PermissionOverwrites: overwrites,
	}

	// 4. Issue the REST request to create the channel
	ch, err := s.Session.GuildChannelCreateComplex(guildID, data)
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
	_, err := s.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
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
	return err
}

// SendCheckoutPrompt sends the interactive Clock-Out button prompt to a specific Discord channel.
func (s *DiscordService) SendCheckoutPrompt(channelID string) error {
	_, err := s.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
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
	return err
}
