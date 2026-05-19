package usecases

import (
	"fmt"
	"chantry/server/internal/pocketbase"
)

// ReportSummary defines the mathematical aggregations for the Streamlit cards.
type ReportSummary struct {
	TotalRecords          int     `json:"total_records"`
	TotalPresents         int     `json:"total_presents"`
	TotalAbsents          int     `json:"total_absents"`
	TotalLates            int     `json:"total_lates"`
	AttendanceRatePercent float64 `json:"attendance_rate_percent"`
}

// ExportRecordDTO defines the flat structure for the data table and CSV export.
type ExportRecordDTO struct {
	StudentName      string `json:"student_name"`
	StudentDiscordID string `json:"student_discord_id"`
	RoleName         string `json:"role_name"`
	Date             string `json:"date"`
	ClockIn          string `json:"clock_in"`
	ClockOut         string `json:"clock_out"`
	Status           string `json:"status"`
}

// ExportReportResponse wraps the summary and records into a single API response.
type ExportReportResponse struct {
	Summary ReportSummary     `json:"summary"`
	Records []ExportRecordDTO `json:"records"`
}

// ReportUsecase handles the BI logic and data aggregation for reports.
type ReportUsecase struct {
	repo *pocketbase.Repository
}

// NewReportUsecase instantiates a new ReportUsecase.
func NewReportUsecase(repo *pocketbase.Repository) *ReportUsecase {
	return &ReportUsecase{
		repo: repo,
	}
}

// GenerateExportReport fetches records from the database and processes the mathematical aggregations.
func (u *ReportUsecase) GenerateExportReport(guildID, roleID, startDate, endDate string) (ExportReportResponse, error) {
	// 1. Fetch raw records from DB
	attendances, err := u.repo.GetAttendancesByDateRange(guildID, roleID, startDate, endDate)
	if err != nil {
		return ExportReportResponse{}, fmt.Errorf("error fetching attendances from repo: %w", err)
	}

	// 2. Initialize aggregators
	var (
		totalRecords  = len(attendances)
		totalPresents = 0
		totalAbsents  = 0
		totalLates    = 0
		records       = make([]ExportRecordDTO, 0, totalRecords)
	)

	// 3. Process each record
	for _, rec := range attendances {
		// Map relations safely
		studentName := rec.Expand.Student.Nickname
		if studentName == "" {
			studentName = rec.Expand.Student.Username
		}
		
		roleName := rec.Expand.Student.Role.Name
		if roleName == "" {
			roleName = "Unassigned"
		}

		// Count statuses based on rules
		switch rec.Status {
		case "completed":
			totalPresents++
		case "absent":
			totalAbsents++
		case "late":
			totalLates++
		}

		// Map to flat DTO
		records = append(records, ExportRecordDTO{
			StudentName:      studentName,
			StudentDiscordID: rec.Expand.Student.DiscordID,
			RoleName:         roleName,
			Date:             rec.Date,
			ClockIn:          rec.ClockIn,
			ClockOut:         rec.ClockOut,
			Status:           rec.Status,
		})
	}

	// 4. Calculate Rate
	attendanceRate := 0.0
	if totalRecords > 0 {
		// Formula: (completed + late) / total_registos * 100
		attendanceRate = (float64(totalPresents+totalLates) / float64(totalRecords)) * 100.0
	}

	// 5. Construct final response
	return ExportReportResponse{
		Summary: ReportSummary{
			TotalRecords:          totalRecords,
			TotalPresents:         totalPresents,
			TotalAbsents:          totalAbsents,
			TotalLates:            totalLates,
			AttendanceRatePercent: attendanceRate,
		},
		Records: records,
	}, nil
}
