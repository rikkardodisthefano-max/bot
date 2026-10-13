package handlers

import (
	"fmt"
	"strings"

	"go-games/config"
	"go-games/database"
	"github.com/bwmarrin/discordgo"
)

// RateLimitGate roda antes da IA. Ele não chama a Gemini: prepara o contexto
// e bloqueia atomicamente mensagens que não devem chegar ao processamento da IA.
func RateLimitGate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m == nil || m.Author == nil || m.Author.Bot || m.GuildID == "" || m.ID == "" {
		return
	}

	channel, err := s.State.Channel(m.ChannelID)
	if err != nil || channel == nil || !strings.HasPrefix(channel.Name, "ticket-") {
		return
	}

	// A pausa global é verificada ANTES de !voltaria. Assim nenhum usuário
	// consegue contornar /pausar-ia reativando um ticket individualmente.
	if database.GetGlobalAIPaused() {
		database.BlockAIMessage(m.ID, "IA globalmente pausada")
		return
	}

	content := strings.TrimSpace(m.Content)
	ticket := config.Store.GetTicket(m.ChannelID)
	if strings.EqualFold(content, "!voltaria") || ticket.IsPausado {
		database.ExemptAIMessage(m.ID)
		return
	}

	allowed, reason := database.CanAIRequest(m.GuildID)
	if !allowed {
		if database.BlockAIMessage(m.ID, reason) {
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("⏳ **Limite da IA atingido.** %s. Tente novamente em instantes ou solicite atendimento humano.", reason))
		}
		return
	}
	database.PrepareAIMessage(m.ID, m.GuildID)
}
