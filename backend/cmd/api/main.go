package main

import (
	"log"
	"os"
	"time"

	"chantry/backend/internal/config"
	"chantry/backend/internal/discord"
	"chantry/backend/internal/handlers"
	_ "chantry/backend/internal/migrations" // Automatically registers Go migrations
	pbclient "chantry/backend/internal/pocketbase"
	"chantry/backend/internal/usecases"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

func main() {
	// If executed with no arguments (default, e.g. for go-server container),
	// or specifically requested with "api", run the Fiber API server.
	if len(os.Args) < 2 || os.Args[1] == "api" {
		runFiberApp()
		return
	}

	// Otherwise, run PocketBase CLI (supporting serve, migrate, etc.)
	log.Println("⚡ Starting PocketBase Server...")
	app := pocketbase.New()

	// Register migration commands to auto-run Go migrations
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: true,
	})

	if err := app.Start(); err != nil {
		log.Fatalf("❌ FATAL: PocketBase server error: %v", err)
	}
}

// runFiberApp executes the original Discord Integration daemon (Fiber API)
func runFiberApp() {
	log.Println("🚀 Starting Discord Go Daemon...")

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

	// 3. Initialize PocketBase Client & Authenticate
	pbClient := pbclient.NewClient(cfg.PocketBaseURL)
	if err := pbClient.Authenticate(cfg.PBAdminEmail, cfg.PBAdminPassword); err != nil {
		log.Fatalf("❌ FATAL: Erro ao autenticar no PocketBase: %v", err)
	}
	log.Println("✅ PocketBase Client autenticado com sucesso")

	// 4. Initialize HTTP handlers/controllers
	pbRepo := pbclient.NewRepository(pbClient)
	syncUsecase := usecases.NewSyncUsecase(discordService, pbRepo)
	syncHandler := handlers.NewSyncHandler(syncUsecase)

	discordHandler := handlers.NewDiscordHandler(discordService)

	provisionUsecase := usecases.NewProvisionUsecase(discordService, pbRepo)
	provisionHandler := handlers.NewProvisionHandler(provisionUsecase)

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
	api.Get("/discord/guilds/:guildId/categories", discordHandler.HandleGetCategories)
	api.Post("/discord/guilds/:guildId/categories", discordHandler.HandleCreateCategory)

	// Synchronization Route (Logical Upsert)
	api.Post("/sync/guilds/:guildId/members", syncHandler.HandleSyncMembers)
	api.Post("/sync/guilds/:guildId/advanced", syncHandler.HandleAdvancedSync)

	// Provisioning Route (1-on-1 Private Channels Batch)
	api.Post("/provision/guilds/:guildId/channels", provisionHandler.HandleProvisionChannels)

	// Get port from environment or fallback to 12000
	port := os.Getenv("PORT")
	if port == "" {
		port = "12000"
	}

	log.Printf("🚀 Fiber app listening on port %s...", port)
	// Start server on the designated safe port
	if err := app.Listen(":" + port); err != nil {
		panic(err)
	}
}
