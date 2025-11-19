package migrations

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
)

type User struct {
	ID               int       `db:"id"`
	FirstName        string    `db:"first_name"`
	LastName         string    `db:"last_name"`
	Email            string    `db:"email"`
	Password         string    `db:"password"`
	Role             string    `db:"role"`
	ProfileImage     string    `db:"profile_image"`
	City             string    `db:"city"`
	Profession       string    `db:"profession"`
	Gender           string    `db:"gender"`
	Age              *int      `db:"age"`
	CountryID        *int      `db:"country_id"`
	StripeCustomerID string    `db:"stripe_customer_id"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

var db *sql.DB

// Init sets the DB connection for migrations and queries
func Init(database *sql.DB) {
	db = database
}

// Migrate creates required tables if they do not exist
func Migrate() error {
	log.Printf("[MIGRATION] 🔄 Starting database migrations...")
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}

	log.Printf("[MIGRATION] Creating users table if not exists...")
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
		log.Printf("[MIGRATION] ❌ ERROR creating users table: %v", err)
		return err
	}
	log.Printf("[MIGRATION] ✅ users table ready")

	// Ensure optional columns exist (profile_image, city, profession, gender, age, country_id)
	if err := ensureColumnExists("users", "profile_image", "profile_image VARCHAR(255) DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumnExists("users", "city", "city VARCHAR(100) DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumnExists("users", "profession", "profession VARCHAR(100) DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumnExists("users", "gender", "gender VARCHAR(50) DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumnExists("users", "age", "age INT DEFAULT NULL"); err != nil {
		return err
	}
	if err := ensureColumnExists("users", "country_id", "country_id INT DEFAULT NULL"); err != nil {
		return err
	}

	// Subscriptions related tables
	log.Printf("[MIGRATION] Creating subscription_plans table if not exists...")
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
		files INT NOT NULL DEFAULT 0,
		stripe_product_id VARCHAR(100) NULL,
		stripe_price_id VARCHAR(100) NULL
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
	if _, err := db.Exec(createPlans); err != nil {
		log.Printf("[MIGRATION] ❌ ERROR creating subscription_plans table: %v", err)
		return err
	}
	log.Printf("[MIGRATION] ✅ subscription_plans table ready")

	log.Printf("[MIGRATION] Creating subscriptions table if not exists...")
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
		log.Printf("[MIGRATION] ❌ ERROR creating subscriptions table: %v", err)
		return err
	}
	log.Printf("[MIGRATION] ✅ subscriptions table ready")

	// Ensure new Stripe-related columns exist (idempotent) for backward compatibility
	if err := ensureColumnExists("subscription_plans", "stripe_product_id", "stripe_product_id VARCHAR(100) NULL"); err != nil {
		return err
	}
	if err := ensureColumnExists("subscription_plans", "stripe_price_id", "stripe_price_id VARCHAR(100) NULL"); err != nil {
		return err
	}
	if err := ensureColumnExists("users", "stripe_customer_id", "stripe_customer_id VARCHAR(100) NULL"); err != nil {
		return err
	}

	// Password resets table for recovery tokens
	log.Printf("[MIGRATION] Creating password_resets table if not exists...")
	createPasswordResets := `
	CREATE TABLE IF NOT EXISTS password_resets (
		id INT AUTO_INCREMENT PRIMARY KEY,
		email VARCHAR(191) NOT NULL,
		token VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP NOT NULL,
		INDEX idx_email_token (email, token),
		INDEX idx_expires (expires_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
	if _, err := db.Exec(createPasswordResets); err != nil {
		log.Printf("[MIGRATION] ❌ ERROR creating password_resets table: %v", err)
		return err
	}
	log.Printf("[MIGRATION] ✅ password_resets table ready")

	// Medical categories table for quizzes/tests
	log.Printf("[MIGRATION] Creating medical_categories table if not exists...")
	createMedicalCategories := `
	CREATE TABLE IF NOT EXISTS medical_categories (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(191) NOT NULL UNIQUE,
		description TEXT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
	if _, err := db.Exec(createMedicalCategories); err != nil {
		log.Printf("[MIGRATION] ❌ ERROR creating medical_categories table: %v", err)
		return err
	}
	log.Printf("[MIGRATION] ✅ medical_categories table ready")

	// Test history table for tracking completed tests/quizzes
	log.Printf("[MIGRATION] Creating test_history table if not exists...")
	createTestHistory := `
	CREATE TABLE IF NOT EXISTS test_history (
		id INT AUTO_INCREMENT PRIMARY KEY,
		user_id INT NOT NULL,
		category_id INT NULL,
		test_name VARCHAR(255) NOT NULL,
		score_obtained INT NOT NULL DEFAULT 0,
		max_score INT NOT NULL DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (category_id) REFERENCES medical_categories(id) ON DELETE SET NULL,
		INDEX idx_user_created (user_id, created_at),
		INDEX idx_user_category (user_id, category_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
	if _, err := db.Exec(createTestHistory); err != nil {
		log.Printf("[MIGRATION] ❌ ERROR creating test_history table: %v", err)
		return err
	}
	log.Printf("[MIGRATION] ✅ test_history table ready")

	// Vector Stores tables for multi-vector-store management
	log.Printf("[MIGRATION] Creating vector_stores table if not exists...")
	createVectorStores := `
	CREATE TABLE IF NOT EXISTS vector_stores (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL COMMENT 'Nombre descriptivo del vector store',
		vector_store_id VARCHAR(255) NOT NULL UNIQUE COMMENT 'ID del vector store en OpenAI (vs_xxx)',
		description TEXT COMMENT 'Descripción del propósito del vector store',
		category VARCHAR(100) COMMENT 'Categoría (medicina_general, cardiologia, neurologia, etc)',
		is_default BOOLEAN DEFAULT FALSE COMMENT 'Si es el vector store por defecto',
		file_count INT DEFAULT 0 COMMENT 'Número de archivos en el vector store',
		total_bytes BIGINT DEFAULT 0 COMMENT 'Tamaño total en bytes',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		INDEX idx_category (category),
		INDEX idx_default (is_default)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`
	if _, err := db.Exec(createVectorStores); err != nil {
		log.Printf("[MIGRATION] ❌ ERROR creating vector_stores table: %v", err)
		return err
	}
	log.Printf("[MIGRATION] ✅ vector_stores table ready")

	log.Printf("[MIGRATION] Creating vector_store_files table if not exists...")
	createVectorStoreFiles := `
	CREATE TABLE IF NOT EXISTS vector_store_files (
		id INT AUTO_INCREMENT PRIMARY KEY,
		vector_store_id VARCHAR(255) NOT NULL COMMENT 'ID del vector store en OpenAI',
		file_id VARCHAR(255) NOT NULL COMMENT 'ID del archivo en OpenAI (file-xxx)',
		filename VARCHAR(500) NOT NULL COMMENT 'Nombre original del archivo',
		file_size BIGINT COMMENT 'Tamaño del archivo en bytes',
		status VARCHAR(50) DEFAULT 'processing' COMMENT 'Estado: processing, completed, failed',
		uploaded_by INT COMMENT 'ID del admin que subió el archivo',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		UNIQUE KEY unique_file_vector (vector_store_id, file_id),
		INDEX idx_vector_store (vector_store_id),
		INDEX idx_status (status)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`
	if _, err := db.Exec(createVectorStoreFiles); err != nil {
		log.Printf("[MIGRATION] ❌ ERROR creating vector_store_files table: %v", err)
		return err
	}
	log.Printf("[MIGRATION] ✅ vector_store_files table ready")

	// Seed default vector store if it doesn't exist
	if err := seedDefaultVectorStore(); err != nil {
		log.Printf("[MIGRATION] ⚠️ WARNING seeding default vector store: %v", err)
	}

	// Asegurar que todos los usuarios existentes tengan un rol
	if err := ensureUsersHaveRole(); err != nil {
		log.Printf("[MIGRATION] ⚠️ WARNING ensuring users have roles: %v", err)
	}

	// Crear usuario admin por defecto si no existe
	if err := createDefaultAdminUser(); err != nil {
		log.Printf("[MIGRATION] ⚠️ WARNING creating default admin user: %v", err)
		// No retornar error para no bloquear la migración completa
	}

	log.Printf("[MIGRATION] ✅ All migrations completed successfully")
	return nil
}

// ensureUsersHaveRole actualiza todos los usuarios sin rol para que tengan 'user'
func ensureUsersHaveRole() error {
	log.Printf("[MIGRATION] Ensuring all users have a role...")

	updateQuery := "UPDATE users SET role = 'user' WHERE role IS NULL OR role = ''"
	result, err := db.Exec(updateQuery)
	if err != nil {
		return fmt.Errorf("error updating users without role: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("[MIGRATION] ✅ Updated %d users to have 'user' role", rowsAffected)
	} else {
		log.Printf("[MIGRATION] ✅ All users already have roles assigned")
	}

	return nil
}

// createDefaultAdminUser crea el usuario administrador por defecto si no existe
func createDefaultAdminUser() error {
	adminEmail := "drleonardoherrerac@gmail.com"

	// Verificar si el usuario ya existe
	var count int
	checkQuery := "SELECT COUNT(*) FROM users WHERE email = ?"
	if err := db.QueryRow(checkQuery, adminEmail).Scan(&count); err != nil {
		return fmt.Errorf("error checking admin user: %v", err)
	}

	if count > 0 {
		log.Printf("[MIGRATION] ✅ Admin user already exists: %s", adminEmail)
		return nil
	}

	// Crear el usuario admin
	// Nota: La contraseña debería ser hasheada en producción
	// Por ahora usamos una contraseña temporal que el admin debe cambiar
	insertQuery := `
		INSERT INTO users (first_name, last_name, email, password, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())
	`

	_, err := db.Exec(
		insertQuery,
		"Leonardo",
		"Herrera",
		adminEmail,
		"admin123",    // Contraseña temporal - DEBE SER CAMBIADA
		"super_admin", // Rol para panel admin
	)

	if err != nil {
		return fmt.Errorf("error creating admin user: %v", err)
	}

	log.Printf("[MIGRATION] ✅ Default admin user created: %s (password: admin123 - PLEASE CHANGE)", adminEmail)
	return nil
}

// ensureColumnExists checks information_schema and adds the column if missing
func ensureColumnExists(table, column, columnDef string) error {
	var cnt int
	q := `SELECT COUNT(1) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`
	if err := db.QueryRow(q, table, column).Scan(&cnt); err != nil {
		return err
	}
	if cnt == 0 {
		log.Printf("[MIGRATION] Adding column %s.%s", table, column)
		_, err := db.Exec("ALTER TABLE " + table + " ADD COLUMN " + columnDef)
		if err != nil {
			log.Printf("[MIGRATION] ERROR adding column %s.%s: %v", table, column, err)
			return err
		}
		log.Printf("[MIGRATION] Successfully added column %s.%s", table, column)
		return nil
	}
	log.Printf("[MIGRATION] Column %s.%s already exists", table, column)
	return nil
}

// seedDefaultVectorStore inserts the default global vector store if it doesn't exist
func seedDefaultVectorStore() error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}

	log.Printf("[MIGRATION] Seeding default vector stores...")

	// Vector stores a crear
	vectorStores := []struct {
		Name          string
		VectorStoreID string
		Description   string
		Category      string
		IsDefault     bool
	}{
		{
			Name:          "Biblioteca Médica General",
			VectorStoreID: "vs_680fc484cef081918b2b9588b701e2f4",
			Description:   "Vector store principal con libros de medicina general y especialidades",
			Category:      "Medicina General",
			IsDefault:     true,
		},
		{
			Name:          "Banco de Preguntas",
			VectorStoreID: "vs_691deb92da488191aaeefba2b80406d7",
			Description:   "Base de datos de preguntas y cuestionarios médicos",
			Category:      "Cuestionarios",
			IsDefault:     false,
		},
	}

	insertQuery := `
		INSERT INTO vector_stores (name, vector_store_id, description, category, is_default, file_count)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE name = VALUES(name), description = VALUES(description), category = VALUES(category)
	`

	for _, vs := range vectorStores {
		_, err := db.Exec(
			insertQuery,
			vs.Name,
			vs.VectorStoreID,
			vs.Description,
			vs.Category,
			vs.IsDefault,
			0,
		)

		if err != nil {
			log.Printf("[MIGRATION] ⚠️ Warning creating vector store %s: %v", vs.VectorStoreID, err)
		} else {
			log.Printf("[MIGRATION] ✅ Vector store ready: %s (%s)", vs.Name, vs.VectorStoreID)
		}
	}

	return nil
}

// SeedDefaultUser inserts a default user if it doesn't exist, using env vars
func SeedDefaultUser() error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	email := os.Getenv("DEFAULT_USER_EMAIL")
	password := os.Getenv("DEFAULT_USER_PASSWORD")
	if email == "" || password == "" {
		return nil
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(1) FROM users WHERE email = ?", email).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		res, err := db.Exec(
			"INSERT INTO users (first_name, last_name, email, password, role) VALUES (?, ?, ?, ?, ?)",
			"Leonardo", "Herrera", email, password, "super_admin",
		)
		if err != nil {
			return err
		}
		uid, _ := res.LastInsertId()
		// Asigna subscripción Free automáticamente si existe plan
		_ = EnsureFreeSubscriptionForUser(int(uid))
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

// SeedMedicalCategories inserts a default set of medical specialties used by quizzes/tests
func SeedMedicalCategories() error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(1) FROM medical_categories").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	categories := []string{
		"Alergología",
		"Inmunología clínica",
		"Cardiología",
		"Endocrinología",
		"Gastroenterología",
		"Geriatría",
		"Hematología",
		"Infectología",
		"Medicina interna",
		"Medicina física y rehabilitación",
		"Nefrología",
		"Neumología",
		"Neurología",
		"Oncología médica",
		"Pediatría",
		"Psiquiatría",
		"Reumatología",
		"Cirugía general",
		"Cirugía cardiovascular",
		"Cirugía torácica",
		"Cirugía plástica estética y reconstructiva",
		"Cirugía maxilofacial",
		"Cirugía pediátrica",
		"Neurocirugía",
		"Traumatología y ortopedia",
		"Cirugía de mano",
		"Cirugía colorrectal",
		"Cirugía vascular periférica",
		"Dermatología",
		"Ginecología y obstetricia",
		"Oftalmología",
		"Otorrinolaringología",
		"Urología",
		"Anatomía patológica",
		"Genética médica",
		"Medicina nuclear",
		"Radiología",
		"Medicina de laboratorio",
		"Microbiología clínica",
		"Anestesiología",
		"Medicina del deporte",
		"Medicina de urgencias y emergencias",
		"Medicina del trabajo",
		"Medicina paliativa",
		"Medicina preventiva",
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT INTO medical_categories (name) VALUES (?)")
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, c := range categories {
		if _, err := stmt.Exec(c); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// GetUserByEmail retrieves a user from DB by email
func GetUserByEmail(email string) *User {
	if db == nil {
		return nil
	}
	row := db.QueryRow("SELECT id, first_name, last_name, email, password, role, IFNULL(profile_image,''), IFNULL(city,''), IFNULL(profession,''), IFNULL(gender,''), age, country_id, created_at, updated_at FROM users WHERE email = ? LIMIT 1", email)
	var u User
	if err := row.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Password, &u.Role, &u.ProfileImage, &u.City, &u.Profession, &u.Gender, &u.Age, &u.CountryID, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil
	}
	return &u
}

// GetUserByID retrieves a user by its ID
func GetUserByID(id int) *User {
	if db == nil {
		return nil
	}
	row := db.QueryRow("SELECT id, first_name, last_name, email, password, role, IFNULL(profile_image,''), IFNULL(city,''), IFNULL(profession,''), IFNULL(gender,''), age, country_id, created_at, updated_at FROM users WHERE id = ? LIMIT 1", id)
	var u User
	if err := row.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Password, &u.Role, &u.ProfileImage, &u.City, &u.Profession, &u.Gender, &u.Age, &u.CountryID, &u.CreatedAt, &u.UpdatedAt); err != nil {
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

// UpdateUserProfile updates first/last name and optional city/profession/gender/age/country
func UpdateUserProfile(id int, firstName, lastName, city, profession, gender string, age, countryID *int) error {
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
	if gender == "" {
		gender = cur.Gender
	}
	if age == nil {
		age = cur.Age
	}
	if countryID == nil {
		countryID = cur.CountryID
	}
	_, err := db.Exec("UPDATE users SET first_name = ?, last_name = ?, city = ?, profession = ?, gender = ?, age = ?, country_id = ?, updated_at = NOW() WHERE id = ?", firstName, lastName, city, profession, gender, age, countryID, id)
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

// CreatePasswordResetToken creates a new password reset token for the given email
func CreatePasswordResetToken(email, token string, expiresAt time.Time) error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	_, err := db.Exec(
		"INSERT INTO password_resets (email, token, expires_at) VALUES (?, ?, ?)",
		email, token, expiresAt,
	)
	return err
}

// ValidatePasswordResetToken checks if the token is valid and not expired
func ValidatePasswordResetToken(email, token string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("db is not initialized")
	}
	var count int
	query := "SELECT COUNT(1) FROM password_resets WHERE email = ? AND token = ? AND expires_at > NOW()"
	if err := db.QueryRow(query, email, token).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeletePasswordResetToken removes used or expired tokens for the given email
func DeletePasswordResetToken(email string) error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	_, err := db.Exec("DELETE FROM password_resets WHERE email = ?", email)
	return err
}

// CleanExpiredPasswordResets removes expired tokens (cleanup job)
func CleanExpiredPasswordResets() error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	_, err := db.Exec("DELETE FROM password_resets WHERE expires_at < NOW()")
	return err
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
		p.id, p.name, p.currency, p.price, p.billing, p.consultations, p.questionnaires, p.clinical_cases, p.files,
		CASE WHEN p.price>0 THEN 1 ELSE 0 END AS statistics
		FROM subscriptions s JOIN subscription_plans p ON s.plan_id = p.id
		WHERE s.user_id = ? ORDER BY s.id DESC LIMIT 1`
	row := db.QueryRow(query, userID)
	var sID, uID, planID, freq, sConsult, sQuest, sClin, sFiles int
	var start time.Time
	var end sql.NullTime
	var pID, pConsult, pQuest, pClin, pFiles int
	var statistics int
	var pName, pCurrency, pBilling string
	var pPrice float64
	if err := row.Scan(&sID, &uID, &planID, &start, &end, &freq, &sConsult, &sQuest, &sClin, &sFiles,
		&pID, &pName, &pCurrency, &pPrice, &pBilling, &pConsult, &pQuest, &pClin, &pFiles, &statistics); err != nil {
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
		"statistics":     statistics,
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
			"statistics":     statistics,
		},
	}
	if end.Valid {
		sub["end_date"] = end.Time.Format(time.RFC3339)
	}
	return sub, nil
}

// ResetActiveSubscriptionQuotas resets the latest subscription quotas to plan defaults.
func ResetActiveSubscriptionQuotas(userID int) error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	row := db.QueryRow("SELECT id, plan_id FROM subscriptions WHERE user_id=? ORDER BY id DESC LIMIT 1", userID)
	var subID, planID int
	if err := row.Scan(&subID, &planID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	prow := db.QueryRow("SELECT consultations, questionnaires, clinical_cases, files FROM subscription_plans WHERE id=? LIMIT 1", planID)
	var c1, c2, c3, c4 int
	if err := prow.Scan(&c1, &c2, &c3, &c4); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE subscriptions SET consultations=?, questionnaires=?, clinical_cases=?, files=? WHERE id=?`, c1, c2, c3, c4, subID); err != nil {
		return err
	}
	log.Printf("[QUOTA][AUTO-RESET] user_id=%d sub_id=%d plan_id=%d consultations=%d questionnaires=%d clinical_cases=%d files=%d", userID, subID, planID, c1, c2, c3, c4)
	return nil
}
