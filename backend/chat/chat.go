package chat

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ema-backend/openai"
	"ema-backend/sse"
)

type Handler struct {
	AI *openai.Client
}

func NewHandler(ai *openai.Client) *Handler {
	return &Handler{AI: ai}
}

func (h *Handler) Start(c *gin.Context) {
	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt requerido"})
		return
	}
	ch, err := h.AI.StreamMessage(c, req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var sb strings.Builder
	for token := range ch {
		sb.WriteString(token)
	}
	c.JSON(http.StatusOK, gin.H{"thread_id": uuid.NewString(), "text": sb.String()})
}

func (h *Handler) Message(c *gin.Context) {
	var req struct {
		ThreadID string `json:"thread_id"`
		Prompt   string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parámetros inválidos"})
		return
	}
	stream, err := h.AI.StreamMessage(c, req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sse.Stream(c, stream)
}
