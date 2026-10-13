package database

import "fmt"

func InitBuyerRole() error {
	if DB == nil {
		return fmt.Errorf("banco de dados não inicializado")
	}
	_, err := DB.Exec(`CREATE TABLE IF NOT EXISTS buyer_roles (guild_id TEXT PRIMARY KEY, role_id TEXT NOT NULL DEFAULT '')`)
	return err
}

func SetBuyerRole(guildID, roleID string) error {
	if DB == nil {
		return fmt.Errorf("banco de dados não inicializado")
	}
	_, err := DB.Exec(`INSERT INTO buyer_roles (guild_id, role_id) VALUES (?, ?) ON CONFLICT(guild_id) DO UPDATE SET role_id=excluded.role_id`, guildID, roleID)
	return err
}

func GetBuyerRole(guildID string) string {
	if DB == nil {
		return ""
	}
	var roleID string
	if err := DB.QueryRow(`SELECT role_id FROM buyer_roles WHERE guild_id=?`, guildID).Scan(&roleID); err != nil {
		return ""
	}
	return roleID
}
