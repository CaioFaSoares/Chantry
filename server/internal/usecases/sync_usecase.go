package usecases

import (
	"fmt"
	"log"

	"chantry/server/internal/discord"
	"chantry/server/internal/pocketbase"
)

// Metrics consolidates the results of the logical upsert execution
type Metrics struct {
	TotalProcessed int `json:"total_processed"`
	NewInserted    int `json:"new_inserted"`
	Updated        int `json:"updated"`
}

// StudentSyncPayload defines the input structure for student role sync
type StudentSyncPayload struct {
	PrimaryRoleID    string   `json:"primary_role_id"`
	SecondaryRoleIDs []string `json:"secondary_role_ids"`
}

// ManagerSyncRule maps a Discord role to a pre-defined manager role access level
type ManagerSyncRule struct {
	RoleID      string `json:"role_id"`
	ManagerType string `json:"manager_type"` // admin, mentor, pedagogy
}

// AdvancedSyncPayload holds parameters for the multi-role student and manager sync
type AdvancedSyncPayload struct {
	Students StudentSyncPayload `json:"students"`
	Managers []ManagerSyncRule  `json:"managers"`
}

// AdvancedSyncMetrics holds mutation summaries for students and managers
type AdvancedSyncMetrics struct {
	StudentsProcessed int `json:"students_processed"`
	StudentsInserted  int `json:"students_inserted"`
	StudentsUpdated   int `json:"students_updated"`
	ManagersProcessed int `json:"managers_processed"`
	ManagersInserted  int `json:"managers_inserted"`
	ManagersUpdated   int `json:"managers_updated"`
}

// SyncUsecase orchestrates the logic to fetch, validate and persist Discord members into PocketBase
type SyncUsecase struct {
	discordService *discord.DiscordService
	pbRepo         *pocketbase.Repository
}

// NewSyncUsecase creates a new instance of SyncUsecase
func NewSyncUsecase(discordService *discord.DiscordService, pbRepo *pocketbase.Repository) *SyncUsecase {
	return &SyncUsecase{
		discordService: discordService,
		pbRepo:         pbRepo,
	}
}

// EnsureRoleExists resolves or creates a Role in PocketBase, returning its 15-character ID
func (u *SyncUsecase) EnsureRoleExists(guildID, roleID, pbGuildID string) (string, error) {
	var roleRecord pocketbase.RoleRecord
	found, err := u.pbRepo.FindFirstByDiscordID("roles", roleID, &roleRecord)
	if err != nil {
		return "", err
	}

	if !found {
		roleName := "Sincronizado via Discord"
		if dRoles, err := u.discordService.Session.GuildRoles(guildID); err == nil {
			for _, r := range dRoles {
				if r.ID == roleID {
					roleName = r.Name
					break
				}
			}
		} else {
			log.Printf("⚠️ Warning: Failed to fetch Roles for Guild %s from Discord API. Error: %v", guildID, err)
		}

		newRole := pocketbase.RoleRecord{
			DiscordID: roleID,
			Name:      roleName,
			GuildID:   pbGuildID,
		}
		if err := u.pbRepo.CreateRecord("roles", newRole, &roleRecord); err != nil {
			return "", err
		}
		log.Printf("✅ Auto-created missing Role record: %s (PB ID: %s)", roleName, roleRecord.ID)
	}

	return roleRecord.ID, nil
}

// SyncStudentsByRole performs cursor-paginated member searches in Discord and syncs them to PocketBase
func (u *SyncUsecase) SyncStudentsByRole(guildID string, roleID string) (Metrics, error) {
	metrics := Metrics{}

	// 1. Ensure Guild exists in PocketBase (Silent Upsert)
	var guildRecord pocketbase.GuildRecord
	found, err := u.pbRepo.FindFirstByDiscordID("guilds", guildID, &guildRecord)
	if err != nil {
		return metrics, fmt.Errorf("failed to query guild in pocketbase: %w", err)
	}

	if !found {
		guildName := "Sincronizado via Discord"
		// Query Discord REST API for Guild information
		if dGuild, err := u.discordService.Session.Guild(guildID); err == nil && dGuild != nil {
			guildName = dGuild.Name
		} else {
			log.Printf("⚠️ Warning: Failed to fetch Guild %s details from Discord API, using fallback name. Error: %v", guildID, err)
		}

		newGuild := pocketbase.GuildRecord{
			DiscordID: guildID,
			Name:      guildName,
			Status:    "active",
		}
		if err := u.pbRepo.CreateRecord("guilds", newGuild, &guildRecord); err != nil {
			return metrics, fmt.Errorf("failed to auto-create guild record in pocketbase: %w", err)
		}
		log.Printf("✅ Auto-created missing Guild record: %s (PB ID: %s)", guildName, guildRecord.ID)
	}

	// 2. Ensure Role exists in PocketBase (Silent Upsert)
	var roleRecord pocketbase.RoleRecord
	found, err = u.pbRepo.FindFirstByDiscordID("roles", roleID, &roleRecord)
	if err != nil {
		return metrics, fmt.Errorf("failed to query role in pocketbase: %w", err)
	}

	if !found {
		roleName := "Sincronizado via Discord"
		// Query Discord REST API to fetch Guild Roles
		if dRoles, err := u.discordService.Session.GuildRoles(guildID); err == nil {
			for _, r := range dRoles {
				if r.ID == roleID {
					roleName = r.Name
					break
				}
			}
		} else {
			log.Printf("⚠️ Warning: Failed to fetch Roles for Guild %s from Discord API, using fallback name. Error: %v", guildID, err)
		}

		newRole := pocketbase.RoleRecord{
			DiscordID: roleID,
			Name:      roleName,
			GuildID:   guildRecord.ID, // PocketBase 15-char record ID relation
		}
		if err := u.pbRepo.CreateRecord("roles", newRole, &roleRecord); err != nil {
			return metrics, fmt.Errorf("failed to auto-create role record in pocketbase: %w", err)
		}
		log.Printf("✅ Auto-created missing Role record: %s (PB ID: %s)", roleName, roleRecord.ID)
	}

	// 3. Fetch active Discord guild members holding the role
	members, err := u.discordService.GetGuildMembersByRole(guildID, roleID)
	if err != nil {
		return metrics, fmt.Errorf("failed to fetch guild members by role from Discord API: %w", err)
	}

	// 4. Perform logical upsert for each student member
	for _, m := range members {
		metrics.TotalProcessed++

		var student pocketbase.StudentRecord
		studentFound, err := u.pbRepo.FindFirstByDiscordAndGuild("students", m.ID, guildRecord.ID, &student)
		if err != nil {
			log.Printf("⚠️ Error querying student %s (%s) in guild %s: %v", m.Username, m.ID, guildRecord.ID, err)
			continue
		}

		if studentFound {
			// Compare dynamic data to check if partial PATCH is required (optimizing API requests)
			if student.Username != m.Username || student.Nickname != m.Nickname || student.RoleID != roleRecord.ID || student.GuildID != guildRecord.ID {
				updateData := map[string]interface{}{
					"username":  m.Username,
					"nickname":  m.Nickname,
					"role_id":   roleRecord.ID,
					"guild_id":  guildRecord.ID,
				}
				var updatedStudent pocketbase.StudentRecord
				if err := u.pbRepo.UpdateRecord("students", student.ID, updateData, &updatedStudent); err != nil {
					log.Printf("⚠️ Error updating student record %s (ID: %s) in PocketBase: %v", m.Username, student.ID, err)
					continue
				}
				metrics.Updated++
			}
		} else {
			// Create a brand new active student record
			newStudent := pocketbase.StudentRecord{
				DiscordID: m.ID,
				Username:  m.Username,
				Nickname:  m.Nickname,
				RoleID:    roleRecord.ID,
				GuildID:   guildRecord.ID,
				Status:    "active",
			}
			var createdStudent pocketbase.StudentRecord
			if err := u.pbRepo.CreateRecord("students", newStudent, &createdStudent); err != nil {
				log.Printf("⚠️ Error creating student record %s in PocketBase: %v", m.Username, err)
				continue
			}
			metrics.NewInserted++
		}
	}

	return metrics, nil
}

// AdvancedSync performs multi-role students and managers synchronization
func (u *SyncUsecase) AdvancedSync(guildID string, payload AdvancedSyncPayload) (AdvancedSyncMetrics, error) {
	metrics := AdvancedSyncMetrics{}

	// 1. Ensure Guild exists in PocketBase (Silent Upsert)
	var guildRecord pocketbase.GuildRecord
	found, err := u.pbRepo.FindFirstByDiscordID("guilds", guildID, &guildRecord)
	if err != nil {
		return metrics, fmt.Errorf("failed to query guild in pocketbase: %w", err)
	}

	if !found {
		guildName := "Sincronizado via Discord"
		if dGuild, err := u.discordService.Session.Guild(guildID); err == nil && dGuild != nil {
			guildName = dGuild.Name
		}
		newGuild := pocketbase.GuildRecord{
			DiscordID: guildID,
			Name:      guildName,
			Status:    "active",
		}
		if err := u.pbRepo.CreateRecord("guilds", newGuild, &guildRecord); err != nil {
			return metrics, fmt.Errorf("failed to auto-create guild record in pocketbase: %w", err)
		}
	}

	// 2. Perform Students Sync if primary role is provided
	if payload.Students.PrimaryRoleID != "" {
		// Resolve Primary Role PB ID
		primaryRolePBID, err := u.EnsureRoleExists(guildID, payload.Students.PrimaryRoleID, guildRecord.ID)
		if err != nil {
			return metrics, fmt.Errorf("failed to resolve primary role: %w", err)
		}

		// Resolve all Secondary Roles PB IDs
		secondaryPBIDsMap := make(map[string]string)
		for _, secRoleID := range payload.Students.SecondaryRoleIDs {
			pbID, err := u.EnsureRoleExists(guildID, secRoleID, guildRecord.ID)
			if err != nil {
				log.Printf("⚠️ Warning: Failed to resolve secondary role %s: %v", secRoleID, err)
				continue
			}
			secondaryPBIDsMap[secRoleID] = pbID
		}

		// Fetch Primary Students from Discord
		studentsList, err := u.discordService.GetGuildMembersByRole(guildID, payload.Students.PrimaryRoleID)
		if err != nil {
			return metrics, fmt.Errorf("failed to fetch students by primary role: %w", err)
		}

		for _, m := range studentsList {
			metrics.StudentsProcessed++

			// Intersect secondary roles of this member
			var studentSecPBIDs []string
			for _, memberRoleID := range m.Roles {
				if pbID, ok := secondaryPBIDsMap[memberRoleID]; ok {
					studentSecPBIDs = append(studentSecPBIDs, pbID)
				}
			}

			var student pocketbase.StudentRecord
			studentFound, err := u.pbRepo.FindFirstByDiscordAndGuild("students", m.ID, guildRecord.ID, &student)
			if err != nil {
				log.Printf("⚠️ Error querying student %s (%s) in guild %s: %v", m.Username, m.ID, guildRecord.ID, err)
				continue
			}

			// Helper to compare slices of strings
			slicesEqual := func(a, b []string) bool {
				if len(a) != len(b) {
					return false
				}
				for i := range a {
					if a[i] != b[i] {
						return false
					}
				}
				return true
			}

			if studentFound {
				// Check for changes
				if student.Username != m.Username ||
					student.Nickname != m.Nickname ||
					student.RoleID != primaryRolePBID ||
					student.GuildID != guildRecord.ID ||
					!slicesEqual(student.SecondaryRoles, studentSecPBIDs) {

					updateData := map[string]interface{}{
						"username":        m.Username,
						"nickname":        m.Nickname,
						"role_id":         primaryRolePBID,
						"secondary_roles": studentSecPBIDs,
						"guild_id":        guildRecord.ID,
					}
					var updatedStudent pocketbase.StudentRecord
					if err := u.pbRepo.UpdateRecord("students", student.ID, updateData, &updatedStudent); err != nil {
						log.Printf("⚠️ Error updating student %s: %v", m.Username, err)
						continue
					}
					metrics.StudentsUpdated++
				}
			} else {
				// Insert new student
				newStudent := pocketbase.StudentRecord{
					DiscordID:      m.ID,
					Username:       m.Username,
					Nickname:       m.Nickname,
					RoleID:         primaryRolePBID,
					SecondaryRoles: studentSecPBIDs,
					GuildID:        guildRecord.ID,
					Status:         "active",
				}
				var createdStudent pocketbase.StudentRecord
				if err := u.pbRepo.CreateRecord("students", newStudent, &createdStudent); err != nil {
					log.Printf("⚠️ Error creating student %s: %v", m.Username, err)
					continue
				}
				metrics.StudentsInserted++
			}
		}
	}

	// 3. Perform Managers Sync
	for _, mgrRule := range payload.Managers {
		if mgrRule.RoleID == "" || mgrRule.ManagerType == "" {
			continue
		}

		mgrList, err := u.discordService.GetGuildMembersByRole(guildID, mgrRule.RoleID)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to fetch managers for role %s: %v", mgrRule.RoleID, err)
			continue
		}

		for _, m := range mgrList {
			metrics.ManagersProcessed++

			var manager pocketbase.ManagerRecord
			mgrFound, err := u.pbRepo.FindFirstByDiscordID("managers", m.ID, &manager)
			if err != nil {
				log.Printf("⚠️ Error querying manager %s (%s): %v", m.Username, m.ID, err)
				continue
			}

			// Format Name (fallback to Username)
			name := m.Nickname
			if name == "" {
				name = m.Username
			}

			if mgrFound {
				// Guild relation cascade append without duplicate
				guildPBIDExists := false
				for _, gID := range manager.Guilds {
					if gID == guildRecord.ID {
						guildPBIDExists = true
						break
					}
				}
				updatedGuilds := manager.Guilds
				if !guildPBIDExists {
					updatedGuilds = append(updatedGuilds, guildRecord.ID)
				}

				if manager.Name != name || manager.Role != mgrRule.ManagerType || !guildPBIDExists {
					updateData := map[string]interface{}{
						"name":   name,
						"role":   mgrRule.ManagerType,
						"guilds": updatedGuilds,
					}
					var updatedMgr pocketbase.ManagerRecord
					if err := u.pbRepo.UpdateRecord("managers", manager.ID, updateData, &updatedMgr); err != nil {
						log.Printf("⚠️ Error updating manager %s: %v", name, err)
						continue
					}
					metrics.ManagersUpdated++
				}
			} else {
				// Insert new manager
				newManager := pocketbase.ManagerRecord{
					DiscordID: m.ID,
					Name:      name,
					Role:      mgrRule.ManagerType,
					Guilds:    []string{guildRecord.ID},
				}
				var createdMgr pocketbase.ManagerRecord
				if err := u.pbRepo.CreateRecord("managers", newManager, &createdMgr); err != nil {
					log.Printf("⚠️ Error creating manager %s: %v", name, err)
					continue
				}
				metrics.ManagersInserted++
			}
		}
	}

	return metrics, nil
}

