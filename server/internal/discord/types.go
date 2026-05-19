package discord

// DiscordCategory represents a minimized category output DTO to prevent internal data leaks
type DiscordCategory struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

// CreateCategoryRequest represents the HTTP request payload needed to create a new Discord channel category
type CreateCategoryRequest struct {
	Name     string `json:"name"`
	Position int    `json:"position"`
}

// DiscordChannel represents a minimized category output DTO to prevent internal data leaks
type DiscordChannel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parent_id"` // Categoria Pai
}
