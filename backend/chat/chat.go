package chat

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	// Prefer creating a real Assistants thread if an assistant is configured
	threadID := ""
	if h.AI.AssistantID != "" && len(h.AI.AssistantID) >= 5 && h.AI.AssistantID[:5] == "asst_" {
		if tid, err := h.AI.CreateThread(c); err == nil {
			threadID = tid
		}
	}
	if threadID == "" {
		// Fallback to a client-side UUID if Assistants thread can't be created
		threadID = uuid.NewString()
	}
	var initial strings.Builder
	_ = req
	c.JSON(http.StatusOK, gin.H{"thread_id": threadID, "text": initial.String()})
}

func (h *Handler) Message(c *gin.Context) {
	ct := c.GetHeader("Content-Type")

	// Handle multipart/form-data (PDF upload + prompt + thread_id), matching frontend FormData
	if strings.HasPrefix(ct, "multipart/form-data") {
		start := time.Now()
		// file is optional
		upFile, _ := c.FormFile("file") // ignore error; file is optional
		prompt := c.PostForm("prompt")
		// threadID := c.PostForm("thread_id") // currently unused by backend

		// If it's an audio file, try to transcribe and append to the prompt.
		if upFile != nil {
			ext := strings.ToLower(filepath.Ext(upFile.Filename))
			if ext == ".mp3" || ext == ".wav" || ext == ".m4a" || ext == ".aac" || ext == ".flac" || ext == ".ogg" || ext == ".webm" {
				// Save to a temp path; Gin provides a way to save uploaded file
				tmpDir := "./tmp"
				_ = os.MkdirAll(tmpDir, 0o755)
				tmp := filepath.Join(tmpDir, uuid.NewString()+ext)
				// Save uploaded file
				_ = c.SaveUploadedFile(upFile, tmp)
				if text, err := h.AI.TranscribeFile(c, tmp); err == nil && strings.TrimSpace(text) != "" {
					if strings.TrimSpace(prompt) != "" {
						prompt = prompt + "\n\n[Transcripción de audio]:\n" + text
					} else {
						prompt = text
					}
				}
				// best-effort: we won't delete temp here; in real app use os.Remove after use
			} else if ext == ".pdf" {
				// Validations: extension and basic size
				if upFile.Size <= 0 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "archivo vacío"})
					return
				}
				// Limits by env
				maxFiles := 0
				maxMB := 0
				if v := os.Getenv("VS_MAX_FILES"); v != "" {
					if n, err := strconv.Atoi(v); err == nil {
						maxFiles = n
					}
				}
				if v := os.Getenv("VS_MAX_MB"); v != "" {
					if n, err := strconv.Atoi(v); err == nil {
						maxMB = n
					}
				}
				formThreadID := c.PostForm("thread_id")
				if h.AI.AssistantID != "" && strings.HasPrefix(formThreadID, "thread_") {
					// Check counts
					filesCount := h.AI.CountThreadFiles(formThreadID)
					if maxFiles > 0 && filesCount >= maxFiles {
						c.JSON(http.StatusBadRequest, gin.H{"error": "límite de archivos alcanzado"})
						return
					}
					currBytes := h.AI.GetSessionBytes(formThreadID)
					if maxMB > 0 {
						nextMB := (currBytes + upFile.Size) / (1024 * 1024)
						if int(nextMB) > maxMB {
							c.JSON(http.StatusBadRequest, gin.H{"error": "límite de tamaño por sesión superado"})
							return
						}
					}
				}
				// New flow: rely on vector store ingestion
				tmpDir := "./tmp"
				_ = os.MkdirAll(tmpDir, 0o755)
				tmp := filepath.Join(tmpDir, uuid.NewString()+ext)
				_ = c.SaveUploadedFile(upFile, tmp)
				if h.AI.AssistantID != "" && strings.HasPrefix(formThreadID, "thread_") {
					vsID, err := h.AI.EnsureVectorStore(c, formThreadID)
					if err != nil {
						log.Printf("ERROR EnsureVectorStore: %v", err)
						// Fallback: responder sin RAG (sin documento)
						fb := strings.TrimSpace(prompt)
						if fb == "" {
							fb = "Responde al usuario. Nota: no se pudo procesar el documento adjunto."
						} else {
							fb = fb + "\n\nNota: no se pudo procesar el documento adjunto."
						}
						stream, ferr := h.AI.StreamMessage(c, fb)
						if ferr != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": ferr.Error()})
							return
						}
						sse.Stream(c, stream)
						return
					}
					log.Printf("DEBUG: Successfully created/found vector store: %s", vsID)

					fileID, err := h.AI.UploadAssistantFile(c, formThreadID, tmp)
					if err != nil {
						log.Printf("ERROR UploadAssistantFile: %v", err)
						// Fallback: responder sin RAG (sin documento)
						fb := strings.TrimSpace(prompt)
						if fb == "" {
							fb = "Responde al usuario. Nota: no se pudo procesar el documento adjunto."
						} else {
							fb = fb + "\n\nNota: no se pudo procesar el documento adjunto."
						}
						stream, ferr := h.AI.StreamMessage(c, fb)
						if ferr != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": ferr.Error()})
							return
						}
						sse.Stream(c, stream)
						return
					}
					log.Printf("DEBUG: Successfully uploaded file: %s", fileID)
					// short wait; if still processing -> 202. Allow override via VS_POLL_SEC for tests.
					pStart := time.Now()
					pollSec := 8
					if v := os.Getenv("VS_POLL_SEC"); v != "" {
						if n, err := strconv.Atoi(v); err == nil && n >= 0 {
							pollSec = n
						}
					}
					if err := h.AI.PollFileProcessed(c, fileID, time.Duration(pollSec)*time.Second); err != nil {
						log.Printf("{\"event\":\"file.processing\",\"thread\":%q,\"vs\":%q,\"file_id\":%q,\"status\":\"processing\",\"elapsed_ms\":%d}", formThreadID, vsID, fileID, time.Since(pStart).Milliseconds())
						// No 500: informar procesamiento en curso
						c.JSON(http.StatusAccepted, gin.H{"status": "processing", "message": "archivo en procesamiento, intenta en 1–2 min", "file_id": fileID})
						return
					}
					// processed: add to vector store and proceed to run
					if err := h.AI.AddFileToVectorStore(c, vsID, fileID); err != nil {
						log.Printf("ERROR AddFileToVectorStore: %v", err)
						// Fallback: responder sin RAG (sin documento)
						fb := strings.TrimSpace(prompt)
						if fb == "" {
							fb = "Responde al usuario. Nota: no se pudo procesar el documento adjunto."
						} else {
							fb = fb + "\n\nNota: no se pudo procesar el documento adjunto."
						}
						stream, ferr := h.AI.StreamMessage(c, fb)
						if ferr != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": ferr.Error()})
							return
						}
						sse.Stream(c, stream)
						return
					}
					log.Printf("DEBUG: Successfully added file %s to vector store %s", fileID, vsID)
					// increase session bytes
					h.AI.AddSessionBytes(formThreadID, upFile.Size)
					base := strings.TrimSpace(prompt)
					if base == "" {
						// Hint to assistant: focus on the most recently uploaded file for this thread
						base = "Por favor, resume el documento adjunto más reciente (archivo: " + filepath.Base(upFile.Filename) + ")."
					}
					log.Printf("{\"event\":\"run.start\",\"thread\":%q,\"vs\":%q,\"file_id\":%q}", formThreadID, vsID, fileID)
					stream, err := h.AI.StreamAssistantMessage(c, formThreadID, base)
					if err != nil {
						// Fallback: responder sin RAG (sin documento)
						fb := base + "\n\nNota: no se pudo usar el documento adjunto."
						fbs, ferr := h.AI.StreamMessage(c, fb)
						if ferr != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": ferr.Error()})
							return
						}
						c.Header("X-Assistant-Start-Ms", time.Since(start).String())
						sse.Stream(c, fbs)
						return
					}
					c.Header("X-Assistant-Start-Ms", time.Since(start).String())
					sse.Stream(c, stream)
					return
				}
			} else {
				// Other files: keep prompt as-is for now
			}
		}

		// If Assistants is configured and thread_id provided, use Assistants flow
		formThreadID := c.PostForm("thread_id")
		if h.AI.AssistantID != "" && len(h.AI.AssistantID) >= 5 && h.AI.AssistantID[:5] == "asst_" && strings.HasPrefix(formThreadID, "thread_") {
			// allow client thread id but we create our own thread on /start; we just send the prompt here
			stream, err := h.AI.StreamAssistantMessage(c, formThreadID, prompt)
			if err != nil {
				// Fallback: responder sin Assistants si falla
				log.Printf("ERROR StreamAssistantMessage (multipart): %v", err)
				fbStream, ferr := h.AI.StreamMessage(c, prompt)
				if ferr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": ferr.Error()})
					return
				}
				c.Header("X-Assistant-Start-Ms", time.Since(start).String())
				sse.Stream(c, fbStream)
				return
			}
			c.Header("X-Assistant-Start-Ms", time.Since(start).String())
			sse.Stream(c, stream)
			return
		}
		stream, err := h.AI.StreamMessage(c, prompt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Add simple timing header for client/testing
		c.Header("X-Assistant-Start-Ms", time.Since(start).String())
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
	start := time.Now()
	// Use Assistants flow when configured
	if h.AI.AssistantID != "" && len(h.AI.AssistantID) >= 5 && h.AI.AssistantID[:5] == "asst_" && strings.HasPrefix(req.ThreadID, "thread_") {
		stream, err := h.AI.StreamAssistantMessage(c, req.ThreadID, req.Prompt)
		if err != nil {
			// Fallback: responder sin Assistants si falla
			log.Printf("ERROR StreamAssistantMessage (json): %v", err)
			fbStream, ferr := h.AI.StreamMessage(c, req.Prompt)
			if ferr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": ferr.Error()})
				return
			}
			c.Header("X-Assistant-Start-Ms", time.Since(start).String())
			sse.Stream(c, fbStream)
			return
		}
		c.Header("X-Assistant-Start-Ms", time.Since(start).String())
		sse.Stream(c, stream)
		return
	}
	stream, err := h.AI.StreamMessage(c, req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("X-Assistant-Start-Ms", time.Since(start).String())
	sse.Stream(c, stream)
}

// Delete removes OpenAI artifacts for a given thread (files, vector store, thread) and returns 204.
func (h *Handler) Delete(c *gin.Context) {
	var req struct {
		ThreadID string `json:"thread_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.ThreadID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "thread_id requerido"})
		return
	}
	// Best-effort cleanup; even if some resources are missing, we aim to clear local state
	_ = h.AI.DeleteThreadArtifacts(c, req.ThreadID)
	c.Status(http.StatusNoContent)
}
