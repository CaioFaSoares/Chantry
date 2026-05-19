package handlers

import (
	"log"

	"chantry/server/internal/pocketbase"
	"chantry/server/internal/usecases"

	"github.com/gofiber/fiber/v2"
)

type TestHandler struct {
	testUsecase *usecases.TestUsecase
	pbRepo      *pocketbase.Repository
}

func NewTestHandler(testUsecase *usecases.TestUsecase, pbRepo *pocketbase.Repository) *TestHandler {
	return &TestHandler{
		testUsecase: testUsecase,
		pbRepo:      pbRepo,
	}
}

type TriggerTestClockInRequest struct {
	GuildID         string `json:"guild_id"`
	ChannelID       string `json:"channel_id"`
	TesterDiscordID string `json:"tester_discord_id"`
}

// HandleTestAttendanceTrigger parses parameters and invokes TriggerTestClockIn on the usecase.
func (h *TestHandler) HandleTestAttendanceTrigger(c *fiber.Ctx) error {
	var req TriggerTestClockInRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or malformed JSON request body",
		})
	}

	if req.GuildID == "" || req.ChannelID == "" || req.TesterDiscordID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "guild_id, channel_id, and tester_discord_id are all required",
		})
	}

	if err := h.testUsecase.TriggerTestClockIn(req.GuildID, req.ChannelID, req.TesterDiscordID); err != nil {
		log.Printf("❌ ERROR [TestAttendanceTrigger]: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to trigger clock-in test: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Mensagem de teste enviada com sucesso para o canal.",
	})
}

// HandleGetManagers returns the list of managers associated with a guild.
func (h *TestHandler) HandleGetManagers(c *fiber.Ctx) error {
	guildDiscordID := c.Params("guildId")
	if guildDiscordID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The guildId parameter is required in the path route",
		})
	}

	// 1. Resolve Discord Guild ID to PocketBase Guild Record
	var guild pocketbase.GuildRecord
	found, err := h.pbRepo.FindFirstByDiscordID("guilds", guildDiscordID, &guild)
	if err != nil {
		log.Printf("❌ ERROR [GetManagers] resolving Guild %s: %v", guildDiscordID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to resolve Guild: " + err.Error(),
		})
	}
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Guild not found in local database mapping",
		})
	}

	// 2. Fetch managers associated with this guild
	managers, err := h.pbRepo.FindManagersByGuild(guild.ID)
	if err != nil {
		log.Printf("❌ ERROR [GetManagers] fetching managers for Guild %s: %v", guild.ID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch managers: " + err.Error(),
		})
	}

	return c.JSON(managers)
}
