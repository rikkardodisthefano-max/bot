package database

import (
	"log"
	"time"
)

func InitInteractionStore() {
	if DB == nil { return }
	_, err := DB.Exec(`CREATE TABLE IF NOT EXISTS processed_interactions (
		interaction_id TEXT PRIMARY KEY,
		processed_at INTEGER NOT NULL
	);`)
	if err != nil { log.Println("Erro ao criar tabela de interacoes processadas:", err); return }
	_, _ = DB.Exec("DELETE FROM processed_interactions WHERE processed_at < ?", time.Now().Add(-24*time.Hour).Unix())
}

func ClaimInteraction(id string) bool {
	if DB == nil || id == "" { return false }
	result, err := DB.Exec("INSERT OR IGNORE INTO processed_interactions (interaction_id, processed_at) VALUES (?, ?)", id, time.Now().Unix())
	if err != nil { log.Println("Erro ao registrar interacao:", err); return false }
	n, err := result.RowsAffected()
	return err == nil && n == 1
}
