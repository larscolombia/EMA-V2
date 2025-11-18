package email

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

func send(to, subject, body string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = user
	}
	if host == "" || port == "" || user == "" || pass == "" || from == "" {
		return fmt.Errorf("SMTP environment variables missing")
	}
	addr := fmt.Sprintf("%s:%s", host, port)
	auth := smtp.PlainAuth("", user, pass, host)
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, to, subject, body))
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

func SendWelcome(to string) error {
	subject := "Bienvenido"
	body := "Gracias por registrarte. ¡Bienvenido!"
	if err := send(to, subject, body); err != nil {
		return err
	}
	log.Printf("[EMAIL] welcome sent to %s", to)
	return nil
}

func SendPasswordChanged(to string) error {
	subject := "Contraseña actualizada"
	body := "Tu contraseña ha sido cambiada satisfactoriamente. Si no fuiste tú, contacta soporte."
	if err := send(to, subject, body); err != nil {
		return err
	}
	log.Printf("[EMAIL] password change notification sent to %s", to)
	return nil
}

// SendPasswordReset envía el correo con el enlace de recuperación de contraseña
func SendPasswordReset(to, resetLink string) error {
	subject := "Recuperación de contraseña - EMA"
	body := fmt.Sprintf(`Hola,

Recibimos una solicitud para restablecer tu contraseña.

Haz clic en el siguiente enlace para crear una nueva contraseña:
%s

Este enlace expirará en 1 hora.

Si no solicitaste este cambio, puedes ignorar este correo.

Saludos,
Equipo EMA`, resetLink)
	if err := send(to, subject, body); err != nil {
		return err
	}
	log.Printf("[EMAIL] password reset sent to %s", to)
	return nil
}

// SendUpgradeSuggestion envía un correo promocionando los planes premium.
func SendUpgradeSuggestion(to string) error {
	subject := "Actualiza a un plan premium"
	body := "Aprovecha las funcionalidades avanzadas cambiándote a un plan premium."
	if err := send(to, subject, body); err != nil {
		return err
	}
	log.Printf("[EMAIL] upgrade suggestion sent to %s", to)
	return nil
}
