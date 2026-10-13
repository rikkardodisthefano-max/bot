package admin

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"go-games/database"
)

func AddVIP(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var userID, guildID, plan string
	var days int
	for _, o := range i.ApplicationCommandData().Options {
		switch o.Name {
		case "usuario":
			user := o.UserValue(s)
			if user != nil { userID = user.ID }
		case "guild_id": guildID = o.StringValue()
		case "plano": plan = o.StringValue()
		case "dias": days = int(o.IntValue())
		}
	}
	if userID == "" || guildID == "" || plan == "" || days <= 0 {
		respond(s, i, "❌ Preencha todos os dados corretamente.")
		return
	}

	token, err := database.AddOrUpdateVIP(userID, guildID, plan, days)
	if err != nil {
		respond(s, i, "❌ Erro ao registrar cliente: "+err.Error())
		return
	}

	roleStatus := ""
	buyerRoleID := database.GetBuyerRole(i.GuildID)
	if buyerRoleID == "" {
		roleStatus = "\n\nℹ️ Nenhum cargo de comprador foi configurado neste servidor. Use `/set-cargo-comprador`."
	} else if err := s.GuildMemberRoleAdd(i.GuildID, userID, buyerRoleID); err != nil {
		roleStatus = "\n\n⚠️ Não consegui adicionar o cargo <@&" + buyerRoleID + ">. Verifique se o bot tem **Gerenciar Cargos**, se o usuário está neste servidor e se o cargo está abaixo do cargo mais alto do bot."
	} else {
		roleStatus = "\n\n✅ Cargo de comprador <@&" + buyerRoleID + "> atribuído automaticamente."
	}

	embed := &discordgo.MessageEmbed{
		Title: "🎉 Cliente registrado",
		Description: fmt.Sprintf("**Cliente:** <@%s>\n**Plano:** `%s`\n**Duração:** `%d dias`\n**Servidor contratado:** `%s`\n\n🔑 **Token:** `%s`%s", userID, plan, days, guildID, token, roleStatus),
		Color: 0x57F287,
		Footer: &discordgo.MessageEmbedFooter{Text: "Bot-Pro • Gerenciamento de clientes"},
	}
	respondEmbed(s, i, embed)
}

func ResetarFaturaCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var userID string
	var days int
	for _, o := range i.ApplicationCommandData().Options {
		switch o.Name {
		case "usuario":
			user := o.UserValue(s)
			if user != nil { userID = user.ID }
		case "dias": days = int(o.IntValue())
		}
	}
	if userID == "" || days <= 0 { respond(s, i, "❌ Informe um usuário e uma quantidade de dias válida."); return }
	if err := database.ResetarFatura(userID, days); err != nil { respond(s, i, "❌ Não foi possível renovar: "+err.Error()); return }
	respond(s, i, fmt.Sprintf("✅ Fatura de <@%s> renovada por mais **%d dias**.", userID, days))
}

func ListaCompradores(s *discordgo.Session, i *discordgo.InteractionCreate) {
	pagos, err := database.GetSubscriptionsByStatus("pago")
	if err != nil { respond(s, i, "❌ Erro ao consultar clientes: "+err.Error()); return }
	pendentes, err := database.GetSubscriptionsByStatus("pendente")
	if err != nil { respond(s, i, "❌ Erro ao consultar pagamentos pendentes: "+err.Error()); return }

	active := ""
	for _, v := range pagos { active += fmt.Sprintf("• <@%s> — `%s` — vence **%s**\n", v.UserID, v.PlanName, v.ExpiresAt.Format("02/01/2006")) }
	if active == "" { active = "Nenhum cliente ativo." }
	pending := ""
	for _, v := range pendentes { pending += fmt.Sprintf("• <@%s> — `%s` — ⚠️ pendente\n", v.UserID, v.GuildID) }
	if pending == "" { pending = "Nenhum pagamento pendente." }

	embed := &discordgo.MessageEmbed{
		Title: "📊 Central de Clientes",
		Description: "Visão geral das assinaturas cadastradas.",
		Fields: []*discordgo.MessageEmbedField{{Name: "🟢 Ativos", Value: active}, {Name: "🔴 Pendentes", Value: pending}},
		Color: 0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{Text: "Bot-Pro • Central de clientes"},
	}
	respondEmbed(s, i, embed)
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, text string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: text, Flags: discordgo.MessageFlagsEphemeral}})
}

func respondEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, e *discordgo.MessageEmbed) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{e}, Flags: discordgo.MessageFlagsEphemeral}})
}
