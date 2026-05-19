package usecases

import (
	"fmt"
	"time"

	"chantry/server/internal/discord"
	"chantry/server/internal/pocketbase"
)

// SquadDashboardStudentDTO represents a single student in the squad dashboard
type SquadDashboardStudentDTO struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Has1on1  bool   `json:"has_1on1"`
}

// SquadDashboardMetricsDTO represents aggregated metrics for a squad
type SquadDashboardMetricsDTO struct {
	TotalStudents            int     `json:"total_students"`
	Provisioned1on1Channels  int     `json:"provisioned_1on1_channels"`
	AverageAttendancePercent float64 `json:"average_attendance_percent"`
}

// SquadDashboardInfoDTO represents core squad info
type SquadDashboardInfoDTO struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	SquadChannelID string `json:"squad_channel_id"`
}

// SquadDashboardResponse is the main BFF payload
type SquadDashboardResponse struct {
	SquadInfo    SquadDashboardInfoDTO      `json:"squad_info"`
	Metrics      SquadDashboardMetricsDTO   `json:"metrics"`
	TextChannels []discord.DiscordChannel   `json:"text_channels"`
	Students     []SquadDashboardStudentDTO `json:"students"`
}

// UIUsecase handles complex BFF data aggregations
type UIUsecase struct {
	discordService *discord.DiscordService
	repo           *pocketbase.Repository
}

// NewUIUsecase creates a new UIUsecase
func NewUIUsecase(discordService *discord.DiscordService, repo *pocketbase.Repository) *UIUsecase {
	return &UIUsecase{
		discordService: discordService,
		repo:           repo,
	}
}

// GetSquadDashboardData aggregates all data needed for the Squad Management Dashboard
func (u *UIUsecase) GetSquadDashboardData(roleID string) (*SquadDashboardResponse, error) {
	// 1. Fetch Role Record
	var role pocketbase.RoleRecord
	found, err := u.repo.FindByID("roles", roleID, &role)
	if err != nil || !found {
		return nil, fmt.Errorf("role not found or error: %v", err)
	}

	// 2. Fetch Guild Record to get Discord ID
	var guild pocketbase.GuildRecord
	foundGuild, errGuild := u.repo.FindByID("guilds", role.GuildID, &guild)
	if errGuild != nil || !foundGuild {
		return nil, fmt.Errorf("guild not found for role")
	}

	// 3. Fetch Discord Categories/Channels to filter text channels
	// The PRD mentions text channels. Let's use GetGuildCategories which we already have,
	// or create a new method to just get text channels.
	// Since we don't have GetTextChannels natively exposed on discordService easily without adding to it,
	// I'll fetch categories which usually contain channels if the struct supports it, 
	// or we can fetch channels directly via Discord API.
	// We'll add GetTextChannels to DiscordService to keep it clean.
	textChannels, err := u.discordService.GetGuildTextChannels(guild.DiscordID)
	if err != nil {
		// Log but don't fail the whole dashboard
		textChannels = []discord.DiscordChannel{}
	}

	// 4. Fetch Active Students in this Role
	students, err := u.repo.FindActiveStudentsByRole(role.GuildID, role.ID)
	if err != nil {
		students = []pocketbase.StudentRecord{}
	}

	// 5. Build Student DTOs and Calculate Basic Metrics
	var studentDTOs []SquadDashboardStudentDTO
	provisionedCount := 0

	for _, s := range students {
		hasChannel := s.ChannelID != ""
		if hasChannel {
			provisionedCount++
		}
		
		nickname := s.Nickname
		if nickname == "" {
			nickname = s.Username
		}

		studentDTOs = append(studentDTOs, SquadDashboardStudentDTO{
			ID:       s.ID,
			Username: s.Username,
			Nickname: nickname,
			Has1on1:  hasChannel,
		})
	}

	totalStudents := len(studentDTOs)

	// 6. Calculate Average Attendance (Basic Approximation for PRD)
	// We'll fetch attendances for the last 30 days for this role.
	avgAttendance := 0.0
	if totalStudents > 0 {
		startDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
		endDate := time.Now().Format("2006-01-02")
		
		attendances, _ := u.repo.GetAttendancesByDateRange(role.GuildID, role.ID, startDate, endDate)
		
		if len(attendances) > 0 {
			completedCount := 0
			for _, att := range attendances {
				if att.Status == "completed" || att.Status == "late" {
					completedCount++
				}
			}
			avgAttendance = (float64(completedCount) / float64(len(attendances))) * 100.0
		}
	}

	// 7. Assemble Response
	return &SquadDashboardResponse{
		SquadInfo: SquadDashboardInfoDTO{
			ID:             role.ID,
			Name           : role.Name,
			SquadChannelID : role.SquadChannelID,
		},
		Metrics: SquadDashboardMetricsDTO{
			TotalStudents:            totalStudents,
			Provisioned1on1Channels:  provisionedCount,
			AverageAttendancePercent: avgAttendance,
		},
		TextChannels: textChannels,
		Students:     studentDTOs,
	}, nil
}
