package login

import (
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

const tokenValue = "dummy-token"

func Handler(c *gin.Context) {
	var creds Credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	user := migrations.GetUserByEmail(creds.Email)
	if user != nil && user.Password == creds.Password {
		userRes := gin.H{
			"id":            user.ID,
			"first_name":    user.FirstName,
			"last_name":     user.LastName,
			"email":         user.Email,
			"full_name":     user.FirstName + " " + user.LastName,
			"status":        true,
			"language":      "es",
			"dark_mode":     0,
			"created_at":    time.Now().Format(time.RFC3339),
			"updated_at":    time.Now().Format(time.RFC3339),
			"profile_image": "",
		}
		c.JSON(http.StatusOK, gin.H{"token": tokenValue, "user": userRes})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales inválidas"})
	}
}

func SessionHandler(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token != tokenValue {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
		return
	}
	user := migrations.GetUserByEmail("leonardoherrerac10@gmail.com")
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
		"created_at":    time.Now().Format(time.RFC3339),
		"updated_at":    time.Now().Format(time.RFC3339),
		"profile_image": "",
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenValue, "user": userRes})
}
