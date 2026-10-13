package config

import (
	"go-games/database"
	"sync"
)

type ServerConfig struct {
	GuildID          string `json:"guild_id"`
	UnverifiedRoleID string `json:"unverified_role_id"`
	VerifiedRoleID   string `json:"verified_role_id"`
	SystemPrompt     string `json:"system_prompt"`
	WelcomeChannel   string `json:"welcome_channel"`
	WelcomeMessage   string `json:"welcome_message"`
	AutoRole         string `json:"auto_role"`
}

type TicketState struct {
	IsPausado bool `json:"is_pausado"`
}

type ConfigStore struct {
	mu      sync.RWMutex
	Servers map[string]*ServerConfig
	Tickets map[string]*TicketState
}

var Store = &ConfigStore{
	Servers: make(map[string]*ServerConfig),
	Tickets: make(map[string]*TicketState),
}

func (c *ConfigStore) GetServer(guildID string) *ServerConfig {
	c.mu.RLock()
	if cfg, exists := c.Servers[guildID]; exists {
		c.mu.RUnlock()
		return cfg
	}
	c.mu.RUnlock()

	dbCfg := database.GetServerConfig(guildID)
	cfg := &ServerConfig{
		GuildID:          dbCfg.GuildID,
		UnverifiedRoleID: dbCfg.UnverifiedRoleID,
		VerifiedRoleID:   dbCfg.VerifiedRoleID,
		SystemPrompt:     dbCfg.SystemPrompt,
		WelcomeChannel:   dbCfg.WelcomeChannel,
		WelcomeMessage:   dbCfg.WelcomeMessage,
		AutoRole:         dbCfg.AutoRole,
	}

	c.mu.Lock()
	c.Servers[guildID] = cfg
	c.mu.Unlock()

	return cfg
}

func (c *ConfigStore) SaveServer(cfg *ServerConfig) {
	c.mu.Lock()
	c.Servers[cfg.GuildID] = cfg
	c.mu.Unlock()

	database.SaveServerConfig(&database.ServerConfigData{
		GuildID:          cfg.GuildID,
		UnverifiedRoleID: cfg.UnverifiedRoleID,
		VerifiedRoleID:   cfg.VerifiedRoleID,
		SystemPrompt:     cfg.SystemPrompt,
		WelcomeChannel:   cfg.WelcomeChannel,
		WelcomeMessage:   cfg.WelcomeMessage,
		AutoRole:         cfg.AutoRole,
	})
}

func (c *ConfigStore) GetTicket(channelID string) *TicketState {
	c.mu.RLock()
	if ticket, exists := c.Tickets[channelID]; exists {
		c.mu.RUnlock()
		return ticket
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	t := &TicketState{IsPausado: false}
	c.Tickets[channelID] = t
	return t
}

func (c *ConfigStore) SaveTicket(channelID string, ticket *TicketState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Tickets[channelID] = ticket
}

func (c *ConfigStore) Load() {
	c.mu.Lock()
	defer c.mu.Unlock()

	dbConfigs := database.LoadAllServerConfigs()
	for guildID, dbCfg := range dbConfigs {
		c.Servers[guildID] = &ServerConfig{
			GuildID:          dbCfg.GuildID,
			UnverifiedRoleID: dbCfg.UnverifiedRoleID,
			VerifiedRoleID:   dbCfg.VerifiedRoleID,
			SystemPrompt:     dbCfg.SystemPrompt,
			WelcomeChannel:   dbCfg.WelcomeChannel,
			WelcomeMessage:   dbCfg.WelcomeMessage,
			AutoRole:         dbCfg.AutoRole,
		}
	}
}