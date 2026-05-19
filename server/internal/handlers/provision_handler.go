package handlers

import (
	"log"

	"chantry/server/internal/usecases"
	"github.com/gofiber/fiber/v2"
)

type ProvisionHandler struct {
	provisionUsecase *usecases.ProvisionUsecase
}

// NewProvisionHandler instantiates a new ProvisionHandler.
func NewProvisionHandler(usecase *usecases.ProvisionUsecase) *ProvisionHandler {
	return &ProvisionHandler{
		provisionUsecase: usecase,
	}
}

type ProvisionChannelsRequest struct {
	CategoryID string `json:"category_id"`
	RoleID     string `json:"role_id"`
}

// HandleProvisionChannels parses the HTTP inputs, invokes the ProvisionUsecase, and returns metrics.
func (h *ProvisionHandler) HandleProvisionChannels(c *fiber.Ctx) error {
	guildID := c.Params("guildId")
	if guildID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The guildId path parameter is required in the route",
		})
	}

	var req ProvisionChannelsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or malformed JSON request body",
		})
	}

	if req.CategoryID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The category_id field is required in the JSON body",
		})
	}

	if req.RoleID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The role_id field is required in the JSON body",
		})
	}

	log.Printf("[PROVISION-API] Triggered batch provisioning for Guild: %s, Category: %s, Role: %s", guildID, req.CategoryID, req.RoleID)
	metrics, err := h.provisionUsecase.BatchCreatePrivateChannels(guildID, req.CategoryID, req.RoleID)
	if err != nil {
		log.Printf("❌ ERROR [HandleProvisionChannels] for Guild ID %s: %v", guildID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Provisionamento concluído",
		"metrics": metrics,
	})
}

type HealChannelsRequest struct {
	CategoryID string `json:"category_id"`
}

// HandleHealChannels parses path params and body, invokes the Usecase to heal student channel mapping, and returns HealMetrics.
func (h *ProvisionHandler) HandleHealChannels(c *fiber.Ctx) error {
	guildID := c.Params("guildId")
	if guildID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The guildId path parameter is required in the route",
		})
	}

	var req HealChannelsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or malformed JSON request body",
		})
	}

	if req.CategoryID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The category_id field is required in the JSON body",
		})
	}

	log.Printf("[PROVISION-API] Triggered channel Auto-Healing for Guild: %s, Category: %s", guildID, req.CategoryID)
	metrics, err := h.provisionUsecase.HealChannelsByCategory(guildID, req.CategoryID)
	if err != nil {
		log.Printf("❌ ERROR [HandleHealChannels] for Guild ID %s: %v", guildID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Processo de auto-healing concluído",
		"metrics": metrics,
	})
}

// HandleGetProvisionPageData handles GET /api/ui/provision-page/:guildId, returning the aggregated data for the page.
func (h *ProvisionHandler) HandleGetProvisionPageData(c *fiber.Ctx) error {
	guildID := c.Params("guildId")
	if guildID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The guildId path parameter is required in the route",
		})
	}

	data, err := h.provisionUsecase.GetProvisionPageData(guildID)
	if err != nil {
		log.Printf("❌ ERROR [HandleGetProvisionPageData] for Guild ID %s: %v", guildID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(data)
}

