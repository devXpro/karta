package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"karta/internal/models"
)

// Database represents the SQLite database connection and operations
type Database struct {
	db *sql.DB
}

// User represents a Telegram user in the database
type User struct {
	ID       int64     `json:"id"`
	ChatID   int64     `json:"chat_id"`
	Username string    `json:"username"`
	JoinedAt time.Time `json:"joined_at"`
	Active   bool      `json:"active"`
}

// QueueHistory represents historical queue data
type QueueHistory struct {
	ID        int64             `json:"id"`
	QueueData *models.QueueData `json:"queue_data"`
	CreatedAt time.Time         `json:"created_at"`
}

// NewDatabase creates a new database connection and initializes tables
func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	database := &Database{db: db}

	if err := database.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	log.Printf("Database initialized successfully at %s", dbPath)
	return database, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// initTables creates the necessary database tables
func (d *Database) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chat_id INTEGER UNIQUE NOT NULL,
			username TEXT,
			joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			active BOOLEAN DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS queue_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			queue_data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_chat_id ON users(chat_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_active ON users(active)`,
		`CREATE INDEX IF NOT EXISTS idx_queue_history_created_at ON queue_history(created_at)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}

	return nil
}

// AddUser adds a new user to the database or updates existing user
func (d *Database) AddUser(chatID int64, username string) error {
	query := `INSERT OR REPLACE INTO users (chat_id, username, joined_at, active) 
			  VALUES (?, ?, COALESCE((SELECT joined_at FROM users WHERE chat_id = ?), CURRENT_TIMESTAMP), 1)`

	_, err := d.db.Exec(query, chatID, username, chatID)
	if err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}

	log.Printf("User added/updated: chat_id=%d, username=%s", chatID, username)
	return nil
}

// GetActiveUsers returns all active users
func (d *Database) GetActiveUsers() ([]User, error) {
	query := `SELECT id, chat_id, username, joined_at, active FROM users WHERE active = 1`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var username sql.NullString

		err := rows.Scan(&user.ID, &user.ChatID, &username, &user.JoinedAt, &user.Active)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if username.Valid {
			user.Username = username.String
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// DeactivateUser marks a user as inactive
func (d *Database) DeactivateUser(chatID int64) error {
	query := `UPDATE users SET active = 0 WHERE chat_id = ?`

	_, err := d.db.Exec(query, chatID)
	if err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	log.Printf("User deactivated: chat_id=%d", chatID)
	return nil
}

// SaveQueueHistory saves queue data to history
func (d *Database) SaveQueueHistory(queueData *models.QueueData) error {
	jsonData, err := json.Marshal(queueData)
	if err != nil {
		return fmt.Errorf("failed to marshal queue data: %w", err)
	}

	query := `INSERT INTO queue_history (queue_data) VALUES (?)`

	_, err = d.db.Exec(query, string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to save queue history: %w", err)
	}

	return nil
}

// GetLatestQueueData returns the most recent queue data from history
func (d *Database) GetLatestQueueData() (*models.QueueData, error) {
	query := `SELECT queue_data FROM queue_history ORDER BY created_at DESC LIMIT 1`

	var jsonData string
	err := d.db.QueryRow(query).Scan(&jsonData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No data found
		}
		return nil, fmt.Errorf("failed to query latest queue data: %w", err)
	}

	var queueData models.QueueData
	if err := json.Unmarshal([]byte(jsonData), &queueData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal queue data: %w", err)
	}

	return &queueData, nil
}

// CleanOldHistory removes queue history older than specified duration
func (d *Database) CleanOldHistory(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	query := `DELETE FROM queue_history WHERE created_at < ?`

	result, err := d.db.Exec(query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to clean old history: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Cleaned %d old history records", rowsAffected)

	return nil
}

// GetUserCount returns the total number of active users
func (d *Database) GetUserCount() (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE active = 1`

	var count int
	err := d.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get user count: %w", err)
	}

	return count, nil
}
