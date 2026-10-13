package admin

import (
    "crypto/sha256"
    "crypto/subtle"
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/bwmarrin/discordgo"
    "go-games/database"
)

const adminSessionDuration = time.Hour

var authMu sync.Mutex

func isAdministrator(i *discordgo.InteractionCreate) bool {
    if i == nil || i.Member == nil || i.Member.User == nil {
        return false
    }
    if i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
        return false
    }
    ownerID := strings.TrimSpace(os.Getenv("BOT_OWNER_ID"))
    return ownerID == "" || ownerID == i.Member.User.ID
}

func ensureAdminAccess(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
    if !isAdministrator(i) {
        respond(s, i, "❌ Você não tem acesso a esta função.")
        return false
    }

    userID := i.Member.User.ID
    if database.AdminIsLocked(userID) {
        respond(s, i, "🔒 Seu acesso administrativo está bloqueado permanentemente após 3 tentativas incorretas.")
        return false
    }
    if database.AdminIsAuthenticated(userID) {
        return true
    }

    err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseModal,
        Data: &discordgo.InteractionResponseData{
            CustomID: "admin_login",
            Title:    "🔐 Acesso administrativo",
            Components: []discordgo.MessageComponent{
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{
                        CustomID:    "admin_password",
                        Label:       "Senha de administrador",
                        Style:       discordgo.TextInputShort,
                        Placeholder: "Digite a senha administrativa",
                        Required:    true,
                        MinLength:   8,
                        MaxLength:   200,
                    },
                }},
            },
        },
    })
    if err != nil {
        log.Printf("erro ao abrir login administrativo: %v", err)
    }
    return false
}

func expectedPasswordHash() ([]byte, error) {
    password := os.Getenv("BOT_ADMIN_PASSWORD")
    if password == "" {
        return nil, fmt.Errorf("BOT_ADMIN_PASSWORD não configurada")
    }
    sum := sha256.Sum256([]byte(password))
    return sum[:], nil
}

func setActivity(s *discordgo.Session, kind, text string) (int, error) {
    text = strings.TrimSpace(text)
    if text == "" {
        return 0, fmt.Errorf("o texto da atividade não pode ficar vazio")
    }
    switch kind {
    case "jogando":
        return 0, s.UpdateGameStatus(0, text)
    case "ouvindo":
        return 2, s.UpdateListeningStatus(text)
    case "assistindo":
        return 3, s.UpdateWatchStatus(0, text)
    case "competindo":
        return 5, s.UpdateGameStatus(0, text)
    default:
        return 0, fmt.Errorf("tipo de atividade inválido")
    }
}

func ActivityCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
    if !ensureAdminAccess(s, i) {
        return
    }
    var kind, text string
    for _, option := range i.ApplicationCommandData().Options {
        switch option.Name {
        case "tipo":
            kind = option.StringValue()
        case "texto":
            text = option.StringValue()
        }
    }
    activityType, err := setActivity(s, kind, text)
    if err != nil {
        respond(s, i, "❌ "+err.Error())
        return
    }
    if err := database.SaveGlobalActivity(activityType, strings.TrimSpace(text)); err != nil {
        log.Printf("erro ao salvar atividade: %v", err)
    }
    respond(s, i, "✅ **Atividade atualizada!**")
}

func TicketTemplateCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
    if ensureAdminAccess(s, i) {
        openTicketModal(s, i)
    }
}

func LimitsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
    if ensureAdminAccess(s, i) {
        openLimitsModal(s, i)
    }
}

func AdminPanelCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
    if !ensureAdminAccess(s, i) {
        return
    }
    cfg := database.GetBotSettings(i.GuildID)
    title, message := ticketDefaults(cfg)
    _, activity := database.GetGlobalActivity()
    if activity == "" {
        activity = "Não configurada"
    }

    embed := &discordgo.MessageEmbed{
        Title:       "⚙️ Painel administrativo",
        Description: "Central protegida para configurar atendimento, atividade e limites da IA.",
        Color:       0x5865F2,
        Fields: []*discordgo.MessageEmbedField{
            {Name: "🎮 Atividade", Value: activity, Inline: false},
            {Name: "🎫 Ticket", Value: fmt.Sprintf("**%s**\n%s", title, message), Inline: false},
            {Name: "🤖 Limites", Value: fmt.Sprintf("**RPM:** `%d`\n**RPD:** `%d`", cfg.RPM, cfg.RPD), Inline: true},
        },
        Footer: &discordgo.MessageEmbedFooter{Text: "RikksLabels • Painel protegido"},
    }

    components := []discordgo.MessageComponent{
        discordgo.ActionsRow{Components: []discordgo.MessageComponent{
            discordgo.Button{Label: "Atividade", Style: discordgo.PrimaryButton, CustomID: "admin_activity"},
            discordgo.Button{Label: "Ticket", Style: discordgo.SecondaryButton, CustomID: "admin_ticket"},
            discordgo.Button{Label: "Limites IA", Style: discordgo.SuccessButton, CustomID: "admin_limits"},
        }},
    }

    _ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Embeds:     []*discordgo.MessageEmbed{embed},
            Components: components,
            Flags:      discordgo.MessageFlagsEphemeral,
        },
    })
}

func ComponentHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
    if i.Type != discordgo.InteractionMessageComponent {
        return
    }
    switch i.MessageComponentData().CustomID {
    case "admin_activity":
        if ensureAdminAccess(s, i) {
            openActivityModal(s, i)
        }
    case "admin_ticket":
        if ensureAdminAccess(s, i) {
            openTicketModal(s, i)
        }
    case "admin_limits":
        if ensureAdminAccess(s, i) {
            openLimitsModal(s, i)
        }
    }
}

func ModalHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
    if i.Type != discordgo.InteractionModalSubmit {
        return
    }
    data := i.ModalSubmitData()
    switch data.CustomID {
    case "admin_login":
        handleLogin(s, i, data)
    case "admin_ticket_template":
        handleTicket(s, i, data)
    case "admin_limits":
        handleLimits(s, i, data)
    case "admin_activity":
        handleActivity(s, i, data)
    }
}

func handleLogin(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
    if !isAdministrator(i) || i.Member == nil || i.Member.User == nil {
        respond(s, i, "❌ Acesso negado.")
        return
    }
    userID := i.Member.User.ID
    if database.AdminIsLocked(userID) {
        respond(s, i, "🔒 Seu acesso está bloqueado permanentemente.")
        return
    }
    expected, err := expectedPasswordHash()
    if err != nil {
        respond(s, i, "❌ "+err.Error())
        return
    }
    provided := sha256.Sum256([]byte(modalValue(data, "admin_password")))
    authMu.Lock()
    defer authMu.Unlock()
    if subtle.ConstantTimeCompare(provided[:], expected) != 1 {
        attempts, locked := database.RegisterAdminFailure(userID)
        if locked {
            respond(s, i, "🔒 **Acesso bloqueado permanentemente.** Você atingiu 3 tentativas incorretas.")
        } else {
            respond(s, i, fmt.Sprintf("❌ Senha incorreta. Tentativa **%d/3**.", attempts))
        }
        return
    }
    if err := database.AuthenticateAdmin(userID, time.Now().Add(adminSessionDuration)); err != nil {
        respond(s, i, "❌ Não foi possível salvar a sessão administrativa.")
        return
    }
    respond(s, i, "🔓 **Acesso liberado por 1 hora.**")
}

func handleTicket(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
    if !ensureModalAdmin(s, i) {
        return
    }
    title := strings.TrimSpace(modalValue(data, "ticket_title"))
    message := strings.TrimSpace(modalValue(data, "ticket_message"))
    footer := strings.TrimSpace(modalValue(data, "ticket_footer"))
    if title == "" || message == "" {
        respond(s, i, "❌ Título e mensagem são obrigatórios.")
        return
    }
    if err := database.SaveTicketTemplate(i.GuildID, title, message, footer); err != nil {
        respond(s, i, "❌ Erro ao salvar o modelo: "+err.Error())
        return
    }
    respond(s, i, "✅ **Modelo do ticket atualizado!** Novos tickets usarão essa mensagem.")
}

func handleLimits(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
    if !ensureModalAdmin(s, i) {
        return
    }
    rpm, rpmErr := strconv.Atoi(strings.TrimSpace(modalValue(data, "ai_rpm")))
    rpd, rpdErr := strconv.Atoi(strings.TrimSpace(modalValue(data, "ai_rpd")))
    if rpmErr != nil || rpdErr != nil || rpm < 0 || rpd < 0 {
        respond(s, i, "❌ RPM e RPD devem ser números inteiros maiores ou iguais a zero.")
        return
    }
    if err := database.SaveAILimits(i.GuildID, rpm, rpd); err != nil {
        respond(s, i, "❌ Erro ao salvar limites: "+err.Error())
        return
    }
    respond(s, i, fmt.Sprintf("✅ **Limites atualizados!**\nRPM: `%d`\nRPD: `%d`\n\n`0` = ilimitado.", rpm, rpd))
}

func handleActivity(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
    if !ensureModalAdmin(s, i) {
        return
    }
    kind := strings.ToLower(strings.TrimSpace(modalValue(data, "activity_type")))
    text := strings.TrimSpace(modalValue(data, "activity_text"))
    activityType, err := setActivity(s, kind, text)
    if err != nil {
        respond(s, i, "❌ "+err.Error())
        return
    }
    if err := database.SaveGlobalActivity(activityType, text); err != nil {
        log.Printf("erro ao salvar atividade: %v", err)
    }
    respond(s, i, "✅ **Atividade atualizada!**")
}

func openTicketModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
    cfg := database.GetBotSettings(i.GuildID)
    title, message := ticketDefaults(cfg)
    err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseModal,
        Data: &discordgo.InteractionResponseData{
            CustomID: "admin_ticket_template",
            Title:    "🎫 Personalizar ticket",
            Components: []discordgo.MessageComponent{
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "ticket_title", Label: "Título do embed", Style: discordgo.TextInputShort, Value: title, Required: true, MaxLength: 256},
                }},
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "ticket_message", Label: "Mensagem", Style: discordgo.TextInputParagraph, Placeholder: "Use {usuario} e {categoria}.", Value: message, Required: true, MaxLength: 2000},
                }},
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "ticket_footer", Label: "Rodapé (opcional)", Style: discordgo.TextInputShort, Value: cfg.TicketFooter, Required: false, MaxLength: 200},
                }},
            },
        },
    })
    if err != nil {
        log.Printf("erro ao abrir modal de ticket: %v", err)
    }
}

func openLimitsModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
    cfg := database.GetBotSettings(i.GuildID)
    err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseModal,
        Data: &discordgo.InteractionResponseData{
            CustomID: "admin_limits",
            Title:    "⚙️ Limites da IA",
            Components: []discordgo.MessageComponent{
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "ai_rpm", Label: "RPM — chamadas por minuto", Style: discordgo.TextInputShort, Value: strconv.Itoa(cfg.RPM), Required: true, MaxLength: 6},
                }},
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "ai_rpd", Label: "RPD — chamadas por dia", Style: discordgo.TextInputShort, Value: strconv.Itoa(cfg.RPD), Required: true, MaxLength: 9},
                }},
            },
        },
    })
    if err != nil {
        log.Printf("erro ao abrir modal de limites: %v", err)
    }
}

func openActivityModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
    err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseModal,
        Data: &discordgo.InteractionResponseData{
            CustomID: "admin_activity",
            Title:    "🎮 Alterar atividade",
            Components: []discordgo.MessageComponent{
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "activity_type", Label: "Tipo", Style: discordgo.TextInputShort, Value: "jogando", Placeholder: "jogando, assistindo, ouvindo ou competindo", Required: true, MaxLength: 20},
                }},
                discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                    discordgo.TextInput{CustomID: "activity_text", Label: "Texto da atividade", Style: discordgo.TextInputShort, Placeholder: "Ex.: suporte 24h | RikksLabels", Required: true, MaxLength: 128},
                }},
            },
        },
    })
    if err != nil {
        log.Printf("erro ao abrir modal de atividade: %v", err)
    }
}

func ensureModalAdmin(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
    if !isAdministrator(i) || i.Member == nil || i.Member.User == nil {
        respond(s, i, "🔒 Acesso administrativo negado.")
        return false
    }
    userID := i.Member.User.ID
    if database.AdminIsLocked(userID) || !database.AdminIsAuthenticated(userID) {
        respond(s, i, "🔒 Sua sessão administrativa expirou ou está bloqueada. Use o comando novamente.")
        return false
    }
    return true
}

func modalValue(data discordgo.ModalSubmitInteractionData, customID string) string {
    for _, row := range data.Components {
        actions, ok := row.(*discordgo.ActionsRow)
        if !ok {
            continue
        }
        for _, component := range actions.Components {
            input, ok := component.(*discordgo.TextInput)
            if ok && input.CustomID == customID {
                return input.Value
            }
        }
    }
    return ""
}

func ticketDefaults(cfg database.BotSettings) (string, string) {
    title := strings.TrimSpace(cfg.TicketTitle)
    message := strings.TrimSpace(cfg.TicketMessage)
    if title == "" {
        title = "🤖 Suporte Automático & Atendimento"
    }
    if message == "" {
        message = "Olá {usuario}!\n\nDescreva seu problema abaixo. Nossa IA irá analisar sua solicitação e, quando necessário, encaminhará para a equipe."
    }
    return title, message
}
