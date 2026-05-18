package handlers

import (
	"log"

	"chantry/backend/internal/pocketbase"
	"github.com/gofiber/fiber/v2"
)

// ReportHandler handles requests for analytical report queries.
type ReportHandler struct {
	repo *pocketbase.Repository
}

// NewReportHandler instantiates a new ReportHandler.
func NewReportHandler(repo *pocketbase.Repository) *ReportHandler {
	return &ReportHandler{
		repo: repo,
	}
}

// AttendanceReportDTO defines the response payload structure for attendance records.
type AttendanceReportDTO struct {
	AttendanceID    string `json:"attendance_id"`
	StudentName     string `json:"student_name"`
	StudentNickname string `json:"student_nickname"`
	Date            string `json:"date"`
	ClockIn         string `json:"clock_in"`
	ClockOut        string `json:"clock_out"`
	Status          string `json:"status"`
	Source          string `json:"source"`
}

// HandleGetAttendances extracts path parameters and query arguments, queries PocketBase,
// maps relational structures, and serves a cleaned JSON daily attendance list.
func (h *ReportHandler) HandleGetAttendances(c *fiber.Ctx) error {
	guildID := c.Params("guildId")
	if guildID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The guildId path parameter is required",
		})
	}

	dateStr := c.Query("date")
	if dateStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The date query parameter is required (format: YYYY-MM-DD)",
		})
	}

	roleID := c.Query("role_id")
	if roleID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "The role_id query parameter is required",
		})
	}

	log.Printf("[REPORT] Fetching attendances for Guild: %s, Role: %s, Date: %s", guildID, roleID, dateStr)

	records, err := h.repo.GetAttendancesByDateAndRole(guildID, roleID, dateStr)
	if err != nil {
		log.Printf("❌ ERROR [ReportHandler.HandleGetAttendances]: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	report := make([]AttendanceReportDTO, 0, len(records))
	for _, rec := range records {
		studentName := rec.Expand.Student.Username
		studentNickname := rec.Expand.Student.Nickname
		if studentNickname == "" {
			studentNickname = studentName
		}

		report = append(report, AttendanceReportDTO{
			AttendanceID:    rec.ID,
			StudentName:     studentName,
			StudentNickname: studentNickname,
			Date:            rec.Date,
			ClockIn:         rec.ClockIn,
			ClockOut:        rec.ClockOut,
			Status:          rec.Status,
			Source:          rec.Source,
		})
	}

	return c.Status(fiber.StatusOK).JSON(report)
}
