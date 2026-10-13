package database

import (
	"database/sql"
	"fmt"
	"time"
)

type BotSettings struct {
	GuildID       string
	ActivityType  int
	Activity      string
	TicketTitle   string
	TicketMessage string
	TicketFooter  string
	RPM           int
	RPD           int
	AIPaused      bool
}

func InitAdminSettings() {
	if DB == nil {
		return
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS bot_settings (guild_id TEXT PRIMARY KEY, activity_type INTEGER DEFAULT 0, activity TEXT DEFAULT '', ticket_title TEXT DEFAULT '', ticket_message TEXT DEFAULT '', ticket_footer TEXT DEFAULT '', rpm INTEGER DEFAULT 10, rpd INTEGER DEFAULT 500);`,
		`CREATE TABLE IF NOT EXISTS ai_usage (guild_id TEXT PRIMARY KEY, rpm_count INTEGER DEFAULT 0, rpm_window INTEGER DEFAULT 0, rpd_count INTEGER DEFAULT 0, rpd_day TEXT DEFAULT '');`,
		`CREATE TABLE IF NOT EXISTS admin_auth (user_id TEXT PRIMARY KEY, failed_attempts INTEGER DEFAULT 0, locked INTEGER DEFAULT 0, authenticated_until INTEGER DEFAULT 0);`,
	}
	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			panic(fmt.Errorf("erro criando configuracoes administrativas: %w", err))
		}
	}

	// Migração segura para bancos criados por versões anteriores.
	var exists int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('bot_settings') WHERE name='ai_paused'`).Scan(&exists); err != nil {
		panic(fmt.Errorf("erro verificando migração de ai_paused: %w", err))
	}
	if exists == 0 {
		if _, err := DB.Exec(`ALTER TABLE bot_settings ADD COLUMN ai_paused INTEGER DEFAULT 0`); err != nil {
			panic(fmt.Errorf("erro adicionando coluna ai_paused: %w", err))
		}
	}
}

func GetBotSettings(guildID string) BotSettings {
	cfg := BotSettings{GuildID: guildID, RPM: 10, RPD: 500}
	if DB == nil {
		return cfg
	}
	var paused int
	_ = DB.QueryRow(`SELECT activity_type, activity, ticket_title, ticket_message, ticket_footer, rpm, rpd, ai_paused FROM bot_settings WHERE guild_id = ?`, guildID).Scan(&cfg.ActivityType, &cfg.Activity, &cfg.TicketTitle, &cfg.TicketMessage, &cfg.TicketFooter, &cfg.RPM, &cfg.RPD, &paused)
	cfg.AIPaused = paused == 1
	return cfg
}

func SaveActivity(guildID string, activityType int, activity string) error {
	_, err := DB.Exec(`INSERT INTO bot_settings (guild_id, activity_type, activity) VALUES (?, ?, ?) ON CONFLICT(guild_id) DO UPDATE SET activity_type=excluded.activity_type, activity=excluded.activity`, guildID, activityType, activity)
	return err
}

func SaveTicketTemplate(guildID, title, message, footer string) error {
	_, err := DB.Exec(`INSERT INTO bot_settings (guild_id, ticket_title, ticket_message, ticket_footer) VALUES (?, ?, ?, ?) ON CONFLICT(guild_id) DO UPDATE SET ticket_title=excluded.ticket_title, ticket_message=excluded.ticket_message, ticket_footer=excluded.ticket_footer`, guildID, title, message, footer)
	return err
}

func SaveAILimits(guildID string, rpm, rpd int) error {
	_, err := DB.Exec(`INSERT INTO bot_settings (guild_id, rpm, rpd) VALUES (?, ?, ?) ON CONFLICT(guild_id) DO UPDATE SET rpm=excluded.rpm, rpd=excluded.rpd`, guildID, rpm, rpd)
	return err
}

func SetGlobalAIPaused(paused bool) error {
	value := 0
	if paused {
		value = 1
	}
	_, err := DB.Exec(`INSERT INTO bot_settings (guild_id, ai_paused) VALUES ('__global__', ?) ON CONFLICT(guild_id) DO UPDATE SET ai_paused=excluded.ai_paused`, value)
	return err
}

func GetGlobalAIPaused() bool {
	if DB == nil {
		return false
	}
	var paused int
	if err := DB.QueryRow(`SELECT ai_paused FROM bot_settings WHERE guild_id='__global__'`).Scan(&paused); err != nil {
		return false
	}
	return paused == 1
}

func CheckAndConsumeAIRequest(guildID string) (bool, string) {
	if DB == nil {
		return false, "banco indisponível"
	}
	cfg := GetBotSettings(guildID)
	if cfg.RPM == 0 && cfg.RPD == 0 {
		return true, ""
	}
	now := time.Now().UTC()
	minute := now.Truncate(time.Minute).Unix()
	day := now.Format("2006-01-02")
	tx, err := DB.Begin()
	if err != nil {
		return false, "não foi possível abrir o controle de limite"
	}
	defer tx.Rollback()
	var rpmCount, rpdCount int
	var rpmWindow int64
	var rpdDay string
	err = tx.QueryRow(`SELECT rpm_count, rpm_window, rpd_count, rpd_day FROM ai_usage WHERE guild_id=?`, guildID).Scan(&rpmCount, &rpmWindow, &rpdCount, &rpdDay)
	if err == sql.ErrNoRows {
		if _, err = tx.Exec(`INSERT INTO ai_usage (guild_id, rpm_count, rpm_window, rpd_count, rpd_day) VALUES (?, 0, ?, 0, ?)`, guildID, minute, day); err != nil {
			return false, "não foi possível inicializar o controle de limite"
		}
		rpmWindow, rpdDay = minute, day
	} else if err != nil {
		return false, "não foi possível ler o controle de limite"
	}
	if rpmWindow != minute {
		rpmCount = 0
		rpmWindow = minute
	}
	if rpdDay != day {
		rpdCount = 0
		rpdDay = day
	}
	if cfg.RPM > 0 && rpmCount >= cfg.RPM {
		return false, "limite de RPM atingido"
	}
	if cfg.RPD > 0 && rpdCount >= cfg.RPD {
		return false, "limite de RPD atingido"
	}
	rpmCount++
	rpdCount++
	if _, err = tx.Exec(`UPDATE ai_usage SET rpm_count=?, rpm_window=?, rpd_count=?, rpd_day=? WHERE guild_id=?`, rpmCount, rpmWindow, rpdCount, rpdDay, guildID); err != nil {
		return false, "não foi possível atualizar o consumo"
	}
	if err = tx.Commit(); err != nil {
		return false, "não foi possível confirmar o consumo"
	}
	return true, ""
}

func AdminIsLocked(userID string) bool {
	if DB == nil {
		return false
	}
	var locked int
	if err := DB.QueryRow(`SELECT locked FROM admin_auth WHERE user_id=?`, userID).Scan(&locked); err != nil {
		return false
	}
	return locked == 1
}

func AdminIsAuthenticated(userID string) bool {
	if DB == nil {
		return false
	}
	var until int64
	if err := DB.QueryRow(`SELECT authenticated_until FROM admin_auth WHERE user_id=? AND locked=0`, userID).Scan(&until); err != nil {
		return false
	}
	return until > time.Now().Unix()
}

func RegisterAdminFailure(userID string) (int, bool) {
	if DB == nil {
		return 3, true
	}
	_, _ = DB.Exec(`INSERT INTO admin_auth (user_id, failed_attempts, locked) VALUES (?,1,0) ON CONFLICT(user_id) DO UPDATE SET failed_attempts=failed_attempts+1`, userID)
	var attempts int
	_ = DB.QueryRow(`SELECT failed_attempts FROM admin_auth WHERE user_id=?`, userID).Scan(&attempts)
	locked := attempts >= 3
	if locked {
		_, _ = DB.Exec(`UPDATE admin_auth SET locked=1 WHERE user_id=?`, userID)
	}
	return attempts, locked
}

func AuthenticateAdmin(userID string, until time.Time) error {
	_, err := DB.Exec(`INSERT INTO admin_auth (user_id, failed_attempts, locked, authenticated_until) VALUES (?,0,0,?) ON CONFLICT(user_id) DO UPDATE SET failed_attempts=0, locked=0, authenticated_until=?`, userID, until.Unix(), until.Unix())
	return err
}
