package handlers

import (
	"log"

	"chantry/server/internal/usecases"
	"github.com/gofiber/fiber/v2"
)

// SyncHandler coordinates incoming synchronization HTTP requests
type SyncHandler struct {
	syncUsecase *usecases.SyncUsecase
}

// NewSyncHandler creates a new instance of SyncHandler
func NewSyncHandler(usecase *usecases.SyncUsecase) *SyncHandler {
	return &SyncHandler{
		syncUsecase: usecase,
	}
}

// SyncMembersRequest maps the incoming POST request body
type SyncMembersRequest struct {
	RoleID string `json:"role_id"`
}

// HandleSyncMembers parses inputs, triggers the synchronization usecase, and returns the mutational metrics
func (h *SyncHandler) HandleSyncMembers(c *fiber.Ctx) error {
	guildID := c.Params("guildId")
	if guildID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The guildId path parameter is required in the route",
		})
	}

	var req SyncMembersRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or malformed JSON request body",
		})
	}

	if req.RoleID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The role_id field is required in the JSON body",
		})
	}

	log.Printf("[SYNC] Triggered member synchronization for Guild: %s, Role: %s", guildID, req.RoleID)
	metrics, err := h.syncUsecase.SyncStudentsByRole(guildID, req.RoleID)
	if err != nil {
		log.Printf("❌ ERROR [SyncMembers] for Guild ID %s and Role ID %s: %v", guildID, req.RoleID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Sincronização concluída com sucesso",
		"metrics": metrics,
	})
}

// HandleAdvancedSync handles the advanced multi-role and manager sync request
func (h *SyncHandler) HandleAdvancedSync(c *fiber.Ctx) error {
	guildID := c.Params("guildId")
	if guildID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The guildId path parameter is required in the route",
		})
	}

	var payload usecases.AdvancedSyncPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or malformed JSON request body",
		})
	}

	log.Printf("[ADV-SYNC] Triggered advanced synchronization for Guild: %s", guildID)
	metrics, err := h.syncUsecase.AdvancedSync(guildID, payload)
	if err != nil {
		log.Printf("❌ ERROR [AdvancedSync] for Guild ID %s: %v", guildID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Sincronização avançada concluída com sucesso",
		"metrics": metrics,
	})
}
