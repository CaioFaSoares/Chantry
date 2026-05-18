package main

import (
	"log"
	"os"
	"time"

	"chantry/backend/internal/config"
	"chantry/backend/internal/discord"
	"chantry/backend/internal/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	// 1. Load Configurations from environment (fails early if invalid)
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ FATAL: Erro ao carregar as configurações: %v", err)
	}

	// 2. Initialize Discord Client Service
	discordService, err := discord.NewDiscordService(cfg.DiscordBotToken)
	if err != nil {
		log.Fatalf("❌ FATAL: Erro ao inicializar o cliente Discord: %v", err)
	}
	log.Println("✅ Cliente Discord inicializado com sucesso")

	// 3. Initialize HTTP handlers/controllers
	discordHandler := handlers.NewDiscordHandler(discordService)

	// Initialize Fiber App with dynamic configuration
	app := fiber.New(fiber.Config{
		AppName: "Chantry Go Daemon v0.1.0",
	})

	// Logger Middleware for basic connection tracking
	app.Use(logger.New())

	// CORS Middleware to allow Streamlit frontend query the health route securely
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	// Healthcheck Route
	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":    "ok",
			"service":   "chantry-go-daemon",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Discord Integration Endpoints (REST API Proxy)
	api := app.Group("/api")
	api.Get("/discord/guilds", discordHandler.HandleGetGuilds)
	api.Get("/discord/guilds/:guildId/roles", discordHandler.HandleGetGuildRoles)
	api.Get("/discord/guilds/:guildId/members", discordHandler.HandleGetGuildMembers)

	// Get port from environment or fallback to 12000
	port := os.Getenv("PORT")
	if port == "" {
		port = "12000"
	}

	// Start server on the designated safe port
	if err := app.Listen(":" + port); err != nil {
		panic(err)
	}
}
