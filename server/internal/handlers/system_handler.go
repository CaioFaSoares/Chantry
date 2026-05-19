package handlers

import (
	"chantry/server/internal/discord"
	"chantry/server/internal/pocketbase"
	"log"

	"github.com/gofiber/fiber/v2"
)

type SystemHandler struct {
	pbRepo         *pocketbase.Repository
	discordService *discord.DiscordService
	clientID       string
}

func NewSystemHandler(pbRepo *pocketbase.Repository, discordService *discord.DiscordService, clientID string) *SystemHandler {
	return &SystemHandler{
		pbRepo:         pbRepo,
		discordService: discordService,
		clientID:       clientID,
	}
}

func (h *SystemHandler) HandleGetHealth(c *fiber.Ctx) error {
	// 1. Check Discord WS state
	discordWS := "disconnected"
	if h.discordService != nil && h.discordService.Session != nil && h.discordService.Session.State.User != nil {
		discordWS = "connected"
	}

	// 2. Query counts from PocketBase and determine PB health
	pbStatus := "healthy"
	totalGuilds, err := h.pbRepo.CountRecords("guilds")
	if err != nil {
		log.Printf("⚠️ [HealthCheck] Failed to count guilds: %v", err)
		pbStatus = "unhealthy"
	}

	totalStudents, err := h.pbRepo.CountRecords("students")
	if err != nil {
		log.Printf("⚠️ [HealthCheck] Failed to count students: %v", err)
		pbStatus = "unhealthy"
	}

	totalAttendances, err := h.pbRepo.CountRecords("attendances")
	if err != nil {
		log.Printf("⚠️ [HealthCheck] Failed to count attendances: %v", err)
		pbStatus = "unhealthy"
	}

	// 3. Assemble response following the PRD contract
	response := fiber.Map{
		"status": "online",
		"services": fiber.Map{
			"go_daemon":  "healthy",
			"pocketbase": pbStatus,
			"discord_ws": discordWS,
		},
		"metrics": fiber.Map{
			"total_guilds":      totalGuilds,
			"total_students":    totalStudents,
			"total_attendances": totalAttendances,
		},
		"env": fiber.Map{
			"discord_client_id": h.clientID,
		},
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
