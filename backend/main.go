package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"ema-backend/casos_clinico"
	"ema-backend/casos_interactivos"
	"ema-backend/categories"
	"ema-backend/chat"
	"ema-backend/conn"
	"ema-backend/conversations_ia"
	"ema-backend/countries"
	"ema-backend/login"
	"ema-backend/marketing"
	"ema-backend/migrations"
	"ema-backend/openai"
	"ema-backend/profile"
	"ema-backend/quota"
	"ema-backend/subscriptions"
	"ema-backend/testsapi"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// BuildCommit is injected at build time via: -ldflags "-X main.BuildCommit=$(git rev-parse --short HEAD)"
// If empty (e.g., during `go run`), it falls back to "dev".
var BuildCommit string

func main() {
	// Ensure BuildCommit has a fallback for local runs.
	if BuildCommit == "" {
		BuildCommit = "dev"
	}
	// Load environment variables from .env if present
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file found, using environment variables: %v", err)
	}

	// Connect to MySQL
	db, err := conn.NewMySQL()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	// Initialize migrations package with DB and run migrations
	migrations.Init(db)
	if err := migrations.Migrate(); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}
	if err := migrations.SeedDefaultUser(); err != nil {
		log.Printf("seed default user failed: %v", err)
	}
	if err := migrations.SeedDefaultPlans(); err != nil {
		log.Printf("seed default plans failed: %v", err)
	}
	if err := migrations.SeedMedicalCategories(); err != nil {
		log.Printf("seed medical categories failed: %v", err)
	}

	mk := marketing.NewService(db)
	go mk.Start()

	r := gin.Default()
	// Replace default Recovery with custom JSON-aware recovery for /conversations/*
	r.Use(func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[panic] path=%s method=%s err=%v", c.Request.URL.Path, c.Request.Method, rec)
				// If conversations endpoint, force JSON body so client sees code
				if strings.HasPrefix(c.Request.URL.Path, "/conversations/") {
					c.AbortWithStatusJSON(500, gin.H{"error": "internal_panic", "detail": fmt.Sprintf("%v", rec)})
					return
				}
				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	})
	// Attach token expiry header middleware globally
	r.Use(login.TokenExpiryHeader())

	// Request logging middleware (minimal)
	r.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		c.Next()
		dur := time.Since(start).Milliseconds()
		log.Printf("[req] %s %s status=%d dur_ms=%d", method, path, c.Writer.Status(), dur)
	})

	// Middleware to handle 413 errors with better messages for file uploads
	r.Use(func(c *gin.Context) {
		c.Next()

		// Check if response was 413 and convert to JSON if it's a conversations endpoint
		if c.Writer.Status() == 413 && strings.HasPrefix(c.Request.URL.Path, "/conversations/") {
			// Clear any existing response and send JSON
			c.Header("Content-Type", "application/json")
			log.Printf("[413] file_too_large path=%s", c.Request.URL.Path)

			// Don't write if response already started
			if !c.Writer.Written() {
				c.JSON(413, gin.H{
					"error":       "archivo demasiado grande",
					"code":        "file_too_large_nginx",
					"detail":      "El archivo excede el límite permitido de 100 MB.",
					"max_size_mb": 100,
				})
			}
		}
	})

	// Initialize OpenAI client BEFORE routes that reference it
	openai.SetPersistDB(db)
	ai := openai.NewClient()

	// Health check extendido (needs ai)
	r.GET("/health", func(c *gin.Context) {
		assistantID := ai.GetAssistantID()
		c.JSON(http.StatusOK, gin.H{
			"status":               "ok",
			"build":                BuildCommit,
			"assistant_configured": strings.HasPrefix(assistantID, "asst_"),
		})
	})

	// Auth routes expected by Flutter
	r.POST("/login", login.Handler)
	r.GET("/session", login.SessionHandler)
	r.POST("/logout", login.LogoutHandler)
	r.POST("/session/refresh", login.RefreshHandler)
	r.POST("/register", login.RegisterHandler)
	r.POST("/password/forgot", login.ForgotPasswordHandler)
	r.POST("/password/change", login.ChangePasswordHandler)

	// Profile routes and static media
	profile.RegisterRoutes(r)
	mediaRoot := strings.TrimSpace(os.Getenv("MEDIA_ROOT"))
	if mediaRoot == "" {
		mediaRoot = "./media"
	}
	if err := os.MkdirAll(mediaRoot, 0755); err != nil {
		log.Printf("[media] failed to ensure MEDIA_ROOT '%s': %v", mediaRoot, err)
	} else {
		log.Printf("[media] serving /media from %s", mediaRoot)
	}
	r.Static("/media", mediaRoot)

	// Subscriptions & quota
	subRepo := subscriptions.NewRepository(db)
	subHandler := subscriptions.NewHandler(subRepo)
	subHandler.RegisterRoutes(r)
	// Quota validator setup: user resolver via migrations (minimal projection)
	quota.RegisterUserResolver(func(email string) *quota.UserLite {
		u := migrations.GetUserByEmail(email)
		if u == nil {
			return nil
		}
		return &quota.UserLite{ID: u.ID, Email: u.Email}
	})
	qValidator := quota.NewValidator(subRepo)

	// Countries route (simple list)
	countries.RegisterRoutes(r)

	// Medical categories
	catRepo := categories.NewRepository(db)
	catHandler := categories.NewHandler(catRepo)
	catHandler.RegisterRoutes(r)

	// Chat/OpenAI endpoints (optional if keys provided) - client already initialized above
	chatHandler := chat.NewHandler(ai)
	chatHandler.SetQuotaValidator(qValidator.ValidateAndConsume)
	r.POST("/asistente/start", chatHandler.Start)
	r.POST("/asistente/message", chatHandler.Message)
	// Cleanup endpoint to delete OpenAI artifacts for a thread
	r.POST("/asistente/delete", chatHandler.Delete)
	// Vector store maintenance & inspection
	r.POST("/asistente/vector/reset", chatHandler.VectorReset)
	r.GET("/asistente/vector/files", chatHandler.VectorFiles)

	// Nuevo chat migrado (Assistants v2 estricto)
	convHandler := conversations_ia.NewHandler(ai)
	convHandler.SetQuotaValidator(qValidator.ValidateAndConsume)
	r.POST("/conversations/start", convHandler.Start)
	r.POST("/conversations/message", convHandler.Message)
	r.POST("/conversations/message/debug", convHandler.MessageDebug) // Versión JSON para debugging en Postman
	// Debug config (non-secret) – can be protected later behind ENV
	r.GET("/conversations/debug/config", convHandler.DebugConfig)
	// Paridad: limpieza y vector store
	r.POST("/conversations/delete", convHandler.Delete)
	r.POST("/conversations/vector/reset", convHandler.VectorReset)
	r.GET("/conversations/vector/files", convHandler.VectorFiles)

	// Quotas endpoint for current user (Authorization token required)
	r.GET("/me/quotas", func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			c.JSON(401, gin.H{"error": "token requerido"})
			return
		}
		email, ok := login.GetEmailFromToken(token)
		if !ok {
			c.JSON(401, gin.H{"error": "sesión inválida"})
			return
		}
		u := migrations.GetUserByEmail(email)
		if u == nil {
			c.JSON(404, gin.H{"error": "usuario no encontrado"})
			return
		}
		sub, err := subRepo.GetActiveSubscription(u.ID)
		if err != nil || sub == nil {
			c.JSON(404, gin.H{"error": "suscripción no encontrada"})
			return
		}
		c.JSON(200, gin.H{"consultations": sub.Consultations, "questionnaires": sub.Questionnaires, "clinical_cases": sub.ClinicalCases, "files": sub.Files})
	})

	// Active subscription summary (plan info)
	r.GET("/me/subscription", func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			c.JSON(401, gin.H{"error": "token requerido"})
			return
		}
		email, ok := login.GetEmailFromToken(token)
		if !ok {
			c.JSON(401, gin.H{"error": "sesión inválida"})
			return
		}
		u := migrations.GetUserByEmail(email)
		if u == nil {
			c.JSON(404, gin.H{"error": "usuario no encontrado"})
			return
		}
		sub, err := subRepo.GetActiveSubscription(u.ID)
		if err != nil || sub == nil {
			c.JSON(404, gin.H{"error": "suscripción no encontrada"})
			return
		}
		c.JSON(200, gin.H{"plan": sub.Plan, "subscription_id": sub.ID})
	})

	// Plans + active plan id (for UI highlight)
	r.GET("/me/plans", func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			c.JSON(401, gin.H{"error": "token requerido"})
			return
		}
		email, ok := login.GetEmailFromToken(token)
		if !ok {
			c.JSON(401, gin.H{"error": "sesión inválida"})
			return
		}
		u := migrations.GetUserByEmail(email)
		if u == nil {
			c.JSON(404, gin.H{"error": "usuario no encontrado"})
			return
		}
		plans, err := subRepo.GetPlans()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		sub, _ := subRepo.GetActiveSubscription(u.ID)
		activeID := 0
		if sub != nil {
			activeID = sub.PlanID
		}
		list := []gin.H{}
		for _, p := range plans {
			list = append(list, gin.H{"id": p.ID, "name": p.Name, "price": p.Price, "billing": p.Billing, "consultations": p.Consultations, "questionnaires": p.Questionnaires, "clinical_cases": p.ClinicalCases, "files": p.Files, "currency": p.Currency, "active": p.ID == activeID})
		}
		c.JSON(200, gin.H{"plans": list, "active_plan_id": activeID})
	})

	// Tests (quizzes) endpoints for Flutter
	testsHandler := testsapi.DefaultHandler()
	testsHandler.SetQuotaValidator(qValidator.ValidateAndConsume)
	// Inject category name resolver for better prompts when user selects a category
	testsHandler.SetCategoryResolver(func(ctx context.Context, ids []int) ([]string, error) {
		return catRepo.NamesByIDs(ctx, ids)
	})
	testsHandler.RegisterRoutes(r)

	// Clinical cases endpoints (analytical & interactive)
	clinicalHandler := casos_clinico.DefaultHandler()
	clinicalHandler.SetQuotaValidator(qValidator.ValidateAndConsume)
	clinicalHandler.RegisterRoutes(r)

	// New interactive flow with stricter JSON turn contract
	interactivosHandler := casos_interactivos.DefaultHandler()
	interactivosHandler.SetQuotaValidator(qValidator.ValidateAndConsume)
	interactivosHandler.RegisterRoutes(r)

	// Removed legacy stats stub endpoints (now served via /user-overview aggregate)

	// Debug endpoint to reset clinical_cases quota for current token (ONLY for local dev)
	r.POST("/debug/reset-clinical-cases", func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			c.JSON(401, gin.H{"error": "token requerido"})
			return
		}
		email, ok := login.GetEmailFromToken(token)
		if !ok {
			c.JSON(401, gin.H{"error": "sesión inválida"})
			return
		}
		u := migrations.GetUserByEmail(email)
		if u == nil {
			c.JSON(404, gin.H{"error": "usuario no encontrado"})
			return
		}
		var body struct {
			Value int `json:"value"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Value < 0 {
			c.JSON(400, gin.H{"error": "value inválido"})
			return
		}
		sub, err := subRepo.GetActiveSubscription(u.ID)
		if err != nil || sub == nil {
			c.JSON(404, gin.H{"error": "suscripción no encontrada"})
			return
		}
		if err := subRepo.SetQuotaValue(sub.ID, "clinical_cases", body.Value); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok", "subscription_id": sub.ID, "clinical_cases": body.Value})
	})

	// Dev helper: forzar creación de suscripción (evita esperar webhook). Requiere APP_ENV=dev
	r.POST("/dev/force-subscribe", func(c *gin.Context) {
		if os.Getenv("APP_ENV") != "dev" {
			c.JSON(403, gin.H{"error": "solo disponible en dev"})
			return
		}
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			c.JSON(401, gin.H{"error": "token requerido"})
			return
		}
		email, ok := login.GetEmailFromToken(token)
		if !ok {
			c.JSON(401, gin.H{"error": "sesión inválida"})
			return
		}
		u := migrations.GetUserByEmail(email)
		if u == nil {
			c.JSON(404, gin.H{"error": "usuario no encontrado"})
			return
		}
		var body struct {
			PlanID    int `json:"plan_id"`
			Frequency int `json:"frequency"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.PlanID == 0 {
			c.JSON(400, gin.H{"error": "plan_id requerido"})
			return
		}
		plan, err := subRepo.GetPlanByID(body.PlanID)
		if err != nil || plan == nil {
			c.JSON(404, gin.H{"error": "plan no encontrado"})
			return
		}
		s := &subscriptions.Subscription{UserID: u.ID, PlanID: plan.ID, StartDate: time.Now(), Frequency: body.Frequency}
		if err := subRepo.CreateSubscription(s); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok", "subscription_id": s.ID, "plan_id": plan.ID})
	})

	// Inspect active subscription quotas quickly
	r.GET("/me/quota", func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			c.JSON(401, gin.H{"error": "token requerido"})
			return
		}
		email, ok := login.GetEmailFromToken(token)
		if !ok {
			c.JSON(401, gin.H{"error": "sesión inválida"})
			return
		}
		u := migrations.GetUserByEmail(email)
		if u == nil {
			c.JSON(404, gin.H{"error": "usuario no encontrado"})
			return
		}
		sub, err := subRepo.GetActiveSubscription(u.ID)
		if err != nil || sub == nil {
			c.JSON(404, gin.H{"error": "suscripción no encontrada"})
			return
		}
		c.JSON(200, gin.H{
			"subscription_id":          sub.ID,
			"plan":                     sub.Plan.Name,
			"consultations_remaining":  sub.Consultations,
			"questionnaires_remaining": sub.Questionnaires,
			"clinical_cases_remaining": sub.ClinicalCases,
			"files_remaining":          sub.Files,
			"plan_limits": gin.H{
				"consultations":  sub.Plan.Consultations,
				"questionnaires": sub.Plan.Questionnaires,
				"clinical_cases": sub.Plan.ClinicalCases,
				"files":          sub.Plan.Files,
			},
		})
	})

	// Dev reset quotas to plan defaults
	r.POST("/dev/reset-quotas", func(c *gin.Context) {
		if os.Getenv("APP_ENV") != "dev" {
			c.JSON(403, gin.H{"error": "solo disponible en dev"})
			return
		}
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			c.JSON(401, gin.H{"error": "token requerido"})
			return
		}
		email, ok := login.GetEmailFromToken(token)
		if !ok {
			c.JSON(401, gin.H{"error": "sesión inválida"})
			return
		}
		u := migrations.GetUserByEmail(email)
		if u == nil {
			c.JSON(404, gin.H{"error": "usuario no encontrado"})
			return
		}
		sub, err := subRepo.GetActiveSubscription(u.ID)
		if err != nil || sub == nil {
			c.JSON(404, gin.H{"error": "suscripción no encontrada"})
			return
		}
		if err := subRepo.ResetSubscriptionQuotasToPlan(sub.ID); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		// Reload
		fresh, _ := subRepo.GetActiveSubscription(u.ID)
		c.JSON(200, gin.H{"status": "ok", "subscription_id": sub.ID, "plan": fresh.Plan.Name, "consultations": fresh.Consultations, "questionnaires": fresh.Questionnaires, "clinical_cases": fresh.ClinicalCases, "files": fresh.Files})
	})

	// Dev diagnostic: dump plans + subscriptions for a user and highlight anomalies
	r.GET("/dev/diagnose-subscriptions", func(c *gin.Context) {
		if os.Getenv("APP_ENV") != "dev" {
			c.JSON(403, gin.H{"error": "solo disponible en dev"})
			return
		}
		uidStr := c.Query("user_id")
		if uidStr == "" {
			c.JSON(400, gin.H{"error": "user_id requerido"})
			return
		}
		uid, err := strconv.Atoi(uidStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "user_id inválido"})
			return
		}
		// Load plans
		plansRows, err := db.Query(`SELECT id,name,consultations,questionnaires,clinical_cases,files,price FROM subscription_plans`)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer plansRows.Close()
		plans := []gin.H{}
		for plansRows.Next() {
			var id, c1, c2, c3, c4 int
			var name string
			var price float64
			if err := plansRows.Scan(&id, &name, &c1, &c2, &c3, &c4, &price); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			plans = append(plans, gin.H{"id": id, "name": name, "consultations": c1, "questionnaires": c2, "clinical_cases": c3, "files": c4, "price": price})
		}
		// Load all subscriptions for user
		sRows, err := db.Query(`SELECT id, plan_id, consultations, questionnaires, clinical_cases, files, start_date, end_date, frequency FROM subscriptions WHERE user_id=? ORDER BY id DESC`, uid)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer sRows.Close()
		subs := []gin.H{}
		for sRows.Next() {
			var id, planID, cons, ques, clin, files, freq int
			var start, endStr sql.NullTime
			var end interface{} = nil
			if err := sRows.Scan(&id, &planID, &cons, &ques, &clin, &files, &start, &endStr, &freq); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			if endStr.Valid {
				end = endStr.Time
			}
			subs = append(subs, gin.H{"id": id, "plan_id": planID, "consultations": cons, "questionnaires": ques, "clinical_cases": clin, "files": files, "start_date": start.Time, "end_date": end, "frequency": freq})
		}
		// Determine active (first entry in subs slice) and compare with plan
		anomalies := []string{}
		if len(subs) > 0 {
			active := subs[0]
			var planMatch gin.H
			for _, p := range plans {
				if p["id"] == active["plan_id"] {
					planMatch = p
					break
				}
			}
			if planMatch != nil {
				// Check if active quotas are all zero while plan has >0
				zeros := (active["consultations"].(int) == 0 && active["questionnaires"].(int) == 0 && active["clinical_cases"].(int) == 0)
				planHas := (planMatch["consultations"].(int) > 0 || planMatch["questionnaires"].(int) > 0 || planMatch["clinical_cases"].(int) > 0)
				if zeros && planHas {
					anomalies = append(anomalies, "active_subscription_zero_remaining_vs_plan_limits")
				}
				// Individual mismatch (remaining > plan limit) improbable, check
				if active["consultations"].(int) > planMatch["consultations"].(int) {
					anomalies = append(anomalies, "consultations_remaining_exceeds_plan_limit")
				}
			}
		}
		c.JSON(200, gin.H{"user_id": uid, "plans": plans, "subscriptions": subs, "anomalies": anomalies})
	})

	// Admin self-repair: if active subscription remaining quotas are all zero but plan limits > 0, reset them.
	r.POST("/admin/repair-quotas", func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			c.JSON(401, gin.H{"error": "token requerido"})
			return
		}
		email, ok := login.GetEmailFromToken(token)
		if !ok {
			c.JSON(401, gin.H{"error": "sesión inválida"})
			return
		}
		u := migrations.GetUserByEmail(email)
		if u == nil {
			c.JSON(404, gin.H{"error": "usuario no encontrado"})
			return
		}
		sub, err := subRepo.GetActiveSubscription(u.ID)
		if err != nil || sub == nil {
			c.JSON(404, gin.H{"error": "suscripción no encontrada"})
			return
		}
		zeros := sub.Consultations == 0 && sub.Questionnaires == 0 && sub.ClinicalCases == 0 && sub.Files > 0 // allow files>0
		planHas := sub.Plan.Consultations > 0 || sub.Plan.Questionnaires > 0 || sub.Plan.ClinicalCases > 0
		if !zeros || !planHas {
			c.JSON(200, gin.H{"status": "no_action", "consultations": sub.Consultations, "clinical_cases": sub.ClinicalCases})
			return
		}
		if err := subRepo.ResetSubscriptionQuotasToPlan(sub.ID); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		fresh, _ := subRepo.GetActiveSubscription(u.ID)
		c.JSON(200, gin.H{"status": "repaired", "subscription_id": fresh.ID, "plan": fresh.Plan.Name, "consultations": fresh.Consultations, "questionnaires": fresh.Questionnaires, "clinical_cases": fresh.ClinicalCases, "files": fresh.Files})
	})

	// Debug: listar todas las suscripciones (o del usuario actual) con cuotas y plan
	r.GET("/debug/subscriptions", func(c *gin.Context) {
		userParam := c.Query("user")
		uid := 0
		if userParam != "" {
			if v, err := strconv.Atoi(userParam); err == nil {
				uid = v
			}
		}
		subs, err := subRepo.GetSubscriptions(uid)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		// Determine active per user (highest id per user_id)
		latest := map[int]int{} // user_id -> max subscription id
		for _, s := range subs {
			if s.ID > latest[s.UserID] {
				latest[s.UserID] = s.ID
			}
		}
		out := []gin.H{}
		for _, s := range subs {
			plan := gin.H{}
			if s.Plan != nil {
				plan = gin.H{"id": s.Plan.ID, "name": s.Plan.Name, "consultations": s.Plan.Consultations, "questionnaires": s.Plan.Questionnaires, "clinical_cases": s.Plan.ClinicalCases, "files": s.Plan.Files, "price": s.Plan.Price}
			}
			out = append(out, gin.H{"id": s.ID, "user_id": s.UserID, "plan_id": s.PlanID, "plan": plan, "consultations": s.Consultations, "questionnaires": s.Questionnaires, "clinical_cases": s.ClinicalCases, "files": s.Files, "start_date": s.StartDate, "end_date": s.EndDate, "active": s.ID == latest[s.UserID]})
		}
		c.JSON(200, gin.H{"subscriptions": out})
	})

	// Debug: resync cuotas de la suscripción activa con los valores del plan
	r.POST("/debug/resync-quotas", func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			c.JSON(401, gin.H{"error": "token requerido"})
			return
		}
		email, ok := login.GetEmailFromToken(token)
		if !ok {
			c.JSON(401, gin.H{"error": "sesión inválida"})
			return
		}
		u := migrations.GetUserByEmail(email)
		if u == nil {
			c.JSON(404, gin.H{"error": "usuario no encontrado"})
			return
		}
		sub, err := subRepo.GetActiveSubscription(u.ID)
		if err != nil || sub == nil {
			c.JSON(404, gin.H{"error": "suscripción no encontrada"})
			return
		}
		plan, err := subRepo.GetPlanByID(sub.PlanID)
		if err != nil || plan == nil {
			c.JSON(404, gin.H{"error": "plan no encontrado"})
			return
		}
		// Set each field explicitly
		if err := subRepo.SetQuotaValue(sub.ID, "consultations", plan.Consultations); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if err := subRepo.SetQuotaValue(sub.ID, "questionnaires", plan.Questionnaires); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if err := subRepo.SetQuotaValue(sub.ID, "clinical_cases", plan.ClinicalCases); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if err := subRepo.SetQuotaValue(sub.ID, "files", plan.Files); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok", "subscription_id": sub.ID, "plan_id": plan.ID, "consultations": plan.Consultations, "questionnaires": plan.Questionnaires, "clinical_cases": plan.ClinicalCases, "files": plan.Files})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Servidor HTTP personalizado con timeouts generosos para streaming/RAG
	// WriteTimeout debe ser >90s para permitir OpenAI runs largos con file_search
	// AUMENTADO a 5 min debido a latencias extremas de OpenAI API (180s+ para crear runs)
	srv := &http.Server{
		Addr:           ":" + port,
		Handler:        r,
		ReadTimeout:    30 * time.Second,  // Suficiente para subir archivos grandes
		WriteTimeout:   300 * time.Second, // 5 minutos para tolerar latencias extremas de OpenAI
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	log.Printf("[server] Starting HTTP server on port %s with WriteTimeout=%v", port, srv.WriteTimeout)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}

}
