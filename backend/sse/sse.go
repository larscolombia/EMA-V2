package sse

import (
	"net/http"
	"strings"

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
		// Soportar mensajes multi-línea: cada línea debe ir precedida por 'data: '
		// para que el cliente SSE no pierda contenido entre saltos de línea.
		// Además preservamos los '\n' originales añadiéndolos dentro del token excepto en la última línea.
		lines := strings.Split(msg, "\n")
		for i, line := range lines {
			token := line
			if i < len(lines)-1 { // reinyectar salto de línea perdido por Split
				token += "\n"
			}
			_, _ = c.Writer.Write([]byte("data: " + token + "\n"))
		}
		// Termina el evento con una línea en blanco
		_, _ = c.Writer.Write([]byte("\n"))
		flusher.Flush()
	}
	// Write done marker
	_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}
