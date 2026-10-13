package admin

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func EditBotNameCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	name := ""
	for _, option := range i.ApplicationCommandData().Options {
		if option.Name == "nome" {
			name = strings.TrimSpace(option.StringValue())
		}
	}

	if len([]rune(name)) < 2 || len([]rune(name)) > 32 {
		respond(s, i, "❌ O nome precisa ter entre **2 e 32 caracteres**.")
		return
	}

	user, err := s.UserUpdate(name, "", "")
	if err != nil {
		respond(s, i, "❌ Não foi possível alterar o nome do bot: "+err.Error())
		return
	}

	respond(s, i, fmt.Sprintf("✅ Nome do bot alterado para **%s**.", user.Username))
}

func EditBotAvatarCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	url := ""
	for _, option := range i.ApplicationCommandData().Options {
		if option.Name == "url" {
			url = strings.TrimSpace(option.StringValue())
		}
	}

	if !strings.HasPrefix(strings.ToLower(url), "https://") && !strings.HasPrefix(strings.ToLower(url), "http://") {
		respond(s, i, "❌ Informe uma URL de imagem válida começando com `https://`.")
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		respond(s, i, "❌ Não consegui baixar essa imagem: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respond(s, i, fmt.Sprintf("❌ A imagem respondeu com HTTP `%d`.", resp.StatusCode))
		return
	}

	const maxAvatarSize = 8 << 20
	if resp.ContentLength > maxAvatarSize {
		respond(s, i, "❌ A imagem é grande demais. Use uma imagem de até **8 MB**.")
		return
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAvatarSize+1))
	if err != nil {
		respond(s, i, "❌ Não consegui ler a imagem: "+err.Error())
		return
	}
	if len(data) > maxAvatarSize {
		respond(s, i, "❌ A imagem é grande demais. Use uma imagem de até **8 MB**.")
		return
	}

	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		respond(s, i, "❌ O arquivo informado não parece ser uma imagem válida.")
		return
	}

	avatar := "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
	if _, err := s.UserUpdate("", avatar, ""); err != nil {
		respond(s, i, "❌ Não foi possível alterar a foto do bot: "+err.Error())
		return
	}

	respond(s, i, "✅ **Foto de perfil do bot atualizada!**")
}
