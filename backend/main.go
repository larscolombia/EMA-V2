package main

import (
	"log"
	"net/http"
	"os"

	"ema-backend/chat"
	"ema-backend/conn"
	"ema-backend/countries"
	"ema-backend/login"
	"ema-backend/migrations"
	"ema-backend/openai"
	"ema-backend/profile"
	"ema-backend/subscriptions"

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

	// Chat/OpenAI endpoints (optional if keys provided)
	ai := openai.NewClient()
	chatHandler := chat.NewHandler(ai)
	r.POST("/asistente/start", chatHandler.Start)
	r.POST("/asistente/message", chatHandler.Message)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}

}
