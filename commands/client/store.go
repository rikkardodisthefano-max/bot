package client

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"go-games/database"
)

func PainelCliente(s *discordgo.Session, i *discordgo.InteractionCreate) {
	sub, err := database.GetUserSubscription(i.Member.User.ID)
	if err != nil {
		respond(s, i, "❌ Nenhuma licença vinculada à sua conta foi encontrada.")
		return
	}

	dias := int(time.Until(sub.ExpiresAt).Hours() / 24)
	if dias < 0 { dias = 0 }

	status := "🟢 Ativo"
	if sub.Status != "pago" { status = "🔴 Pendente" }

	embed := &discordgo.MessageEmbed{
		Title: "🖥️ Painel do Cliente — Bot-Pro",
		Description: fmt.Sprintf(
			"**Plano:** `%s`\n**Status:** %s\n**Dias restantes:** `%d dias`\n**Vencimento:** `%s`\n**Servidor vinculado:** `%s`\n**Token:** `%s`\n\n"+
				"Para renovar sua assinatura, abra um ticket no canal de renovação da fatura: <#1548353694511136888>.\n\n"+
				"Use este canal para falar com a equipe sobre pagamento, renovação e envio de comprovante.",
			sub.PlanName,
			status,
			dias,
			sub.ExpiresAt.Format("02/01/2006"),
			sub.GuildID,
			sub.Token,
		),
		Color: 0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{Text: "Bot-Pro • Painel do cliente"},
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, text string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: text, Flags: discordgo.MessageFlagsEphemeral},
	})
}
