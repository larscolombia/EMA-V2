package migrations

import (
	"database/sql"
	"fmt"
	"time"
)

type User struct {
	ID           int       `db:"id"`
	FirstName    string    `db:"first_name"`
	LastName     string    `db:"last_name"`
	Email        string    `db:"email"`
	Password     string    `db:"password"`
	Role         string    `db:"role"`
	ProfileImage string    `db:"profile_image"`
	City         string    `db:"city"`
	Profession   string    `db:"profession"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
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

	// Subscriptions related tables
	createPlans := `
	CREATE TABLE IF NOT EXISTS subscription_plans (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		currency VARCHAR(10) NOT NULL DEFAULT 'USD',
		price DECIMAL(10,2) NOT NULL DEFAULT 0.00,
		billing VARCHAR(50) NOT NULL DEFAULT 'Mensual',
		consultations INT NOT NULL DEFAULT 0,
		questionnaires INT NOT NULL DEFAULT 0,
		clinical_cases INT NOT NULL DEFAULT 0,
		files INT NOT NULL DEFAULT 0
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
	if _, err := db.Exec(createPlans); err != nil {
		return err
	}
	createSubs := `
	CREATE TABLE IF NOT EXISTS subscriptions (
		id INT AUTO_INCREMENT PRIMARY KEY,
		user_id INT NOT NULL,
		plan_id INT NOT NULL,
		start_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		end_date DATETIME NULL,
		frequency INT NOT NULL DEFAULT 0,
		consultations INT NOT NULL DEFAULT 0,
		questionnaires INT NOT NULL DEFAULT 0,
		clinical_cases INT NOT NULL DEFAULT 0,
		files INT NOT NULL DEFAULT 0,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (plan_id) REFERENCES subscription_plans(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
	if _, err := db.Exec(createSubs); err != nil {
		return err
	}
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

// SeedDefaultPlans inserts some default plans if none exist
func SeedDefaultPlans() error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(1) FROM subscription_plans").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		// Free plan
		if _, err := db.Exec(`INSERT INTO subscription_plans (name, currency, price, billing, consultations, questionnaires, clinical_cases, files) VALUES ('Free','USD',0.00,'Mensual',3,10,5,2)`); err != nil {
			return err
		}
		// Pro plan
		if _, err := db.Exec(`INSERT INTO subscription_plans (name, currency, price, billing, consultations, questionnaires, clinical_cases, files) VALUES ('Pro','USD',9.99,'Mensual',30,50,25,100)`); err != nil {
			return err
		}
		// Premium plan
		if _, err := db.Exec(`INSERT INTO subscription_plans (name, currency, price, billing, consultations, questionnaires, clinical_cases, files) VALUES ('Premium','USD',19.99,'Mensual',100,200,100,500)`); err != nil {
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
	if firstName == "" {
		firstName = cur.FirstName
	}
	if lastName == "" {
		lastName = cur.LastName
	}
	if city == "" {
		city = cur.City
	}
	if profession == "" {
		profession = cur.Profession
	}
	_, err := db.Exec("UPDATE users SET first_name = ?, last_name = ?, city = ?, profession = ?, updated_at = NOW() WHERE id = ?", firstName, lastName, city, profession, id)
	return err
}

// UpdateUserPassword updates the password for the given user id
func UpdateUserPassword(id int, password string) error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	_, err := db.Exec("UPDATE users SET password = ?, updated_at = NOW() WHERE id = ?", password, id)
	return err
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

// EnsureFreeSubscriptionForUser creates a Free subscription for the user if none exists
func EnsureFreeSubscriptionForUser(userID int) error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(1) FROM subscriptions WHERE user_id = ?", userID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	// Find Free plan
	var planID, consultations, questionnaires, clinicalCases, files int
	row := db.QueryRow("SELECT id, consultations, questionnaires, clinical_cases, files FROM subscription_plans WHERE name = 'Free' LIMIT 1")
	switch err := row.Scan(&planID, &consultations, &questionnaires, &clinicalCases, &files); err {
	case nil:
		// ok
	case sql.ErrNoRows:
		// Fallback to any plan (cheapest/first)
		row2 := db.QueryRow("SELECT id, consultations, questionnaires, clinical_cases, files FROM subscription_plans ORDER BY price ASC, id ASC LIMIT 1")
		if err2 := row2.Scan(&planID, &consultations, &questionnaires, &clinicalCases, &files); err2 != nil {
			return err2
		}
	default:
		return err
	}
	// Create subscription initialized with plan quotas
	_, err := db.Exec(`INSERT INTO subscriptions (user_id, plan_id, start_date, frequency, consultations, questionnaires, clinical_cases, files)
		VALUES (?,?, NOW(), 0, ?, ?, ?, ?)`, userID, planID, consultations, questionnaires, clinicalCases, files)
	return err
}

// GetActiveSubscriptionForUser returns the most recent subscription of a user joined with plan
func GetActiveSubscriptionForUser(userID int) (map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("db is not initialized")
	}
	query := `SELECT s.id, s.user_id, s.plan_id, s.start_date, s.end_date, s.frequency,
		s.consultations, s.questionnaires, s.clinical_cases, s.files,
		p.id, p.name, p.currency, p.price, p.billing, p.consultations, p.questionnaires, p.clinical_cases, p.files
		FROM subscriptions s JOIN subscription_plans p ON s.plan_id = p.id
		WHERE s.user_id = ? ORDER BY s.id DESC LIMIT 1`
	row := db.QueryRow(query, userID)
	var sID, uID, planID, freq, sConsult, sQuest, sClin, sFiles int
	var start time.Time
	var end sql.NullTime
	var pID, pConsult, pQuest, pClin, pFiles int
	var pName, pCurrency, pBilling string
	var pPrice float64
	if err := row.Scan(&sID, &uID, &planID, &start, &end, &freq, &sConsult, &sQuest, &sClin, &sFiles,
		&pID, &pName, &pCurrency, &pPrice, &pBilling, &pConsult, &pQuest, &pClin, &pFiles); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	sub := map[string]interface{}{
		"id":             sID,
		"user_id":        uID,
		"plan_id":        planID,
		"start_date":     start.Format(time.RFC3339),
		"end_date":       nil,
		"frequency":      freq,
		"consultations":  sConsult,
		"questionnaires": sQuest,
		"clinical_cases": sClin,
		"files":          sFiles,
		"subscription_plan": map[string]interface{}{
			"id":             pID,
			"name":           pName,
			"currency":       pCurrency,
			"price":          pPrice,
			"billing":        pBilling,
			"consultations":  pConsult,
			"questionnaires": pQuest,
			"clinical_cases": pClin,
			"files":          pFiles,
		},
	}
	if end.Valid {
		sub["end_date"] = end.Time.Format(time.RFC3339)
	}
	return sub, nil
}
