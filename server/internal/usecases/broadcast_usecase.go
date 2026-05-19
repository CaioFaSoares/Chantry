package usecases

import (
	"fmt"
	"log"
	"time"

	"chantry/server/internal/discord"
	"chantry/server/internal/pocketbase"
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
		for _, roleID := range targetRoleIDs {
			if len(roleID) == 15 {
				targetPBIDs[roleID] = true
			} else if pbID, ok := discordRoleToPBID[roleID]; ok {
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

// BroadcastPageData encapsulates data returned to build the Broadcast Center page in high performance.
type BroadcastPageData struct {
	GuildID                 string                       `json:"guild_pb_id"`
	AnnouncementChannelID   string                       `json:"announcement_channel_id"`
	AnnouncementChannelName string                       `json:"announcement_channel_name"`
	Roles                   []pocketbase.RoleRecord      `json:"roles"`
	Broadcasts              []pocketbase.BroadcastRecord `json:"broadcasts"`
}

// GetBroadcastPageData resolves and aggregates all information required by the Broadcast Center Streamlit UI in one roundtrip.
func (u *BroadcastUsecase) GetBroadcastPageData(guildDiscordID string) (BroadcastPageData, error) {
	var data BroadcastPageData
	data.Roles = []pocketbase.RoleRecord{}
	data.Broadcasts = []pocketbase.BroadcastRecord{}

	// 1. Resolve Guild
	var guild pocketbase.GuildRecord
	found, err := u.pbRepo.FindFirstByDiscordID("guilds", guildDiscordID, &guild)
	if err != nil {
		return data, fmt.Errorf("failed to resolve guild: %w", err)
	}
	if !found {
		return data, fmt.Errorf("guild %s not found in DB", guildDiscordID)
	}

	data.GuildID = guild.ID
	data.AnnouncementChannelID = guild.AnnouncementChannelID


	// Resolve the announcement channel name
	if guild.AnnouncementChannelID != "" {
		channel, err := u.discordService.Session.Channel(guild.AnnouncementChannelID)
		if err == nil && channel != nil {
			data.AnnouncementChannelName = "#" + channel.Name
		} else {
			data.AnnouncementChannelName = "Canal Configurado (" + guild.AnnouncementChannelID + ")"
		}
	} else {
		data.AnnouncementChannelName = "Não Configurado"
	}

	// 2. Fetch Guild Roles mapped in local DB
	roles, err := u.pbRepo.FindRolesByGuild(guild.ID)
	if err != nil {
		return data, fmt.Errorf("failed to fetch mapped roles: %w", err)
	}
	data.Roles = roles

	// 3. Fetch Broadcasts History sorted decrescendo
	broadcasts, err := u.pbRepo.FindBroadcastsByGuild(guild.ID)
	if err != nil {
		return data, fmt.Errorf("failed to fetch broadcasts: %w", err)
	}
	data.Broadcasts = broadcasts

	return data, nil
}

// CreateBroadcast provisions a new scheduled broadcast inside the PocketBase database.
func (u *BroadcastUsecase) CreateBroadcast(record *pocketbase.BroadcastRecord) error {
	record.Status = "scheduled"
	record.MetricsSent = 0
	record.MetricsErrors = 0

	var created pocketbase.BroadcastRecord
	err := u.pbRepo.CreateRecord("broadcasts", record, &created)
	if err != nil {
		return fmt.Errorf("failed to create broadcast: %w", err)
	}
	*record = created
	return nil
}

// CancelBroadcast cancels (physically deletes) a scheduled broadcast if it hasn't started processing yet.
func (u *BroadcastUsecase) CancelBroadcast(broadcastID string) error {
	var record pocketbase.BroadcastRecord
	found, err := u.pbRepo.FindByID("broadcasts", broadcastID, &record)
	if err != nil {
		return fmt.Errorf("failed to search broadcast: %w", err)
	}
	if !found {
		return fmt.Errorf("broadcast not found")
	}

	if record.Status != "scheduled" {
		return fmt.Errorf("cannot cancel broadcast that is already '%s'", record.Status)
	}

	err = u.pbRepo.DeleteRecord("broadcasts", broadcastID)
	if err != nil {
		return fmt.Errorf("failed to delete broadcast record: %w", err)
	}

	return nil
}

