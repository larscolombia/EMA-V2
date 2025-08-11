package profile

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ema-backend/login"
	"ema-backend/migrations"

	"github.com/gin-gonic/gin"
)

// getUserByID returns user by id
func getUserByID(id int) *migrations.User {
	row := migrations.GetUserByID(id)
	return row
}

// RegisterRoutes registers profile endpoints
func RegisterRoutes(r *gin.Engine) {
	r.GET("/user-detail/:id", getProfile)
	r.POST("/user-detail/:id", updateProfile)
}

func getProfile(c *gin.Context) {
	log.Printf("[PROFILE][GET] incoming request: path=%s headers=%v", c.Request.URL.Path, c.Request.Header)
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[PROFILE][GET] invalid id param: %s", idStr)
		c.JSON(http.StatusBadRequest, gin.H{"message": "ID inválido"})
		return
	}
	user := getUserByID(id)
	if user == nil {
		log.Printf("[PROFILE][GET] user not found for id=%d", id)
		c.JSON(http.StatusNotFound, gin.H{"message": "Usuario no encontrado"})
		return
	}
	// Ensure user has at least a Free subscription so app quotas work
	if err := migrations.EnsureFreeSubscriptionForUser(user.ID); err != nil {
		log.Printf("[PROFILE][GET] ensure free subscription failed for userID=%d: %v", user.ID, err)
	}
	// Attach user's latest subscription joined with plan as active_subscription
	activeSub, err := migrations.GetActiveSubscriptionForUser(user.ID)
	if err != nil {
		log.Printf("[PROFILE][GET] fetch active subscription failed for userID=%d: %v", user.ID, err)
	}
	resp := userToMap(user)
	if activeSub != nil {
		resp["active_subscription"] = activeSub
	}
	log.Printf("[PROFILE][GET] success id=%d email=%s hasActiveSub=%t", id, user.Email, activeSub != nil)
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func updateProfile(c *gin.Context) {
	log.Printf("[PROFILE][POST] incoming request: path=%s headers=%v", c.Request.URL.Path, c.Request.Header)
	idStr := c.Param("id")
	idParam, _ := strconv.Atoi(idStr)
	// Auth
	auth := c.GetHeader("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	email, ok := login.GetEmailFromToken(token)
	if !ok || token == "" {
		log.Printf("[PROFILE][POST] unauthorized: missing/invalid token. authHeader='%s' token='%s'", auth, token)
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Token inválido o ausente"})
		return
	}
	// Load user by email
	user := migrations.GetUserByEmail(email)
	if user == nil {
		log.Printf("[PROFILE][POST] session email found but user not in DB: %s", email)
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Usuario no encontrado"})
		return
	}
	if idParam != 0 && idParam != user.ID {
		log.Printf("[PROFILE][POST] id mismatch: param=%d sessionUserID=%d email=%s", idParam, user.ID, email)
		// Continue but log mismatch
	}

	// Is multipart with image?
	ct := c.GetHeader("Content-Type")
	log.Printf("[PROFILE][POST] content-type=%s", ct)
	if strings.HasPrefix(ct, "multipart/form-data") {
		// handle profile image upload
		file, err := c.FormFile("profile_image")
		if err != nil {
			log.Printf("[PROFILE][POST] multipart without 'profile_image' field: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": "Imagen no proporcionada"})
			return
		}
		log.Printf("[PROFILE][POST] uploading image: filename=%s size=%d userID=%d", file.Filename, file.Size, user.ID)
		// Create media folder per user
		base := filepath.Join("media", fmt.Sprintf("user_%d", user.ID))
		if err := os.MkdirAll(base, 0755); err != nil {
			log.Printf("[PROFILE][POST] failed to create media dir '%s': %v", base, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "No se pudo crear carpeta de medios"})
			return
		}
		// Save file
		ext := filepath.Ext(file.Filename)
		if ext == "" {
			ext = ".jpg"
		}
		fileName := fmt.Sprintf("profile%s", ext)
		dst := filepath.Join(base, fileName)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			log.Printf("[PROFILE][POST] failed to save image to '%s': %v", dst, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "No se pudo guardar la imagen"})
			return
		}
		// Store relative path in DB
		rel := filepath.ToSlash(dst)
		if err := migrations.UpdateUserProfileImage(user.ID, rel); err != nil {
			log.Printf("[PROFILE][POST] failed updating DB with image path '%s' for userID=%d: %v", rel, user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "No se pudo actualizar el usuario"})
			return
		}
		updated := migrations.GetUserByID(user.ID)
		log.Printf("[PROFILE][POST] image updated successfully for userID=%d path=%s", user.ID, rel)
		c.JSON(http.StatusOK, gin.H{"data": userToMap(updated)})
		return
	}

	// Otherwise JSON body with fields to update
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[PROFILE][POST] failed reading body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Cuerpo inválido"})
		return
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[PROFILE][POST] invalid JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "JSON inválido"})
		return
	}

	firstName := strings.TrimSpace(getString(payload, "first_name"))
	lastName := strings.TrimSpace(getString(payload, "last_name"))
	city := strings.TrimSpace(getString(payload, "city"))
	profession := strings.TrimSpace(getString(payload, "profession"))
	log.Printf("[PROFILE][POST] update fields userID=%d first_name='%s' last_name='%s' city='%s' profession='%s'", user.ID, firstName, lastName, city, profession)

	if err := migrations.UpdateUserProfile(user.ID, firstName, lastName, city, profession); err != nil {
		log.Printf("[PROFILE][POST] DB update failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "No se pudo actualizar"})
		return
	}
	updated := migrations.GetUserByID(user.ID)
	log.Printf("[PROFILE][POST] JSON update success for userID=%d", user.ID)
	c.JSON(http.StatusOK, gin.H{"data": userToMap(updated)})
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func userToMap(u *migrations.User) map[string]interface{} {
	if u == nil {
		return nil
	}
	return map[string]interface{}{
		"id":            u.ID,
		"first_name":    u.FirstName,
		"last_name":     u.LastName,
		"email":         u.Email,
		"full_name":     u.FirstName + " " + u.LastName,
		"status":        true,
		"language":      "es",
		"dark_mode":     0,
		"created_at":    u.CreatedAt.Format(time.RFC3339),
		"updated_at":    u.UpdatedAt.Format(time.RFC3339),
		"profile_image": u.ProfileImage,
		"city":          u.City,
		"profession":    u.Profession,
	}
}
