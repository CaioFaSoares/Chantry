package cron

import (
	"log"
	"time"
	_ "time/tzdata"

	"chantry/server/internal/discord"
	"chantry/server/internal/pocketbase"
)

// StartDynamicCron launches a background ticker that queries PocketBase for roles
// configured to check-in at the current minute (in the provided timezone) and sends interactive buttons.
func StartDynamicCron(repo *pocketbase.Repository, discordService *discord.DiscordService, timezone string) {
	log.Println("⏰ CRON: Starting dynamic cron scheduler...")

	// Load Timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		log.Printf("⚠️ CRON WARNING: Failed to load %s location: %v. Falling back to UTC/Local.", timezone, err)
		loc = time.Local
	} else {
		log.Printf("⏰ CRON: Using %s timezone database.", timezone)
	}

	// 1-minute ticker
	ticker := time.NewTicker(1 * time.Minute)

	// Run ticker loop in a background goroutine
	go func() {
		for range ticker.C {
			now := time.Now().In(loc)
			currentTime := now.Format("15:04")

			// Query PocketBase for roles scheduled for currentTime
			roles, err := repo.FindRolesByCheckInTime(currentTime)
			if err != nil {
				log.Printf("❌ CRON ERROR: Failed to fetch scheduled roles for %s: %v", currentTime, err)
				continue
			}

			if len(roles) == 0 {
				continue
			}

			log.Printf("⏰ CRON: Found %d role(s) scheduled for check-in at %s", len(roles), currentTime)

			for _, role := range roles {
				// Resolve the PocketBase Guild ID to check records
				var guild pocketbase.GuildRecord
				found, err := repo.FindByID("guilds", role.GuildID, &guild)
				if err != nil || !found {
					log.Printf("❌ CRON ERROR: Failed to resolve Guild ID %s for Role %s: %v", role.GuildID, role.Name, err)
					continue
				}

				// Fetch active students for this guild and role
				students, err := repo.FindActiveStudentsByRole(guild.ID, role.ID)
				if err != nil {
					log.Printf("❌ CRON ERROR: Failed to fetch active students for Role %s: %v", role.Name, err)
					continue
				}

				if len(students) == 0 {
					log.Printf("⏰ CRON: No active students mapped to Role %s", role.Name)
					continue
				}

				log.Printf("⏰ CRON: Dispatching check-in buttons for Role %s to %d active student(s)...", role.Name, len(students))

				successCount := 0
				for _, student := range students {
					if student.ChannelID == "" {
						log.Printf("⚠️ CRON WARNING: Student %s (%s) does not have a provisioned private channel. Skipping.", student.Nickname, student.DiscordID)
						continue
					}

					// Send the attendance buttons to the private channel
					err := discordService.SendAttendanceButtons(student.ChannelID)
					if err != nil {
						log.Printf("❌ CRON ERROR: Failed to send attendance buttons to channel %s for student %s: %v", student.ChannelID, student.Nickname, err)
					} else {
						successCount++
					}

					// Small safety delay (100ms) to avoid hammering Discord's API rate limits
					time.Sleep(100 * time.Millisecond)
				}

				log.Printf("✅ CRON: Successfully dispatched buttons to %d/%d student(s) in Role %s", successCount, len(students), role.Name)
			}
		}
	}()
}

// StartClockOutTicker launches a background 1-minute ticker that checks for students currently in 'pending_checkout'
// who have exceeded their role's checkout_cooldown window. If exceeded, it dispatches the check-out button prompt.
func StartClockOutTicker(repo *pocketbase.Repository, discordService *discord.DiscordService, timezone string) {
	log.Println("⏰ CLOCK-OUT TICKER: Starting clock-out dynamic ticker...")

	// Load Timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		log.Printf("⚠️ CLOCK-OUT TICKER WARNING: Failed to load %s location: %v. Falling back to UTC/Local.", timezone, err)
		loc = time.Local
	}

	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for range ticker.C {
			attendances, err := repo.FindPendingCheckouts()
			if err != nil {
				log.Printf("❌ CLOCK-OUT TICKER ERROR: Failed to retrieve pending checkouts: %v", err)
				continue
			}

			if len(attendances) == 0 {
				continue
			}

			log.Printf("⏰ CLOCK-OUT TICKER: Found %d pending attendance record(s) to verify.", len(attendances))

			for _, record := range attendances {
				if record.ClockIn == "" {
					continue
				}

				// Robust parsing of PocketBase DateTime string
				var clockInTime time.Time
				var parseErr error

				// RFC3339 format
				clockInTime, parseErr = time.Parse(time.RFC3339, record.ClockIn)
				if parseErr != nil {
					// Fallback to SQLite raw format
					clockInTime, parseErr = time.Parse("2006-01-02 15:04:05.000Z", record.ClockIn)
					if parseErr != nil {
						clockInTime, parseErr = time.Parse("2006-01-02 15:04:05", record.ClockIn)
					}
				}

				if parseErr != nil {
					log.Printf("❌ CLOCK-OUT TICKER ERROR: Failed to parse clock_in timestamp '%s' for record %s: %v", record.ClockIn, record.ID, parseErr)
					continue
				}

				// Localize in Sao Paulo timezone
				localClockIn := clockInTime.In(loc)

				// Determine cooldown duration
				cooldown := record.Expand.Student.Role.CheckoutCooldown
				if cooldown <= 0 {
					cooldown = 4 // Safe fallback (4 hours)
				}

				// Target trigger time
				targetTime := localClockIn.Add(time.Duration(cooldown) * time.Hour)
				now := time.Now().In(loc)

				log.Printf("⏰ CLOCK-OUT TICKER: Checking record %s (Student %s) - ClockIn: %s, Cooldown: %d hrs, Target: %s, Current: %s",
					record.ID, record.StudentID, localClockIn.Format("15:04:05"), cooldown, targetTime.Format("15:04:05"), now.Format("15:04:05"))

				if now.After(targetTime) {
					channelID := record.Expand.Student.ChannelID
					if channelID == "" {
						log.Printf("⚠️ CLOCK-OUT TICKER WARNING: Student %s has no channel_id configured. Skipping.", record.StudentID)
						continue
					}

					// Dispatch the button prompt to their private channel
					err := discordService.SendCheckoutPrompt(channelID)
					if err != nil {
						log.Printf("❌ CLOCK-OUT TICKER ERROR: Failed to send checkout prompt to channel %s: %v", channelID, err)
						continue
					}

					// Update database to avoid spam
					err = repo.UpdateRecord("attendances", record.ID, map[string]interface{}{
						"checkout_prompt_sent": true,
					}, nil)
					if err != nil {
						log.Printf("❌ CLOCK-OUT TICKER ERROR: Failed to update attendance record %s in PocketBase: %v", record.ID, err)
					} else {
						log.Printf("✅ CLOCK-OUT TICKER: Successfully sent checkout prompt to channel %s and flagged record %s.", channelID, record.ID)
					}

					// Sleep briefly between Discord API requests to avoid rate limits
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}()
}
