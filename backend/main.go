package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"ema-backend/casos_clinico"
	"ema-backend/casos_interactivos"
	"ema-backend/categories"
	"ema-backend/chat"
	"ema-backend/conn"
	"ema-backend/countries"
	"ema-backend/login"
	"ema-backend/marketing"
	"ema-backend/migrations"
	"ema-backend/openai"
	"ema-backend/profile"
	"ema-backend/subscriptions"
	"ema-backend/testsapi"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
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

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth routes expected by Flutter
	r.POST("/login", login.Handler)
	r.GET("/session", login.SessionHandler)
	r.POST("/logout", login.LogoutHandler)
	r.POST("/register", login.RegisterHandler)
	r.POST("/password/forgot", login.ForgotPasswordHandler)
	r.POST("/password/change", login.ChangePasswordHandler)

	// Profile routes and static media
	profile.RegisterRoutes(r)
	r.Static("/media", "./media")

	// Subscriptions routes
	subRepo := subscriptions.NewRepository(db)
	subHandler := subscriptions.NewHandler(subRepo)
	subHandler.RegisterRoutes(r)

	// Countries route (simple list)
	countries.RegisterRoutes(r)

	// Medical categories
	catRepo := categories.NewRepository(db)
	catHandler := categories.NewHandler(catRepo)
	catHandler.RegisterRoutes(r)

	// Chat/OpenAI endpoints (optional if keys provided)
	openai.SetPersistDB(db)
	ai := openai.NewClient()
	chatHandler := chat.NewHandler(ai)
	r.POST("/asistente/start", chatHandler.Start)
	r.POST("/asistente/message", chatHandler.Message)
	// Cleanup endpoint to delete OpenAI artifacts for a thread
	r.POST("/asistente/delete", chatHandler.Delete)

	// Tests (quizzes) endpoints for Flutter
	testsHandler := testsapi.DefaultHandler()
	// Inject category name resolver for better prompts when user selects a category
	testsHandler.SetCategoryResolver(func(ctx context.Context, ids []int) ([]string, error) {
		return catRepo.NamesByIDs(ctx, ids)
	})
	testsHandler.RegisterRoutes(r)

	// Clinical cases endpoints (analytical & interactive)
	clinicalHandler := casos_clinico.DefaultHandler()
	clinicalHandler.RegisterRoutes(r)

	// New interactive flow with stricter JSON turn contract
	interactivosHandler := casos_interactivos.DefaultHandler()
	interactivosHandler.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}

}
