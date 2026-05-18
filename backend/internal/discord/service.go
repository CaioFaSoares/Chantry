package discord

// SimpleEntity represents a minimized output structure (DTO) to prevent
// internal data leaks and leak only public-facing structural properties.
type SimpleEntity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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
