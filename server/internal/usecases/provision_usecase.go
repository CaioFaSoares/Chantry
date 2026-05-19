package usecases

import (
	"fmt"
	"log"
	"strings"
	"time"

	"chantry/server/internal/discord"
	"chantry/server/internal/pocketbase"

	"github.com/bwmarrin/discordgo"
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

// HealMetrics holds recovery metrics for the Disaster Recovery (Auto-Healing) batch run.
type HealMetrics struct {
	ChannelsScanned      int `json:"channels_scanned"`
	SuccessfullyMapped   int `json:"successfully_mapped"`
	UnmappedChannels     int `json:"unmapped_channels"`
	StudentsStillPending int `json:"students_still_pending"`
}

// HealChannelsByCategory scans all text channels under a Discord category, matches them against
// students' usernames in PocketBase, and updates the missing channel_id fields.
func (u *ProvisionUsecase) HealChannelsByCategory(guildDiscordID string, categoryDiscordID string) (HealMetrics, error) {
	metrics := HealMetrics{}

	// 1. Resolve PocketBase Guild ID
	var guildRecord pocketbase.GuildRecord
	found, err := u.PBRepository.FindFirstByDiscordID("guilds", guildDiscordID, &guildRecord)
	if err != nil {
		return metrics, fmt.Errorf("failed to query guild in database: %w", err)
	}
	if !found {
		return metrics, fmt.Errorf("guild %s not found in database, please sync first", guildDiscordID)
	}

	// 2. Fetch all channels in the Guild from Discord API
	discordChannels, err := u.DiscordService.Session.GuildChannels(guildDiscordID)
	if err != nil {
		return metrics, fmt.Errorf("failed to fetch channels from Discord API: %w", err)
	}

	// 3. Filter channels physically belonging to the target Category and that are text channels
	var categoryChannels []*discordgo.Channel
	for _, ch := range discordChannels {
		if ch.ParentID == categoryDiscordID && ch.Type == discordgo.ChannelTypeGuildText {
			categoryChannels = append(categoryChannels, ch)
		}
	}
	metrics.ChannelsScanned = len(categoryChannels)
	log.Printf("🛠️ [HEAL] Found %d text channels under Category %s in Discord", metrics.ChannelsScanned, categoryDiscordID)

	// 4. Fetch all active/existing students for this guild from PocketBase
	students, err := u.PBRepository.FindStudentsByGuild(guildRecord.ID)
	if err != nil {
		return metrics, fmt.Errorf("failed to fetch students for guild %s: %w", guildRecord.ID, err)
	}

	// 5. Build O(1) maps of students indexed by their Discord ID and lowercase username
	studentByDiscordID := make(map[string]*pocketbase.StudentRecord)
	studentByUsername := make(map[string]*pocketbase.StudentRecord)
	for i := range students {
		if students[i].DiscordID != "" {
			studentByDiscordID[students[i].DiscordID] = &students[i]
		}
		uName := strings.ToLower(students[i].Username)
		if uName != "" {
			studentByUsername[uName] = &students[i]
		}
	}

	// 6. Match and update missing channel IDs
	for _, ch := range categoryChannels {
		chName := strings.ToLower(ch.Name)

		var student *pocketbase.StudentRecord
		foundByOverwrite := false

		// A. First Strategy: Check Permission Overwrites for Member matching Student's Discord ID
		for _, ow := range ch.PermissionOverwrites {
			if ow.Type == discordgo.PermissionOverwriteTypeMember {
				if s, exists := studentByDiscordID[ow.ID]; exists {
					student = s
					foundByOverwrite = true
					log.Printf("🎯 [HEAL] Matched channel %s (%s) to student %s (%s) via Discord ID permission overwrite!", ch.Name, ch.ID, s.Nickname, s.ID)
					break
				}
			}
		}

		// B. Second Strategy (Fallback): If not found by permission overwrite, match by name prefix
		if !foundByOverwrite {
			if strings.HasPrefix(chName, "1-on-1-") {
				targetUsername := strings.TrimPrefix(chName, "1-on-1-")
				if s, exists := studentByUsername[targetUsername]; exists {
					student = s
					log.Printf("ℹ️ [HEAL] Matched channel %s (%s) to student %s (%s) via Name Fallback", ch.Name, ch.ID, s.Nickname, s.ID)
				}
			}
		}

		if student == nil {
			log.Printf("⚠️ [HEAL] Warning: Discord channel %s (%s) has no matching student (by permission or username fallback)", ch.Name, ch.ID)
			metrics.UnmappedChannels++
			continue
		}

		// If student has no channel_id, restore/heal it!
		if student.ChannelID == "" {
			log.Printf("🔗 [HEAL] Healing: Mapping Discord channel %s (%s) to student %s (%s)", ch.Name, ch.ID, student.Nickname, student.ID)
			
			updateData := map[string]interface{}{
				"channel_id": ch.ID,
			}
			var updatedStudent pocketbase.StudentRecord
			err = u.PBRepository.UpdateRecord("students", student.ID, updateData, &updatedStudent)
			if err != nil {
				log.Printf("❌ [HEAL] Error updating student %s in PocketBase: %v", student.Nickname, err)
				// Don't interrupt the whole process for one record failure
				continue
			}

			// Update local map state so we don't count it as pending
			student.ChannelID = ch.ID
			metrics.SuccessfullyMapped++
		} else if student.ChannelID != ch.ID {
			log.Printf("ℹ️ [HEAL] Student %s already has channel %s in database. Skipping override.", student.Nickname, student.ChannelID)
		} else {
			log.Printf("ℹ️ [HEAL] Channel %s is already correctly mapped in PocketBase for %s.", ch.Name, student.Nickname)
		}
	}

	// 7. Count how many students are still pending a channel association
	for _, student := range students {
		if student.ChannelID == "" {
			metrics.StudentsStillPending++
		}
	}

	log.Printf("🏁 [HEAL] Auto-Healing run completed. Metrics: %+v", metrics)
	return metrics, nil
}

// ProvisionPageMetrics holds metric configurations for the provision page BFF.
type ProvisionPageMetrics struct {
	TotalStudentsWithoutChannels int `json:"total_students_without_channels"`
	TotalStudents                int `json:"total_students"`
}

// ProvisionPageData holds the aggregated data payload for the provision page BFF.
type ProvisionPageData struct {
	Categories            []discord.DiscordCategory `json:"categories"`
	Roles                 []discord.SimpleEntity    `json:"roles"`
	Metrics               ProvisionPageMetrics      `json:"metrics"`
	AnnouncementChannelID string                    `json:"announcement_channel_id"`
	TextChannels          []discord.DiscordChannel  `json:"text_channels"`
}

// GetProvisionPageData gathers and returns all categories, roles, text channels, and announcement channel configuration for a guild.
func (u *ProvisionUsecase) GetProvisionPageData(guildDiscordID string) (ProvisionPageData, error) {
	var data ProvisionPageData
	data.Categories = []discord.DiscordCategory{}
	data.Roles = []discord.SimpleEntity{}
	data.TextChannels = []discord.DiscordChannel{}

	// 1. Fetch categories from Discord
	categories, err := u.DiscordService.GetGuildCategories(guildDiscordID)
	if err != nil {
		return data, fmt.Errorf("failed to fetch Discord categories: %w", err)
	}
	data.Categories = categories

	// 2. Fetch roles from Discord
	roles, err := u.DiscordService.GetGuildRoles(guildDiscordID)
	if err != nil {
		return data, fmt.Errorf("failed to fetch Discord roles: %w", err)
	}
	data.Roles = roles

	// 3. Fetch text channels from Discord
	textChannels, err := u.DiscordService.GetGuildTextChannels(guildDiscordID)
	if err != nil {
		return data, fmt.Errorf("failed to fetch Discord text channels: %w", err)
	}
	data.TextChannels = textChannels

	// 4. Query local Guild Record for announcement channel
	var guildRecord pocketbase.GuildRecord
	found, err := u.PBRepository.FindFirstByDiscordID("guilds", guildDiscordID, &guildRecord)
	if err != nil {
		log.Printf("⚠️ Warning: Failed to query guild %s in database: %v", guildDiscordID, err)
	}
	if found {
		data.AnnouncementChannelID = guildRecord.AnnouncementChannelID

		// Count students without channels in PocketBase
		students, err := u.PBRepository.FindStudentsByGuild(guildRecord.ID)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to fetch students for guild %s: %v", guildRecord.ID, err)
		} else {
			count := 0
			for _, student := range students {
				if student.ChannelID == "" {
					count++
				}
			}
			data.Metrics.TotalStudentsWithoutChannels = count
			data.Metrics.TotalStudents = len(students)
		}
	}

	return data, nil
}

