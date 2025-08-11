package login

import (
	"net/http"

	"ema-backend/migrations"
	"github.com/gin-gonic/gin"
)

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Handler(c *gin.Context) {
	var creds Credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	user := migrations.GetUserByEmail(creds.Email)
	if user != nil && user.Password == creds.Password {
		c.JSON(http.StatusOK, gin.H{"message": "Inicio de sesión exitoso"})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales inválidas"})
	}
}
