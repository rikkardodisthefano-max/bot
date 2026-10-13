package database

import "fmt"

// InitAIGuardStrict adiciona a segunda barreira do limite: mesmo que duas
// mensagens passem pelo pré-check ao mesmo tempo, só a primeira pode entrar
// em processed_messages quando RPM/RPD já estiverem no teto.
func InitAIGuardStrict() {
	if DB == nil { return }
	q := `CREATE TRIGGER IF NOT EXISTS ai_limit_before_claim
	BEFORE INSERT ON processed_messages
	WHEN NOT EXISTS (SELECT 1 FROM ai_exempt_messages WHERE message_id=NEW.message_id)
	 AND EXISTS (SELECT 1 FROM ai_message_context WHERE message_id=NEW.message_id)
	 AND EXISTS (SELECT 1 FROM bot_settings b JOIN ai_message_context c ON c.guild_id=b.guild_id WHERE c.message_id=NEW.message_id AND ((b.rpm>0 AND COALESCE((SELECT CASE WHEN u.rpm_window=(strftime('%s','now')-(strftime('%s','now')%60)) THEN u.rpm_count ELSE 0 END FROM ai_usage u WHERE u.guild_id=b.guild_id),0)>=b.rpm) OR (b.rpd>0 AND COALESCE((SELECT CASE WHEN u.rpd_day=strftime('%Y-%m-%d','now') THEN u.rpd_count ELSE 0 END FROM ai_usage u WHERE u.guild_id=b.guild_id),0)>=b.rpd)))
BEGIN SELECT RAISE(IGNORE); END;`
	if _, err := DB.Exec(q); err != nil { panic(fmt.Errorf("erro criando trava de RPM/RPD: %w", err)) }
}
