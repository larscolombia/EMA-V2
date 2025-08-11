package main

import (
        "ema-backend/chat"
        "ema-backend/conn"
        "ema-backend/login"
        "ema-backend/migrations"
        "ema-backend/openai"
        "ema-backend/subscriptions"
        "github.com/gin-gonic/gin"
)

func main() {
        r := gin.Default()

        db, err := conn.NewMySQL()
        if err != nil {
                panic(err)
        }
        if err := migrations.Run(db); err != nil {
                panic(err)
        }
        subRepo := subscriptions.NewRepository(db)
        subHandler := subscriptions.NewHandler(subRepo)
        subHandler.RegisterRoutes(r)

        r.POST("/login", login.Handler)
        r.GET("/session", login.SessionHandler)

        ai := openai.NewClient()
        chatHandler := chat.NewHandler(ai)
        r.POST("/asistente/start", chatHandler.Start)
        r.POST("/asistente/message", chatHandler.Message)

        r.Run(":8080")
}

