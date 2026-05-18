package usecases

import (
	"fmt"
	"log"
	"time"

	"chantry/backend/internal/discord"
	"chantry/backend/internal/pocketbase"
)

// ProvisionMetrics holds execution metrics for the batch provisioning run.
type ProvisionMetrics struct {
	TotalStudents      int `json:"total_students"`
	ChannelsCreated    int `json:"channels_created"`
	AlreadyProvisioned int `json:"already_provisioned"`
	Errors             int `json:"errors"`
}

type ProvisionUsecase struct {
	DiscordService *discord.DiscordService
	PBRepository   *pocketbase.Repository
}

// NewProvisionUsecase instantiates a new ProvisionUsecase.
func NewProvisionUsecase(ds *discord.DiscordService, pbr *pocketbase.Repository) *ProvisionUsecase {
	return &ProvisionUsecase{
		DiscordService: ds,
		PBRepository:   pbr,
	}
}

// BatchCreatePrivateChannels orchestrates creating private Discord channels in batches,
// resolving Discord snowflakes to internal PocketBase IDs, and handling rate limits.
func (u *ProvisionUsecase) BatchCreatePrivateChannels(guildDiscordID, categoryDiscordID, roleDiscordID string) (ProvisionMetrics, error) {
	metrics := ProvisionMetrics{}

	// 1. Resolve IDs (Discord Snowflake -> 15-char PocketBase internal record ID)
	var guildRecord pocketbase.GuildRecord
	found, err := u.PBRepository.FindFirstByDiscordID("guilds", guildDiscordID, &guildRecord)
	if err != nil {
		return metrics, fmt.Errorf("failed to query guild in database: %w", err)
	}
	if !found {
		return metrics, fmt.Errorf("guild %s not found in database, please sync first", guildDiscordID)
	}

	var roleRecord pocketbase.RoleRecord
	found, err = u.PBRepository.FindFirstByDiscordID("roles", roleDiscordID, &roleRecord)
	if err != nil {
		return metrics, fmt.Errorf("failed to query role in database: %w", err)
	}
	if !found {
		return metrics, fmt.Errorf("role %s not found in database, please sync first", roleDiscordID)
	}

	// 2. Query pending students
	pendingStudents, err := u.PBRepository.FindStudentsPendingProvision(guildRecord.ID, roleRecord.ID)
	if err != nil {
		return metrics, fmt.Errorf("failed to fetch pending students: %w", err)
	}

	metrics.TotalStudents = len(pendingStudents)
	if len(pendingStudents) == 0 {
		log.Printf("[PROVISION] All students for guild %s / role %s already have provisioned channels.", guildDiscordID, roleDiscordID)
		return metrics, nil
	}

	// 3. Retrieve guild managers to apply overwrites
	managers, err := u.PBRepository.FindManagersByGuild(guildRecord.ID)
	if err != nil {
		log.Printf("⚠️ WARNING [ProvisionUsecase]: Failed to retrieve managers for Guild %s: %v. Continuing without managers...", guildRecord.ID, err)
	}

	managerDiscordIDs := make([]string, 0)
	for _, m := range managers {
		if m.DiscordID != "" {
			managerDiscordIDs = append(managerDiscordIDs, m.DiscordID)
		}
	}

	log.Printf("🚀 [PROVISION] Starting batch creation of %d private channels on Discord...", len(pendingStudents))

	// 4. Provision loop
	for _, student := range pendingStudents {
		if student.DiscordID == "" {
			log.Printf("⚠️ Skipping student %s (PB ID: %s): missing Discord ID.", student.Nickname, student.ID)
			metrics.Errors++
			continue
		}

		log.Printf("⏳ [PROVISION] Creating private channel for student %s (%s)...", student.Nickname, student.DiscordID)

		// Create private channel using the 1-on-1 Factory
		newChannel, err := u.DiscordService.CreatePrivateChannel(
			guildDiscordID,
			categoryDiscordID,
			student.DiscordID,
			student.Nickname,
			managerDiscordIDs,
		)
		if err != nil {
			log.Printf("❌ ERROR [PROVISION] Failed to create channel for student %s: %v", student.Nickname, err)
			metrics.Errors++
			continue
		}

		log.Printf("✅ [PROVISION] Channel %s (%s) created successfully. Updating PocketBase...", newChannel.Name, newChannel.ID)

		// 5. Update PocketBase record immediately (transactional durability)
		updateData := map[string]interface{}{
			"channel_id": newChannel.ID,
		}
		var updatedStudent pocketbase.StudentRecord
		err = u.PBRepository.UpdateRecord("students", student.ID, updateData, &updatedStudent)
		if err != nil {
			log.Printf("❌ ERROR [PROVISION] Failed to update channel_id in database for student %s: %v", student.Nickname, err)
			metrics.Errors++
			// Continue since the Discord channel was successfully created
			continue
		}

		metrics.ChannelsCreated++

		// 6. Rate Limit Cooldown (800ms)
		log.Printf("💤 [PROVISION] Pausing for 800ms before provisioning next channel...")
		time.Sleep(800 * time.Millisecond)
	}

	log.Printf("🏁 [PROVISION] Batch complete. Metrics: %+v", metrics)
	return metrics, nil
}
