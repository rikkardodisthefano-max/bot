package admin

import (
	"fmt"
	"log"
	"strings"

	"go-games/database"
	"github.com/bwmarrin/discordgo"
)

// PauseAICommand pausa ou retoma a IA em todos os tickets deste servidor.
// A pausa é global e persistida no SQLite, portanto continua válida após reinício.
func PauseAICommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !ensureAdminAccess(s, i) {
		return
	}

	action := "pausar"
	for _, option := range i.ApplicationCommandData().Options {
		if option.Name == "acao" {
			action = strings.ToLower(strings.TrimSpace(option.StringValue()))
		}
	}

	paused := action != "retomar"
	if err := database.SetGlobalAIPaused(paused); err != nil {
		log.Printf("Erro alterando pausa global da IA: %v", err)
		respond(s, i, "❌ Não foi possível salvar o estado global da IA.")
		return
	}

	if paused {
		respond(s, i, "⏸️ **IA pausada em todos os tickets deste servidor.**\n\nNenhum novo atendimento será processado pela IA até você usar `/pausarIA` novamente com **retomar**.")
		return
	}

	respond(s, i, fmt.Sprintf("▶️ **IA reativada em todos os tickets deste servidor.**\n\nOs tickets que não estiverem pausados individualmente voltaram a receber atendimento automático."))
}
