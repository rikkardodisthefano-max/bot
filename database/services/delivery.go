package services

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
)

// EstoqueEmMemoria simula o estoque de itens/keys no servidor (RAM)
type EstoqueManager struct {
	mu     sync.Mutex
	Itens  map[string][]string // Ex: "roblox_code" -> ["CODE1", "CODE2"]
}

var Estoque = &EstoqueManager{
	Itens: make(map[string][]string),
}

// EntregarItemTexto puxa um item do estoque e envia no privado (DM) do cliente
func EntregarItemTexto(s *discordgo.Session, userID string, produtoID string) (string, error) {
	Estoque.mu.Lock()
	defer Estoque.mu.Unlock()

	lista, existe := Estoque.Itens[produtoID]
	if !existe || len(lista) == 0 {
		return "", fmt.Errorf("estoque esgotado para o produto: %s", produtoID)
	}

	// Pega o primeiro item e remove do estoque
	itemEntregue := lista[0]
	Estoque.Itens[produtoID] = lista[1:]

	// Abre o canal de Mensagem Direta (DM) com o usuário
	dm, err := s.UserChannelCreate(userID)
	if err != nil {
		return itemEntregue, fmt.Errorf("não foi possível abrir a DM com o usuário: %v", err)
	}

	// Envia o item no privado
	msg := fmt.Sprintf("🎉 **Pagamento Confirmado!**\n\nAqui está o seu item/código:\n`%s`\n\n*Obrigado pela compra!*", itemEntregue)
	_, err = s.ChannelMessageSend(dm.ID, msg)
	if err != nil {
		return itemEntregue, fmt.Errorf("erro ao enviar mensagem na DM: %v", err)
	}

	return itemEntregue, nil
}