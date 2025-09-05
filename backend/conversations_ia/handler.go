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
    "fmt"
    "log"
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
    // Nuevos métodos para búsqueda específica en RAG y PubMed
    SearchInVectorStore(ctx context.Context, vectorStoreID, query string) (string, error)
    SearchPubMed(ctx context.Context, query string) (string, error)
    StreamAssistantWithSpecificVectorStore(ctx context.Context, threadID, prompt, vectorStoreID string) (<-chan string, error)
}

type Handler struct {
    AI AIClient
    quotaValidator func(ctx context.Context, c *gin.Context, flow string) error
}

// SmartMessage implementa el flujo mejorado: 1) RAG específico, 2) PubMed fallback, 3) citar fuente
func (h *Handler) SmartMessage(ctx context.Context, threadID, prompt string) (<-chan string, string, error) {
    const targetVectorID = "vs_680fc484cef081918b2b9588b701e2f4"
    
    // Primero intentar búsqueda en el vector store específico
    log.Printf("[conv][SmartMessage][start] thread=%s target_vector=%s", threadID, targetVectorID)
    
    ragResult, err := h.AI.SearchInVectorStore(ctx, targetVectorID, prompt)
    if err == nil && strings.TrimSpace(ragResult) != "" {
        // Encontramos información en el RAG, usamos el assistant con este vector específico
        log.Printf("[conv][SmartMessage][rag_found] thread=%s chars=%d", threadID, len(ragResult))
        
        ragPrompt := fmt.Sprintf(`Responde la siguiente pregunta usando EXCLUSIVAMENTE la información encontrada en el vector store: "%s"

Pregunta: %s

Información encontrada: %s

IMPORTANTE: 
- Solo usa la información proporcionada arriba
- Cita textualmente fragmentos relevantes
- Al final indica claramente: "Fuente: Base de conocimiento médico interno"
- Si la información no es suficiente para responder completamente, indica qué aspectos no se pueden responder
`, targetVectorID, prompt, ragResult)

        stream, err := h.AI.StreamAssistantWithSpecificVectorStore(ctx, threadID, ragPrompt, targetVectorID)
        return stream, "rag", err
    }
    
    // Si no encontramos en el RAG, buscar en PubMed
    log.Printf("[conv][SmartMessage][rag_empty] thread=%s, trying_pubmed", threadID)
    
    pubmedResult, err := h.AI.SearchPubMed(ctx, prompt)
    if err == nil && strings.TrimSpace(pubmedResult) != "" {
        log.Printf("[conv][SmartMessage][pubmed_found] thread=%s chars=%d", threadID, len(pubmedResult))
        
        pubmedPrompt := fmt.Sprintf(`Responde la siguiente pregunta usando EXCLUSIVAMENTE la información de PubMed proporcionada:

Pregunta: %s

Información de PubMed: %s

IMPORTANTE: 
- Solo usa la información de PubMed proporcionada arriba
- Incluye las referencias PMID cuando estén disponibles
- Al final indica claramente: "Fuente: PubMed (https://pubmed.ncbi.nlm.nih.gov/)"
- Mantén el rigor científico y cita estudios específicos cuando sea posible
`, prompt, pubmedResult)

        // Para el caso de PubMed, crear un stream simple que devuelva el resultado formateado
        ch := make(chan string, 1)
        go func() {
            defer close(ch)
            ch <- pubmedPrompt
        }()
        return ch, "pubmed", nil
    }
    
    // Si no encontramos en ninguna fuente, responder que no hay información
    log.Printf("[conv][SmartMessage][no_sources] thread=%s", threadID)
    
    noInfoPrompt := fmt.Sprintf(`La pregunta "%s" no pudo ser respondida porque:

1. No se encontró información relevante en la base de conocimiento médico interno
2. No se encontró información relevante en PubMed

Recomendaciones:
- Reformula tu pregunta con términos más específicos
- Verifica la ortografía de términos médicos
- Considera consultar fuentes médicas adicionales o un profesional de la salud

Fuente: Búsqueda sin resultados en fuentes configuradas`, prompt)

    ch := make(chan string, 1)
    go func() {
        defer close(ch)
        ch <- noInfoPrompt
    }()
    
    return ch, "none", nil
}

func NewHandler(ai AIClient) *Handler { return &Handler{AI: ai} }
func (h *Handler) SetQuotaValidator(fn func(ctx context.Context, c *gin.Context, flow string) error) { h.quotaValidator = fn }

// DebugConfig: expone estado mínimo de configuración (sin filtrar secretos) para diagnóstico remoto.
// Retorna si assistant está configurado y un prefijo enmascarado del ID.
func (h *Handler) DebugConfig(c *gin.Context) {
    id := strings.TrimSpace(h.AI.GetAssistantID())
    masked := ""
    if len(id) > 10 { masked = id[:6] + "..." + id[len(id)-4:] } else { masked = id }
    c.JSON(http.StatusOK, gin.H{
        "assistant_configured": strings.HasPrefix(id, "asst_"),
        "assistant_id_masked": masked,
    })
}

// Start: crea SIEMPRE un thread real Assistants. Error si no hay assistant configurado.
func (h *Handler) Start(c *gin.Context) {
    c.Header("X-Route-Matched", "/conversations/start")
    if h.AI.GetAssistantID() == "" {
        log.Printf("[conv][Start][error] assistant_id_empty")
        c.JSON(http.StatusServiceUnavailable, gin.H{"error":"assistant no configurado"}); return
    }
    start := time.Now()
    log.Printf("[conv][Start][begin] assistant_id=%s", h.AI.GetAssistantID())
    tid, err := h.AI.CreateThread(c.Request.Context())
    if err != nil || !strings.HasPrefix(tid, "thread_") {
    code := classifyErr(err)
    // Incluimos más detalles para facilitar debug remoto
    log.Printf("[conv][Start][error] create_thread code=%s err=%v assistant_id=%s", code, err, h.AI.GetAssistantID())
    status := http.StatusInternalServerError
    if code == "assistant_not_configured" { status = http.StatusServiceUnavailable }
    // Añadimos headers porque algunos clientes/proxies pueden ocultar body en 500
    c.Header("X-Conv-Error-Code", code)
    if err != nil { c.Header("X-Conv-Error-Detail", sanitize(err.Error())) }
    c.JSON(status, gin.H{"error":"no se pudo crear thread","code":code,"detail":errMsg(err)}); return
    }
    elapsed := time.Since(start)
    log.Printf("[conv][Start][ok] thread=%s elapsed_ms=%d", tid, elapsed.Milliseconds())
    c.Header("X-Assistant-Start-Ms", elapsed.String())
    c.JSON(http.StatusOK, gin.H{"thread_id": tid, "strict_threads": true, "text":""})
}

// Message: soporta JSON simple o multipart (PDF/audio)
func (h *Handler) Message(c *gin.Context) {
    wall := time.Now()
    // Fail fast when Assistants are not configured for this handler
    if h.AI.GetAssistantID() == "" {
        log.Printf("[conv][Message][error] assistant_id_empty")
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "assistant no configurado"})
        return
    }
    if h.quotaValidator != nil {
        if err := h.quotaValidator(c.Request.Context(), c, "chat_message"); err != nil {
            field,_ := c.Get("quota_error_field"); reason,_ := c.Get("quota_error_reason")
            resp := gin.H{"error":"chat quota exceeded"}
            if f,ok:=field.(string);ok&&f!=""{resp["field"]=f}
            if r,ok:=reason.(string);ok&&r!=""{resp["reason"]=r}
            log.Printf("[conv][Message][quota][denied] field=%v reason=%v", field, reason)
            c.JSON(http.StatusForbidden, resp); return
        }
        if v,ok:=c.Get("quota_remaining"); ok { log.Printf("[conv][Message][quota] remaining=%v", v) }
    }
    ct := c.GetHeader("Content-Type")
    log.Printf("[conv][Message][begin] ct=%s", ct)
    if strings.HasPrefix(ct, "multipart/form-data") { h.handleMultipart(c); return }
    var req struct { ThreadID string `json:"thread_id"`; Prompt string `json:"prompt"` }
    if err := c.ShouldBindJSON(&req); err != nil || !strings.HasPrefix(req.ThreadID, "thread_") {
        log.Printf("[conv][Message][json][error] bind_or_thread_invalid thread=%s err=%v", req.ThreadID, err)
        c.JSON(http.StatusBadRequest, gin.H{"error":"parámetros inválidos"}); return
    }
    start := time.Now()
    log.Printf("[conv][Message][json][smart.begin] thread=%s prompt_len=%d", req.ThreadID, len(req.Prompt))
    
    // Usar el nuevo flujo inteligente que busca en RAG específico y luego PubMed
    stream, source, err := h.SmartMessage(c.Request.Context(), req.ThreadID, req.Prompt)
    if err != nil {
        code := classifyErr(err)
        log.Printf("[conv][Message][json][smart.error] thread=%s code=%s err=%v", req.ThreadID, code, err)
        status := http.StatusInternalServerError
        if code == "assistant_not_configured" { status = http.StatusServiceUnavailable }
        c.JSON(status, gin.H{"error": errMsg(err), "code": code}); return
    }
    if v,ok:=c.Get("quota_remaining");ok { c.Header("X-Quota-Remaining", toString(v)) }
    c.Header("X-Assistant-Start-Ms", time.Since(start).String())
    c.Header("X-Thread-ID", req.ThreadID)
    c.Header("X-Strict-Threads", "1")
    c.Header("X-Source-Used", source) // Indicar qué fuente se usó
    log.Printf("[conv][Message][json][smart.stream] thread=%s source=%s prep_elapsed_ms=%d total_elapsed_ms=%d", req.ThreadID, source, time.Since(start).Milliseconds(), time.Since(wall).Milliseconds())
    sseMaybeCapture(c, stream, req.ThreadID)
}

// handleMultipart replica lógica esencial de PDF/audio del chat original, sin fallback Chat Completions.
func (h *Handler) handleMultipart(c *gin.Context) {
    prompt := c.PostForm("prompt")
    threadID := c.PostForm("thread_id")
    if !strings.HasPrefix(threadID, "thread_") { log.Printf("[conv][Message][multipart][error] invalid_thread=%s", threadID); c.JSON(http.StatusBadRequest, gin.H{"error":"thread_id inválido"}); return }
    upFile, err := c.FormFile("file")
    
    // Si no hay archivo pero tampoco hay error, probablemente fue rechazado por Nginx por tamaño
    if upFile == nil && err != nil {
        log.Printf("[conv][Message][multipart][error] no_file_received err=%v", err)
        // Si parece ser un error de tamaño (common en uploads grandes)
        if strings.Contains(strings.ToLower(err.Error()), "size") || strings.Contains(strings.ToLower(err.Error()), "large") {
            c.JSON(http.StatusRequestEntityTooLarge, gin.H{
                "error": "archivo demasiado grande",
                "code": "file_too_large_nginx", 
                "detail": "El archivo fue rechazado por ser muy grande. El límite máximo es 100 MB.",
                "max_size_mb": 100,
            })
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{"error": "no se pudo recibir el archivo", "detail": err.Error()})
        return
    }
    start := time.Now()
    log.Printf("[conv][Message][multipart][begin] thread=%s has_file=%v prompt_len=%d", threadID, upFile!=nil, len(prompt))
    if upFile == nil { // solo texto - usar flujo inteligente
        stream, source, err := h.SmartMessage(c.Request.Context(), threadID, prompt)
        if err != nil { code:=classifyErr(err); log.Printf("[conv][Message][multipart][smart.error] thread=%s code=%s err=%v", threadID, code, err); status:=http.StatusInternalServerError; if code=="assistant_not_configured" {status=http.StatusServiceUnavailable}; c.JSON(status, gin.H{"error": errMsg(err), "code":code}); return }
        if v,ok:=c.Get("quota_remaining");ok { c.Header("X-Quota-Remaining", toString(v)) }
        c.Header("X-Assistant-Start-Ms", time.Since(start).String())
        c.Header("X-Thread-ID", threadID)
        c.Header("X-Strict-Threads", "1")
        c.Header("X-Source-Used", source) // Indicar qué fuente se usó
        log.Printf("[conv][Message][multipart][smart.stream] thread=%s source=%s elapsed_ms=%d", threadID, source, time.Since(start).Milliseconds())
        sseMaybeCapture(c, stream, threadID); return
    }
    ext := strings.ToLower(filepath.Ext(upFile.Filename))
    tmpDir := "./tmp"; _ = os.MkdirAll(tmpDir, 0o755)
    tmp := filepath.Join(tmpDir, upFile.Filename)
    if err := c.SaveUploadedFile(upFile, tmp); err != nil { log.Printf("[conv][Message][multipart][error] save_upload err=%v", err); c.JSON(http.StatusInternalServerError, gin.H{"error":"upload failed"}); return }
    log.Printf("[conv][Message][multipart][file] thread=%s name=%s size=%d ext=%s", threadID, upFile.Filename, upFile.Size, ext)
    // Audio -> transcripción
    if isAudioExt(ext) {
        if text, err := h.AI.TranscribeFile(c, tmp); err == nil && strings.TrimSpace(text) != "" {
            if strings.TrimSpace(prompt) != "" { prompt += "\n\n[Transcripción]:\n" + text } else { prompt = text }
            log.Printf("[conv][Message][multipart][audio.transcribed] chars=%d", len(text))
        }
        stream, source, err := h.SmartMessage(c.Request.Context(), threadID, prompt)
        if err != nil { code:=classifyErr(err); log.Printf("[conv][Message][multipart][audio.error] thread=%s code=%s err=%v", threadID, code, err); status:=http.StatusInternalServerError; if code=="assistant_not_configured" {status=http.StatusServiceUnavailable}; c.JSON(status, gin.H{"error": errMsg(err), "code":code}); return }
        if v,ok:=c.Get("quota_remaining");ok { c.Header("X-Quota-Remaining", toString(v)) }
        c.Header("X-Assistant-Start-Ms", time.Since(start).String())
        c.Header("X-Thread-ID", threadID)
        c.Header("X-Strict-Threads", "1")
        c.Header("X-Source-Used", source) // Indicar qué fuente se usó
        log.Printf("[conv][Message][multipart][audio.stream] thread=%s source=%s elapsed_ms=%d", threadID, source, time.Since(start).Milliseconds())
        sseMaybeCapture(c, stream, threadID); return
    }
    if ext == ".pdf" { h.handlePDF(c, threadID, prompt, upFile, tmp, start); return }
    // Otros archivos: solo manda prompt (ignora archivo) - usar flujo inteligente
    stream, source, err := h.SmartMessage(c.Request.Context(), threadID, prompt)
    if err != nil { code:=classifyErr(err); log.Printf("[conv][Message][multipart][other.error] thread=%s code=%s err=%v", threadID, code, err); status:=http.StatusInternalServerError; if code=="assistant_not_configured" {status=http.StatusServiceUnavailable}; c.JSON(status, gin.H{"error": errMsg(err), "code":code}); return }
    if v,ok:=c.Get("quota_remaining");ok { c.Header("X-Quota-Remaining", toString(v)) }
    c.Header("X-Assistant-Start-Ms", time.Since(start).String())
    c.Header("X-Thread-ID", threadID)
    c.Header("X-Strict-Threads", "1")
    c.Header("X-Source-Used", source) // Indicar qué fuente se usó
    log.Printf("[conv][Message][multipart][other.stream] thread=%s source=%s elapsed_ms=%d", threadID, source, time.Since(start).Milliseconds())
    sseMaybeCapture(c, stream, threadID)
}

func (h *Handler) handlePDF(c *gin.Context, threadID, prompt string, upFile *multipart.FileHeader, tmp string, start time.Time) {
    if upFile.Size <= 0 { log.Printf("[conv][PDF][error] empty_file thread=%s", threadID); c.JSON(http.StatusBadRequest, gin.H{"error":"archivo vacío"}); return }
    
    // Verificar tamaño individual del archivo (100MB = 104857600 bytes)
    maxFileSizeBytes := int64(100 * 1024 * 1024) // 100MB
    if upFile.Size > maxFileSizeBytes {
        sizeMB := float64(upFile.Size) / (1024 * 1024)
        log.Printf("[conv][PDF][error] file_too_large thread=%s size_mb=%.1f max_mb=100", threadID, sizeMB)
        c.JSON(http.StatusRequestEntityTooLarge, gin.H{
            "error": "archivo demasiado grande", 
            "code": "file_too_large",
            "detail": fmt.Sprintf("El archivo pesa %.1f MB. El límite máximo es 100 MB.", sizeMB),
            "max_size_mb": 100,
        })
        return
    }
    
    maxFiles, _ := strconv.Atoi(os.Getenv("VS_MAX_FILES"))
    maxMB, _ := strconv.Atoi(os.Getenv("VS_MAX_MB"))
    if maxFiles > 0 && h.AI.CountThreadFiles(threadID) >= maxFiles { log.Printf("[conv][PDF][error] max_files thread=%s", threadID); c.JSON(http.StatusBadRequest, gin.H{"error":"límite de archivos alcanzado"}); return }
    if maxMB > 0 { nextMB := (h.AI.GetSessionBytes(threadID)+upFile.Size)/(1024*1024); if int(nextMB) > maxMB { log.Printf("[conv][PDF][error] max_mb thread=%s nextMB=%d max=%d", threadID, nextMB, maxMB); c.JSON(http.StatusBadRequest, gin.H{"error":"límite de tamaño por sesión superado"}); return } }
    vsID, err := h.AI.EnsureVectorStore(c.Request.Context(), threadID)
    if err != nil { log.Printf("[conv][PDF][error] ensure_vector err=%v", err); c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)}); return }
    log.Printf("[conv][PDF][vs.ready] thread=%s vs=%s", threadID, vsID)
    fileID, err := h.AI.UploadAssistantFile(c.Request.Context(), threadID, tmp)
    if err != nil { log.Printf("[conv][PDF][error] upload err=%v", err); c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)}); return }
    log.Printf("[conv][PDF][upload.ok] thread=%s file_id=%s name=%s size=%d", threadID, fileID, upFile.Filename, upFile.Size)
    pollSec := 8; if v:=os.Getenv("VS_POLL_SEC"); v!="" { if n,err:=strconv.Atoi(v); err==nil && n>=0 { pollSec = n } }
    pStart := time.Now()
    if err := h.AI.PollFileProcessed(c.Request.Context(), fileID, time.Duration(pollSec)*time.Second); err != nil {
        log.Printf("[conv][PDF][processing] thread=%s file_id=%s waited_ms=%d", threadID, fileID, time.Since(pStart).Milliseconds())
        c.JSON(http.StatusAccepted, gin.H{"status":"processing","file_id":fileID}); return
    }
    log.Printf("[conv][PDF][processed] thread=%s file_id=%s process_wait_ms=%d", threadID, fileID, time.Since(pStart).Milliseconds())
    if err := h.AI.AddFileToVectorStore(c.Request.Context(), vsID, fileID); err != nil { log.Printf("[conv][PDF][error] add_to_vs err=%v", err); c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)}); return }
    log.Printf("[conv][PDF][vs.added] thread=%s vs=%s file_id=%s", threadID, vsID, fileID)
    h.AI.AddSessionBytes(threadID, upFile.Size)
    // Consumir cuota de archivo como en chat original
    if h.quotaValidator != nil {
        if err := h.quotaValidator(c.Request.Context(), c, "file_upload"); err != nil {
            field,_ := c.Get("quota_error_field"); reason,_ := c.Get("quota_error_reason")
            resp := gin.H{"error":"file quota exceeded"}
            if f,ok:=field.(string);ok&&f!=""{resp["field"]=f}
            if r,ok:=reason.(string);ok&&r!=""{resp["reason"]=r}
            log.Printf("[conv][PDF][quota][denied] field=%v reason=%v", field, reason)
            c.JSON(http.StatusForbidden, resp); return
        } else { if v,ok:=c.Get("quota_remaining"); ok { log.Printf("[conv][PDF][quota] remaining=%v", v) } }
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
    if err != nil { log.Printf("[conv][PDF][error] stream err=%v", err); c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)}); return }
    if v,ok:=c.Get("quota_remaining");ok { c.Header("X-Quota-Remaining", toString(v)) }
    c.Header("X-RAG","1")
    c.Header("X-RAG-File", filepath.Base(upFile.Filename))
    c.Header("X-RAG-Prompt", ragPromptTag)
    c.Header("X-Assistant-Start-Ms", time.Since(start).String())
    c.Header("X-Thread-ID", threadID)
    c.Header("X-Strict-Threads", "1")
    log.Printf("[conv][PDF][stream] thread=%s file=%s rag_prompt=%s elapsed_ms=%d", threadID, upFile.Filename, ragPromptTag, time.Since(start).Milliseconds())
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

// classifyErr produce un code simbólico para facilitar observabilidad lado cliente.
func classifyErr(err error) string {
    if err == nil { return "" }
    e := strings.ToLower(err.Error())
    switch {
    case strings.Contains(e, "not configured"):
        return "assistant_not_configured"
    case strings.Contains(e, "401") || strings.Contains(e, "unauthorized"):
        return "openai_unauthorized"
    case strings.Contains(e, "timeout"):
        return "openai_timeout"
    case strings.Contains(e, "rate limit"):
        return "openai_rate_limited"
    case strings.Contains(e, "413") || strings.Contains(e, "entity too large") || strings.Contains(e, "file too large"):
        return "file_too_large"
    default:
        return "openai_error"
    }
}

// Delete: limpieza de artifacts (paridad)
func (h *Handler) Delete(c *gin.Context) {
    var req struct { ThreadID string `json:"thread_id"` }
    if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.ThreadID)=="" { log.Printf("[conv][Delete][error] bind thread_id=%s err=%v", req.ThreadID, err); c.JSON(http.StatusBadRequest, gin.H{"error":"thread_id requerido"}); return }
    log.Printf("[conv][Delete][begin] thread=%s", req.ThreadID)
    _ = h.AI.DeleteThreadArtifacts(c.Request.Context(), req.ThreadID)
    log.Printf("[conv][Delete][done] thread=%s", req.ThreadID)
    c.Status(http.StatusNoContent)
}

// VectorReset: fuerza vector store limpio
func (h *Handler) VectorReset(c *gin.Context) {
    var req struct { ThreadID string `json:"thread_id"` }
    if err := c.ShouldBindJSON(&req); err != nil || !strings.HasPrefix(req.ThreadID, "thread_") { log.Printf("[conv][VectorReset][error] bind_or_invalid thread=%s err=%v", req.ThreadID, err); c.JSON(http.StatusBadRequest, gin.H{"error":"thread_id inválido"}); return }
    log.Printf("[conv][VectorReset][begin] thread=%s", req.ThreadID)
    vsID, err := h.AI.ForceNewVectorStore(c.Request.Context(), req.ThreadID)
    if err != nil { log.Printf("[conv][VectorReset][error] force_new err=%v", err); c.JSON(http.StatusInternalServerError, gin.H{"error":errMsg(err)}); return }
    log.Printf("[conv][VectorReset][done] thread=%s vs=%s", req.ThreadID, vsID)
    c.JSON(http.StatusOK, gin.H{"status":"reset","vector_store_id":vsID})
}

// VectorFiles: lista archivos
func (h *Handler) VectorFiles(c *gin.Context) {
    threadID := strings.TrimSpace(c.Query("thread_id"))
    if !strings.HasPrefix(threadID, "thread_") { log.Printf("[conv][VectorFiles][error] invalid_thread=%s", threadID); c.JSON(http.StatusBadRequest, gin.H{"error":"thread_id inválido"}); return }
    log.Printf("[conv][VectorFiles][begin] thread=%s", threadID)
    files, err := h.AI.ListVectorStoreFiles(c.Request.Context(), threadID)
    if err != nil { log.Printf("[conv][VectorFiles][error] list err=%v", err); c.JSON(http.StatusInternalServerError, gin.H{"error":errMsg(err)}); return }
    vsID := h.AI.GetVectorStoreID(threadID)
    log.Printf("[conv][VectorFiles][ok] thread=%s vs=%s files=%d", threadID, vsID, len(files))
    c.JSON(http.StatusOK, gin.H{"thread_id":threadID, "vector_store_id":vsID, "files":files})
}

