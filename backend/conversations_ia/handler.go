package conversations_ia

// Nuevo paquete para migrar SOLO el chat principal a la nueva lógica (Assistants v2 estricta)
// Mantiene endpoints separados para que el frontend pueda apuntar aquí sin tocar el paquete chat existente.
// Características iniciales:
// - /conversations/start: siempre crea thread assistant real (error si falla)
// - /conversations/message: acepta JSON {thread_id,prompt} o multipart (file opcional + prompt + thread_id)
// - Solo flujo Assistants; NO fallback a Chat Completions; si falla retorna error controlado.
// - Reutiliza openai.Client existente (AIClient-like métodos). No duplicamos lógica de vector store, solo la usamos.
// - Soporta PDF igual que el chat original (ingestión y RAG) para no romper UX del frontend.

import (
    "context"
    "mime/multipart"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
)

// Interface mínima que necesitamos (coincide con subset de chat.AIClient)
type AIClient interface {
    GetAssistantID() string
    CreateThread(ctx context.Context) (string, error)
    StreamAssistantMessage(ctx context.Context, threadID, prompt string) (<-chan string, error)
    EnsureVectorStore(ctx context.Context, threadID string) (string, error)
    UploadAssistantFile(ctx context.Context, threadID, filePath string) (string, error)
    PollFileProcessed(ctx context.Context, fileID string, timeout time.Duration) error
    AddFileToVectorStore(ctx context.Context, vsID, fileID string) error
    AddSessionBytes(threadID string, delta int64)
    CountThreadFiles(threadID string) int
    GetSessionBytes(threadID string) int64
    TranscribeFile(ctx context.Context, filePath string) (string, error)
    // Paridad adicional
    DeleteThreadArtifacts(ctx context.Context, threadID string) error
    ForceNewVectorStore(ctx context.Context, threadID string) (string, error)
    ListVectorStoreFiles(ctx context.Context, threadID string) ([]string, error)
    GetVectorStoreID(threadID string) string
}

type Handler struct {
    AI AIClient
    quotaValidator func(ctx context.Context, c *gin.Context, flow string) error
}

func NewHandler(ai AIClient) *Handler { return &Handler{AI: ai} }
func (h *Handler) SetQuotaValidator(fn func(ctx context.Context, c *gin.Context, flow string) error) { h.quotaValidator = fn }

// Start: crea SIEMPRE un thread real Assistants. Error si no hay assistant configurado.
func (h *Handler) Start(c *gin.Context) {
    if h.AI.GetAssistantID() == "" { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"assistant no configurado"}); return }
    start := time.Now()
    tid, err := h.AI.CreateThread(c.Request.Context())
    if err != nil || !strings.HasPrefix(tid, "thread_") { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"no se pudo crear thread","detail":errMsg(err)}); return }
    c.Header("X-Assistant-Start-Ms", time.Since(start).String())
    c.JSON(http.StatusOK, gin.H{"thread_id": tid, "strict_threads": true, "text":""})
}

// Message: soporta JSON simple o multipart (PDF/audio)
func (h *Handler) Message(c *gin.Context) {
    if h.quotaValidator != nil {
        if err := h.quotaValidator(c.Request.Context(), c, "chat_message"); err != nil {
            field,_ := c.Get("quota_error_field"); reason,_ := c.Get("quota_error_reason")
            resp := gin.H{"error":"chat quota exceeded"}
            if f,ok:=field.(string);ok&&f!=""{resp["field"]=f}
            if r,ok:=reason.(string);ok&&r!=""{resp["reason"]=r}
            c.JSON(http.StatusForbidden, resp); return
        }
    }
    ct := c.GetHeader("Content-Type")
    if strings.HasPrefix(ct, "multipart/form-data") { h.handleMultipart(c); return }
    var req struct { ThreadID string `json:"thread_id"`; Prompt string `json:"prompt"` }
    if err := c.ShouldBindJSON(&req); err != nil || !strings.HasPrefix(req.ThreadID, "thread_") {
        c.JSON(http.StatusBadRequest, gin.H{"error":"parámetros inválidos"}); return
    }
    start := time.Now()
    stream, err := h.AI.StreamAssistantMessage(c.Request.Context(), req.ThreadID, req.Prompt)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)}); return }
    if v,ok:=c.Get("quota_remaining");ok { c.Header("X-Quota-Remaining", toString(v)) }
    c.Header("X-Assistant-Start-Ms", time.Since(start).String())
    c.Header("X-Thread-ID", req.ThreadID)
    c.Header("X-Strict-Threads", "1")
    sseMaybeCapture(c, stream, req.ThreadID)
}

// handleMultipart replica lógica esencial de PDF/audio del chat original, sin fallback Chat Completions.
func (h *Handler) handleMultipart(c *gin.Context) {
    prompt := c.PostForm("prompt")
    threadID := c.PostForm("thread_id")
    if !strings.HasPrefix(threadID, "thread_") { c.JSON(http.StatusBadRequest, gin.H{"error":"thread_id inválido"}); return }
    upFile, _ := c.FormFile("file")
    start := time.Now()
    if upFile == nil { // solo texto
        stream, err := h.AI.StreamAssistantMessage(c.Request.Context(), threadID, prompt)
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)}); return }
        if v,ok:=c.Get("quota_remaining");ok { c.Header("X-Quota-Remaining", toString(v)) }
        c.Header("X-Assistant-Start-Ms", time.Since(start).String())
        c.Header("X-Thread-ID", threadID)
        c.Header("X-Strict-Threads", "1")
        sseMaybeCapture(c, stream, threadID); return
    }
    ext := strings.ToLower(filepath.Ext(upFile.Filename))
    tmpDir := "./tmp"; _ = os.MkdirAll(tmpDir, 0o755)
    tmp := filepath.Join(tmpDir, upFile.Filename)
    if err := c.SaveUploadedFile(upFile, tmp); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"upload failed"}); return }
    // Audio -> transcripción
    if isAudioExt(ext) {
        if text, err := h.AI.TranscribeFile(c, tmp); err == nil && strings.TrimSpace(text) != "" {
            if strings.TrimSpace(prompt) != "" { prompt += "\n\n[Transcripción]:\n" + text } else { prompt = text }
        }
        stream, err := h.AI.StreamAssistantMessage(c.Request.Context(), threadID, prompt)
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)}); return }
        if v,ok:=c.Get("quota_remaining");ok { c.Header("X-Quota-Remaining", toString(v)) }
        c.Header("X-Assistant-Start-Ms", time.Since(start).String())
        c.Header("X-Thread-ID", threadID)
        c.Header("X-Strict-Threads", "1")
        sseMaybeCapture(c, stream, threadID); return
    }
    if ext == ".pdf" { h.handlePDF(c, threadID, prompt, upFile, tmp, start); return }
    // Otros archivos: solo manda prompt (ignora archivo)
    stream, err := h.AI.StreamAssistantMessage(c.Request.Context(), threadID, prompt)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)}); return }
    if v,ok:=c.Get("quota_remaining");ok { c.Header("X-Quota-Remaining", toString(v)) }
    c.Header("X-Assistant-Start-Ms", time.Since(start).String())
    c.Header("X-Thread-ID", threadID)
    c.Header("X-Strict-Threads", "1")
    sseMaybeCapture(c, stream, threadID)
}

func (h *Handler) handlePDF(c *gin.Context, threadID, prompt string, upFile *multipart.FileHeader, tmp string, start time.Time) {
    if upFile.Size <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error":"archivo vacío"}); return }
    maxFiles, _ := strconv.Atoi(os.Getenv("VS_MAX_FILES"))
    maxMB, _ := strconv.Atoi(os.Getenv("VS_MAX_MB"))
    if maxFiles > 0 && h.AI.CountThreadFiles(threadID) >= maxFiles { c.JSON(http.StatusBadRequest, gin.H{"error":"límite de archivos alcanzado"}); return }
    if maxMB > 0 { nextMB := (h.AI.GetSessionBytes(threadID)+upFile.Size)/(1024*1024); if int(nextMB) > maxMB { c.JSON(http.StatusBadRequest, gin.H{"error":"límite de tamaño por sesión superado"}); return } }
    vsID, err := h.AI.EnsureVectorStore(c.Request.Context(), threadID)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)}); return }
    fileID, err := h.AI.UploadAssistantFile(c.Request.Context(), threadID, tmp)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)}); return }
    pollSec := 8; if v:=os.Getenv("VS_POLL_SEC"); v!="" { if n,err:=strconv.Atoi(v); err==nil && n>=0 { pollSec = n } }
    if err := h.AI.PollFileProcessed(c.Request.Context(), fileID, time.Duration(pollSec)*time.Second); err != nil {
        c.JSON(http.StatusAccepted, gin.H{"status":"processing","file_id":fileID}); return
    }
    if err := h.AI.AddFileToVectorStore(c.Request.Context(), vsID, fileID); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)}); return }
    h.AI.AddSessionBytes(threadID, upFile.Size)
    // Consumir cuota de archivo como en chat original
    if h.quotaValidator != nil {
        if err := h.quotaValidator(c.Request.Context(), c, "file_upload"); err != nil {
            field,_ := c.Get("quota_error_field"); reason,_ := c.Get("quota_error_reason")
            resp := gin.H{"error":"file quota exceeded"}
            if f,ok:=field.(string);ok&&f!=""{resp["field"]=f}
            if r,ok:=reason.(string);ok&&r!=""{resp["reason"]=r}
            c.JSON(http.StatusForbidden, resp); return
        }
    }
    base := strings.TrimSpace(prompt)
    structured := os.Getenv("STRUCTURED_PDF_SUMMARY") == "1"
    ragPromptTag := "generic-v1"
    if base == "" {
        fname := filepath.Base(upFile.Filename)
        if structured {
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
            ragPromptTag = "structured-v1"
        } else {
            base = "Analiza el documento adjunto (" + fname + ") y genera: 1) Un resumen ejecutivo de 4-6 líneas. 2) Una lista de los temas o secciones principales detectadas. 3) Hallazgos o insights clave. 4) Cualquier métrica, cifra o dato cuantitativo relevante. 5) Riesgos o limitaciones si se infieren del texto. 6) Recomendaciones accionables (máx 3). No inventes datos; si algo no existe omite el apartado correspondiente en lugar de rellenarlo. No uses frases introductorias como 'Claro'."
        }
        if pre := strings.TrimSpace(os.Getenv("DOC_SUMMARY_PREAMBLE")); pre != "" { base = pre + "\n\n" + base }
    }
    stream, err := h.AI.StreamAssistantMessage(c.Request.Context(), threadID, base)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)}); return }
    if v,ok:=c.Get("quota_remaining");ok { c.Header("X-Quota-Remaining", toString(v)) }
    c.Header("X-RAG","1")
    c.Header("X-RAG-File", filepath.Base(upFile.Filename))
    c.Header("X-RAG-Prompt", ragPromptTag)
    c.Header("X-Assistant-Start-Ms", time.Since(start).String())
    c.Header("X-Thread-ID", threadID)
    c.Header("X-Strict-Threads", "1")
    sseMaybeCapture(c, stream, threadID)
}

// Utilidades
func isAudioExt(ext string) bool { switch ext {case ".mp3", ".wav", ".m4a", ".aac", ".flac", ".ogg", ".webm": return true}; return false }
func errMsg(err error) string { if err==nil { return "" }; return err.Error() }
func toString(v interface{}) string { switch t:=v.(type){case string:return t; case int:return strconv.Itoa(t); case int64:return strconv.FormatInt(t,10)}; return "" }

// sseStream mínima (duplicada para aislar del paquete chat existente) – reusa formato: cada token -> data: token\n\n
func sseStream(c *gin.Context, ch <-chan string) {
    c.Writer.Header().Set("Content-Type", "text/event-stream")
    c.Writer.Header().Set("Cache-Control", "no-cache")
    c.Writer.Header().Set("Connection", "keep-alive")
    c.Writer.WriteHeader(http.StatusOK)
    c.Writer.Flush()
    for tok := range ch { if tok == "" { continue }; _, _ = c.Writer.Write([]byte("data: "+sanitize(tok)+"\n\n")); c.Writer.Flush() }
}

// sseMaybeCapture agrega token final __FULL__ en modo test (TEST_CAPTURE_FULL=1) replicando chat original.
func sseMaybeCapture(c *gin.Context, ch <-chan string, threadID string) {
    if os.Getenv("TEST_CAPTURE_FULL") != "1" { sseStream(c, ch); return }
    buf := &strings.Builder{}
    proxy := make(chan string)
    go func(){ for tok := range ch { buf.WriteString(tok); proxy <- tok }; close(proxy) }()
    sseStream(c, proxy)
    c.Writer.Write([]byte("data: __FULL__ "+sanitize(buf.String())+"\n\n"))
}

func sanitize(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " ") }

// Delete: limpieza de artifacts (paridad)
func (h *Handler) Delete(c *gin.Context) {
    var req struct { ThreadID string `json:"thread_id"` }
    if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.ThreadID)=="" { c.JSON(http.StatusBadRequest, gin.H{"error":"thread_id requerido"}); return }
    _ = h.AI.DeleteThreadArtifacts(c.Request.Context(), req.ThreadID)
    c.Status(http.StatusNoContent)
}

// VectorReset: fuerza vector store limpio
func (h *Handler) VectorReset(c *gin.Context) {
    var req struct { ThreadID string `json:"thread_id"` }
    if err := c.ShouldBindJSON(&req); err != nil || !strings.HasPrefix(req.ThreadID, "thread_") { c.JSON(http.StatusBadRequest, gin.H{"error":"thread_id inválido"}); return }
    vsID, err := h.AI.ForceNewVectorStore(c.Request.Context(), req.ThreadID)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":errMsg(err)}); return }
    c.JSON(http.StatusOK, gin.H{"status":"reset","vector_store_id":vsID})
}

// VectorFiles: lista archivos
func (h *Handler) VectorFiles(c *gin.Context) {
    threadID := strings.TrimSpace(c.Query("thread_id"))
    if !strings.HasPrefix(threadID, "thread_") { c.JSON(http.StatusBadRequest, gin.H{"error":"thread_id inválido"}); return }
    files, err := h.AI.ListVectorStoreFiles(c.Request.Context(), threadID)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":errMsg(err)}); return }
    vsID := h.AI.GetVectorStoreID(threadID)
    c.JSON(http.StatusOK, gin.H{"thread_id":threadID, "vector_store_id":vsID, "files":files})
}
