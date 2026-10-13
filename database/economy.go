package database

import (
	"database/sql"
	"fmt"
	"time"
)

// TransferCoins transfere moedas de forma atômica, evitando saldo negativo
// e condições de corrida quando duas transferências acontecem juntas.
func TransferCoins(senderID, receiverID string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("a quantia deve ser maior que zero")
	}
	if senderID == receiverID {
		return fmt.Errorf("não é possível transferir para si mesmo")
	}
	if DB == nil {
		return fmt.Errorf("banco de dados não inicializado")
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var balance int64
	err = tx.QueryRow("SELECT bank FROM economy WHERE user_id = ?", senderID).Scan(&balance)
	if err == sql.ErrNoRows {
		balance = 0
	} else if err != nil {
		return err
	}

	if balance < amount {
		return fmt.Errorf("saldo insuficiente")
	}

	if _, err = tx.Exec(`
		INSERT INTO economy (user_id, bank) VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET bank = economy.bank + excluded.bank`,
		senderID, -amount); err != nil {
		return err
	}

	if _, err = tx.Exec(`
		INSERT INTO economy (user_id, bank) VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET bank = economy.bank + excluded.bank`,
		receiverID, amount); err != nil {
		return err
	}

	return tx.Commit()
}

// ClaimDaily faz o resgate diário em uma única operação SQL.
func ClaimDaily(userID string, reward int64) (bool, time.Duration, error) {
	if DB == nil {
		return false, 0, fmt.Errorf("banco de dados não inicializado")
	}

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)
	nowStr := now.Format(time.RFC3339Nano)
	cutoffStr := cutoff.Format(time.RFC3339Nano)

	result, err := DB.Exec(`
		INSERT INTO economy (user_id, bank, last_daily)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			bank = economy.bank + excluded.bank,
			last_daily = excluded.last_daily
		WHERE economy.last_daily IS NULL
		   OR economy.last_daily = ''
		   OR economy.last_daily <= ?`,
		userID, reward, nowStr, cutoffStr)
	if err != nil {
		return false, 0, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, 0, err
	}
	if rows > 0 {
		return true, 0, nil
	}

	canClaim, remaining := CanClaimDaily(userID)
	return canClaim, remaining, nil
}
