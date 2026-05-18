package handlers

import (
	"log"

	"chantry/backend/internal/discord"
	"github.com/gofiber/fiber/v2"
)

// DiscordHandler coordinates incoming HTTP API requests for Discord integrations
type DiscordHandler struct {
	DiscordService *discord.DiscordService
}

// NewDiscordHandler instantiates a new HTTP controller for Discord endpoints
func NewDiscordHandler(service *discord.DiscordService) *DiscordHandler {
	return &DiscordHandler{
		DiscordService: service,
	}
}

// HandleGetGuilds lists all servers (Guilds) where the bot client is currently active
func (h *DiscordHandler) HandleGetGuilds(c *fiber.Ctx) error {
	guilds, err := h.DiscordService.GetGuilds()
	if err != nil {
		log.Printf("❌ ERROR [GetGuilds]: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(guilds)
}

// HandleGetGuildRoles lists all custom roles associated with a specific server/guild
func (h *DiscordHandler) HandleGetGuildRoles(c *fiber.Ctx) error {
	guildID := c.Params("guildId")
	if guildID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "guildId parameter is required in the path route",
		})
	}

	roles, err := h.DiscordService.GetGuildRoles(guildID)
	if err != nil {
		log.Printf("❌ ERROR [GetGuildRoles] for Guild ID %s: %v", guildID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(roles)
}
