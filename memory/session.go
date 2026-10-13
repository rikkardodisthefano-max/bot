package memory

import "sync"

type EstadoCliente struct {
	GuildID             string
	UserID              string
	ChannelID           string
	HistoricoChat       []string
	TentativasInvalidas int
	AguardandoPix       bool
}

type SessionManager struct {
	mu      sync.RWMutex
	sessoes map[string]*EstadoCliente
}

var GlobalSessions = &SessionManager{
	sessoes: make(map[string]*EstadoCliente),
}

// Get devolve uma cópia da sessão para impedir data races enquanto a IA trabalha.
func (m *SessionManager) Get(sessionID string) (*EstadoCliente, bool) {
	m.mu.RLock()
	estado, ok := m.sessoes[sessionID]
	if !ok {
		m.mu.RUnlock()
		return nil, false
	}

	copia := *estado
	copia.HistoricoChat = append([]string(nil), estado.HistoricoChat...)
	m.mu.RUnlock()

	return &copia, true
}

// AddHistorico cria a sessão automaticamente e mantém apenas as últimas 10 mensagens.
func (m *SessionManager) AddHistorico(sessionID string, mensagem string) {
	if mensagem == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	estado := m.sessoes[sessionID]
	if estado == nil {
		estado = &EstadoCliente{}
		m.sessoes[sessionID] = estado
	}

	estado.HistoricoChat = append(estado.HistoricoChat, mensagem)
	if len(estado.HistoricoChat) > 10 {
		estado.HistoricoChat = estado.HistoricoChat[len(estado.HistoricoChat)-10:]
	}
}

// Set define ou substitui uma sessão.
func (m *SessionManager) Set(sessionID string, estado *EstadoCliente) {
	if estado == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	copia := *estado
	copia.HistoricoChat = append([]string(nil), estado.HistoricoChat...)
	m.sessoes[sessionID] = &copia
}

// Delete limpa a memória do ticket.
func (m *SessionManager) Delete(sessionID string) {
	m.mu.Lock()
	delete(m.sessoes, sessionID)
	m.mu.Unlock()
}
