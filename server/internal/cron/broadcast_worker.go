package cron

import (
	"log"
	"time"

	"chantry/server/internal/discord"
	"chantry/server/internal/pocketbase"
	"chantry/server/internal/usecases"
)

// StartBroadcastWorker launches a background ticker running every 1 minute
// that searches for broadcasts scheduled for the current minute (or past scheduled but unprocessed)
// and executes them, updating status and computing delivery metrics.
func StartBroadcastWorker(repo *pocketbase.Repository, discordService *discord.DiscordService) {
	log.Println("⏰ BROADCAST WORKER: Starting message broadcast background worker...")

	// Initialize the broadcast usecase to reuse its highly tested delivery engine
	broadcastUsecase := usecases.NewBroadcastUsecase(discordService, repo)

	// Encapsulate the processing logic to allow immediate execution
	processBroadcasts := func() {
		// Query scheduled broadcasts that have schedule_time <= now (in UTC)
		nowUTC := time.Now().UTC()
		// PocketBase strictly uses YYYY-MM-DD HH:mm:ss.SSSZ for accurate string date comparison
		cutoff := nowUTC.Format("2006-01-02 15:04:05.000Z")

		scheduled, err := repo.FindScheduledBroadcastsBefore(cutoff)
		if err != nil {
			log.Printf("❌ BROADCAST WORKER ERROR: Failed to fetch scheduled broadcasts: %v", err)
			return
		}

		if len(scheduled) == 0 {
			return
		}

		log.Printf("⏰ BROADCAST WORKER: Found %d scheduled broadcast(s) ready to process.", len(scheduled))

		for _, broadcast := range scheduled {
			log.Printf("📢 BROADCAST WORKER: Processing broadcast %s (Guild ID: %s, Target Type: %s)...", broadcast.ID, broadcast.GuildID, broadcast.TargetType)

			// 1. Immediately change status to "processing" to lock the record
			err = repo.UpdateRecord("broadcasts", broadcast.ID, map[string]interface{}{
				"status": "processing",
			}, nil)
			if err != nil {
				log.Printf("❌ BROADCAST WORKER ERROR: Failed to set status to 'processing' for broadcast %s: %v. Skipping.", broadcast.ID, err)
				continue
			}

			// 2. Fetch the Guild's Discord Snowflake ID from PB
			var guild pocketbase.GuildRecord
			found, err := repo.FindByID("guilds", broadcast.GuildID, &guild)
			if err != nil || !found {
				log.Printf("❌ BROADCAST WORKER ERROR: Failed to resolve Guild ID %s for broadcast %s: %v. Marking failed.", broadcast.GuildID, broadcast.ID, err)
				_ = repo.UpdateRecord("broadcasts", broadcast.ID, map[string]interface{}{
					"status":         "failed",
					"metrics_errors": 1,
				}, nil)
				continue
			}

			// 3. Trigger delivery through our robust Sync/Send engine
			metrics, err := broadcastUsecase.SendBroadcast(guild.DiscordID, broadcast.Content, broadcast.TargetType, broadcast.TargetRoles)

			// 4. Update the record with delivery results and final status
			status := "completed"
			if err != nil && metrics.MessagesSent == 0 {
				status = "failed"
			}

			err = repo.UpdateRecord("broadcasts", broadcast.ID, map[string]interface{}{
				"status":         status,
				"metrics_sent":   metrics.MessagesSent,
				"metrics_errors": metrics.Errors,
			}, nil)

			if err != nil {
				log.Printf("❌ BROADCAST WORKER ERROR: Failed to update final status of broadcast %s in PB: %v", broadcast.ID, err)
			} else {
				log.Printf("✅ BROADCAST WORKER: Broadcast %s finished with status '%s' (Sent: %d, Errors: %d)",
					broadcast.ID, status, metrics.MessagesSent, metrics.Errors)
			}
		}
	}

	// Execute immediately once to catch up on any missed broadcasts while offline
	go processBroadcasts()

	// 1-minute ticker
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for range ticker.C {
			processBroadcasts()
		}
	}()
}
