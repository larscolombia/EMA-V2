package marketing

import (
	"database/sql"
	"log"
	"time"

	"ema-backend/email"
)

// Service gestiona el envío de campañas a usuarios gratuitos.
type Service struct {
	db *sql.DB
}

// NewService crea un nuevo servicio de marketing.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// Start inicia un ticker que periódicamente notifica a los usuarios gratuitos.
func (s *Service) Start() {
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			if err := s.notifyFreeUsers(); err != nil {
				log.Printf("[MARKETING] error notificando usuarios gratuitos: %v", err)
			}
		}
	}()
}

// notifyFreeUsers obtiene a los usuarios con plan gratuito y les envía correos y pushes.
func (s *Service) notifyFreeUsers() error {
	rows, err := s.db.Query(`SELECT u.id, u.email FROM users u
        JOIN subscriptions s2 ON u.id = s2.user_id
        JOIN subscription_plans p ON s2.plan_id = p.id
        WHERE p.price = 0`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var mail string
		if err := rows.Scan(&id, &mail); err != nil {
			return err
		}
		if err := email.SendUpgradeSuggestion(mail); err != nil {
			log.Printf("[MARKETING] fallo enviando correo a %s: %v", mail, err)
		}
		log.Printf("[MARKETING] push enviado a usuario %d", id)
	}
	return rows.Err()
}
