package main

import (
	"ema-backend/login"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.POST("/login", login.Handler)

	r.GET("/session", login.SessionHandler)


	r.Run(":8080")
}
