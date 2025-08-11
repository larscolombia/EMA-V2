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
	ProfileImage string `db:"profile_image"`
	City         string `db:"city"`
	Profession   string `db:"profession"`
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
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS profile_image VARCHAR(255) DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS city VARCHAR(100) DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS profession VARCHAR(100) DEFAULT ''")
	return nil
}

// SeedDefaultUser inserts a default user if it doesn't exist
func SeedDefaultUser() error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
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
	row := db.QueryRow("SELECT id, first_name, last_name, email, password, role, IFNULL(profile_image,''), IFNULL(city,''), IFNULL(profession,''), created_at, updated_at FROM users WHERE email = ? LIMIT 1", email)
	var u User
	if err := row.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Password, &u.Role, &u.ProfileImage, &u.City, &u.Profession, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil
	}
	return &u
}

// GetUserByID retrieves a user by its ID
func GetUserByID(id int) *User {
	if db == nil {
		return nil
	}
	row := db.QueryRow("SELECT id, first_name, last_name, email, password, role, IFNULL(profile_image,''), IFNULL(city,''), IFNULL(profession,''), created_at, updated_at FROM users WHERE id = ? LIMIT 1", id)
	var u User
	if err := row.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Password, &u.Role, &u.ProfileImage, &u.City, &u.Profession, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil
	}
	return &u
}

// UpdateUserProfileImage updates the profile_image path
func UpdateUserProfileImage(id int, path string) error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	_, err := db.Exec("UPDATE users SET profile_image = ?, updated_at = NOW() WHERE id = ?", path, id)
	return err
}

// UpdateUserProfile updates first/last name and optional city/profession
func UpdateUserProfile(id int, firstName, lastName, city, profession string) error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	cur := GetUserByID(id)
	if cur == nil {
		return fmt.Errorf("user not found")
	}
	if firstName == "" { firstName = cur.FirstName }
	if lastName == "" { lastName = cur.LastName }
	if city == "" { city = cur.City }
	if profession == "" { profession = cur.Profession }
	_, err := db.Exec("UPDATE users SET first_name = ?, last_name = ?, city = ?, profession = ?, updated_at = NOW() WHERE id = ?", firstName, lastName, city, profession, id)
	return err
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


