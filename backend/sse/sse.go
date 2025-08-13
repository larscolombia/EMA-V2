package sse

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Stream writes raw SSE lines in the form:
//
//	data: <token>\n\n
//
// and finishes with:
//
//	data: [DONE]\n\n
//
// This matches the frontend's simple 'data:' line parsing.
func Stream(c *gin.Context, ch <-chan string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	for msg := range ch {
		// Write raw token line
		_, _ = c.Writer.Write([]byte("data: " + msg + "\n\n"))
		flusher.Flush()
	}
	// Write done marker
	_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}
