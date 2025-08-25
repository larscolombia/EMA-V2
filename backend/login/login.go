package login

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	mailer "ema-backend/email"
	"ema-backend/migrations"

	"github.com/gin-gonic/gin"
)

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

// blacklist for manual logout (tokens -> expiry). Not persisted; acceptable for MVP.
var blacklist = map[string]int64{}

// tokenPayload minimal JWT-like payload
type tokenPayload struct {
	Email string `json:"email"`
	Exp   int64  `json:"exp"`
	Rem   bool   `json:"rem"` // remember flag
	Jti   string `json:"jti"` // unique id
}

func sessionDurations(remember bool) time.Duration {
	defHours := 12
	if v := os.Getenv("SESSION_DEFAULT_HOURS"); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { defHours = n } }
	remDays := 30
	if v := os.Getenv("SESSION_REMEMBER_DAYS"); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { remDays = n } }
	if remember { return time.Hour * 24 * time.Duration(remDays) }
	return time.Hour * time.Duration(defHours)
}

func sessionSecret() []byte {
	s := os.Getenv("SESSION_SECRET")
	if s == "" { s = "dev-insecure-secret" }
	return []byte(s)
}

func signToken(email string, dur time.Duration, remember bool) (string, int64, error) {
	exp := time.Now().Add(dur).Unix()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(tokenPayload{Email: email, Exp: exp, Rem: remember, Jti: generateJTI()})
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	mac := hmac.New(sha256.New, sessionSecret())
	mac.Write([]byte(header + "." + payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return header + "." + payload + "." + sig, exp, nil
}

func parseToken(token string) (tokenPayload, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 { return tokenPayload{}, false }
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, sessionSecret())
	mac.Write([]byte(unsigned))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) { return tokenPayload{}, false }
	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil { return tokenPayload{}, false }
	var tp tokenPayload
	if err := json.Unmarshal(pb, &tp); err != nil { return tokenPayload{}, false }
	if tp.Exp < time.Now().Unix() { return tokenPayload{}, false }
	if exp, blk := blacklist[token]; blk && exp >= time.Now().Unix() { return tokenPayload{}, false }
	return tp, true
}

func generateJTI() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil { return time.Now().Format("20060102150405") }
	return hex.EncodeToString(b)
}

// GetEmailFromToken validates signature + expiry and returns email
func GetEmailFromToken(token string) (string, bool) {
	tp, ok := parseToken(token)
	if !ok { return "", false }
	return tp.Email, true
}

func Handler(c *gin.Context) {
	var creds Credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// Normalize inputs
	creds.Email = strings.TrimSpace(strings.ToLower(creds.Email))
	creds.Password = strings.TrimSpace(creds.Password)

	user := migrations.GetUserByEmail(creds.Email)
	if user != nil && user.Password == creds.Password {
		dur := sessionDurations(creds.Remember)
		token, exp, _ := signToken(user.Email, dur, creds.Remember)

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
	c.JSON(http.StatusOK, gin.H{"token": token, "user": userRes, "expires_at": exp, "remember": creds.Remember})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales inválidas"})
	}
}

func SessionHandler(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" { c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"}); return }
	tp, ok := parseToken(token)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
		return
	}
	email := tp.Email
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
	if token == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "Token requerido"}); return }
	// Add to blacklist until its natural expiry (best effort)
	if tp, ok := parseToken(token); ok { blacklist[token] = tp.Exp }
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
	if err := mailer.SendWelcome(p.Email); err != nil {
		log.Printf("send welcome email failed for %s: %v", p.Email, err)
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

type ChangePasswordPayload struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func ChangePasswordHandler(c *gin.Context) {
	var p ChangePasswordPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}
	auth := c.GetHeader("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	userEmail, ok := GetEmailFromToken(token)
	if !ok || token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
		return
	}
	user := migrations.GetUserByEmail(userEmail)
	if user == nil || user.Password != p.OldPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales inválidas"})
		return
	}
	if err := migrations.UpdateUserPassword(user.ID, p.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar la contraseña"})
		return
	}
	if err := mailer.SendPasswordChanged(user.Email); err != nil {
		log.Printf("send password change email failed for %s: %v", user.Email, err)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Contraseña actualizada"})
}

// RefreshHandler issues a new token preserving remember flag while previous token is blacklisted.
func RefreshHandler(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" { c.JSON(http.StatusUnauthorized, gin.H{"error":"Token requerido"}); return }
	tp, ok := parseToken(token)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error":"Token inválido o expirado"}); return }
	dur := time.Until(time.Unix(tp.Exp,0))
	// Recalculate full duration based on remember flag if remaining <50% to extend period
	baseDur := sessionDurations(tp.Rem)
	if dur < baseDur/2 { dur = baseDur } // extend window
	newToken, newExp, _ := signToken(tp.Email, dur, tp.Rem)
	// Blacklist old token
	blacklist[token] = tp.Exp
	c.JSON(http.StatusOK, gin.H{"token": newToken, "expires_at": newExp, "remember": tp.Rem})
}

// TokenExpiryHeader middleware adds X-Token-Expires-At when token válido.
func TokenExpiryHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token != "" {
			if tp, ok := parseToken(token); ok {
				c.Writer.Header().Set("X-Token-Expires-At", strconv.FormatInt(tp.Exp,10))
				if tp.Rem { c.Writer.Header().Set("X-Token-Remember", "1") }
			}
		}
		c.Next()
	}
}
