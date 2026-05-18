package discord

import (
	"log"
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
