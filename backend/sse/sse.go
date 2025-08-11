package sse

import (
	"io"

	"github.com/gin-gonic/gin"
)

func Stream(c *gin.Context, ch <-chan string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Stream(func(w io.Writer) bool {
		if msg, ok := <-ch; ok {
			c.SSEvent("message", msg)
			return true
		}
		c.SSEvent("message", "[DONE]")
		return false
	})
}
