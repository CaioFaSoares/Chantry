package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chantry/server/internal/config"
	"chantry/server/internal/cron"
	"chantry/server/internal/discord"
	"chantry/server/internal/handlers"
	_ "chantry/server/internal/migrations" // Automatically registers Go migrations
	pbclient "chantry/server/internal/pocketbase"
	"chantry/server/internal/usecases"

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

	configHandler := handlers.NewConfigHandler(pbRepo)
	reportUsecase := usecases.NewReportUsecase(pbRepo)
	reportHandler := handlers.NewReportHandler(pbRepo, reportUsecase)

	testUsecase := usecases.NewTestUsecase(pbRepo, discordService)
	testHandler := handlers.NewTestHandler(testUsecase, pbRepo)

	broadcastUsecase := usecases.NewBroadcastUsecase(discordService, pbRepo)
	broadcastHandler := handlers.NewBroadcastHandler(broadcastUsecase)

	uiUsecase := usecases.NewUIUsecase(discordService, pbRepo)
	uiHandler := handlers.NewUIHandler(uiUsecase)

	systemHandler := handlers.NewSystemHandler(pbRepo, discordService, cfg.DiscordAppID)

	// Initialize AttendanceUsecase and register Interaction handlers for Gateway
	attendanceUsecase := usecases.NewAttendanceUsecase(pbRepo)
	discord.RegisterInteractionHandlers(discordService.Session, attendanceUsecase)
	log.Println("✅ Handlers de Interação do Discord registrados")

	// Start background async Broadcast worker
	cron.StartBroadcastWorker(pbRepo, discordService)


	// Set global timezone
	loc, err := time.LoadLocation(cfg.Timezone)
	if err == nil {
		time.Local = loc
		log.Printf("🌍 Timezone set to %s", cfg.Timezone)
	} else {
		log.Printf("⚠️ Failed to load timezone %s: %v", cfg.Timezone, err)
	}

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
		now := time.Now()
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":    "ok",
			"service":   "chantry-go-daemon",
			"timestamp": now.Format(time.RFC3339),
			"timezone":  cfg.Timezone,
		})
	})

	// Discord Integration Endpoints (REST API Proxy)
	api := app.Group("/api")
	api.Get("/system/health", systemHandler.HandleGetHealth)
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
	api.Post("/provision/guilds/:guildId/heal", provisionHandler.HandleHealChannels)
	api.Get("/ui/provision-page/:guildId", provisionHandler.HandleGetProvisionPageData)

	// Targeted Broadcast Route
	api.Post("/broadcast/guilds/:guildId/send", broadcastHandler.HandleSendBroadcast)
	api.Get("/ui/broadcast-page/:guildId", broadcastHandler.HandleGetBroadcastPageData)
	api.Post("/broadcasts", broadcastHandler.HandleCreateBroadcast)
	api.Delete("/broadcasts/:id", broadcastHandler.HandleCancelBroadcast)


	// Configuration Routes (Schedules & Shifts)
	api.Get("/config/guilds/:guildId/roles", configHandler.HandleGetGuildRolesConfig)
	api.Patch("/config/roles/:roleId", configHandler.HandleUpdateRoleConfig)
	api.Patch("/config/roles/:roleId/channel", configHandler.HandleUpdateSquadChannel)
	api.Patch("/config/guilds/:guildId", configHandler.HandleUpdateGuildConfig)

	// BFF UI Routes
	api.Get("/ui/squads/:roleId", uiHandler.HandleSquadDashboard)

	// Analytical Report Routes (Daily Attendance Dashboard)
	api.Get("/reports/guilds/:guildId/attendances", reportHandler.HandleGetAttendances)
	api.Get("/reports/guilds/:guildId/export", reportHandler.HandleExportReport)

	// Sandbox Dry Run Test Routes
	api.Post("/test/attendance/trigger", testHandler.HandleTestAttendanceTrigger)
	api.Get("/test/guilds/:guildId/managers", testHandler.HandleGetManagers)

	// Start dynamic cron background scheduler
	cron.StartDynamicCron(pbRepo, discordService, cfg.Timezone)
	// Start background clock-out check ticker
	cron.StartClockOutTicker(pbRepo, discordService, cfg.Timezone)

	// Get port from environment or fallback to 12000
	port := os.Getenv("PORT")
	if port == "" {
		port = "12000"
	}

	// Start Fiber in a non-blocking goroutine
	go func() {
		log.Printf("🚀 Fiber app listening on port %s...", port)
		if err := app.Listen(":" + port); err != nil {
			log.Printf("⚠️ Fiber server listening stopped: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Println("⚡ Gracefully shutting down daemon...")

	// Close Discord Gateway connection
	if err := discordService.Close(); err != nil {
		log.Printf("⚠️ Warning: erro ao fechar sessão do Discord: %v", err)
	} else {
		log.Println("✅ Conexão WebSocket do Discord fechada com sucesso")
	}

	// Shutdown Fiber app
	if err := app.Shutdown(); err != nil {
		log.Printf("⚠️ Warning: erro ao desligar Fiber: %v", err)
	} else {
		log.Println("✅ Servidor Fiber finalizado com sucesso")
	}
}
