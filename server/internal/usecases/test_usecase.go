package usecases

import (
	"fmt"
	"log"

	"chantry/server/internal/discord"
	"chantry/server/internal/pocketbase"
)

type TestUsecase struct {
	pbRepo         *pocketbase.Repository
	discordService *discord.DiscordService
}

func NewTestUsecase(pbRepo *pocketbase.Repository, discordService *discord.DiscordService) *TestUsecase {
	return &TestUsecase{
		pbRepo:         pbRepo,
		discordService: discordService,
	}
}

// TriggerTestClockIn ensures the tester exists in PocketBase students and triggers check-in buttons in Discord.
func (u *TestUsecase) TriggerTestClockIn(guildDiscordID, channelID, testerDiscordID string) error {
	// 1. Resolve Discord Guild ID to PocketBase Guild Record
	var guild pocketbase.GuildRecord
	foundGuild, err := u.pbRepo.FindFirstByDiscordID("guilds", guildDiscordID, &guild)
	if err != nil {
		return fmt.Errorf("failed to resolve guild from Discord ID: %w", err)
	}
	if !foundGuild {
		return fmt.Errorf("guild with Discord ID %s not found in local database", guildDiscordID)
	}

	// 2. Check if the tester is already in the students table
	var student pocketbase.StudentRecord
	foundStudent, err := u.pbRepo.FindFirstByDiscordID("students", testerDiscordID, &student)
	if err != nil {
		return fmt.Errorf("failed to query student status: %w", err)
	}

	if !foundStudent {
		log.Printf("ℹ️ Tester %s not found in students database. Creating a mock student profile...", testerDiscordID)
		newStudent := pocketbase.StudentRecord{
			DiscordID: testerDiscordID,
			Username:  "tester_" + testerDiscordID,
			Nickname:  "TESTER (" + testerDiscordID + ")",
			GuildID:   guild.ID, // PocketBase Guild relation ID
			Status:    "inactive",
		}
		var createdStudent pocketbase.StudentRecord
		if err := u.pbRepo.CreateRecord("students", &newStudent, &createdStudent); err != nil {
			return fmt.Errorf("failed to insert mock tester student record: %w", err)
		}
		log.Printf("✅ Mock student profile created successfully with PocketBase ID %s", createdStudent.ID)
	} else {
		log.Printf("ℹ️ Tester %s already exists as student %s", testerDiscordID, student.ID)
	}

	// 3. Trigger check-in buttons message on the Discord channel
	if err := u.discordService.SendAttendanceButtons(channelID); err != nil {
		return fmt.Errorf("failed to send test clock-in buttons to Discord channel: %w", err)
	}

	return nil
}
