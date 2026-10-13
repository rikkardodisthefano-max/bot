package database

import (
	"database/sql"
	"fmt"
	"time"
)

// InitAIGuard cria o contexto e os gatilhos usados pelo ClaimMessage existente.
// Assim o limite é consumido na mesma operação atômica que marca a mensagem.
func InitAIGuard() {
	if DB == nil { return }
	queries := []string{
		`CREATE TABLE IF NOT EXISTS ai_message_context (message_id TEXT PRIMARY KEY, guild_id TEXT NOT NULL, created_at INTEGER NOT NULL);`,
		`CREATE TABLE IF NOT EXISTS ai_blocked_messages (message_id TEXT PRIMARY KEY, reason TEXT NOT NULL, created_at INTEGER NOT NULL);`,
		`CREATE TABLE IF NOT EXISTS ai_exempt_messages (message_id TEXT PRIMARY KEY, created_at INTEGER NOT NULL);`,
		`CREATE TRIGGER IF NOT EXISTS ai_blocked_claim BEFORE INSERT ON processed_messages WHEN EXISTS (SELECT 1 FROM ai_blocked_messages WHERE message_id=NEW.message_id) BEGIN SELECT RAISE(IGNORE); END;`,
		`CREATE TRIGGER IF NOT EXISTS ai_count_claim AFTER INSERT ON processed_messages WHEN NOT EXISTS (SELECT 1 FROM ai_exempt_messages WHERE message_id=NEW.message_id) BEGIN INSERT INTO ai_usage(guild_id,rpm_count,rpm_window,rpd_count,rpd_day) SELECT guild_id,1,strftime('%s','now')-(strftime('%s','now')%60),1,strftime('%Y-%m-%d','now') FROM ai_message_context WHERE message_id=NEW.message_id ON CONFLICT(guild_id) DO UPDATE SET rpm_count=CASE WHEN rpm_window=(strftime('%s','now')-(strftime('%s','now')%60)) THEN rpm_count+1 ELSE 1 END,rpm_window=strftime('%s','now')-(strftime('%s','now')%60),rpd_count=CASE WHEN rpd_day=strftime('%Y-%m-%d','now') THEN rpd_count+1 ELSE 1 END,rpd_day=strftime('%Y-%m-%d','now'); END;`,
	}
	for _, q := range queries { if _, err := DB.Exec(q); err != nil { panic(fmt.Errorf("erro inicializando controle da IA: %w", err)) } }
}

func CanAIRequest(guildID string) (bool, string) {
	cfg := GetBotSettings(guildID)
	if cfg.RPM == 0 && cfg.RPD == 0 { return true, "" }
	var rpmCount, rpdCount int
	var rpmWindow int64
	var rpdDay string
	err := DB.QueryRow(`SELECT rpm_count,rpm_window,rpd_count,rpd_day FROM ai_usage WHERE guild_id=?`, guildID).Scan(&rpmCount,&rpmWindow,&rpdCount,&rpdDay)
	if err != nil && err != sql.ErrNoRows { return false, "não foi possível verificar o limite" }
	now := time.Now().UTC(); minute := now.Truncate(time.Minute).Unix(); day := now.Format("2006-01-02")
	if rpmWindow != minute { rpmCount = 0 }
	if rpdDay != day { rpdCount = 0 }
	if cfg.RPM > 0 && rpmCount >= cfg.RPM { return false, fmt.Sprintf("RPM configurado em %d", cfg.RPM) }
	if cfg.RPD > 0 && rpdCount >= cfg.RPD { return false, fmt.Sprintf("RPD configurado em %d", cfg.RPD) }
	return true, ""
}

func PrepareAIMessage(messageID, guildID string) { if DB != nil { _, _ = DB.Exec(`INSERT OR REPLACE INTO ai_message_context(message_id,guild_id,created_at) VALUES(?,?,?)`,messageID,guildID,time.Now().Unix()) } }
func ExemptAIMessage(messageID string) { if DB != nil { _, _ = DB.Exec(`INSERT OR REPLACE INTO ai_exempt_messages(message_id,created_at) VALUES(?,?)`,messageID,time.Now().Unix()) } }
func BlockAIMessage(messageID, reason string) bool { if DB == nil { return false }; r,e:=DB.Exec(`INSERT OR IGNORE INTO ai_blocked_messages(message_id,reason,created_at) VALUES(?,?,?)`,messageID,reason,time.Now().Unix()); if e!=nil{return false}; n,_:=r.RowsAffected(); return n==1 }
