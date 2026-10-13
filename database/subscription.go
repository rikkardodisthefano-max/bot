package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type Subscription struct {
	UserID string
	GuildID string
	PlanName string
	Status string
	Token string
	ExpiresAt time.Time
}

func InitSubDB() {
	if DB == nil { return }
	_, err := DB.Exec(`CREATE TABLE IF NOT EXISTS subscriptions (
		user_id TEXT PRIMARY KEY,
		guild_id TEXT,
		plan_name TEXT,
		status TEXT,
		token TEXT,
		expires_at DATETIME
	);`)
	if err != nil { fmt.Printf("Erro ao criar tabela de assinaturas: %v\n", err) }
}

func generateToken() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil { return "RL-INVALID" }
	return "RL-" + hex.EncodeToString(b)
}

func AddOrUpdateVIP(userID, guildID, planName string, days int) (string, error) {
	if DB == nil { return "", fmt.Errorf("banco de dados não inicializado") }
	if days <= 0 { return "", fmt.Errorf("a duração deve ser maior que zero") }
	expiresAt := time.Now().AddDate(0, 0, days)
	token := generateToken()
	_, err := DB.Exec(`INSERT INTO subscriptions (user_id,guild_id,plan_name,status,token,expires_at)
		VALUES (?,?,?,'pago',?,?)
		ON CONFLICT(user_id) DO UPDATE SET guild_id=excluded.guild_id,plan_name=excluded.plan_name,status='pago',token=excluded.token,expires_at=excluded.expires_at;`, userID,guildID,planName,token,expiresAt)
	return token, err
}

func ResetarFatura(userID string, additionalDays int) error {
	if DB == nil { return fmt.Errorf("banco de dados não inicializado") }
	if additionalDays <= 0 { return fmt.Errorf("a quantidade de dias deve ser maior que zero") }
	var current time.Time
	if err := DB.QueryRow("SELECT expires_at FROM subscriptions WHERE user_id = ?", userID).Scan(&current); err != nil { return err }
	base := time.Now()
	if current.After(base) { base = current }
	_, err := DB.Exec("UPDATE subscriptions SET status='pago', expires_at=? WHERE user_id=?", base.AddDate(0,0,additionalDays), userID)
	return err
}

func GetSubscriptionsByStatus(status string) ([]Subscription, error) {
	if DB == nil { return nil, fmt.Errorf("banco de dados não inicializado") }
	rows, err := DB.Query("SELECT user_id,guild_id,plan_name,status,token,expires_at FROM subscriptions WHERE status=? ORDER BY expires_at ASC", status)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.UserID,&s.GuildID,&s.PlanName,&s.Status,&s.Token,&s.ExpiresAt); err != nil { return nil, err }
		result = append(result,s)
	}
	return result, rows.Err()
}

func GetUserSubscription(userID string) (*Subscription, error) {
	if DB == nil { return nil, fmt.Errorf("banco de dados não inicializado") }
	var s Subscription
	err := DB.QueryRow("SELECT user_id,guild_id,plan_name,status,token,expires_at FROM subscriptions WHERE user_id=?", userID).Scan(&s.UserID,&s.GuildID,&s.PlanName,&s.Status,&s.Token,&s.ExpiresAt)
	if err != nil { return nil, err }
	return &s, nil
}
