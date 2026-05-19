package handlers

import (
	"log"
	"regexp"

	"chantry/server/internal/pocketbase"
	"github.com/gofiber/fiber/v2"
)

// ConfigHandler coordinates incoming HTTP API requests for system configurations.
type ConfigHandler struct {
	repo *pocketbase.Repository
}

// NewConfigHandler instantiates a new HTTP controller for configuration endpoints.
func NewConfigHandler(repo *pocketbase.Repository) *ConfigHandler {
	return &ConfigHandler{
		repo: repo,
	}
}

// HandleGetGuildRolesConfig fetches roles and their configurations associated with a Discord guild ID.
func (h *ConfigHandler) HandleGetGuildRolesConfig(c *fiber.Ctx) error {
	guildDiscordID := c.Params("guildId")
	if guildDiscordID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The guildId parameter is required in the path route",
		})
	}

	// 1. Resolve Discord Guild ID to PocketBase Guild Record
	var guild pocketbase.GuildRecord
	found, err := h.repo.FindFirstByDiscordID("guilds", guildDiscordID, &guild)
	if err != nil {
		log.Printf("❌ ERROR [GetGuildRolesConfig] resolving Guild %s: %v", guildDiscordID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to resolve Guild: " + err.Error(),
		})
	}
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Guild not found in local database mapping",
		})
	}

	// 2. Fetch all mapped roles for that Guild from PocketBase
	roles, err := h.repo.FindRolesByGuild(guild.ID)
	if err != nil {
		log.Printf("❌ ERROR [GetGuildRolesConfig] fetching roles for Guild %s: %v", guild.ID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch roles from database: " + err.Error(),
		})
	}

	return c.JSON(roles)
}

// UpdateRoleConfigRequest maps the incoming request body for PATCH /api/config/roles/:roleId
type UpdateRoleConfigRequest struct {
	Shift            string `json:"shift"`
	CheckInTime      string `json:"check_in_time"`
	CheckoutCooldown int    `json:"checkout_cooldown"`
	IsMonitored      *bool  `json:"is_monitored"`
	IsActive         *bool  `json:"is_active"`
}

// HandleUpdateRoleConfig updates the turn and schedule configuration of a target role.
func (h *ConfigHandler) HandleUpdateRoleConfig(c *fiber.Ctx) error {
	roleID := c.Params("roleId")
	if roleID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The roleId parameter is required in the path route",
		})
	}

	var req UpdateRoleConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body format",
		})
	}

	// 1. Validate Shift Value if provided
	if req.Shift != "" && req.Shift != "morning" && req.Shift != "afternoon" && req.Shift != "night" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid shift. Must be one of: morning, afternoon, night",
		})
	}

	// 2. Validate check_in_time format (must be "HH:MM" if provided)
	if req.CheckInTime != "" {
		matched, _ := regexp.MatchString(`^(0[0-9]|1[0-9]|2[0-3]):[0-5][0-9]$`, req.CheckInTime)
		if !matched {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid check_in_time format. Must be HH:MM (24-hour), e.g. 08:00 or 14:30",
			})
		}
	}

	// 3. Prepare data map for PocketBase UpdateRecord conditionally
	updateData := map[string]interface{}{}
	if req.Shift != "" {
		updateData["shift"] = req.Shift
	}
	if req.CheckInTime != "" {
		updateData["check_in_time"] = req.CheckInTime
	}
	if req.CheckoutCooldown > 0 {
		updateData["checkout_cooldown"] = req.CheckoutCooldown
	}
	if req.IsMonitored != nil {
		updateData["is_monitored"] = *req.IsMonitored
	}
	if req.IsActive != nil {
		updateData["is_active"] = *req.IsActive
	}

	var updated pocketbase.RoleRecord
	if err := h.repo.UpdateRecord("roles", roleID, &updateData, &updated); err != nil {
		log.Printf("❌ ERROR [UpdateRoleConfig] updating Role ID %s: %v", roleID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update role configuration in PocketBase: " + err.Error(),
		})
	}

	log.Printf("✅ CONFIG: Updated Role %s (%s): Shift=%s, CheckInTime=%s, Cooldown=%d, Monitored=%v, Active=%v",
		updated.Name, updated.ID, updated.Shift, updated.CheckInTime, updated.CheckoutCooldown, updated.IsMonitored, updated.IsActive)

	return c.Status(fiber.StatusOK).JSON(updated)
}

// UpdateGuildConfigRequest maps the incoming request body for PATCH /api/config/guilds/:guildId
type UpdateGuildConfigRequest struct {
	AnnouncementChannelID string `json:"announcement_channel_id"`
}

// HandleUpdateGuildConfig updates the guild configuration in PocketBase (such as the announcement channel).
func (h *ConfigHandler) HandleUpdateGuildConfig(c *fiber.Ctx) error {
	guildDiscordID := c.Params("guildId")
	if guildDiscordID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The guildId path parameter is required in the route",
		})
	}

	var req UpdateGuildConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body format",
		})
	}

	// 1. Resolve Discord Guild ID to PocketBase Guild Record
	var guild pocketbase.GuildRecord
	found, err := h.repo.FindFirstByDiscordID("guilds", guildDiscordID, &guild)
	if err != nil {
		log.Printf("❌ ERROR [UpdateGuildConfig] resolving Guild %s: %v", guildDiscordID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to resolve Guild: " + err.Error(),
		})
	}
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Guild not found in local database mapping",
		})
	}

	// 2. Perform partial update (PATCH) of announcement channel ID
	updateData := map[string]interface{}{
		"announcement_channel_id": req.AnnouncementChannelID,
	}

	var updated pocketbase.GuildRecord
	if err := h.repo.UpdateRecord("guilds", guild.ID, &updateData, &updated); err != nil {
		log.Printf("❌ ERROR [UpdateGuildConfig] updating Guild ID %s: %v", guild.ID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update guild configuration in PocketBase: " + err.Error(),
		})
	}

	log.Printf("✅ CONFIG: Updated Guild Announcement Channel for %s (%s): %s",
		updated.Name, updated.DiscordID, updated.AnnouncementChannelID)

	return c.Status(fiber.StatusOK).JSON(updated)
}


// UpdateSquadChannelRequest represents the payload to update a role's squad_channel_id
type UpdateSquadChannelRequest struct {
	SquadChannelID string `json:"squad_channel_id"`
}

// HandleUpdateSquadChannel updates only the squad_channel_id for a specific role
func (h *ConfigHandler) HandleUpdateSquadChannel(c *fiber.Ctx) error {
	roleID := c.Params("roleId")
	if roleID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The roleId parameter is required",
		})
	}

	var req UpdateSquadChannelRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid JSON body format",
		})
	}

	payload := map[string]interface{}{
		"squad_channel_id": req.SquadChannelID,
	}

	err := h.repo.UpdateRecord("roles", roleID, payload, nil)
	if err != nil {
		log.Printf("❌ ERROR [UpdateSquadChannel] for Role ID %s: %v", roleID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update squad channel in database",
		})
	}

	log.Printf("✅ [ConfigHandler] Squad channel for Role ID %s successfully updated", roleID)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Squad channel updated successfully",
	})
}
