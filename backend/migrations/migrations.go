package migrations

import "database/sql"

type User struct {
        ID        int
        FirstName string
        LastName  string
        Email     string
        Password  string
        Role      string
}

var users = []User{
        {
                ID:        1,
                FirstName: "Leonardo",
                LastName:  "Herrera",
                Email:     "leonardoherrerac10@gmail.com",
                Password:  "supersecret",
                Role:      "super_admin",
        },
}

func GetUserByEmail(email string) *User {
        for i, u := range users {
                if u.Email == email {
                        return &users[i]
                }
        }
        return nil
}

// Run ejecuta las migraciones necesarias para el sistema de planes y suscripciones
func Run(db *sql.DB) error {
        _, err := db.Exec(`CREATE TABLE IF NOT EXISTS subscription_plans (
                id INT AUTO_INCREMENT PRIMARY KEY,
                name VARCHAR(255) NOT NULL,
                currency VARCHAR(10) NOT NULL,
                price DECIMAL(10,2) NOT NULL,
                billing VARCHAR(50) NOT NULL,
                consultations INT NOT NULL,
                questionnaires INT NOT NULL,
                clinical_cases INT NOT NULL,
                files INT NOT NULL
        )`)
        if err != nil {
                return err
        }

        _, err = db.Exec(`CREATE TABLE IF NOT EXISTS subscriptions (
                id INT AUTO_INCREMENT PRIMARY KEY,
                user_id INT NOT NULL,
                plan_id INT NOT NULL,
                start_date DATETIME NOT NULL,
                end_date DATETIME NULL,
                frequency INT NOT NULL,
                consultations INT NOT NULL,
                questionnaires INT NOT NULL,
                clinical_cases INT NOT NULL,
                files INT NOT NULL,
                FOREIGN KEY (plan_id) REFERENCES subscription_plans(id)
        )`)
        if err != nil {
                return err
        }

        _, err = db.Exec(`INSERT INTO subscription_plans (name, currency, price, billing, consultations, questionnaires, clinical_cases, files)
                SELECT * FROM (SELECT 'Free' AS name, 'USD' AS currency, 0 AS price, 'Mensual' AS billing, 5 AS consultations, 5 AS questionnaires, 5 AS clinical_cases, 5 AS files) AS tmp
                WHERE NOT EXISTS (SELECT id FROM subscription_plans WHERE name = 'Free')`)
        return err
}

