package handlers

import (
	"fmt"
	"strings"
	"unicode"

	"go-games/config"
	"go-games/database"
	"go-games/memory"
	"github.com/bwmarrin/discordgo"
)

func ButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent { return }
	data := i.MessageComponentData()
	switch data.CustomID {
	case "btn_abrir_ticket", "btn_criar_ticket": criarTicket(s, i, "")
	case "ticket_menu": categoria := ""; if len(data.Values) > 0 { categoria = data.Values[0] }; criarTicket(s, i, categoria)
	case "btn_fechar_ticket": fecharTicket(s, i)
	}
}

func criarTicket(s *discordgo.Session, i *discordgo.InteractionCreate, categoria string) {
	if i.Member == nil || i.Member.User == nil || i.GuildID == "" { return }
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}}); err != nil { fmt.Printf("Erro ao responder criação do ticket: %v\n", err); return }

	guildID, userID, username := i.GuildID, i.Member.User.ID, i.Member.User.Username
	channelName := "ticket-" + sanitizeChannelPart(username)
	if categoria != "" { channelName += "-" + sanitizeChannelPart(categoria) }
	if len(channelName) > 100 { channelName = channelName[:100] }

	channel, err := s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name: channelName, Type: discordgo.ChannelTypeGuildText,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{ID: guildID, Type: discordgo.PermissionOverwriteTypeRole, Deny: discordgo.PermissionViewChannel},
			{ID: userID, Type: discordgo.PermissionOverwriteTypeMember, Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory},
		},
	})
	if err != nil { msg := "❌ Ocorreu um erro ao criar seu ticket. Verifique se o bot possui **Gerenciar Canais**."; _, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg}); return }

	memory.GlobalSessions.Delete(channel.ID)
	config.Store.SaveTicket(channel.ID, &config.TicketState{IsPausado: false})
	msg := fmt.Sprintf("✅ Seu ticket foi criado com sucesso: <#%s>", channel.ID)
	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg})

	settings := database.GetBotSettings(guildID)
	titulo := strings.TrimSpace(settings.TicketTitle)
	if titulo == "" { titulo = "🤖 Suporte Automático & Atendimento" }
	mensagem := strings.TrimSpace(settings.TicketMessage)
	if mensagem == "" { mensagem = "Olá {usuario}!\n\nDescreva seu problema abaixo. Nossa IA irá analisar sua solicitação e, quando necessário, encaminhará para a equipe." }
	mensagem = strings.ReplaceAll(mensagem, "{usuario}", "<@"+userID+">")
	categoriaTexto := strings.TrimPrefix(categoria, "ticket_")
	categoriaTexto = strings.ReplaceAll(categoriaTexto, "-", " ")
	mensagem = strings.ReplaceAll(mensagem, "{categoria}", categoriaTexto)
	if categoria != "" && !strings.Contains(titulo, categoriaTexto) { titulo += " • " + categoriaTexto }

	embedTicket := &discordgo.MessageEmbed{Title: titulo, Description: mensagem, Color: 0x5865F2}
	if strings.TrimSpace(settings.TicketFooter) != "" { embedTicket.Footer = &discordgo.MessageEmbedFooter{Text: settings.TicketFooter} }
	btnFechar := discordgo.Button{Label: "Fechar Ticket", Style: discordgo.DangerButton, CustomID: "btn_fechar_ticket", Emoji: &discordgo.ComponentEmoji{Name: "🔒"}}
	if _, err := s.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embedTicket}, Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{btnFechar}}}}); err != nil { fmt.Printf("Erro ao enviar mensagem inicial do ticket: %v\n", err) }
}

func fecharTicket(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: "🔒 Encerrando e apagando este ticket...", Flags: discordgo.MessageFlagsEphemeral}}); err != nil { fmt.Printf("Erro ao responder fechamento do ticket: %v\n", err); return }
	memory.GlobalSessions.Delete(i.ChannelID)
	config.Store.SaveTicket(i.ChannelID, &config.TicketState{IsPausado: true})
	if _, err := s.ChannelDelete(i.ChannelID); err != nil { fmt.Printf("Erro ao apagar canal do ticket: %v\n", err) }
}

func sanitizeChannelPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value)); var b strings.Builder; lastDash := false
	for _, r := range value { switch { case r >= 'a' && r <= 'z', r >= '0' && r <= '9': b.WriteRune(r); lastDash = false; case r == '-' || unicode.IsSpace(r): if b.Len() > 0 && !lastDash { b.WriteByte('-'); lastDash = true } } }
	result := strings.Trim(b.String(), "-"); if result == "" { return "usuario" }; return result
}
