package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ema-backend/login"
	"ema-backend/migrations"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
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
	// Aggregated overview endpoint to reduce multiple sequential fetches on app start.
	r.GET("/user-overview/:id", getOverview)
	// Test completion endpoint for statistics tracking
	r.POST("/record-test", func(c *gin.Context) {
		log.Printf("🔥🔥🔥 [MIDDLEWARE] POST /record-test received - Method=%s Path=%s RemoteAddr=%s",
			c.Request.Method, c.Request.URL.Path, c.Request.RemoteAddr)
		recordTest(c)
	})
}

func getProfile(c *gin.Context) {
	log.Printf("[PROFILE][GET] incoming request: path=%s headers=%v", c.Request.URL.Path, c.Request.Header)
	auth := c.GetHeader("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token requerido"})
		return
	}
	email, ok := login.GetEmailFromToken(token)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido o expirado"})
		return
	}
	user := migrations.GetUserByEmail(email)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no encontrado"})
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
		// Structured log for quota visibility
		if sp, ok := activeSub["subscription_plan"].(map[string]interface{}); ok {
			planName, _ := sp["name"].(string)
			log.Printf("[PROFILE][QUOTA] user=%d plan=%s consultations=%v questionnaires=%v clinical_cases=%v files=%v", user.ID, planName,
				activeSub["consultations"], activeSub["questionnaires"], activeSub["clinical_cases"], activeSub["files"])
		}
	}
	log.Printf("[PROFILE][GET] success id=%d email=%s hasActiveSub=%t", user.ID, user.Email, activeSub != nil)
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// getOverview aggregates profile + simple stats in one roundtrip so the mobile app
// can avoid firing 5-6 requests (/user-detail, /user/:id/test-progress, etc.).
// Response shape:
// { data: { profile: {..user fields.., active_subscription: {...}}, stats: { clinical_cases_count, total_tests, test_progress, most_studied_category, chats } } }
// (Maintains backward compatibility by not altering existing /user-detail response.)
func getOverview(c *gin.Context) {
	start := time.Now()
	log.Printf("[OVERVIEW][GET] incoming request: path=%s", c.Request.URL.Path)
	auth := c.GetHeader("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token requerido"})
		return
	}
	email, ok := login.GetEmailFromToken(token)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido o expirado"})
		return
	}
	user := migrations.GetUserByEmail(email)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no encontrado"})
		return
	}

	// Ensure at least a free subscription
	if err := migrations.EnsureFreeSubscriptionForUser(user.ID); err != nil {
		log.Printf("[OVERVIEW][GET] ensure free subscription failed for userID=%d: %v", user.ID, err)
	}
	activeSub, err := migrations.GetActiveSubscriptionForUser(user.ID)
	if err != nil {
		log.Printf("[OVERVIEW][GET] fetch active subscription failed userID=%d: %v", user.ID, err)
	}
	prof := userToMap(user)
	if activeSub != nil {
		// Detect anomaly: remaining zero but plan limits > 0
		if sp, ok := activeSub["subscription_plan"].(map[string]interface{}); ok {
			consZero := activeSub["consultations"].(int) == 0
			clinZero := activeSub["clinical_cases"].(int) == 0
			planCons, _ := sp["consultations"].(int)
			planClin, _ := sp["clinical_cases"].(int)
			planHasAny := (planCons > 0 || planClin > 0)
			// Caso 1: ambos en cero pero plan tiene >0 en alguno -> reset completo
			if consZero && clinZero && planHasAny {
				log.Printf("[OVERVIEW][AUTO-RESET][DETECT] user_id=%d consultations=0 clinical_cases=0 plan_consultations=%v plan_clinical_cases=%v", user.ID, planCons, planClin)
				if err := migrations.ResetActiveSubscriptionQuotas(user.ID); err != nil {
					log.Printf("[OVERVIEW][AUTO-RESET][ERROR] user_id=%d err=%v", user.ID, err)
				} else if refreshed, err2 := migrations.GetActiveSubscriptionForUser(user.ID); err2 == nil && refreshed != nil {
					activeSub = refreshed
				}
			} else if clinZero && planClin > 0 { // Caso 2: solo clinical_cases en cero pero plan lo ofrece
				log.Printf("[OVERVIEW][AUTO-REPAIR][CLINICAL_CASES_ONLY] user_id=%d plan_clinical_cases=%d", user.ID, planClin)
				// Reparar solo clinical_cases usando actualización directa
				if err := migrations.ResetActiveSubscriptionQuotas(user.ID); err != nil {
					log.Printf("[OVERVIEW][AUTO-REPAIR][ERROR] user_id=%d err=%v", user.ID, err)
				} else if refreshed, err2 := migrations.GetActiveSubscriptionForUser(user.ID); err2 == nil && refreshed != nil {
					activeSub = refreshed
				}
			}
		}
		prof["active_subscription"] = activeSub
		if sp, ok := activeSub["subscription_plan"].(map[string]interface{}); ok {
			planName, _ := sp["name"].(string)
			log.Printf("[OVERVIEW][QUOTA] user=%d plan=%s consultations=%v questionnaires=%v clinical_cases=%v files=%v", user.ID, planName,
				activeSub["consultations"], activeSub["questionnaires"], activeSub["clinical_cases"], activeSub["files"])
		}
	}

	// Calcular estadísticas reales basadas en los quotas de la suscripción
	stats := gin.H{
		"clinical_cases_count":  0,
		"total_tests":           0,
		"test_progress":         []any{},
		"most_studied_category": nil,
		"monthly_scores":        []any{},
		"chats":                 []any{},
	}

	if activeSub != nil {
		plan, _ := activeSub["subscription_plan"].(map[string]interface{})

		// Clinical Cases
		planClinical := plan["clinical_cases"].(int)
		remainingClinical := activeSub["clinical_cases"].(int)
		usedClinical := planClinical - remainingClinical
		stats["clinical_cases_count"] = usedClinical

		// Total Tests/Questionnaires
		planTests := plan["questionnaires"].(int)
		remainingTests := activeSub["questionnaires"].(int)
		usedTests := planTests - remainingTests
		stats["total_tests"] = usedTests

		// Chats/Consultations (devuelto como array para compatibilidad)
		planChats := plan["consultations"].(int)
		remainingChats := activeSub["consultations"].(int)
		usedChats := planChats - remainingChats
		// Crear array de usedChats elementos para compatibilidad con frontend
		chatsArray := make([]int, usedChats)
		stats["chats"] = chatsArray

		log.Printf("[OVERVIEW][STATS] userID=%d clinical_used=%d tests_used=%d chats_used=%d",
			user.ID, usedClinical, usedTests, usedChats)
	}

	// Obtener estadísticas detalladas de test_history
	if testProgress, err := migrations.GetTestProgress(user.ID, 10); err == nil && testProgress != nil {
		stats["test_progress"] = testProgress
		log.Printf("[OVERVIEW][STATS] userID=%d test_progress_count=%d", user.ID, len(testProgress))
	}

	if monthlyScores, err := migrations.GetMonthlyScores(user.ID); err == nil && monthlyScores != nil {
		stats["monthly_scores"] = monthlyScores
		log.Printf("[OVERVIEW][STATS] userID=%d monthly_scores_count=%d", user.ID, len(monthlyScores))
	}

	if mostStudied, err := migrations.GetMostStudiedCategory(user.ID); err == nil && mostStudied != nil {
		stats["most_studied_category"] = mostStudied
		log.Printf("[OVERVIEW][STATS] userID=%d most_studied=%v", user.ID, mostStudied)
	}

	duration := time.Since(start)
	log.Printf("[OVERVIEW][GET] success userID=%d hasActiveSub=%t duration=%s", user.ID, activeSub != nil, duration)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"profile": prof, "stats": stats}})
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
			c.JSON(http.StatusBadRequest, gin.H{"message": "Imagen no proporcionada", "code": "image_missing"})
			return
		}
		log.Printf("[PROFILE][POST] uploading image: filename=%s size=%d userID=%d", file.Filename, file.Size, user.ID)

		// Validaciones básicas de tamaño (10 MB por defecto)
		maxMB := 10
		if envMax := os.Getenv("PROFILE_IMAGE_MAX_MB"); envMax != "" {
			if n, err := strconv.Atoi(envMax); err == nil && n > 0 {
				maxMB = n
			}
		}
		if file.Size <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Archivo vacío", "code": "image_empty"})
			return
		}
		if file.Size > int64(maxMB*1024*1024) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"message": "Imagen demasiado grande", "code": "image_too_large", "max_size_mb": maxMB})
			return
		}

		// Check if Cloudinary is configured
		cloudinaryURL := strings.TrimSpace(os.Getenv("CLOUDINARY_URL"))
		if cloudinaryURL == "" {
			log.Printf("[PROFILE][POST] CLOUDINARY_URL not configured, falling back to local storage")
			// Fallback to local storage (existing code)
			mediaRoot := strings.TrimSpace(os.Getenv("MEDIA_ROOT"))
			if mediaRoot == "" {
				mediaRoot = "./media"
			}
			base := filepath.Join(mediaRoot, fmt.Sprintf("user_%d", user.ID))
			if err := os.MkdirAll(base, 0755); err != nil {
				log.Printf("[PROFILE][POST] failed to create media dir '%s': %v", base, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "No se pudo crear carpeta de medios"})
				return
			}
			// Save file
			ext := normalizeImageExt(file.Filename, file.Header.Get("Content-Type"))
			fileName := fmt.Sprintf("profile%s", ext)
			dst := filepath.Join(base, fileName)
			if err := c.SaveUploadedFile(file, dst); err != nil {
				log.Printf("[PROFILE][POST] failed to save image to '%s': %v", dst, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "No se pudo guardar la imagen", "code": "image_save_failed"})
				return
			}
			// Store relative path in DB as a URL path under /media
			relPath, err := filepath.Rel(mediaRoot, dst)
			if err != nil {
				relPath = filepath.Base(dst)
			}
			rel := "/media/" + filepath.ToSlash(relPath)
			if err := migrations.UpdateUserProfileImage(user.ID, rel); err != nil {
				log.Printf("[PROFILE][POST] failed updating DB with image path '%s' for userID=%d: %v", rel, user.ID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "No se pudo actualizar el usuario"})
				return
			}
		} else {
			// Use Cloudinary
			imageURL, err := uploadToCloudinary(file, user.ID)
			if err != nil {
				log.Printf("[PROFILE][POST] failed uploading to Cloudinary for userID=%d: %v", user.ID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "No se pudo subir la imagen", "code": "cloudinary_upload_failed"})
				return
			}

			// Store Cloudinary URL in DB
			if err := migrations.UpdateUserProfileImage(user.ID, imageURL); err != nil {
				log.Printf("[PROFILE][POST] failed updating DB with Cloudinary URL '%s' for userID=%d: %v", imageURL, user.ID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "No se pudo actualizar el usuario"})
				return
			}
		}

		updated := migrations.GetUserByID(user.ID)
		log.Printf("[PROFILE][POST] image updated successfully for userID=%d", user.ID)
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
	gender := strings.TrimSpace(getString(payload, "gender"))

	var age *int
	if ageVal, ok := payload["age"]; ok && ageVal != nil {
		if ageFloat, ok := ageVal.(float64); ok {
			ageInt := int(ageFloat)
			age = &ageInt
		}
	}

	var countryID *int
	if countryVal, ok := payload["country_id"]; ok && countryVal != nil {
		if countryFloat, ok := countryVal.(float64); ok {
			countryInt := int(countryFloat)
			countryID = &countryInt
		}
	}

	log.Printf("[PROFILE][POST] update fields userID=%d first_name='%s' last_name='%s' city='%s' profession='%s' gender='%s' age=%v country_id=%v", user.ID, firstName, lastName, city, profession, gender, age, countryID)

	if err := migrations.UpdateUserProfile(user.ID, firstName, lastName, city, profession, gender, age, countryID); err != nil {
		log.Printf("[PROFILE][POST] DB update failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "No se pudo actualizar"})
		return
	}
	updated := migrations.GetUserByID(user.ID)
	ageVal := "nil"
	if updated.Age != nil {
		ageVal = fmt.Sprintf("%d", *updated.Age)
	}
	countryVal := "nil"
	if updated.CountryID != nil {
		countryVal = fmt.Sprintf("%d", *updated.CountryID)
	}
	log.Printf("[PROFILE][POST] Updated user data: gender='%s' age=%s country_id=%s", updated.Gender, ageVal, countryVal)
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
	m := map[string]interface{}{
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
		"gender":        u.Gender,
	}
	if u.Age != nil {
		m["age"] = *u.Age
	}
	if u.CountryID != nil {
		m["country_id"] = *u.CountryID
		m["country_name"] = getCountryName(*u.CountryID)
	}
	return m
}

// getCountryName returns the country name based on country ID
func getCountryName(countryID int) string {
	countries := map[int]string{
		1: "Colombia",
		2: "México",
		3: "Perú",
		4: "Argentina",
		5: "Chile",
		6: "España",
		7: "Estados Unidos",
	}
	if name, ok := countries[countryID]; ok {
		return name
	}
	return ""
}

// normalizeImageExt returns a safe image extension based on filename and content-type.
// Allowed: .jpg, .jpeg, .png, .webp. Default: .jpg.
func normalizeImageExt(filename, contentType string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return ext
	}
	// Infer from content type if possible
	ct := strings.ToLower(strings.TrimSpace(contentType))
	switch ct {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

// uploadToCloudinary uploads a profile image to Cloudinary and returns the public URL
func uploadToCloudinary(file *multipart.FileHeader, userID int) (string, error) {
	// Check if Cloudinary is configured
	cloudinaryURL := strings.TrimSpace(os.Getenv("CLOUDINARY_URL"))
	if cloudinaryURL == "" {
		return "", fmt.Errorf("CLOUDINARY_URL no está configurado")
	}

	// Initialize Cloudinary client
	cld, err := cloudinary.NewFromURL(cloudinaryURL)
	if err != nil {
		return "", fmt.Errorf("error inicializando Cloudinary: %v", err)
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("error abriendo archivo: %v", err)
	}
	defer src.Close()

	// Configure upload parameters
	uploadParams := uploader.UploadParams{
		PublicID:       fmt.Sprintf("profile_images/user_%d_profile", userID),
		Folder:         "ema_profiles",              // Organize images in a folder
		Transformation: "c_fill,g_face,h_400,w_400", // Resize to 400x400, crop to face
		ResourceType:   "image",
	}

	// Upload to Cloudinary
	ctx := context.Background()
	result, err := cld.Upload.Upload(ctx, src, uploadParams)
	if err != nil {
		return "", fmt.Errorf("error subiendo a Cloudinary: %v", err)
	}

	log.Printf("[CLOUDINARY] Image uploaded successfully: userID=%d, publicID=%s, url=%s",
		userID, result.PublicID, result.SecureURL)

	return result.SecureURL, nil
}

// recordTest handles POST /record-test for recording test completions
func recordTest(c *gin.Context) {
	log.Printf("[RECORD_TEST] 🚀 INICIO - Received POST request to /record-test")
	log.Printf("[RECORD_TEST] Headers: %v", c.Request.Header)

	auth := c.GetHeader("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" {
		log.Printf("[RECORD_TEST] ❌ No token provided")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token requerido"})
		return
	}

	log.Printf("[RECORD_TEST] Token received: %s...", token[:20])

	email, ok := login.GetEmailFromToken(token)
	if !ok {
		log.Printf("[RECORD_TEST] ❌ Invalid token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido o expirado"})
		return
	}

	log.Printf("[RECORD_TEST] Token valid for email: %s", email)
	user := migrations.GetUserByEmail(email)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no encontrado"})
		return
	}

	var payload struct {
		CategoryID    *int   `json:"category_id"`
		TestName      string `json:"test_name"`
		ScoreObtained int    `json:"score_obtained"`
		MaxScore      int    `json:"max_score"`
	}

	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos", "details": err.Error()})
		return
	}

	// Validaciones
	if payload.TestName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "test_name es requerido"})
		return
	}
	if payload.MaxScore <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "max_score debe ser mayor a 0"})
		return
	}
	if payload.ScoreObtained < 0 || payload.ScoreObtained > payload.MaxScore {
		c.JSON(http.StatusBadRequest, gin.H{"error": "score_obtained debe estar entre 0 y max_score"})
		return
	}

	log.Printf("[RECORD_TEST] Attempting to record: userID=%d testName=%s score=%d/%d categoryID=%v",
		user.ID, payload.TestName, payload.ScoreObtained, payload.MaxScore, payload.CategoryID)

	// Registrar en test_history
	if err := migrations.RecordTestCompletion(user.ID, payload.CategoryID, payload.TestName, payload.ScoreObtained, payload.MaxScore); err != nil {
		log.Printf("[RECORD_TEST] ❌ Error recording test for userID=%d: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo registrar el test", "details": err.Error()})
		return
	}

	log.Printf("[RECORD_TEST] ✅ Success: userID=%d testName=%s score=%d/%d categoryID=%v",
		user.ID, payload.TestName, payload.ScoreObtained, payload.MaxScore, payload.CategoryID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test registrado exitosamente",
	})
}
