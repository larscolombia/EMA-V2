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
	"regexp"
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

// VectorSearchResult contiene tanto el contenido encontrado como metadatos de la fuente
type VectorSearchResult struct {
	Content   string `json:"content"`
	Source    string `json:"source"`     // Título del documento o nombre del archivo
	VectorID  string `json:"vector_id"`  // ID del vector store
	HasResult bool   `json:"has_result"` // Indica si se encontró información relevante
}

// AIClientWithMetadata extiende AIClient con capacidades de metadatos
type AIClientWithMetadata interface {
	AIClient
	SearchInVectorStoreWithMetadata(ctx context.Context, vectorStoreID, query string) (*VectorSearchResult, error)
}

type Handler struct {
	AI             AIClient
	quotaValidator func(ctx context.Context, c *gin.Context, flow string) error
}

// Documento representa una pieza de contexto recuperado
type Documento struct {
	Titulo    string
	Contenido string
	Fuente    string // "vector" o "pubmed"
	// Referencias bibliográficas asociadas al contenido (cuando la fuente es PubMed)
	Referencias []string
}

// SmartMessage implementa el flujo mejorado: 1) RAG específico, 2) PubMed fallback, 3) citar fuente
func (h *Handler) SmartMessage(ctx context.Context, threadID, prompt, focusDocID string) (<-chan string, string, error) {
	targetVectorID := booksVectorID()

	// Si focusDocID está especificado, usar EXCLUSIVAMENTE ese documento
	if focusDocID != "" {
		// Prompt restrictivo: solo con el documento específico
		docOnlyPrompt := fmt.Sprintf(`Responde a la consulta usando EXCLUSIVAMENTE la información contenida en el documento con ID: %s

Pregunta del usuario: %s

Instrucciones:
- No utilices conocimiento externo ni otras fuentes.
- Si el documento no contiene información suficiente para responder, di claramente: "El documento no contiene información para responder esta pregunta".
- Cita el nombre del archivo específico si está disponible.
- Estructura la respuesta como: [Respuesta académica] + [Evidencia usada] + [Fuente: Nombre del documento]`, focusDocID, prompt)

		stream, err := h.AI.StreamAssistantWithSpecificVectorStore(ctx, threadID, docOnlyPrompt, h.AI.GetVectorStoreID(threadID))
		if err != nil {
			return nil, "focus_doc", err
		}
		return stream, "focus_doc", nil
	}

	// Si el hilo tiene documentos adjuntos, forzar modo "doc-only":
	// usar EXCLUSIVAMENTE el vector store del hilo y prohibir fuentes externas.
	if hasDocs := h.threadHasDocuments(ctx, threadID); hasDocs {
		vsID := h.AI.GetVectorStoreID(threadID)
		// Prompt restrictivo: solo con documentos del hilo
		docOnlyPrompt := fmt.Sprintf(`Responde a la consulta usando EXCLUSIVAMENTE la información contenida en los documentos adjuntos de este hilo.

Pregunta del usuario: %s

Instrucciones:
- No utilices conocimiento externo ni otras fuentes.
- Si los documentos no contienen información suficiente para responder, di claramente: "No hay suficiente información en los documentos adjuntos para responder".
- Estructura la respuesta como: [Respuesta académica] + [Evidencia usada] + [Fuentes: nombres de archivos y páginas si es posible]`, prompt)
		stream, err := h.AI.StreamAssistantWithSpecificVectorStore(ctx, threadID, docOnlyPrompt, vsID)
		if err != nil {
			return nil, "doc_only", err
		}
		return stream, "doc_only", nil
	}

	// Smalltalk/Saludo: responder cordialmente sin consultar fuentes
	if isSmallTalk(prompt) {
		reply := smallTalkReply(prompt)
		ch := make(chan string, 1)
		ch <- reply
		close(ch)
		return ch, "smalltalk", nil
	}

	// Flujo híbrido: consultar vector y PubMed, fusionar, y pasar contexto al modelo
	log.Printf("[conv][SmartMessage][hybrid.start] thread=%s target_vector=%s", threadID, targetVectorID)

	vdocs := h.buscarVector(ctx, targetVectorID, prompt)
	pdocs := h.buscarPubMed(ctx, prompt)

	ctxVec, ctxPub := fusionarResultados(vdocs, pdocs)

	// Reunir referencias de PubMed ya extraídas y filtradas (>=2020)
	var pubRefs []string
	seenRef := map[string]bool{}
	for _, d := range pdocs {
		for _, r := range d.Referencias {
			k := strings.TrimSpace(strings.ToLower(r))
			if k == "" || seenRef[k] {
				continue
			}
			seenRef[k] = true
			pubRefs = append(pubRefs, r)
		}
	}
	refsBlock := joinRefsForPrompt(pubRefs)

	vecHas := strings.TrimSpace(ctxVec) != ""
	pubHas := strings.TrimSpace(ctxPub) != ""

	if !vecHas && !pubHas {
		// Fallback claro cuando no hay contexto de ninguna fuente
		log.Printf("[conv][SmartMessage][hybrid.empty] thread=%s", threadID)
		fallback := "No se encontró información relevante para responder."
		ch := make(chan string, 1)
		ch <- fallback + "\n\n*(Fuente: Búsqueda sin resultados)*"
		close(ch)
		return ch, "none", nil
	}

	// Preparar input con bloques de contexto estructurados
	input := fmt.Sprintf(
		"Contexto recuperado (priorizar vector store):\n%s\n\n"+
			"Contexto complementario (PubMed):\n%s\n\n"+
			"Referencias (PubMed ≥2020, procesadas):\n%s\n\n"+
			"Pregunta del usuario:\n%s\n\n"+
			"FORMATO DE RESPUESTA OBLIGATORIO:\n"+
			"Estructúrala así:\n\n"+
			"## Respuesta académica:\n"+
			"[Contenido académico preciso y formal sin preámbulos]\n\n"+
			"## Evidencia usada:\n"+
			"[Descripción breve de la evidencia utilizada]\n\n"+
			"## Fuentes:\n"+
			"[Lista de fuentes con formato específico según origen]\n\n"+
			"REGLAS ESTRICTAS:\n"+
			"- Tono académico: preciso, formal y con profundidad\n"+
			"- PRIORIZA SIEMPRE la biblioteca interna (vector store). Si hay conflicto con PubMed, prevalece la biblioteca\n"+
			"- Para fuentes de biblioteca: '- NombreDocumento.pdf (pág. X–Y)' si tienes info de páginas\n"+
			"- Para PubMed: '- Autor et al. Título. Revista Año;Vol(Issue):páginas. DOI/PMID'\n"+
			"- NO inventes fuentes ni páginas. Si no tienes info específica, indica solo el nombre del documento\n"+
			"- NO repitas referencias en el cuerpo de la respuesta\n"+
			"- Sé específico y preciso en las citas\n",
		ctxVec, ctxPub, refsBlock, prompt,
	)

	// Usar el asistente configurado y el vector store de libros para mantener grounding y el historial del hilo
	stream, err := h.AI.StreamAssistantWithSpecificVectorStore(ctx, threadID, input, targetVectorID)
	source := "pubmed"
	if vecHas {
		source = "rag"
	}
	return stream, source, err
}

// isSmallTalk detecta saludos breves y cortesía sin contenido médico
func isSmallTalk(s string) bool {
	t := strings.ToLower(strings.TrimSpace(s))
	if t == "" {
		return false
	}
	// límites: hasta 8 palabras y contiene saludos comunes, sin términos clínicos obvios
	if len(strings.Fields(t)) > 8 {
		return false
	}
	greetings := []string{"hola", "buenos dias", "buenos días", "buenas tardes", "buenas noches", "que tal", "qué tal", "como estas", "cómo estás", "hey", "saludos", "gracias", "adios", "adiós"}
	medicalHints := []string{"sintoma", "síntoma", "diagnost", "tratam", "fiebre", "gripe", "asma", "hipert", "diabet", "virus", "bacteria"}
	hasGreet := false
	for _, g := range greetings {
		if strings.Contains(t, g) {
			hasGreet = true
			break
		}
	}
	if !hasGreet {
		return false
	}
	for _, m := range medicalHints {
		if strings.Contains(t, m) {
			return false
		}
	}
	return true
}

// smallTalkReply construye una respuesta breve y amable
func smallTalkReply(s string) string {
	t := strings.ToLower(s)
	if strings.Contains(t, "gracias") {
		return "¡Con gusto! ¿En qué puedo ayudarte hoy?"
	}
	if strings.Contains(t, "buenos dias") || strings.Contains(t, "buenos días") {
		return "¡Buenos días! ¿En qué puedo ayudarte?"
	}
	if strings.Contains(t, "buenas tardes") {
		return "¡Buenas tardes! ¿En qué puedo ayudarte?"
	}
	if strings.Contains(t, "buenas noches") {
		return "¡Buenas noches! ¿En qué puedo ayudarte?"
	}
	return "¡Hola! Estoy bien, gracias. ¿En qué puedo ayudarte?"
}

func NewHandler(ai AIClient) *Handler { return &Handler{AI: ai} }
func (h *Handler) SetQuotaValidator(fn func(ctx context.Context, c *gin.Context, flow string) error) {
	h.quotaValidator = fn
}

// DebugConfig: expone estado mínimo de configuración (sin filtrar secretos) para diagnóstico remoto.
// Retorna si assistant está configurado y un prefijo enmascarado del ID.
func (h *Handler) DebugConfig(c *gin.Context) {
	id := strings.TrimSpace(h.AI.GetAssistantID())
	masked := ""
	if len(id) > 10 {
		masked = id[:6] + "..." + id[len(id)-4:]
	} else {
		masked = id
	}
	c.JSON(http.StatusOK, gin.H{
		"assistant_configured": strings.HasPrefix(id, "asst_"),
		"assistant_id_masked":  masked,
	})
}

// Start: crea SIEMPRE un thread real Assistants. Error si no hay assistant configurado.
func (h *Handler) Start(c *gin.Context) {
	c.Header("X-Route-Matched", "/conversations/start")
	if h.AI.GetAssistantID() == "" {
		log.Printf("[conv][Start][error] assistant_id_empty")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "assistant no configurado"})
		return
	}
	start := time.Now()
	log.Printf("[conv][Start][begin] assistant_id=%s", h.AI.GetAssistantID())
	tid, err := h.AI.CreateThread(c.Request.Context())
	if err != nil || !strings.HasPrefix(tid, "thread_") {
		code := classifyErr(err)
		// Incluimos más detalles para facilitar debug remoto
		log.Printf("[conv][Start][error] create_thread code=%s err=%v assistant_id=%s", code, err, h.AI.GetAssistantID())
		status := http.StatusInternalServerError
		if code == "assistant_not_configured" {
			status = http.StatusServiceUnavailable
		}
		// Añadimos headers porque algunos clientes/proxies pueden ocultar body en 500
		c.Header("X-Conv-Error-Code", code)
		if err != nil {
			c.Header("X-Conv-Error-Detail", sanitize(err.Error()))
		}
		c.JSON(status, gin.H{"error": "no se pudo crear thread", "code": code, "detail": errMsg(err)})
		return
	}
	elapsed := time.Since(start)
	log.Printf("[conv][Start][ok] thread=%s elapsed_ms=%d", tid, elapsed.Milliseconds())
	c.Header("X-Assistant-Start-Ms", elapsed.String())
	c.JSON(http.StatusOK, gin.H{"thread_id": tid, "strict_threads": true, "text": ""})
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
			field, _ := c.Get("quota_error_field")
			reason, _ := c.Get("quota_error_reason")
			resp := gin.H{"error": "chat quota exceeded"}
			if f, ok := field.(string); ok && f != "" {
				resp["field"] = f
			}
			if r, ok := reason.(string); ok && r != "" {
				resp["reason"] = r
			}
			log.Printf("[conv][Message][quota][denied] field=%v reason=%v", field, reason)
			c.JSON(http.StatusForbidden, resp)
			return
		}
		if v, ok := c.Get("quota_remaining"); ok {
			log.Printf("[conv][Message][quota] remaining=%v", v)
		}
	}
	ct := c.GetHeader("Content-Type")
	log.Printf("[conv][Message][begin] ct=%s", ct)
	if strings.HasPrefix(ct, "multipart/form-data") {
		h.handleMultipart(c)
		return
	}
	var req struct {
		ThreadID   string `json:"thread_id"`
		Prompt     string `json:"prompt"`
		FocusDocID string `json:"focus_doc_id,omitempty"` // ID del PDF específico para limitarse solo a ese documento
	}
	if err := c.ShouldBindJSON(&req); err != nil || !strings.HasPrefix(req.ThreadID, "thread_") {
		log.Printf("[conv][Message][json][error] bind_or_thread_invalid thread=%s err=%v", req.ThreadID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "parámetros inválidos"})
		return
	}
	start := time.Now()
	log.Printf("[conv][Message][json][smart.begin] thread=%s prompt_len=%d prompt_preview=\"%s\"", req.ThreadID, len(req.Prompt), sanitizePreview(req.Prompt))

	// Si el hilo tiene documentos, forzar respuesta solo con esos documentos; de lo contrario flujo inteligente
	var (
		stream <-chan string
		source string
		err    error
	)
	if h.threadHasDocuments(c.Request.Context(), req.ThreadID) {
		vsID := h.AI.GetVectorStoreID(req.ThreadID)
		prompt := fmt.Sprintf(`Tu única fuente de información son los documentos PDF adjuntos de este hilo.

Pregunta: %s

Reglas estrictas:
- No agregues conocimiento externo; no inventes.
- No repitas párrafos o fragmentos textuales completos salvo que se te pida explícitamente.
- Si la pregunta no puede contestarse con la información del PDF, responde exactamente: "El documento no contiene información para responder esta pregunta.".
- Estilo: profesional, claro y preciso; prioriza la precisión antes que la extensión.
- Cita los nombres de archivo de los PDF utilizados si están disponibles; en su defecto indica "documentos adjuntos del hilo".
- Añade al final: "Fuente: documentos adjuntos del hilo".`, req.Prompt)
		stream, err = h.AI.StreamAssistantWithSpecificVectorStore(c.Request.Context(), req.ThreadID, prompt, vsID)
		source = "doc_only"
	} else {
		// Usar el nuevo flujo inteligente que busca en RAG específico y luego PubMed
		stream, source, err = h.SmartMessage(c.Request.Context(), req.ThreadID, req.Prompt, req.FocusDocID)
	}
	if err != nil {
		code := classifyErr(err)
		log.Printf("[conv][Message][json][smart.error] thread=%s code=%s err=%v", req.ThreadID, code, err)
		status := http.StatusInternalServerError
		if code == "assistant_not_configured" {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": errMsg(err), "code": code})
		return
	}
	if v, ok := c.Get("quota_remaining"); ok {
		c.Header("X-Quota-Remaining", toString(v))
	}
	c.Header("X-Assistant-Start-Ms", time.Since(start).String())
	c.Header("X-Thread-ID", req.ThreadID)
	c.Header("X-Strict-Threads", "1")
	c.Header("X-Source-Used", source) // Indicar qué fuente se usó
	if source == "rag" {
		c.Header("X-Books-Vector-ID", "vs_680fc484cef081918b2b9588b701e2f4")
	}
	log.Printf("[conv][Message][json][smart.stream] thread=%s source=%s prep_elapsed_ms=%d total_elapsed_ms=%d", req.ThreadID, source, time.Since(start).Milliseconds(), time.Since(wall).Milliseconds())
	// Emitir señales de etapa antes del contenido
	stages := []string{"__STAGE__:start", "__STAGE__:rag_search"}
	switch source {
	case "doc_only":
		stages = []string{"__STAGE__:start", "__STAGE__:doc_only", "__STAGE__:streaming_answer"}
		sseMaybeCapture(c, wrapWithStages(stages, stream), req.ThreadID)
		return
	case "smalltalk":
		stages = []string{"__STAGE__:start", "__STAGE__:smalltalk", "__STAGE__:streaming_answer"}
		sseMaybeCapture(c, wrapWithStages(stages, stream), req.ThreadID)
		return
	case "rag":
		stages = append(stages, "__STAGE__:rag_found", "__STAGE__:streaming_answer")
	case "pubmed":
		stages = append(stages, "__STAGE__:rag_empty", "__STAGE__:pubmed_search", "__STAGE__:pubmed_found", "__STAGE__:streaming_answer")
	default:
		stages = append(stages, "__STAGE__:rag_empty", "__STAGE__:no_source", "__STAGE__:streaming_answer")
	}
	sseMaybeCapture(c, wrapWithStages(stages, stream), req.ThreadID)
}

// handleMultipart replica lógica esencial de PDF/audio del chat original, sin fallback Chat Completions.
func (h *Handler) handleMultipart(c *gin.Context) {
	prompt := c.PostForm("prompt")
	threadID := c.PostForm("thread_id")
	focusDocID := c.PostForm("focus_doc_id") // Nuevo parámetro para limitar a un documento específico
	if !strings.HasPrefix(threadID, "thread_") {
		log.Printf("[conv][Message][multipart][error] invalid_thread=%s", threadID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "thread_id inválido"})
		return
	}
	upFile, err := c.FormFile("file")

	// Si no hay archivo pero tampoco hay error, probablemente fue rechazado por Nginx por tamaño
	if upFile == nil && err != nil {
		log.Printf("[conv][Message][multipart][error] no_file_received err=%v", err)
		// Si parece ser un error de tamaño (common en uploads grandes)
		if strings.Contains(strings.ToLower(err.Error()), "size") || strings.Contains(strings.ToLower(err.Error()), "large") {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":       "archivo demasiado grande",
				"code":        "file_too_large_nginx",
				"detail":      "El archivo fue rechazado por ser muy grande. El límite máximo es 100 MB.",
				"max_size_mb": 100,
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "no se pudo recibir el archivo", "detail": err.Error()})
		return
	}
	start := time.Now()
	log.Printf("[conv][Message][multipart][begin] thread=%s has_file=%v prompt_len=%d", threadID, upFile != nil, len(prompt))
	if upFile == nil { // solo texto
		// Si el hilo ya tiene documentos, usar doc-only; si no, flujo inteligente
		var (
			stream <-chan string
			source string
			err    error
		)
		if h.threadHasDocuments(c.Request.Context(), threadID) {
			vsID := h.AI.GetVectorStoreID(threadID)
			p := fmt.Sprintf(`Tu única fuente de información son los documentos PDF adjuntos de este hilo.

Pregunta: %s

Reglas estrictas:
- No agregues conocimiento externo; no inventes.
- No repitas párrafos o fragmentos textuales completos salvo que se te pida explícitamente.
- Si la pregunta no puede contestarse con la información del PDF, responde exactamente: "El documento no contiene información para responder esta pregunta.".
- Estilo: profesional, claro y preciso; prioriza la precisión antes que la extensión.
- Cita los nombres de archivo de los PDF utilizados si están disponibles; en su defecto indica "documentos adjuntos del hilo".
- Añade al final: "Fuente: documentos adjuntos del hilo".`, prompt)
			stream, err = h.AI.StreamAssistantWithSpecificVectorStore(c.Request.Context(), threadID, p, vsID)
			source = "doc_only"
		} else {
			stream, source, err = h.SmartMessage(c.Request.Context(), threadID, prompt, focusDocID)
		}
		if err != nil {
			code := classifyErr(err)
			log.Printf("[conv][Message][multipart][smart.error] thread=%s code=%s err=%v", threadID, code, err)
			status := http.StatusInternalServerError
			if code == "assistant_not_configured" {
				status = http.StatusServiceUnavailable
			}
			c.JSON(status, gin.H{"error": errMsg(err), "code": code})
			return
		}
		if v, ok := c.Get("quota_remaining"); ok {
			c.Header("X-Quota-Remaining", toString(v))
		}
		c.Header("X-Assistant-Start-Ms", time.Since(start).String())
		c.Header("X-Thread-ID", threadID)
		c.Header("X-Strict-Threads", "1")
		c.Header("X-Source-Used", source) // Indicar qué fuente se usó
		log.Printf("[conv][Message][multipart][smart.stream] thread=%s source=%s elapsed_ms=%d", threadID, source, time.Since(start).Milliseconds())
		stages := []string{"__STAGE__:start", "__STAGE__:rag_search"}
		if source == "doc_only" {
			stages = []string{"__STAGE__:start", "__STAGE__:doc_only", "__STAGE__:streaming_answer"}
			sseMaybeCapture(c, wrapWithStages(stages, stream), threadID)
			return
		}
		switch source {
		case "smalltalk":
			stages = []string{"__STAGE__:start", "__STAGE__:smalltalk", "__STAGE__:streaming_answer"}
			sseMaybeCapture(c, wrapWithStages(stages, stream), threadID)
			return
		case "rag":
			stages = append(stages, "__STAGE__:rag_found", "__STAGE__:streaming_answer")
		case "pubmed":
			stages = append(stages, "__STAGE__:rag_empty", "__STAGE__:pubmed_search", "__STAGE__:pubmed_found", "__STAGE__:streaming_answer")
		default:
			stages = append(stages, "__STAGE__:rag_empty", "__STAGE__:no_source", "__STAGE__:streaming_answer")
		}
		sseMaybeCapture(c, wrapWithStages(stages, stream), threadID)
		return
	}
	ext := strings.ToLower(filepath.Ext(upFile.Filename))
	// Usar directorio temporal del sistema para mayor compatibilidad (Windows/Linux) y crear subcarpeta
	tmpDir := filepath.Join(os.TempDir(), "ema_uploads")
	if mkErr := os.MkdirAll(tmpDir, 0o755); mkErr != nil {
		log.Printf("[conv][Message][multipart][error] mkdir_tmp err=%v", mkErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed", "code": "tmp_dir_unavailable"})
		return
	}
	// Sanitizar nombre de archivo para evitar caracteres no válidos en Windows y colisiones
	safeBase := sanitizeFilename(upFile.Filename)
	base := strings.TrimSuffix(safeBase, ext)
	if base == "" {
		base = "upload"
	}
	safeName := fmt.Sprintf("%s_%d%s", base, time.Now().UnixNano(), ext)
	tmp := filepath.Join(tmpDir, safeName)
	if err := c.SaveUploadedFile(upFile, tmp); err != nil {
		log.Printf("[conv][Message][multipart][error] save_upload name=%s safe=%s tmp=%s err=%v", upFile.Filename, safeName, tmp, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed", "code": "upload_save_failed"})
		return
	}
	log.Printf("[conv][Message][multipart][file] thread=%s name=%s -> safe=%s size=%d ext=%s", threadID, upFile.Filename, filepath.Base(tmp), upFile.Size, ext)
	// Audio -> transcripción
	if isAudioExt(ext) {
		if text, err := h.AI.TranscribeFile(c.Request.Context(), tmp); err == nil && strings.TrimSpace(text) != "" {
			if strings.TrimSpace(prompt) != "" {
				prompt += "\n\n[Transcripción]:\n" + text
			} else {
				prompt = text
			}
			log.Printf("[conv][Message][multipart][audio.transcribed] chars=%d", len(text))
		}
		stream, source, err := h.SmartMessage(c.Request.Context(), threadID, prompt, "")
		if err != nil {
			code := classifyErr(err)
			log.Printf("[conv][Message][multipart][audio.error] thread=%s code=%s err=%v", threadID, code, err)
			status := http.StatusInternalServerError
			if code == "assistant_not_configured" {
				status = http.StatusServiceUnavailable
			}
			c.JSON(status, gin.H{"error": errMsg(err), "code": code})
			return
		}
		if v, ok := c.Get("quota_remaining"); ok {
			c.Header("X-Quota-Remaining", toString(v))
		}
		c.Header("X-Assistant-Start-Ms", time.Since(start).String())
		c.Header("X-Thread-ID", threadID)
		c.Header("X-Strict-Threads", "1")
		c.Header("X-Source-Used", source) // Indicar qué fuente se usó
		if source == "rag" {
			c.Header("X-Books-Vector-ID", "vs_680fc484cef081918b2b9588b701e2f4")
		}
		log.Printf("[conv][Message][multipart][audio.stream] thread=%s source=%s elapsed_ms=%d", threadID, source, time.Since(start).Milliseconds())
		stages := []string{"__STAGE__:start", "__STAGE__:rag_search"}
		switch source {
		case "rag":
			stages = append(stages, "__STAGE__:rag_found", "__STAGE__:streaming_answer")
		case "pubmed":
			stages = append(stages, "__STAGE__:rag_empty", "__STAGE__:pubmed_search", "__STAGE__:pubmed_found", "__STAGE__:streaming_answer")
		default:
			stages = append(stages, "__STAGE__:rag_empty", "__STAGE__:no_source", "__STAGE__:streaming_answer")
		}
		sseMaybeCapture(c, wrapWithStages(stages, stream), threadID)
		return
	}
	if ext == ".pdf" {
		h.handlePDF(c, threadID, prompt, upFile, tmp, start)
		return
	}
	// Otros archivos: solo manda prompt (ignora archivo) - usar flujo inteligente
	stream, source, err := h.SmartMessage(c.Request.Context(), threadID, prompt, "")
	if err != nil {
		code := classifyErr(err)
		log.Printf("[conv][Message][multipart][other.error] thread=%s code=%s err=%v", threadID, code, err)
		status := http.StatusInternalServerError
		if code == "assistant_not_configured" {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": errMsg(err), "code": code})
		return
	}
	if v, ok := c.Get("quota_remaining"); ok {
		c.Header("X-Quota-Remaining", toString(v))
	}
	c.Header("X-Assistant-Start-Ms", time.Since(start).String())
	c.Header("X-Thread-ID", threadID)
	c.Header("X-Strict-Threads", "1")
	c.Header("X-Source-Used", source) // Indicar qué fuente se usó
	log.Printf("[conv][Message][multipart][other.stream] thread=%s source=%s elapsed_ms=%d", threadID, source, time.Since(start).Milliseconds())
	stages := []string{"__STAGE__:start", "__STAGE__:rag_search"}
	switch source {
	case "rag":
		stages = append(stages, "__STAGE__:rag_found", "__STAGE__:streaming_answer")
	case "pubmed":
		stages = append(stages, "__STAGE__:rag_empty", "__STAGE__:pubmed_search", "__STAGE__:pubmed_found", "__STAGE__:streaming_answer")
	default:
		stages = append(stages, "__STAGE__:rag_empty", "__STAGE__:no_source", "__STAGE__:streaming_answer")
	}
	sseMaybeCapture(c, wrapWithStages(stages, stream), threadID)
}

func (h *Handler) handlePDF(c *gin.Context, threadID, prompt string, upFile *multipart.FileHeader, tmp string, start time.Time) {
	if upFile.Size <= 0 {
		log.Printf("[conv][PDF][error] empty_file thread=%s", threadID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "archivo vacío"})
		return
	}

	// Verificar tamaño individual del archivo (100MB = 104857600 bytes)
	maxFileSizeBytes := int64(100 * 1024 * 1024) // 100MB
	if upFile.Size > maxFileSizeBytes {
		sizeMB := float64(upFile.Size) / (1024 * 1024)
		log.Printf("[conv][PDF][error] file_too_large thread=%s size_mb=%.1f max_mb=100", threadID, sizeMB)
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error":       "archivo demasiado grande",
			"code":        "file_too_large",
			"detail":      fmt.Sprintf("El archivo pesa %.1f MB. El límite máximo es 100 MB.", sizeMB),
			"max_size_mb": 100,
		})
		return
	}

	maxFiles, _ := strconv.Atoi(os.Getenv("VS_MAX_FILES"))
	maxMB, _ := strconv.Atoi(os.Getenv("VS_MAX_MB"))
	if maxFiles > 0 && h.AI.CountThreadFiles(threadID) >= maxFiles {
		log.Printf("[conv][PDF][error] max_files thread=%s", threadID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "límite de archivos alcanzado"})
		return
	}
	if maxMB > 0 {
		nextMB := (h.AI.GetSessionBytes(threadID) + upFile.Size) / (1024 * 1024)
		if int(nextMB) > maxMB {
			log.Printf("[conv][PDF][error] max_mb thread=%s nextMB=%d max=%d", threadID, nextMB, maxMB)
			c.JSON(http.StatusBadRequest, gin.H{"error": "límite de tamaño por sesión superado"})
			return
		}
	}
	vsID, err := h.AI.EnsureVectorStore(c.Request.Context(), threadID)
	if err != nil {
		log.Printf("[conv][PDF][error] ensure_vector err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)})
		return
	}
	log.Printf("[conv][PDF][vs.ready] thread=%s vs=%s", threadID, vsID)
	fileID, err := h.AI.UploadAssistantFile(c.Request.Context(), threadID, tmp)
	if err != nil {
		log.Printf("[conv][PDF][error] upload err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)})
		return
	}
	log.Printf("[conv][PDF][upload.ok] thread=%s file_id=%s name=%s size=%d", threadID, fileID, upFile.Filename, upFile.Size)
	pollSec := 8
	if v := os.Getenv("VS_POLL_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			pollSec = n
		}
	}
	pStart := time.Now()
	if err := h.AI.PollFileProcessed(c.Request.Context(), fileID, time.Duration(pollSec)*time.Second); err != nil {
		log.Printf("[conv][PDF][processing] thread=%s file_id=%s waited_ms=%d", threadID, fileID, time.Since(pStart).Milliseconds())
		c.JSON(http.StatusAccepted, gin.H{"status": "processing", "file_id": fileID})
		return
	}
	log.Printf("[conv][PDF][processed] thread=%s file_id=%s process_wait_ms=%d", threadID, fileID, time.Since(pStart).Milliseconds())
	if err := h.AI.AddFileToVectorStore(c.Request.Context(), vsID, fileID); err != nil {
		log.Printf("[conv][PDF][error] add_to_vs err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)})
		return
	}
	log.Printf("[conv][PDF][vs.added] thread=%s vs=%s file_id=%s", threadID, vsID, fileID)
	h.AI.AddSessionBytes(threadID, upFile.Size)
	// Consumir cuota de archivo como en chat original
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "file_upload"); err != nil {
			field, _ := c.Get("quota_error_field")
			reason, _ := c.Get("quota_error_reason")
			resp := gin.H{"error": "file quota exceeded"}
			if f, ok := field.(string); ok && f != "" {
				resp["field"] = f
			}
			if r, ok := reason.(string); ok && r != "" {
				resp["reason"] = r
			}
			log.Printf("[conv][PDF][quota][denied] field=%v reason=%v", field, reason)
			c.JSON(http.StatusForbidden, resp)
			return
		} else {
			if v, ok := c.Get("quota_remaining"); ok {
				log.Printf("[conv][PDF][quota] remaining=%v", v)
			}
		}
	}
	base := strings.TrimSpace(prompt)
	if base == "" {
		// No generar resumen automático. Solo confirmación y listo para preguntas.
		fname := filepath.Base(upFile.Filename)
		msg := "Documento '" + fname + "' cargado y procesado correctamente. No se generará resumen automático. Puedes hacer preguntas específicas sobre este PDF.\n\nFuente: " + fname
		if v, ok := c.Get("quota_remaining"); ok {
			c.Header("X-Quota-Remaining", toString(v))
		}
		c.Header("X-RAG", "1")
		c.Header("X-Grounded", "1")
		c.Header("X-RAG-File", fname)
		c.Header("X-RAG-Prompt", "doc-only-v1")
		c.Header("X-Assistant-Start-Ms", time.Since(start).String())
		c.Header("X-Thread-ID", threadID)
		c.Header("X-Strict-Threads", "1")
		c.Header("X-Source-Used", "doc_only")
		log.Printf("[conv][PDF][confirm] thread=%s file=%s doc_only=1 elapsed_ms=%d", threadID, upFile.Filename, time.Since(start).Milliseconds())
		one := make(chan string, 1)
		one <- msg
		close(one)
		stages := []string{"__STAGE__:start", "__STAGE__:doc_only", "__STAGE__:streaming_answer"}
		sseMaybeCapture(c, wrapWithStages(stages, one), threadID)
		return
	}
	// Si viene prompt junto al PDF, responder en modo doc-only usando el vector store del hilo
	p := fmt.Sprintf(`Tu única fuente de información son los documentos PDF adjuntos de este hilo.

Pregunta: %s

Reglas estrictas:
- No agregues conocimiento externo; no inventes.
- No repitas párrafos o fragmentos textuales completos salvo que se te pida explícitamente.
- Si la pregunta no puede contestarse con la información del PDF, responde exactamente: "El documento no contiene información para responder esta pregunta.".
- Estilo: profesional, claro y preciso; prioriza la precisión antes que la extensión.
- Cita los nombres de archivo de los PDF utilizados si están disponibles; en su defecto indica "documentos adjuntos del hilo".
- Añade al final: "Fuente: documentos adjuntos del hilo".`, base)
	stream, err := h.AI.StreamAssistantWithSpecificVectorStore(c.Request.Context(), threadID, p, vsID)
	if err != nil {
		log.Printf("[conv][PDF][error] stream err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)})
		return
	}
	if v, ok := c.Get("quota_remaining"); ok {
		c.Header("X-Quota-Remaining", toString(v))
	}
	c.Header("X-RAG", "1")
	c.Header("X-Grounded", "1")
	c.Header("X-RAG-File", filepath.Base(upFile.Filename))
	c.Header("X-RAG-Prompt", "doc-only-v1")
	c.Header("X-Assistant-Start-Ms", time.Since(start).String())
	c.Header("X-Thread-ID", threadID)
	c.Header("X-Strict-Threads", "1")
	c.Header("X-Source-Used", "doc_only")
	log.Printf("[conv][PDF][doc_only.stream] thread=%s file=%s elapsed_ms=%d", threadID, upFile.Filename, time.Since(start).Milliseconds())
	stages := []string{"__STAGE__:start", "__STAGE__:doc_only", "__STAGE__:streaming_answer"}
	sseMaybeCapture(c, wrapWithStages(stages, stream), threadID)
}

// Utilidades
func isAudioExt(ext string) bool {
	switch ext {
	case ".mp3", ".wav", ".m4a", ".aac", ".flac", ".ogg", ".webm":
		return true
	}
	return false
}

// sanitizeFilename reemplaza caracteres inválidos para Windows y normaliza espacios.
// Mantiene solo letras/números/espacios/._- y elimina el resto.
func sanitizeFilename(name string) string {
	// Tomar solo el base name por seguridad
	name = filepath.Base(name)
	// Eliminar caracteres de control y reservados de Windows: <>:"/\|?*
	b := make([]rune, 0, len(name))
	for _, r := range name {
		if r < 0x20 { // control chars
			continue
		}
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			b = append(b, '_')
		default:
			// Permitir runas comunes; opcionalmente restringir más
			b = append(b, r)
		}
	}
	cleaned := strings.TrimSpace(string(b))
	if cleaned == "" {
		return "upload"
	}
	// Compactar espacios consecutivos
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	// Limitar longitud razonable
	if len(cleaned) > 120 {
		cleaned = cleaned[:120]
	}
	return cleaned
}

// threadHasDocuments determina si el hilo tiene archivos en su vector store (para activar modo doc-only)
func (h *Handler) threadHasDocuments(ctx context.Context, threadID string) bool {
	if strings.TrimSpace(threadID) == "" {
		return false
	}
	files, err := h.AI.ListVectorStoreFiles(ctx, threadID)
	if err != nil {
		return false
	}
	return len(files) > 0
}
func errMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	}
	return ""
}

// sseStream mínima (duplicada para aislar del paquete chat existente) – reusa formato: cada token -> data: token\n\n
func sseStream(c *gin.Context, ch <-chan string) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()
	for tok := range ch {
		if tok == "" {
			continue
		}
		// Preservar saltos de línea según protocolo SSE: cada línea con prefijo 'data: '
		lines := strings.Split(tok, "\n")
		for _, ln := range lines {
			_, _ = c.Writer.Write([]byte("data: " + ln + "\n"))
		}
		_, _ = c.Writer.Write([]byte("\n"))
		c.Writer.Flush()
	}
}

// sseMaybeCapture agrega token final __FULL__ en modo test (TEST_CAPTURE_FULL=1) replicando chat original.
func sseMaybeCapture(c *gin.Context, ch <-chan string, threadID string) {
	if os.Getenv("TEST_CAPTURE_FULL") != "1" {
		sseStream(c, ch)
		return
	}
	buf := &strings.Builder{}
	proxy := make(chan string)
	go func() {
		for tok := range ch {
			buf.WriteString(tok)
			proxy <- tok
		}
		close(proxy)
	}()
	sseStream(c, proxy)
	c.Writer.Write([]byte("data: __FULL__ " + sanitize(buf.String()) + "\n\n"))
}

// wrapWithStages emits a sequence of stage markers before forwarding tokens from the main stream.
// Each stage marker is written as its own SSE event (single token), then the underlying stream is proxied.
func wrapWithStages(stages []string, ch <-chan string) <-chan string {
	out := make(chan string)
	go func() {
		// Emit stage markers first
		for _, s := range stages {
			if strings.TrimSpace(s) == "" { // skip empties defensively
				continue
			}
			out <- s
		}
		// Proxy remaining tokens
		for tok := range ch {
			out <- tok
		}
		close(out)
	}()
	return out
}

func sanitize(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " ")
}

// sanitizePreview limita y sanitiza texto para logs
func sanitizePreview(s string) string {
	clean := sanitize(s)
	if len(clean) > 100 {
		return clean[:100] + "..."
	}
	return clean
}

// buscarVector consulta el vector store canónico y devuelve documentos relevantes
func (h *Handler) buscarVector(ctx context.Context, vectorID, query string) []Documento {
	out := []Documento{}
	if clientWithMeta, ok := h.AI.(AIClientWithMetadata); ok {
		if res, err := clientWithMeta.SearchInVectorStoreWithMetadata(ctx, vectorID, query); err == nil && res != nil && res.HasResult && strings.TrimSpace(res.Content) != "" {
			out = append(out, Documento{Titulo: res.Source, Contenido: strings.TrimSpace(res.Content), Fuente: "vector"})
			return out
		}
	}
	if s, err := h.AI.SearchInVectorStore(ctx, vectorID, query); err == nil && strings.TrimSpace(s) != "" {
		// Título legacy esperado por tests cuando no hay metadatos
		out = append(out, Documento{Titulo: "Base de conocimiento médico", Contenido: strings.TrimSpace(s), Fuente: "vector"})
	}
	return out
}

// buscarPubMed consulta PubMed y normaliza el contenido para evitar ruido en el prompt
func (h *Handler) buscarPubMed(ctx context.Context, query string) []Documento {
	out := []Documento{}
	raw, err := h.AI.SearchPubMed(ctx, query)
	if err != nil {
		return out
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	clean := stripModelPreambles(raw)
	// Extraer referencias crudas y filtrarlas por año >= 2020
	refsAll := extractReferenceLines(clean)
	refs := filterRefsByYear(refsAll, 2020)
	// Cuerpo depurado sin referencias embebidas ni inline
	body := removeEmbeddedReferenceSections(removeInlineReferences(clean))
	if len(body) > 2500 {
		body = body[:2500] + "..."
	}
	out = append(out, Documento{Titulo: "PubMed", Contenido: strings.TrimSpace(body), Fuente: "pubmed", Referencias: refs})
	return out
}

// fusionarResultados prepara dos bloques de contexto según contrato requerido
func fusionarResultados(vectorDocs, pubmedDocs []Documento) (ctxVec, ctxPub string) {
	if len(vectorDocs) > 0 {
		var b strings.Builder
		for _, d := range vectorDocs {
			if strings.TrimSpace(d.Contenido) == "" {
				continue
			}
			if d.Titulo != "" {
				fmt.Fprintf(&b, "- %s:\n%s\n\n", d.Titulo, d.Contenido)
			} else {
				fmt.Fprintf(&b, "- %s\n\n", d.Contenido)
			}
		}
		ctxVec = strings.TrimSpace(b.String())
	}
	if len(pubmedDocs) > 0 {
		var b strings.Builder
		for _, d := range pubmedDocs {
			if strings.TrimSpace(d.Contenido) == "" {
				continue
			}
			if d.Titulo != "" {
				fmt.Fprintf(&b, "- %s:\n%s\n\n", d.Titulo, d.Contenido)
			} else {
				fmt.Fprintf(&b, "- %s\n\n", d.Contenido)
			}
		}
		ctxPub = strings.TrimSpace(b.String())
	}
	return
}

// joinDocTitles toma documentos (vector) y devuelve títulos únicos y concisos separados por ", "
func joinDocTitles(docs []Documento) string {
	seen := map[string]bool{}
	out := []string{}
	for _, d := range docs {
		t := strings.TrimSpace(d.Titulo)
		if t == "" {
			continue
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return "(desconocido)"
	}
	// Limitar longitud total razonable
	s := strings.Join(out, ", ")
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}

// booksVectorID devuelve el ID canónico del vector de libros; sin dependencia de env
func booksVectorID() string {
	return "vs_680fc484cef081918b2b9588b701e2f4"
}

// classifyErr produce un code simbólico para facilitar observabilidad lado cliente.
func classifyErr(err error) string {
	if err == nil {
		return ""
	}
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
	var req struct {
		ThreadID string `json:"thread_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.ThreadID) == "" {
		log.Printf("[conv][Delete][error] bind thread_id=%s err=%v", req.ThreadID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "thread_id requerido"})
		return
	}
	log.Printf("[conv][Delete][begin] thread=%s", req.ThreadID)
	_ = h.AI.DeleteThreadArtifacts(c.Request.Context(), req.ThreadID)
	log.Printf("[conv][Delete][done] thread=%s", req.ThreadID)
	c.Status(http.StatusNoContent)
}

// VectorReset: fuerza vector store limpio
func (h *Handler) VectorReset(c *gin.Context) {
	var req struct {
		ThreadID string `json:"thread_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !strings.HasPrefix(req.ThreadID, "thread_") {
		log.Printf("[conv][VectorReset][error] bind_or_invalid thread=%s err=%v", req.ThreadID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "thread_id inválido"})
		return
	}
	log.Printf("[conv][VectorReset][begin] thread=%s", req.ThreadID)
	vsID, err := h.AI.ForceNewVectorStore(c.Request.Context(), req.ThreadID)
	if err != nil {
		log.Printf("[conv][VectorReset][error] force_new err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)})
		return
	}
	log.Printf("[conv][VectorReset][done] thread=%s vs=%s", req.ThreadID, vsID)
	c.JSON(http.StatusOK, gin.H{"status": "reset", "vector_store_id": vsID})
}

// VectorFiles: lista archivos
func (h *Handler) VectorFiles(c *gin.Context) {
	threadID := strings.TrimSpace(c.Query("thread_id"))
	if !strings.HasPrefix(threadID, "thread_") {
		log.Printf("[conv][VectorFiles][error] invalid_thread=%s", threadID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "thread_id inválido"})
		return
	}
	log.Printf("[conv][VectorFiles][begin] thread=%s", threadID)
	files, err := h.AI.ListVectorStoreFiles(c.Request.Context(), threadID)
	if err != nil {
		log.Printf("[conv][VectorFiles][error] list err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)})
		return
	}
	vsID := h.AI.GetVectorStoreID(threadID)
	log.Printf("[conv][VectorFiles][ok] thread=%s vs=%s files=%d", threadID, vsID, len(files))
	c.JSON(http.StatusOK, gin.H{"thread_id": threadID, "vector_store_id": vsID, "files": files})
}

// --- Helpers de formato PubMed ---
// formatPubMedAnswer crea una respuesta estructurada y limpia basada en información de PubMed
func formatPubMedAnswer(query, raw string) string {
	clean := stripModelPreambles(raw)
	refs := extractReferenceLines(clean)

	// Limpiar el contenido principal eliminando referencias duplicadas del cuerpo
	body := removeInlineReferences(clean)

	// NUEVO: Eliminar secciones de referencias en el cuerpo del texto que duplican información
	body = removeEmbeddedReferenceSections(body)

	body = strings.TrimSpace(body)

	if body == "" {
		body = "No se encontró información suficiente en PubMed para responder de manera precisa."
	}

	b := &strings.Builder{}

	// Respuesta principal directa (sin preámbulos)
	fmt.Fprintf(b, "%s\n\n", body)

	// Indicar fuente al final del contenido principal
	b.WriteString("*(Fuente: PubMed)*\n\n")

	// Sección de referencias al final - SOLO UNA VEZ
	b.WriteString("**Referencias:**\n")
	if len(refs) > 0 {
		for _, r := range refs {
			// Formatear referencias de manera consistente
			formattedRef := formatReference(r)
			b.WriteString("- ")
			b.WriteString(formattedRef)
			b.WriteString("\n")
		}
	} else {
		// Fallback a referencia general de PubMed
		b.WriteString("- PubMed: https://pubmed.ncbi.nlm.nih.gov/\n")
	}

	return b.String()
}

// removeEmbeddedReferenceSections elimina secciones de "Referencias:" que aparecen en el cuerpo del texto
func removeEmbeddedReferenceSections(s string) string {
	// Eliminar secciones completas de referencias que aparecen antes del final
	// Patrón para encontrar "Referencias:" o "Referencias seleccionadas:" seguido de una lista con viñetas
	referencePattern := regexp.MustCompile(`(?i)referencias?(?:\s+seleccionadas)?\s*:?[\t ]*\n(?:[\t ]*-[^\n]*\n?)*`)

	// También eliminar secciones inline: desde "Referencias:" hasta el final del texto
	inlinePattern := regexp.MustCompile(`(?is)(?i)\breferencias[^:]*:\s*.*$`)

	// También eliminar la pregunta al final que aparece en algunas respuestas
	questionPattern := regexp.MustCompile(`¿[^?]*\?$`)

	result := referencePattern.ReplaceAllString(s, "")
	result = inlinePattern.ReplaceAllString(result, "")
	result = questionPattern.ReplaceAllString(result, "")

	// Limpiar saltos de línea excesivos que pudieran haber quedado
	result = regexp.MustCompile(`\n\s*\n\s*\n`).ReplaceAllString(result, "\n\n")

	return strings.TrimSpace(result)
}

// removeInlineReferences elimina referencias que aparecen en el cuerpo del texto
func removeInlineReferences(s string) string {
	// Eliminar patrones comunes de referencias inline
	patterns := []string{
		`\(PMID:\s*\d+\)`,
		`\[PMID:\s*\d+\]`,
		`【PMID:\s*\d+】`,
		`\(doi:\s*[^\)]+\)`,
		`\[doi:\s*[^\]]+\]`,
		`\(PubMed\)`,
		`\[PubMed\]`,
	}

	result := s
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "")
	}

	// Limpiar espacios dobles y saltos de línea excesivos, preservando saltos de línea
	result = regexp.MustCompile(` {2,}`).ReplaceAllString(result, " ")
	result = regexp.MustCompile(`\n{3,}`).ReplaceAllString(result, "\n\n")

	return strings.TrimSpace(result)
}

// formatReference formatea una referencia de manera consistente
func formatReference(ref string) string {
	ref = strings.TrimSpace(ref)

	// Si ya tiene formato de referencia académica, mantenerlo
	if strings.Contains(ref, ".") && (strings.Contains(ref, ";") || strings.Contains(ref, ":")) {
		// Limpiar PMID/DOI duplicados al final si ya están en el formato
		if strings.Contains(strings.ToLower(ref), "pmid") && strings.Count(strings.ToLower(ref), "pmid") > 1 {
			// Mantener solo el primer PMID
			parts := strings.Split(ref, "PMID")
			if len(parts) > 2 {
				ref = parts[0] + "PMID" + parts[1]
			}
		}
		return ref
	}

	// Si es solo un PMID, DOI o URL, formatearlo apropiadamente
	lower := strings.ToLower(ref)
	if strings.HasPrefix(lower, "pmid") {
		return ref
	}
	if strings.HasPrefix(lower, "doi") {
		return ref
	}
	if strings.HasPrefix(lower, "http") {
		return ref
	}

	// Para otros casos, devolver tal como está
	return ref
}

// stripModelPreambles elimina frases de sistema/comando comunes y preámbulos innecesarios
func stripModelPreambles(s string) string {
	t := strings.TrimSpace(s)

	// Patrones de preámbulos a eliminar (case insensitive)
	preambles := []string{
		"claro,", "claro.", "claro:", "desde luego,", "desde luego.", "desde luego:",
		"a continuación", "a continuación,", "a continuación:",
		"realizo una búsqueda", "he realizado una búsqueda", "realicé una búsqueda",
		"procedo a buscar", "voy a buscar", "buscaré en", "consulto en",
		"investigaré en pubmed", "haré una búsqueda", "busco información",
		"según la búsqueda", "de acuerdo con la búsqueda", "según los resultados",
		"basándome en la búsqueda", "tras consultar", "después de buscar",
		"priorizando estudios", "a continuación presento", "voy a", "te proporciono",
		"aquí tienes", "aquí está", "he encontrado", "encontré que",
		"según mi búsqueda", "según mi investigación", "según mi consulta",
		"permíteme", "déjame", "me complace", "con gusto",
		"para responder", "para contestar", "en respuesta a",
	}

	lt := strings.ToLower(t)

	// Eliminar preámbulos del inicio
	for _, p := range preambles {
		if strings.HasPrefix(lt, p) {
			// Buscar el final de la oración o línea para cortar apropiadamente
			cutPos := len(p)

			// Buscar hasta el próximo punto, salto de línea o dos puntos
			remaining := t[cutPos:]
			for i, r := range remaining {
				if r == '.' || r == '\n' || r == ':' {
					cutPos += i + 1
					break
				}
			}

			t = strings.TrimSpace(t[cutPos:])
			lt = strings.ToLower(t)
			break
		}
	}

	// Eliminar líneas completas que son solo preámbulos
	lines := strings.Split(t, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lineLower := strings.ToLower(line)
		isPreamble := false

		for _, p := range preambles {
			if strings.Contains(lineLower, p) && len(line) < 100 {
				// Si es una línea corta que contiene preámbulos, probablemente sea solo preámbulo
				isPreamble = true
				break
			}
		}

		if !isPreamble {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// joinRefsForPrompt convierte un slice de referencias en una lista con viñetas apta para incrustar en un prompt
func joinRefsForPrompt(refs []string) string {
	if len(refs) == 0 {
		return "(sin referencias detectadas)"
	}
	b := &strings.Builder{}
	for i, r := range refs {
		if strings.TrimSpace(r) == "" {
			continue
		}
		if i > 10 { // limitar para no inflar el prompt
			break
		}
		fmt.Fprintf(b, "- %s\n", r)
	}
	return b.String()
}

// extractReferenceLines intenta extraer líneas que parecen citas/DOI/PMID o URLs.
func extractReferenceLines(s string) []string {
	out := []string{}
	yearRe := regexp.MustCompile(`\b(19|20)\d{2}\b`)
	volIssueRe := regexp.MustCompile(`\d+\(\d+\)`) // e.g., 50(1)
	startNumHeading := regexp.MustCompile(`^\d+[\.|\)]\s`)

	for _, ln := range strings.Split(s, "\n") {
		l := strings.TrimSpace(ln)
		if l == "" {
			continue
		}
		ll := strings.ToLower(l)

		// Evitar encabezados numerados tipo "2. Gastritis..." si no contienen señales de referencia
		if startNumHeading.MatchString(l) && !strings.Contains(ll, "pmid") && !strings.Contains(ll, "doi") {
			continue
		}

		if strings.Contains(ll, "pmid") || strings.Contains(ll, "doi:") || strings.Contains(ll, "http://") || strings.Contains(ll, "https://") || strings.Contains(ll, "pubmed") {
			out = append(out, l)
			continue
		}
		if yearRe.MatchString(l) && (volIssueRe.MatchString(l) || strings.Contains(l, ";") || strings.Contains(l, ":")) && strings.Contains(l, ".") {
			out = append(out, l)
			continue
		}
	}
	// Deduplicar manteniendo orden
	seen := map[string]bool{}
	uniq := make([]string, 0, len(out))
	for _, r := range out {
		k := strings.ToLower(strings.TrimSpace(r))
		if !seen[k] {
			seen[k] = true
			uniq = append(uniq, r)
		}
	}
	// Limitar a 5 para no saturar
	if len(uniq) > 5 {
		return uniq[:5]
	}
	return uniq
}

// filterRefsByYear filtra referencias que contengan un año >= minYear
func filterRefsByYear(refs []string, minYear int) []string {
	out := make([]string, 0, len(refs))
	yearRe := regexp.MustCompile(`\b(19|20)\d{2}\b`)
	for _, r := range refs {
		yrs := yearRe.FindAllString(r, -1)
		keep := false
		for _, ys := range yrs {
			if y, err := strconv.Atoi(ys); err == nil && y >= minYear {
				keep = true
				break
			}
		}
		if keep {
			out = append(out, r)
		}
	}
	return out
}
