package handlers

import (
	"log"

	"chantry/server/internal/usecases"
	"github.com/gofiber/fiber/v2"
)

// UIHandler coordinates incoming HTTP requests specifically meant for the BFF (Streamlit)
type UIHandler struct {
	uiUsecase *usecases.UIUsecase
}

// NewUIHandler instantiates a new HTTP controller for BFF endpoints
func NewUIHandler(uiUsecase *usecases.UIUsecase) *UIHandler {
	return &UIHandler{
		uiUsecase: uiUsecase,
	}
}

// HandleSquadDashboard aggregates data for the Squad Management screen
func (h *UIHandler) HandleSquadDashboard(c *fiber.Ctx) error {
	roleID := c.Params("roleId")
	if roleID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The roleId path parameter is required",
		})
	}

	log.Printf("[UI BFF] Generating squad dashboard data for Role: %s", roleID)

	data, err := h.uiUsecase.GetSquadDashboardData(roleID)
	if err != nil {
		log.Printf("❌ ERROR [HandleSquadDashboard]: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(data)
}
