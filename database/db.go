package database

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

var DB *sql.DB

type ServerConfigData struct {
	GuildID          string
	UnverifiedRoleID string
	VerifiedRoleID   string
	SystemPrompt     string
	WelcomeChannel   string
	WelcomeMessage   string
	AutoRole         string
}

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite", "./database.db")
	if err != nil {
		log.Fatal("Erro ao abrir o banco de dados:", err)
	}

	warningsTableQuery := `CREATE TABLE IF NOT EXISTS user_warnings (
		guild_id TEXT,
		user_id TEXT,
		warnings INTEGER DEFAULT 0,
		PRIMARY KEY (guild_id, user_id)
	);`
	if _, err = DB.Exec(warningsTableQuery); err != nil {
		log.Fatal("Erro ao criar tabela de avisos:", err)
	}

	economyQuery := `CREATE TABLE IF NOT EXISTS economy (
		user_id TEXT PRIMARY KEY,
		bank INTEGER DEFAULT 0,
		last_daily TEXT
	);`
	if _, err = DB.Exec(economyQuery); err != nil {
		log.Fatal("Erro ao criar tabela de economia:", err)
	}

	configQuery := `CREATE TABLE IF NOT EXISTS server_configs (
		guild_id TEXT PRIMARY KEY,
		unverified_role_id TEXT DEFAULT '',
		verified_role_id TEXT DEFAULT '',
		system_prompt TEXT DEFAULT '',
		welcome_channel TEXT DEFAULT '',
		welcome_message TEXT DEFAULT '',
		auto_role TEXT DEFAULT ''
	);`
	if _, err = DB.Exec(configQuery); err != nil {
		log.Fatal("Erro ao criar tabela de configuracoes:", err)
	}

	// Idempotência da IA: uma mensagem do Discord só pode ser reivindicada
	// uma vez antes de uma chamada à Gemini. Isso protege inclusive contra
	// duas instâncias do processo compartilhando o mesmo SQLite.
	processedMessagesQuery := `CREATE TABLE IF NOT EXISTS processed_messages (
		message_id TEXT PRIMARY KEY,
		processed_at INTEGER NOT NULL
	);`
	if _, err = DB.Exec(processedMessagesQuery); err != nil {
		log.Fatal("Erro ao criar tabela de mensagens processadas:", err)
	}

	// Evita crescimento infinito da tabela. Uma mensagem antiga não precisa
	// continuar bloqueada para sempre.
	cutoff := time.Now().Add(-7 * 24 * time.Hour).Unix()
	if _, err = DB.Exec("DELETE FROM processed_messages WHERE processed_at < ?", cutoff); err != nil {
		log.Println("Aviso ao limpar mensagens processadas antigas:", err)
	}
}

// ClaimMessage marca uma mensagem como processada de forma atômica.
// Retorna true somente para a primeira tentativa. Em caso de erro no banco,
// retorna false de propósito: é melhor não chamar a IA do que gastar RPD em duplicidade.
func ClaimMessage(messageID string) bool {
	if DB == nil || messageID == "" {
		return false
	}

	result, err := DB.Exec(
		"INSERT OR IGNORE INTO processed_messages (message_id, processed_at) VALUES (?, ?)",
		messageID,
		time.Now().Unix(),
	)
	if err != nil {
		log.Println("Erro ao registrar mensagem processada:", err)
		return false
	}

	affected, err := result.RowsAffected()
	if err != nil {
		log.Println("Erro ao verificar mensagem processada:", err)
		return false
	}
	return affected == 1
}

// ==========================================
// FUNÇÕES DE CONFIGURAÇÃO DO SERVIDOR
// ==========================================

func SaveWelcomeChannel(guildID, channelID string) error {
	cfg := GetServerConfig(guildID)
	cfg.WelcomeChannel = channelID
	SaveServerConfig(cfg)
	return nil
}

func GetWelcomeChannel(guildID string) (string, error) {
	cfg := GetServerConfig(guildID)
	return cfg.WelcomeChannel, nil
}

func GetServerConfig(guildID string) *ServerConfigData {
	cfg := &ServerConfigData{GuildID: guildID}
	if DB == nil {
		return cfg
	}

	query := `SELECT unverified_role_id, verified_role_id, system_prompt, welcome_channel, welcome_message, auto_role
		FROM server_configs WHERE guild_id = ?`
	err := DB.QueryRow(query, guildID).Scan(
		&cfg.UnverifiedRoleID, &cfg.VerifiedRoleID, &cfg.SystemPrompt,
		&cfg.WelcomeChannel, &cfg.WelcomeMessage, &cfg.AutoRole,
	)
	if err != nil && err != sql.ErrNoRows {
		log.Println("Erro ao buscar configuracao do servidor:", err)
	}
	return cfg
}

func SaveServerConfig(cfg *ServerConfigData) {
	if DB == nil {
		return
	}

	query := `INSERT INTO server_configs
		(guild_id, unverified_role_id, verified_role_id, system_prompt, welcome_channel, welcome_message, auto_role)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(guild_id) DO UPDATE SET
		unverified_role_id = excluded.unverified_role_id,
		verified_role_id = excluded.verified_role_id,
		system_prompt = excluded.system_prompt,
		welcome_channel = excluded.welcome_channel,
		welcome_message = excluded.welcome_message,
		auto_role = excluded.auto_role;`

	_, err := DB.Exec(query, cfg.GuildID, cfg.UnverifiedRoleID, cfg.VerifiedRoleID, cfg.SystemPrompt, cfg.WelcomeChannel, cfg.WelcomeMessage, cfg.AutoRole)
	if err != nil {
		log.Println("Erro ao salvar configuracao do servidor:", err)
	}
}

func LoadAllServerConfigs() map[string]*ServerConfigData {
	configs := make(map[string]*ServerConfigData)
	if DB == nil {
		log.Println("⚠️ Banco de dados ainda não inicializado.")
		return configs
	}

	rows, err := DB.Query("SELECT guild_id, unverified_role_id, verified_role_id, system_prompt, welcome_channel, welcome_message, auto_role FROM server_configs")
	if err != nil {
		log.Println("Erro ao carregar configuracoes do SQLite:", err)
		return configs
	}
	defer rows.Close()

	for rows.Next() {
		cfg := &ServerConfigData{}
		if err := rows.Scan(&cfg.GuildID, &cfg.UnverifiedRoleID, &cfg.VerifiedRoleID, &cfg.SystemPrompt, &cfg.WelcomeChannel, &cfg.WelcomeMessage, &cfg.AutoRole); err == nil {
			configs[cfg.GuildID] = cfg
		}
	}
	if err = rows.Err(); err != nil {
		log.Println("Erro durante a iteracao das linhas:", err)
	}
	return configs
}

// ==========================================
// FUNÇÕES DE ECONOMIA & AVISOS
// ==========================================

func GetBalance(userID string) int64 {
	if DB == nil {
		return 0
	}
	var bank int64
	err := DB.QueryRow("SELECT bank FROM economy WHERE user_id = ?", userID).Scan(&bank)
	if err == sql.ErrNoRows {
		return 0
	}
	return bank
}

func AddCoins(userID string, amount int64) int64 {
	if DB == nil {
		return 0
	}
	currentCoins := GetBalance(userID)
	newBalance := currentCoins + amount
	_, err := DB.Exec(`INSERT INTO economy (user_id, bank) VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET bank = bank + ?;`, userID, amount, amount)
	if err != nil {
		log.Println("Erro ao atualizar moedas:", err)
	}
	return newBalance
}

func CanClaimDaily(userID string) (bool, time.Duration) {
	if DB == nil {
		return true, 0
	}
	var lastDailyStr string
	err := DB.QueryRow("SELECT last_daily FROM economy WHERE user_id = ?", userID).Scan(&lastDailyStr)
	if err == sql.ErrNoRows || lastDailyStr == "" {
		return true, 0
	}
	lastDaily, err := time.Parse(time.RFC3339, lastDailyStr)
	if err != nil {
		return true, 0
	}
	cooldown := 24 * time.Hour
	elapsed := time.Since(lastDaily)
	if elapsed >= cooldown {
		return true, 0
	}
	return false, cooldown - elapsed
}

func SetDailyClaimed(userID string) {
	if DB == nil {
		return
	}
	now := time.Now().Format(time.RFC3339)
	_, err := DB.Exec(`INSERT INTO economy (user_id, bank, last_daily) VALUES (?, 0, ?)
		ON CONFLICT(user_id) DO UPDATE SET last_daily = ?;`, userID, now, now)
	if err != nil {
		log.Println("Erro ao atualizar o daily:", err)
	}
}

func GetUserWarnings(guildID, userID string) int {
	if DB == nil {
		return 0
	}
	var count int
	err := DB.QueryRow("SELECT warnings FROM user_warnings WHERE guild_id = ? AND user_id = ?", guildID, userID).Scan(&count)
	if err == sql.ErrNoRows {
		return 0
	}
	return count
}

func IncrementUserWarnings(guildID, userID string) int {
	if DB == nil {
		return 0
	}
	_, err := DB.Exec(`INSERT INTO user_warnings (guild_id, user_id, warnings) VALUES (?, ?, 1)
		ON CONFLICT(guild_id, user_id) DO UPDATE SET warnings = warnings + 1;`, guildID, userID)
	if err != nil {
		log.Println("Erro ao atualizar avisos do usuario:", err)
		return GetUserWarnings(guildID, userID)
	}
	return GetUserWarnings(guildID, userID)
}
