package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go-games/commands/admin"
	"go-games/commands/client"
	"go-games/config"
	"go-games/database"
	"go-games/handlers"
	"github.com/bwmarrin/discordgo"
)

func startHealthCheckServer() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "Bot-Pro está online!")
	})
	go func() {
		fmt.Printf("Servidor HTTP rodando na porta %s\n", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			fmt.Printf("Erro no servidor HTTP: %v\n", err)
		}
	}()
}

func main() {
	database.InitDB()
	database.InitSubDB()
	database.InitInteractionStore()
	config.Store.Load()
	startHealthCheckServer()

	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		fmt.Println("ERRO: defina DISCORD_BOT_TOKEN.")
		return
	}
	bot, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Erro ao criar sessão do Discord:", err)
		return
	}
	bot.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuildMembers

	// Bot-Pro fica focado em gestão de vendas e clientes.
	bot.AddHandler(handlers.ButtonHandler)

	var adminPermission int64 = discordgo.PermissionAdministrator
	commandsList := []*discordgo.ApplicationCommand{
		{Name: "add-vip", Description: "Registra ou atualiza um cliente e gera a licença", DefaultMemberPermissions: &adminPermission, Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionUser, Name: "usuario", Description: "Cliente", Required: true}, {Type: discordgo.ApplicationCommandOptionString, Name: "guild_id", Description: "ID do servidor do cliente", Required: true}, {Type: discordgo.ApplicationCommandOptionString, Name: "plano", Description: "Plano contratado", Required: true}, {Type: discordgo.ApplicationCommandOptionInteger, Name: "dias", Description: "Dias de licença", Required: true, MinValue: func() *float64 { v := 1.0; return &v }()}}},
		{Name: "resetar_fatura", Description: "Aprova o pagamento e renova a licença", DefaultMemberPermissions: &adminPermission, Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionUser, Name: "usuario", Description: "Cliente", Required: true}, {Type: discordgo.ApplicationCommandOptionInteger, Name: "dias", Description: "Dias a adicionar", Required: true, MinValue: func() *float64 { v := 1.0; return &v }()}}},
		{Name: "lista-compradores", Description: "Exibe todos os clientes e o status das licenças", DefaultMemberPermissions: &adminPermission},
		{Name: "painel-cliente", Description: "Exibe os dados da sua licença", DefaultMemberPermissions: nil},
		{Name: "editar-nome", Description: "Altera o nome global do bot", DefaultMemberPermissions: &adminPermission, Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "nome", Description: "Novo nome do bot (2 a 32 caracteres)", Required: true, MinLength: func() *int { v := 2; return &v }(), MaxLength: 32}}},
		{Name: "editar-pfp", Description: "Altera a foto de perfil do bot", DefaultMemberPermissions: &adminPermission, Options: []*discordgo.ApplicationCommandOption{{Type: discordgo.ApplicationCommandOptionString, Name: "url", Description: "URL pública da nova imagem", Required: true, MinLength: func() *int { v := 8; return &v }(), MaxLength: 2048}}},
	}

	syncCommands := func(s *discordgo.Session, guildID string) {
		if _, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", []*discordgo.ApplicationCommand{}); err != nil {
			fmt.Printf("⚠️ Erro limpando comandos globais: %v\n", err)
		}
		if _, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, guildID, commandsList); err != nil {
			fmt.Printf("Erro ao sincronizar comandos em %s: %v\n", guildID, err)
		} else {
			fmt.Printf("✅ Comandos sincronizados em %s (%d comandos)\n", guildID, len(commandsList))
		}
	}

	bot.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Printf("🤖 Bot-Pro conectado como %s\n", s.State.User.String())
		for _, guild := range r.Guilds {
			syncCommands(s, guild.ID)
		}
	})
	bot.AddHandler(func(s *discordgo.Session, g *discordgo.GuildCreate) {
		syncCommands(s, g.ID)
	})

	bot.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}
		if !database.ClaimInteraction(i.ID) {
			fmt.Printf("⚠️ Interaction duplicada ignorada: %s\n", i.ID)
			return
		}
		switch i.ApplicationCommandData().Name {
		case "add-vip":
			admin.AddVIP(s, i)
		case "resetar_fatura":
			admin.ResetarFaturaCommand(s, i)
		case "lista-compradores":
			admin.ListaCompradores(s, i)
		case "painel-cliente":
			client.PainelCliente(s, i)
		case "editar-nome":
			admin.EditBotNameCommand(s, i)
		case "editar-pfp":
			admin.EditBotAvatarCommand(s, i)
		}
	})

	if err := bot.Open(); err != nil {
		fmt.Println("Erro ao abrir conexão com Discord:", err)
		return
	}
	defer bot.Close()
	fmt.Printf("🤖 Bot-Pro online. Comandos: %d | Gestão de vendas ativa\n", len(commandsList))
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}
