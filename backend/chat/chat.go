package chat

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ema-backend/sse"
)

type Handler struct {
	AI AIClient
	quotaValidator func(ctx context.Context, c *gin.Context, flow string) error
	mu sync.Mutex
	// maps client provided thread IDs (any format) to OpenAI assistant thread IDs (thread_*)
	threadMap map[string]string
	lastPrompt map[string]string
	// when true, only real assistant thread ids (thread_*) are accepted for /message endpoints.
	// a start request must successfully create an assistant thread; no local UUID fallbacks.
	strictThreads bool
}

func NewHandler(ai AIClient) *Handler {
	strict := os.Getenv("CHAT_STRICT_THREADS") == "1"
	return &Handler{AI: ai, threadMap: make(map[string]string), lastPrompt: make(map[string]string), strictThreads: strict}
}

// SetQuotaValidator allows injection from main
func (h *Handler) SetQuotaValidator(fn func(ctx context.Context, c *gin.Context, flow string) error) { h.quotaValidator = fn }

func (h *Handler) Start(c *gin.Context) {
	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Even if binding fails, proceed with a new thread to be tolerant like Laravel version
		req.Prompt = ""
	}
	// Prefer creating a real Assistants thread if an assistant is configured; expose assistant thread id directly
	threadID := ""
	if h.AI.GetAssistantID() != "" && len(h.AI.GetAssistantID()) >= 5 && strings.HasPrefix(h.AI.GetAssistantID(), "asst_") {
		if tid, err := h.AI.CreateThread(c.Request.Context()); err == nil {
			threadID = tid
		} else if h.strictThreads {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no se pudo crear thread assistant", "detail": err.Error()})
			return
		}
	}
	if threadID == "" { // fallback local id (will map later on first /message if needed)
		if h.strictThreads {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "assistant no disponible"})
			return
		}
		threadID = uuid.NewString()
	}
	var initial strings.Builder
	_ = req
	resp := gin.H{"thread_id": threadID, "text": initial.String()}
	if h.strictThreads { resp["strict_threads"] = true }
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Message(c *gin.Context) {
	// Quota check (counts every message, including uploads) mapped to consultations bucket
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "chat_message"); err != nil {
			field, _ := c.Get("quota_error_field")
			reason, _ := c.Get("quota_error_reason")
			resp := gin.H{"error": "chat quota exceeded"}
			if f, ok := field.(string); ok && f != "" { resp["field"] = f }
			if r, ok := reason.(string); ok && r != "" { resp["reason"] = r }
			c.JSON(http.StatusForbidden, resp)
			return
		}
		if v, ok := c.Get("quota_remaining"); ok {
			// Set generic headers early (may be overridden/refined later for file uploads)
			c.Header("X-Quota-Field", "consultations")
			c.Header("X-Quota-Remaining", toString(v))
		}
	}
	ct := c.GetHeader("Content-Type")

	// Handle multipart/form-data (PDF upload + prompt + thread_id), matching frontend FormData
	if strings.HasPrefix(ct, "multipart/form-data") {
		start := time.Now()
		// file is optional
		upFile, _ := c.FormFile("file") // ignore error; file is optional
		prompt := c.PostForm("prompt")
		clientThreadID := c.PostForm("thread_id")
		if h.strictThreads && (clientThreadID == "" || !strings.HasPrefix(clientThreadID, "thread_")) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "thread_id inválido (usa /asistente/start)"})
			return
		}
		resolvedThreadID := h.resolveAssistantThread(c, clientThreadID)

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
				formThreadID := resolvedThreadID
				if h.AI.GetAssistantID() != "" && strings.HasPrefix(formThreadID, "thread_") {
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
				if h.AI.GetAssistantID() != "" && strings.HasPrefix(formThreadID, "thread_") {
					vsID, err := h.AI.EnsureVectorStore(c.Request.Context(), formThreadID)
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

					fileID, err := h.AI.UploadAssistantFile(c.Request.Context(), formThreadID, tmp)
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
					if err := h.AI.PollFileProcessed(c.Request.Context(), fileID, time.Duration(pollSec)*time.Second); err != nil {
						log.Printf("{\"event\":\"file.processing\",\"thread\":%q,\"vs\":%q,\"file_id\":%q,\"status\":\"processing\",\"elapsed_ms\":%d}", formThreadID, vsID, fileID, time.Since(pStart).Milliseconds())
						// No 500: informar procesamiento en curso
						c.JSON(http.StatusAccepted, gin.H{"status": "processing", "message": "archivo en procesamiento, intenta en 1–2 min", "file_id": fileID})
						return
					}
					// processed: add to vector store and proceed to run
					if err := h.AI.AddFileToVectorStore(c.Request.Context(), vsID, fileID); err != nil {
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
					// Quota: count this successful PDF ingestion as a file_upload usage if validator present
					if h.quotaValidator != nil {
						if err := h.quotaValidator(c.Request.Context(), c, "file_upload"); err != nil {
							// If quota exhausted for files, abort before running assistant
							field, _ := c.Get("quota_error_field")
						reason, _ := c.Get("quota_error_reason")
						resp := gin.H{"error": "file quota exceeded"}
						if f, ok := field.(string); ok && f != "" { resp["field"] = f }
						if r, ok := reason.(string); ok && r != "" { resp["reason"] = r }
						c.JSON(http.StatusForbidden, resp)
						return
						}
						if v, ok := c.Get("quota_remaining"); ok {
							c.Header("X-Quota-Files-Remaining", toString(v)) // legacy
							c.Header("X-Quota-Field", "files")
							c.Header("X-Quota-Remaining", toString(v))
						}
					}
					// increase session bytes
					h.AI.AddSessionBytes(formThreadID, upFile.Size)
					base := strings.TrimSpace(prompt)
					if base == "" {
						fname := filepath.Base(upFile.Filename)
						structured := os.Getenv("STRUCTURED_PDF_SUMMARY") == "1"
						if structured {
							// Prompt estructurado (activable) para informes ejecutivos normalizados
							base = "Elabora un resumen estructurado y conciso del documento adjunto (archivo: " + fname + "). Produce EXACTAMENTE las secciones numeradas en español a continuación, empezando DIRECTAMENTE por '1. Resumen Ejecutivo:' sin frases introductorias (no uses 'Claro', 'Aquí tienes', etc.) ni explicaciones sobre el proceso. Si un punto no está presente en el documento responde 'No especificado' únicamente para ese punto. No inventes información.\n" +
								"1. Resumen Ejecutivo (3-4 líneas).\n" +
								"2. Objetivo o Propósito.\n" +
								"3. Alcance y Componentes Clave.\n" +
								"4. Propuesta de Valor / Diferenciadores.\n" +
								"5. Entregables y Cronograma (si se mencionan).\n" +
								"6. Modelo Comercial / Costos (si aparecen).\n" +
								"7. Riesgos, Supuestos o Limitaciones.\n" +
								"8. Próximos Pasos sugeridos.\n" +
								"Al final agrega una sección 'Recomendación Breve' (1-2 líneas) y nada más. No repitas el nombre del archivo fuera de la primera línea."
						} else {
							// Prompt genérico: permite análisis amplio para otros tipos de documentos
							base = "Analiza el documento adjunto (" + fname + ") y genera: 1) Un resumen ejecutivo de 4-6 líneas. 2) Una lista de los temas o secciones principales detectadas. 3) Hallazgos o insights clave. 4) Cualquier métrica, cifra o dato cuantitativo relevante. 5) Riesgos o limitaciones si se infieren del texto. 6) Recomendaciones accionables (máx 3). No inventes datos; si algo no existe omite el apartado correspondiente en lugar de rellenarlo. No uses frases introductorias como 'Claro'." 
						}
						// Instrucción global inicial (si configurada) que aplica a TODOS los documentos sin importar tipo.
						if pre := strings.TrimSpace(os.Getenv("DOC_SUMMARY_PREAMBLE")); pre != "" {
							base = pre + "\n\n" + base
						}
					}
					log.Printf("{\"event\":\"run.start\",\"thread\":%q,\"vs\":%q,\"file_id\":%q}", formThreadID, vsID, fileID)
						stream, err := h.AI.StreamAssistantMessage(c.Request.Context(), formThreadID, base)
					if err != nil {
						// Fallback: responder sin RAG (sin documento)
						fb := base + "\n\nNota: no se pudo usar el documento adjunto."
						fbs, ferr := h.AI.StreamMessage(c.Request.Context(), fb)
						if ferr != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": ferr.Error()})
							return
						}
						c.Header("X-Assistant-Start-Ms", time.Since(start).String())
						sse.Stream(c, fbs)
						return
					}
						c.Header("X-Assistant-Start-Ms", time.Since(start).String())
					c.Header("X-RAG", "1")
					c.Header("X-RAG-File", filepath.Base(upFile.Filename))
					c.Header("X-RAG-Prompt", "structured-v1")
						if os.Getenv("TEST_CAPTURE_FULL") == "1" {
							// En modo test, interceptamos stream y añadimos token final con contenido completo
							buf := &strings.Builder{}
							proxy := make(chan string)
							go func(){
								for tok := range stream { buf.WriteString(tok); proxy <- tok }
								close(proxy)
							}()
							// Enviar tokens interceptados
							sse.Stream(c, proxy)
							// Agregar línea final FULL (no estándar SSE pero útil en test)
							c.Writer.Write([]byte("data: __FULL__ "+buf.String()+"\n\n"))
							return
						}
						sse.Stream(c, stream)
					return
				}
			} else {
				// Other files: keep prompt as-is for now
			}
		}

		// If Assistants is configured and thread_id provided, use Assistants flow
	formThreadID := h.resolveAssistantThread(c, c.PostForm("thread_id"))
	if h.AI.GetAssistantID() != "" && len(h.AI.GetAssistantID()) >= 5 && strings.HasPrefix(h.AI.GetAssistantID(), "asst_") && strings.HasPrefix(formThreadID, "thread_") {
			// allow client thread id but we create our own thread on /start; we just send the prompt here
			stream, err := h.AI.StreamAssistantMessage(c.Request.Context(), formThreadID, prompt)
			if err != nil {
				// Fallback: responder sin Assistants si falla
				log.Printf("ERROR StreamAssistantMessage (multipart): %v", err)
				fbStream, ferr := h.AI.StreamMessage(c.Request.Context(), prompt)
				if ferr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": ferr.Error()})
					return
				}
				c.Header("X-Assistant-Start-Ms", time.Since(start).String())
				sse.Stream(c, fbStream)
				return
			}
			c.Header("X-Assistant-Start-Ms", time.Since(start).String())
			if v, ok := c.Get("quota_remaining"); ok { c.Header("X-Quota-Remaining", toString(v)) }
			c.Header("X-Thread-ID", formThreadID)
			if h.strictThreads { c.Header("X-Strict-Threads", "1") }
			sse.Stream(c, stream)
			return
		}
		stream, err := h.AI.StreamMessage(c.Request.Context(), prompt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Add simple timing header for client/testing
		c.Header("X-Assistant-Start-Ms", time.Since(start).String())
		if v, ok := c.Get("quota_remaining"); ok { c.Header("X-Quota-Remaining", toString(v)) }
		c.Header("X-Thread-ID", formThreadID)
		if h.strictThreads { c.Header("X-Strict-Threads", "1") }
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
	if h.strictThreads && (req.ThreadID == "" || !strings.HasPrefix(req.ThreadID, "thread_")) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "thread_id inválido (usa /asistente/start)"})
		return
	}
	// Use Assistants flow when configured
	resolved := h.resolveAssistantThread(c, req.ThreadID)
	if h.AI.GetAssistantID() != "" && len(h.AI.GetAssistantID()) >= 5 && strings.HasPrefix(h.AI.GetAssistantID(), "asst_") && strings.HasPrefix(resolved, "thread_") {
		stream, err := h.AI.StreamAssistantMessage(c.Request.Context(), resolved, req.Prompt)
		if err != nil {
			// Fallback: responder sin Assistants si falla
			log.Printf("ERROR StreamAssistantMessage (json): %v", err)
			fbStream, ferr := h.AI.StreamMessage(c.Request.Context(), req.Prompt)
			if ferr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": ferr.Error()})
				return
			}
			c.Header("X-Assistant-Start-Ms", time.Since(start).String())
			if h.strictThreads { c.Header("X-Strict-Threads", "1") }
			sse.Stream(c, fbStream)
			return
		}
		c.Header("X-Assistant-Start-Ms", time.Since(start).String())
		if v, ok := c.Get("quota_remaining"); ok { c.Header("X-Quota-Remaining", toString(v)) }
		c.Header("X-Thread-ID", resolved)
		if h.strictThreads { c.Header("X-Strict-Threads", "1") }
		sse.Stream(c, stream)
		return
	}
	stream, err := h.AI.StreamMessage(c.Request.Context(), req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("X-Assistant-Start-Ms", time.Since(start).String())
	if v, ok := c.Get("quota_remaining"); ok { c.Header("X-Quota-Remaining", toString(v)) }
	c.Header("X-Thread-ID", resolved)
	if h.strictThreads { c.Header("X-Strict-Threads", "1") }
	sse.Stream(c, stream)
}

// VectorReset: fuerza un nuevo vector store limpio para el hilo dado.
func (h *Handler) VectorReset(c *gin.Context) {
	var req struct { ThreadID string `json:"thread_id"` }
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.ThreadID)=="" { c.JSON(http.StatusBadRequest, gin.H{"error":"thread_id requerido"}); return }
	if !strings.HasPrefix(req.ThreadID, "thread_") { c.JSON(http.StatusBadRequest, gin.H{"error":"thread_id inválido"}); return }
	vsID, err := h.AI.ForceNewVectorStore(c.Request.Context(), req.ThreadID)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"status":"reset","vector_store_id":vsID})
}

// VectorFiles: lista archivos asociados al vector store del hilo.
func (h *Handler) VectorFiles(c *gin.Context) {
	threadID := strings.TrimSpace(c.Query("thread_id"))
	if threadID=="" { c.JSON(http.StatusBadRequest, gin.H{"error":"thread_id requerido"}); return }
	if !strings.HasPrefix(threadID, "thread_") { c.JSON(http.StatusBadRequest, gin.H{"error":"thread_id inválido"}); return }
	files, err := h.AI.ListVectorStoreFiles(c.Request.Context(), threadID)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":err.Error()}); return }
	vsID := h.AI.GetVectorStoreID(threadID)
	c.JSON(http.StatusOK, gin.H{"thread_id":threadID, "vector_store_id":vsID, "files":files})
}

// helper to stringify interface
func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t,10)
	default:
		return ""
	}
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
	_ = h.AI.DeleteThreadArtifacts(c.Request.Context(), req.ThreadID)
	c.Status(http.StatusNoContent)
}

// resolveAssistantThread ensures we have a usable OpenAI assistant thread id when Assistants flow is available.
// If the client supplies an id already starting with thread_ it is used directly.
// Otherwise we map the client id to a freshly created assistant thread (once) and reuse it.
func (h *Handler) resolveAssistantThread(ctx context.Context, clientID string) string {
	if strings.TrimSpace(clientID) == "" {
		clientID = uuid.NewString()
	}
	if h.AI == nil || h.AI.GetAssistantID() == "" || len(h.AI.GetAssistantID()) < 5 || !strings.HasPrefix(h.AI.GetAssistantID(), "asst_") {
		return clientID // no assistants flow configured
	}
	if strings.HasPrefix(clientID, "thread_") {
		return clientID
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if mapped, ok := h.threadMap[clientID]; ok {
		return mapped
	}
	tid, err := h.AI.CreateThread(ctx)
	if err != nil || tid == "" {
		return clientID // fallback: return client id (will trigger non-assistants path)
	}
	h.threadMap[clientID] = tid
	log.Printf("[chat] mapped client_thread=%s -> assistant_thread=%s", clientID, tid)
	return tid
}
