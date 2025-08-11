package login

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"strings"
	"time"


	"ema-backend/migrations"
	"github.com/gin-gonic/gin"
)

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// in-memory token store: token -> email
var sessions = map[string]string{}

func generateToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback to timestamp-based token
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b)
}

// GetEmailFromToken returns email for a given session token
func GetEmailFromToken(token string) (string, bool) {
	email, ok := sessions[token]
	return email, ok
}


func Handler(c *gin.Context) {
	var creds Credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	user := migrations.GetUserByEmail(creds.Email)
	if user != nil && user.Password == creds.Password {

		token := generateToken()
		sessions[token] = user.Email

		userRes := gin.H{
			"id":            user.ID,
			"first_name":    user.FirstName,
			"last_name":     user.LastName,
			"email":         user.Email,
			"full_name":     user.FirstName + " " + user.LastName,
			"status":        true,
			"language":      "es",
			"dark_mode":     0,
			"created_at":    user.CreatedAt.Format(time.RFC3339),
			"updated_at":    user.UpdatedAt.Format(time.RFC3339),
			"profile_image": "",
		}
		c.JSON(http.StatusOK, gin.H{"token": token, "user": userRes})

	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales inválidas"})
	}
}


func SessionHandler(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	email, ok := sessions[token]
	if !ok || token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
		return
	}
	user := migrations.GetUserByEmail(email)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no encontrado"})
		return
	}
	userRes := gin.H{
		"id":            user.ID,
		"first_name":    user.FirstName,
		"last_name":     user.LastName,
		"email":         user.Email,
		"full_name":     user.FirstName + " " + user.LastName,
		"status":        true,
		"language":      "es",
		"dark_mode":     0,
		"created_at":    user.CreatedAt.Format(time.RFC3339),
		"updated_at":    user.UpdatedAt.Format(time.RFC3339),
		"profile_image": "",
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": userRes})
}

// LogoutHandler invalidates the token
func LogoutHandler(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token requerido"})
		return
	}
	delete(sessions, token)
	c.JSON(http.StatusOK, gin.H{"message": "Sesión cerrada"})
}

type RegisterPayload struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

func RegisterHandler(c *gin.Context) {
	var p RegisterPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}
	if p.Email == "" || p.Password == "" || p.FirstName == "" || p.LastName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Campos requeridos faltantes"})
		return
	}
	if exists, err := migrations.EmailExists(p.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error validando usuario"})
		return
	} else if exists {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "El correo ya está registrado"})
		return
	}
	if err := migrations.CreateUser(p.FirstName, p.LastName, p.Email, p.Password, "user"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el usuario"})
		return
	}
	c.Status(http.StatusCreated)
}

type ForgotPayload struct {
	Email string `json:"email"`
}

func ForgotPasswordHandler(c *gin.Context) {
	var p ForgotPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}
	// We acknowledge the request to reset the password.
	c.JSON(http.StatusOK, gin.H{"message": "Si el correo existe, se enviarán instrucciones"})
}

