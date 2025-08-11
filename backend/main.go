package main

import (
	"ema-backend/chat"
	"ema-backend/login"
	"ema-backend/openai"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.POST("/login", login.Handler)
	r.GET("/session", login.SessionHandler)

	ai := openai.NewClient()
	chatHandler := chat.NewHandler(ai)
	r.POST("/asistente/start", chatHandler.Start)
	r.POST("/asistente/message", chatHandler.Message)

	r.Run(":8080")
}
