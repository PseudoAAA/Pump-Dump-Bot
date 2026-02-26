package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

func InitDB(conStr string) (*DB, error) {
	conn, err := sql.Open("postgres", conStr)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return &DB{conn: conn}, nil
}

func (db *DB) AddUser(chatID int64, username string) (isExist bool, err error) {
	trialEnds := time.Now().AddDate(0, 0, 7)

	query1 := `
		INSERT INTO users (chat_id, joined_at, sub_until, username)  
		VALUES ($1, $2, $3, $4) 
		ON CONFLICT (chat_id) DO NOTHING;
	`

	result, err := db.conn.Exec(query1, chatID, time.Now(), trialEnds, username)
	if err != nil {
		return false, fmt.Errorf("ошибка выполнения 1 INSERT: %w", err)
	}

	query2 := `
	INSERT INTO usersconfig (chat_id)
	VALUES ($1)
	ON CONFLICT (chat_id) DO NOTHING;
`
	_, err = db.conn.Exec(query2, chatID)
	if err != nil {
		return false, fmt.Errorf("ошибка выполнения 2 INSERT: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("ошибка при получении RowsAffected: %w", err)
	}

	isExist = rows <= 0

	return isExist, err
}

func (db *DB) GetActiveUsers() ([]int64, error) {
	query := `SELECT chat_id FROM users WHERE sub_until > $1`
	rows, err := db.conn.Query(query, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []int64
	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err == nil {
			users = append(users, chatID)
		}
	}
	return users, nil
}

func (db *DB) GetTrialDaysLeft(chatID int64) (int, error) {
	var trialEnds time.Time
	query := `SELECT sub_until FROM users WHERE chat_id = $1`
	err := db.conn.QueryRow(query, chatID).Scan(&trialEnds)
	if err != nil {
		return 0, err
	}

	days := int(time.Until(trialEnds).Hours() / 24)
	if days < 0 {
		return 0, nil
	}
	return days, nil
}

func (db *DB) IsTransactionUsed(txID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM payments WHERE tx_id = $1)`

	err := db.conn.QueryRow(query, txID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки транзакции: %v", err)
	}
	return exists, nil
}

func (db *DB) SaveTransaction(txID string, userID int64, amount float64, days int) error {
	query := `
        INSERT INTO payments (tx_id, user_id, amount, days, created_at) 
        VALUES ($1, $2, $3, $4, NOW())`

	_, err := db.conn.Exec(query, txID, userID, amount, days)
	if err != nil {
		return fmt.Errorf("ошибка сохранения платежа: %v", err)
	}
	return nil
}

func (db *DB) AddSubscriptionDays(chatID int64, daysToAdd int) error {
	var currentEnd time.Time
	err := db.conn.QueryRow(`SELECT sub_until FROM users WHERE chat_id = $1`, chatID).Scan(&currentEnd)
	if err != nil {
		return err
	}

	newEnd := currentEnd
	if time.Now().After(currentEnd) {
		newEnd = time.Now()
	}
	newEnd = newEnd.AddDate(0, 0, daysToAdd)

	_, err = db.conn.Exec(`UPDATE users SET sub_until = $1 WHERE chat_id = $2`, newEnd, chatID)
	return err
}
