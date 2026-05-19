package discord

import (
	"fmt"
	"log"
	"time"

	"chantry/server/internal/pocketbase"

	"github.com/bwmarrin/discordgo"
)

// AttendanceUsecase defines the interface required by the interaction handler.
// This decouples the usecases package and resolves Go import cycles.
type AttendanceUsecase interface {
	RegisterClockIn(discordUserID string) (*pocketbase.AttendanceRecord, error)
	RegisterClockOut(discordUserID string) (*pocketbase.AttendanceRecord, error)
}

// RegisterInteractionHandlers registers the interaction handlers on the given discord session.
func RegisterInteractionHandlers(session *discordgo.Session, attendanceUsecase AttendanceUsecase) {
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// We only process message component (button click) interactions here
		if i.Type != discordgo.InteractionMessageComponent {
			return
		}

		customID := i.MessageComponentData().CustomID
		discordUserID := i.Member.User.ID
		username := i.Member.User.Username

		switch customID {
		case "btn_clock_in":
			log.Printf("📩 [Interaction] Student %s (%s) clicked btn_clock_in", username, discordUserID)

			// Execute business logic in usecase
			record, err := attendanceUsecase.RegisterClockIn(discordUserID)
			if err != nil {
				log.Printf("⚠️ [Interaction] RegisterClockIn error for student %s: %v", username, err)
				respondWithError(s, i, err.Error())
				return
			}

			// Parse UTC clock-in time to local visual string
			clockInTime, parseErr := time.Parse(time.RFC3339, record.ClockIn)
			timeStr := ""
			if parseErr == nil {
				// Display clock-in formatted to HH:MM:SS in UTC/Local
				timeStr = clockInTime.Format("15:04:05")
			} else {
				timeStr = time.Now().Format("15:04:05")
			}

			// Respond immediately within the 3-second window
			respondWithSuccess(s, i, fmt.Sprintf("🟢 Ponto de entrada registrado às %s! Bom trabalho e boa aula.", timeStr))

		case "btn_clock_out":
			log.Printf("📩 [Interaction] Student %s (%s) clicked btn_clock_out", username, discordUserID)

			// Execute business logic in usecase
			record, err := attendanceUsecase.RegisterClockOut(discordUserID)
			if err != nil {
				log.Printf("⚠️ [Interaction] RegisterClockOut error for student %s: %v", username, err)
				respondWithError(s, i, err.Error())
				return
			}

			// Parse UTC clock-out time to local visual string
			clockOutTime, parseErr := time.Parse(time.RFC3339, record.ClockOut)
			timeStr := ""
			if parseErr == nil {
				timeStr = clockOutTime.Format("15:04:05")
			} else {
				timeStr = time.Now().Format("15:04:05")
			}

			// Respond immediately within the 3-second window
			respondWithSuccess(s, i, fmt.Sprintf("🔴 Ponto de saída registrado às %s! Bom descanso e parabéns pelo dia.", timeStr))
		}
	})
}

// respondWithSuccess sends a clean ephemeral message confirming the action to the student
func respondWithSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral, // Ensures only the user clicking sees the message
		},
	})
	if err != nil {
		log.Printf("❌ Failed to respond to interaction %s: %v", i.ID, err)
	}
}

// respondWithError sends an ephemeral warning message with error context
func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, errMessage string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("⚠️ **Atenção:** %s", errMessage),
			Flags:   discordgo.MessageFlagsEphemeral, // Ephemeral to avoid channel clutter
		},
	})
	if err != nil {
		log.Printf("❌ Failed to respond with error to interaction %s: %v", i.ID, err)
	}
}
