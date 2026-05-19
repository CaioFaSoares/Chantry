# Codebase Server Report: Chantry Go Daemon

## Overview
The **Chantry Go Daemon** is the core logic hub of the system, built using the [Fiber](https://gofiber.io/) web framework in Go. It serves a dual purpose:
1. **REST API Server:** Provides endpoints for the Streamlit frontend to manage Discord guilds, sync members, provision channels, and handle broadcasts.
2. **PocketBase Management CLI:** Integrates the PocketBase CLI, allowing for schema migrations and database management directly through the same binary.

## Project Structure (`/server`)
- **`cmd/api/main.go`:** The entry point. Handles configuration loading, service initialization (Discord, PocketBase), and route registration.
- **`internal/handlers/`:** HTTP controllers that parse requests and return JSON responses. They delegate business logic to use cases.
- **`internal/usecases/`:** Contains the core business logic (e.g., how to provision a channel, how to sync members).
- **`internal/discord/`:** Wrapper for the `discordgo` library, handling WebSocket connections and API interactions.
- **`internal/pocketbase/`:** Client and Repository layers for interacting with the PocketBase database.
- **`internal/cron/`:** Background workers for scheduled tasks (attendance prompts, broadcasts).

## Key Integration Points
- **Discord:** Uses `discordgo` for managing channels, roles, and sending interactive components (buttons).
- **PocketBase:** Uses a custom client for authentication and a repository pattern for data access.
- **Graceful Shutdown:** Implements signal handling (`SIGTERM`, `SIGINT`) to close Discord sessions and shut down the Fiber server cleanly.

## Key Symbols
- `main.runFiberApp()`: Initializes the entire application state.
- `handlers.NewSyncHandler()`: Orchestrates the synchronization of Discord members to the local DB.
- `usecases.ProvisionUsecase`: Handles the complex logic of private channel creation and permission management.
