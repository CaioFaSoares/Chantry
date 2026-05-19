package handlers

import (
	"log"

	"chantry/server/internal/discord"
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

// HandleGetGuildMembers fetches and returns server (Guild) members. If roleId query parameter is supplied, it filters by role.
func (h *DiscordHandler) HandleGetGuildMembers(c *fiber.Ctx) error {
	guildID := c.Params("guildId")
	roleID := c.Query("roleId")

	if guildID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "guildId parameter is required in the path route",
		})
	}

	members, err := h.DiscordService.GetGuildMembersByRole(guildID, roleID)
	if err != nil {
		log.Printf("❌ ERROR [GetGuildMembers] for Guild ID %s and Role ID %q: %v", guildID, roleID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(members)
}

// HandleGetCategories lists all categories created on a Discord server
func (h *DiscordHandler) HandleGetCategories(c *fiber.Ctx) error {
	guildID := c.Params("guildId")
	if guildID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "guildId parameter is required in the path route",
		})
	}

	categories, err := h.DiscordService.GetGuildCategories(guildID)
	if err != nil {
		log.Printf("❌ ERROR [GetGuildCategories] for Guild ID %s: %v", guildID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve Discord categories: " + err.Error(),
		})
	}

	return c.JSON(categories)
}

// HandleCreateCategory creates a new channel category on a Discord server
func (h *DiscordHandler) HandleCreateCategory(c *fiber.Ctx) error {
	guildID := c.Params("guildId")
	if guildID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "guildId parameter is required in the path route",
		})
	}

	var req discord.CreateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body format",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Category name cannot be empty",
		})
	}

	category, err := h.DiscordService.CreateCategory(guildID, req.Name, req.Position)
	if err != nil {
		log.Printf("❌ ERROR [CreateCategory] for Guild ID %s with Name %q: %v", guildID, req.Name, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create category on Discord: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(category)
}
