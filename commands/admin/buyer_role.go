package admin

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"go-games/database"
)

func SetBuyerRoleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !isAdministrator(i) {
		respond(s, i, "❌ Você precisa da permissão de Administrador para configurar o cargo de comprador.")
		return
	}

	var role *discordgo.Role
	for _, option := range i.ApplicationCommandData().Options {
		if option.Name == "cargo" {
			role = option.RoleValue(s, i.GuildID)
		}
	}
	if role == nil {
		respond(s, i, "❌ Não consegui encontrar o cargo informado.")
		return
	}
	if role.Managed {
		respond(s, i, "❌ Cargos gerenciados por integrações não podem ser atribuídos pelo bot.")
		return
	}

	if err := database.SetBuyerRole(i.GuildID, role.ID); err != nil {
		respond(s, i, fmt.Sprintf("❌ Não foi possível salvar o cargo de comprador: %v", err))
		return
	}

	respond(s, i, fmt.Sprintf("✅ **Cargo de comprador configurado!**\n\nCargo: <@&%s>\n\nQuando `/add-vip` for usado neste servidor, o cliente receberá este cargo automaticamente.", role.ID))
}
