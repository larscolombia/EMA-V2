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
		// Even if binding fails, proceed with a new thread to be tolerant like Laravel version
		req.Prompt = ""
	}
	// Do not call AI here to avoid first-request failures; just create a thread
	// Frontend will send the real user message via /asistente/message using this thread_id
	threadID := uuid.NewString()
	// Optional small greeting; keep empty to avoid UI assumptions
	var initial strings.Builder
	_ = req // keep for potential future logic
	c.JSON(http.StatusOK, gin.H{"thread_id": threadID, "text": initial.String()})
}

func (h *Handler) Message(c *gin.Context) {
	ct := c.GetHeader("Content-Type")

	// Handle multipart/form-data (PDF upload + prompt + thread_id), matching frontend FormData
	if strings.HasPrefix(ct, "multipart/form-data") {
		// file is optional; we can accept and ignore or process later
		_, _ = c.FormFile("file") // ignore error; file is optional
		prompt := c.PostForm("prompt")
		// threadID := c.PostForm("thread_id") // currently unused by backend

		stream, err := h.AI.StreamMessage(c, prompt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		sse.Stream(c, stream)
		return
	}

	// Default JSON body path
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
