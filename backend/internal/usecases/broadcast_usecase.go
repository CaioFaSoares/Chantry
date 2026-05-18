package usecases

import (
	"fmt"
	"log"
	"time"

	"chantry/backend/internal/discord"
	"chantry/backend/internal/pocketbase"
)

// BroadcastMetrics consolida os resultados do processamento
type BroadcastMetrics struct {
	TargetType   string `json:"target_type"`
	MessagesSent int    `json:"messages_sent"`
	Errors       int    `json:"errors"`
}

type BroadcastUsecase struct {
	discordService *discord.DiscordService
	pbRepo         *pocketbase.Repository
}

func NewBroadcastUsecase(discordService *discord.DiscordService, pbRepo *pocketbase.Repository) *BroadcastUsecase {
	return &BroadcastUsecase{
		discordService: discordService,
		pbRepo:         pbRepo,
	}
}

// SendBroadcast orquestra o envio baseado nos filtros e no rate limit
func (u *BroadcastUsecase) SendBroadcast(guildDiscordID, content, targetType string, targetRoleIDs []string) (BroadcastMetrics, error) {
	metrics := BroadcastMetrics{
		TargetType: targetType,
	}

	// 1. Resolver a guilda no PocketBase
	var guildRecord pocketbase.GuildRecord
	found, err := u.pbRepo.FindFirstByDiscordID("guilds", guildDiscordID, &guildRecord)
	if err != nil {
		return metrics, fmt.Errorf("falha ao consultar guilda no banco: %w", err)
	}
	if !found {
		return metrics, fmt.Errorf("servidor Discord %s não encontrado no banco de dados", guildDiscordID)
	}

	// 2. Tratar fluxo PUBLIC (Megafone)
	if targetType == "public" {
		if guildRecord.AnnouncementChannelID == "" {
			return metrics, fmt.Errorf("canal de avisos não configurado")
		}

		log.Printf("[BROADCAST-PUBLIC] Enviando aviso para o canal %s da guilda %s", guildRecord.AnnouncementChannelID, guildRecord.Name)
		_, err := u.discordService.Session.ChannelMessageSend(guildRecord.AnnouncementChannelID, content)
		if err != nil {
			metrics.Errors++
			return metrics, fmt.Errorf("falha ao enviar mensagem pública no Discord: %w", err)
		}

		metrics.MessagesSent = 1
		return metrics, nil
	}

	// 3. Tratar fluxo PRIVATE (Disparo Direcionado 1-on-1)
	// Buscar todos os alunos associados a esta guilda
	allStudents, err := u.pbRepo.FindStudentsByGuild(guildRecord.ID)
	if err != nil {
		return metrics, fmt.Errorf("falha ao listar alunos da guilda: %w", err)
	}

	// Filtrar apenas alunos ativos com canal privado provisionado
	var activeStudents []pocketbase.StudentRecord
	for _, student := range allStudents {
		if student.Status == "active" && student.ChannelID != "" {
			activeStudents = append(activeStudents, student)
		}
	}

	var targetStudents []pocketbase.StudentRecord

	// Filtrar audiência se targetRoleIDs não estiver vazio
	if len(targetRoleIDs) > 0 {
		// Obter os cargos cadastrados nesta guilda no banco
		dbRoles, err := u.pbRepo.FindRolesByGuild(guildRecord.ID)
		if err != nil {
			return metrics, fmt.Errorf("falha ao recuperar cargos da guilda: %w", err)
		}

		// Criar mapa de tradução: Discord Role ID (Snowflake) -> PocketBase ID (15 chars)
		discordRoleToPBID := make(map[string]string)
		for _, role := range dbRoles {
			discordRoleToPBID[role.DiscordID] = role.ID
		}

		// Construir set de IDs do PocketBase válidos para o disparo
		targetPBIDs := make(map[string]bool)
		for _, disRoleID := range targetRoleIDs {
			if pbID, ok := discordRoleToPBID[disRoleID]; ok {
				targetPBIDs[pbID] = true
			}
		}

		// Filtrar estudantes cujo cargo primário ou secundários interceptem os alvos
		for _, s := range activeStudents {
			matched := false
			// Verificar cargo primário
			if targetPBIDs[s.RoleID] {
				matched = true
			} else {
				// Verificar cargos secundários
				for _, secRoleID := range s.SecondaryRoles {
					if targetPBIDs[secRoleID] {
						matched = true
						break
					}
				}
			}

			if matched {
				targetStudents = append(targetStudents, s)
			}
		}
	} else {
		// Sem filtros = disparar para todos os alunos ativos da guilda com canal
		targetStudents = activeStudents
	}

	log.Printf("[BROADCAST-PRIVATE] Iniciando loop de disparos direcionados para %d alunos", len(targetStudents))

	// 4. Iniciar loop de disparos com Rate Limit Engine (500ms)
	for i, s := range targetStudents {
		if i > 0 {
			time.Sleep(500 * time.Millisecond) // Pausa preventiva anti-spam
		}

		_, err := u.discordService.Session.ChannelMessageSend(s.ChannelID, content)
		if err != nil {
			metrics.Errors++
			log.Printf("⚠️ Erro ao enviar DM para o aluno %s (%s) no canal %s: %v", s.Username, s.DiscordID, s.ChannelID, err)
		} else {
			metrics.MessagesSent++
		}
	}

	return metrics, nil
}
