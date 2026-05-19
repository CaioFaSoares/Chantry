package usecases

import (
	"errors"
	"fmt"
	"time"

	"chantry/server/internal/pocketbase"
)

type AttendanceUsecase struct {
	repo *pocketbase.Repository
}

func NewAttendanceUsecase(repo *pocketbase.Repository) *AttendanceUsecase {
	return &AttendanceUsecase{
		repo: repo,
	}
}

// RegisterClockIn handles the clock-in logic for a student on the current day.
// It resolves the student by their Discord ID, checks if they already have an attendance record today,
// and if not, creates a new attendance record with "pending_checkout" status.
func (u *AttendanceUsecase) RegisterClockIn(discordUserID string) (*pocketbase.AttendanceRecord, error) {
	var student pocketbase.StudentRecord
	found, err := u.repo.FindFirstByDiscordID("students", discordUserID, &student)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar estudante pelo Discord ID: %w", err)
	}
	if !found {
		return nil, errors.New("estudante não encontrado no banco de dados. Fale com a equipe pedagógica para ativar o seu registro")
	}

	todayStr := time.Now().UTC().Format("2006-01-02")
	var existing pocketbase.AttendanceRecord
	foundAtt, err := u.repo.FindAttendanceByStudentAndDate(student.ID, todayStr, &existing)
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar ponto de hoje: %w", err)
	}
	if foundAtt {
		return nil, fmt.Errorf("você já registrou o seu ponto hoje (status: %s)", existing.Status)
	}

	// Create new daily attendance
	nowISO := time.Now().UTC().Format(time.RFC3339)
	newRecord := pocketbase.AttendanceRecord{
		StudentID: student.ID,
		Date:      todayStr + " 00:00:00.000Z", // Store daily normalized date
		ClockIn:   nowISO,
		Status:    "pending_checkout",
		Source:    "discord_bot",
		Notes:     "Registrado via botão no Discord",
	}

	var created pocketbase.AttendanceRecord
	if err := u.repo.CreateRecord("attendances", &newRecord, &created); err != nil {
		return nil, fmt.Errorf("erro ao criar registro de presença: %w", err)
	}

	return &created, nil
}

// RegisterClockOut handles the clock-out logic for a student on the current day.
// It resolves the student by their Discord ID, verifies their pending check-out state,
// and updates the status to "completed" with the clock-out timestamp.
func (u *AttendanceUsecase) RegisterClockOut(discordUserID string) (*pocketbase.AttendanceRecord, error) {
	var student pocketbase.StudentRecord
	found, err := u.repo.FindFirstByDiscordID("students", discordUserID, &student)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar estudante pelo Discord ID: %w", err)
	}
	if !found {
		return nil, errors.New("estudante não encontrado no banco de dados")
	}

	todayStr := time.Now().UTC().Format("2006-01-02")
	var existing pocketbase.AttendanceRecord
	foundAtt, err := u.repo.FindAttendanceByStudentAndDate(student.ID, todayStr, &existing)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar presença de hoje: %w", err)
	}
	if !foundAtt {
		return nil, errors.New("registro de entrada (check-in) não encontrado para hoje. Registre a entrada antes de bater a saída")
	}

	if existing.Status == "completed" {
		return nil, errors.New("você já realizou o check-out hoje. Tenha um bom descanso")
	}

	if existing.Status != "pending_checkout" {
		return nil, fmt.Errorf("seu status de presença hoje é '%s'. Não é possível realizar o check-out", existing.Status)
	}

	// Update existing record
	nowISO := time.Now().UTC().Format(time.RFC3339)
	updateData := map[string]interface{}{
		"clock_out": nowISO,
		"status":    "completed",
	}

	var updated pocketbase.AttendanceRecord
	if err := u.repo.UpdateRecord("attendances", existing.ID, &updateData, &updated); err != nil {
		return nil, fmt.Errorf("erro ao atualizar registro de saída: %w", err)
	}

	return &updated, nil
}
