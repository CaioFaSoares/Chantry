package handlers

import (
	"log"

	"chantry/backend/internal/usecases"
	"github.com/gofiber/fiber/v2"
)

type SendBroadcastRequest struct {
	Content    string   `json:"content"`
	TargetType string   `json:"target_type"` // public, private
	RoleIDs    []string `json:"role_ids"`    // opcional
}

type BroadcastHandler struct {
	broadcastUsecase *usecases.BroadcastUsecase
}

func NewBroadcastHandler(usecase *usecases.BroadcastUsecase) *BroadcastHandler {
	return &BroadcastHandler{
		broadcastUsecase: usecase,
	}
}

func (h *BroadcastHandler) HandleSendBroadcast(c *fiber.Ctx) error {
	guildID := c.Params("guildId")
	if guildID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "O parâmetro de rota guildId é obrigatório",
		})
	}

	var req SendBroadcastRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Corpo da requisição JSON inválido ou malformado",
		})
	}

	// Validações básicas de negócio
	if req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "O conteúdo da mensagem (content) não pode ser vazio",
		})
	}

	if req.TargetType != "public" && req.TargetType != "private" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "O tipo de destino (target_type) deve ser 'public' ou 'private'",
		})
	}

	log.Printf("[BROADCAST] Recebida solicitação de disparo para a guilda: %s, tipo: %s", guildID, req.TargetType)

	// Chamar Usecase de disparo
	metrics, err := h.broadcastUsecase.SendBroadcast(guildID, req.Content, req.TargetType, req.RoleIDs)
	if err != nil {
		log.Printf("❌ ERRO [HandleSendBroadcast] na guilda %s: %v", guildID, err)

		// Retornar Bad Request caso o erro seja de configuração pendente
		if err.Error() == "canal de avisos não configurado" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Canal de avisos não configurado para este servidor. Configure nas opções de infraestrutura.",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Erro ao processar o disparo: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Broadcast concluído com sucesso",
		"metrics": metrics,
	})
}
