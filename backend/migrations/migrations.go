package migrations

import (
	"database/sql"
	"fmt"
	"time"
)

type User struct {
	ID        int    `db:"id"`
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string `db:"email"`
	Password  string `db:"password"`
	Role      string `db:"role"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

var db *sql.DB

// Init sets the DB connection for migrations and queries
func Init(database *sql.DB) {
	db = database
}

// Migrate creates required tables if they do not exist
func Migrate() error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	// create database-level settings are assumed handled externally; ensure users table
	createUsers := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		first_name VARCHAR(100) NOT NULL,
		last_name VARCHAR(100) NOT NULL,
		email VARCHAR(191) NOT NULL UNIQUE,
		password VARCHAR(191) NOT NULL,
		role VARCHAR(50) NOT NULL DEFAULT 'user',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
	if _, err := db.Exec(createUsers); err != nil {
		return err
	}
	return nil
}

// SeedDefaultUser inserts a default user if it doesn't exist
func SeedDefaultUser() error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	// check if email exists
	var count int
	if err := db.QueryRow("SELECT COUNT(1) FROM users WHERE email = ?", "leonardoherrerac10@gmail.com").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		_, err := db.Exec(
			"INSERT INTO users (first_name, last_name, email, password, role) VALUES (?, ?, ?, ?, ?)",
			"Leonardo", "Herrera", "leonardoherrerac10@gmail.com", "supersecret", "super_admin",
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetUserByEmail retrieves a user from DB by email
func GetUserByEmail(email string) *User {
	if db == nil {
		return nil
	}
	row := db.QueryRow("SELECT id, first_name, last_name, email, password, role, created_at, updated_at FROM users WHERE email = ? LIMIT 1", email)
	var u User
	if err := row.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Password, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil
	}
	return &u
}

// EmailExists checks if a user with the given email exists
func EmailExists(email string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("db is not initialized")
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(1) FROM users WHERE email = ?", email).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateUser inserts a new user record
func CreateUser(firstName, lastName, email, password, role string) error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	_, err := db.Exec(
		"INSERT INTO users (first_name, last_name, email, password, role) VALUES (?, ?, ?, ?, ?)",
		firstName, lastName, email, password, role,
	)
	return err
}
