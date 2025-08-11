package main

import (
	"ema-backend/login"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.POST("/login", login.Handler)

	r.Run(":8080")
}
