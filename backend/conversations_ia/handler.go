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
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"ema-backend/openai"

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
	PollVectorStoreFileIndexed(ctx context.Context, vsID, fileID string, timeout time.Duration) error
	AddSessionBytes(threadID string, delta int64)
	CountThreadFiles(threadID string) int
	GetSessionBytes(threadID string) int64
	TranscribeFile(ctx context.Context, filePath string) (string, error)
	// Paridad adicional
	DeleteThreadArtifacts(ctx context.Context, threadID string) error
	ForceNewVectorStore(ctx context.Context, threadID string) (string, error)
	ListVectorStoreFiles(ctx context.Context, threadID string) ([]string, error)
	GetVectorStoreID(threadID string) string
	// Limpiar archivos del vector store para prevenir mixing de PDFs
	ClearVectorStoreFiles(ctx context.Context, vsID string) error
	// Nuevos métodos para búsqueda específica en RAG y PubMed
	SearchInVectorStore(ctx context.Context, vectorStoreID, query string) (string, error)
	SearchPubMed(ctx context.Context, query string) (string, error)
	StreamAssistantWithSpecificVectorStore(ctx context.Context, threadID, prompt, vectorStoreID string) (<-chan string, error)
	// Nueva versión que separa userMessage (se guarda en thread) de instructions (temporal para el run)
	StreamAssistantWithInstructions(ctx context.Context, threadID, userMessage, instructions, vectorStoreID string) (<-chan string, error)
	// Análisis de imágenes médicas con GPT-4o Vision
	StreamAssistantMessageWithImage(ctx context.Context, threadID, prompt, imagePath string) (<-chan string, error)
	// Obtener historial conversacional para enriquecer búsquedas
	GetThreadMessages(ctx context.Context, threadID string, limit int) ([]openai.ThreadMessage, error)
	// QuickVectorSearch retorna contenido Y el nombre real del archivo (no adivinado)
	QuickVectorSearch(ctx context.Context, vectorStoreID, query string) (*openai.VectorSearchResult, error)
	// ExtractPDFMetadataFromPath analiza un PDF y detecta si tiene texto extraíble
	ExtractPDFMetadataFromPath(filePath string) *openai.PDFMetadata
}

// SmartResponse encapsula tanto el stream generado como los metadatos necesarios para validar la respuesta antes de exponerla al usuario.
type SmartResponse struct {
	Stream           <-chan string
	Source           string
	AllowedSources   []string
	PubMedReferences []string
	Topic            TopicSnapshot
	Prompt           string
	HasVectorContext bool
	HasPubMedContext bool
	FallbackReason   string
}

// AIClientWithMetadata extiende AIClient con capacidades de metadatos
type AIClientWithMetadata interface {
	AIClient
	SearchInVectorStoreWithMetadata(ctx context.Context, vectorStoreID, query string) (*openai.VectorSearchResult, error)
}

type Handler struct {
	AI             AIClient
	quotaValidator func(ctx context.Context, c *gin.Context, flow string) error
	topicMu        sync.RWMutex
	threadTopics   map[string]*topicState
}

// topicState persiste la información temática por hilo para mantener coherencia entre preguntas y respuestas.
type topicState struct {
	Keywords     []string
	LastPrompt   string
	MessageCount int
	LastUpdated  time.Time
}

// TopicSnapshot representa una lectura inmutable del estado temático de un hilo en el momento de la petición.
type TopicSnapshot struct {
	ThreadID       string
	Keywords       []string
	IsFirstMessage bool
	MessageCount   int
}

var keywordCleaner = strings.NewReplacer(
	",", " ", ".", " ", ";", " ", ":", " ", "!", " ", "?", " ", "¿", " ", "¡", " ", "(", " ", ")", " ",
	"[", " ", "]", " ", "{", " ", "}", " ", "\n", " ", "\t", " ", "\r", " ", "\"", " ", "'", " ",
	"-", " ", "_", " ", "/", " ", "\\", " ",
)

var topicStopwords = map[string]struct{}{
	"el": {}, "la": {}, "los": {}, "las": {}, "un": {}, "una": {}, "unos": {}, "unas": {},
	"de": {}, "del": {}, "al": {}, "a": {}, "en": {}, "por": {}, "para": {}, "con": {}, "sin": {},
	"que": {}, "qué": {}, "cual": {}, "cuál": {}, "cuales": {}, "cuáles": {}, "como": {}, "cómo": {},
	"es": {}, "son": {}, "ser": {}, "estar": {}, "hay": {}, "sobre": {}, "segun": {}, "según": {},
	"the": {}, "and": {}, "for": {}, "from": {}, "into": {}, "about": {}, "what": {}, "when": {},
	"which": {}, "that": {}, "this": {}, "these": {}, "those": {}, "can": {}, "could": {}, "would": {},
}

func (h *Handler) snapshotTopic(threadID string) TopicSnapshot {
	h.topicMu.RLock()
	defer h.topicMu.RUnlock()
	st, ok := h.threadTopics[threadID]
	if !ok || st == nil {
		return TopicSnapshot{ThreadID: threadID, IsFirstMessage: true}
	}
	snap := TopicSnapshot{
		ThreadID:       threadID,
		Keywords:       append([]string{}, st.Keywords...),
		MessageCount:   st.MessageCount,
		IsFirstMessage: st.MessageCount == 0,
	}
	return snap
}

func (h *Handler) recordTopicInteraction(threadID, prompt string, resp *SmartResponse) {
	if resp == nil {
		return
	}
	keywords := extractTopicKeywords(prompt, resp.Topic.Keywords)
	h.topicMu.Lock()
	defer h.topicMu.Unlock()
	st, ok := h.threadTopics[threadID]
	if !ok || st == nil {
		st = &topicState{}
		h.threadTopics[threadID] = st
	}
	st.LastPrompt = prompt
	st.MessageCount++
	st.LastUpdated = time.Now()
	if len(keywords) > 0 {
		st.Keywords = keywords
	}
}

func extractTopicKeywords(prompt string, fallback []string) []string {
	lowered := strings.ToLower(prompt)
	cleaned := keywordCleaner.Replace(lowered)
	tokens := strings.Fields(cleaned)
	if len(tokens) == 0 {
		return append([]string{}, fallback...)
	}
	seen := make(map[string]struct{}, len(tokens))
	var out []string
	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if len(tok) < 4 {
			continue
		}
		if _, skip := topicStopwords[tok]; skip {
			continue
		}
		if _, done := seen[tok]; done {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
		if len(out) >= 6 {
			break
		}
	}
	if len(out) == 0 {
		return append([]string{}, fallback...)
	}
	return out
}

// Documento representa una pieza de contexto recuperado
type Documento struct {
	Titulo    string
	Contenido string
	Fuente    string // "vector" o "pubmed"
	// Referencias bibliográficas asociadas al contenido (cuando la fuente es PubMed)
	Referencias []string
	// Metadatos del PDF si está disponible (para formateo APA)
	Metadata interface{} // PDFMetadata
}

// buildContextualizedQuery enriquece el prompt actual con contexto conversacional previo
// para mejorar la relevancia de las búsquedas en vector stores, especialmente en preguntas
// de seguimiento como "Y cuál sería el tratamiento?" que necesitan contexto de la pregunta anterior.
func (h *Handler) buildContextualizedQuery(ctx context.Context, threadID, currentPrompt string) string {
	// Si el prompt ya es largo y detallado, probablemente no necesita enriquecimiento
	if len(currentPrompt) > 100 {
		return currentPrompt
	}

	// Obtener últimos 3 mensajes del historial (para no sobrecargar)
	messages, err := h.AI.GetThreadMessages(ctx, threadID, 6) // 3 pares user+assistant
	if err != nil || len(messages) == 0 {
		log.Printf("[conv][contextualize][no_history] thread=%s using_original_prompt", threadID)
		return currentPrompt
	}

	// Construir contexto con los últimos intercambios
	var contextParts []string

	// Tomar máximo los últimos 2 mensajes (1 user + 1 assistant) para mantener contexto reciente
	recentCount := 0
	for i := len(messages) - 1; i >= 0 && recentCount < 2; i-- {
		msg := messages[i]
		// Truncar mensajes muy largos para no exceder límites
		content := msg.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		contextParts = append([]string{fmt.Sprintf("%s: %s", msg.Role, content)}, contextParts...)
		recentCount++
	}

	if len(contextParts) == 0 {
		return currentPrompt
	}

	// Construir query enriquecida
	enrichedQuery := fmt.Sprintf("Contexto previo:\n%s\n\nPregunta actual: %s",
		strings.Join(contextParts, "\n"),
		currentPrompt)

	log.Printf("[conv][contextualize][enriched] thread=%s original_len=%d enriched_len=%d",
		threadID, len(currentPrompt), len(enrichedQuery))

	return enrichedQuery
}

// buildConversationContext obtiene los últimos N mensajes del historial como contexto
// para que el Assistant mantenga coherencia temática sin contaminar las búsquedas vectoriales
func (h *Handler) buildConversationContext(ctx context.Context, threadID string, limit int) string {
	fetchStart := time.Now()
	messages, err := h.AI.GetThreadMessages(ctx, threadID, limit)
	log.Printf("[conv][buildContext][fetch] thread=%s limit=%d elapsed_ms=%d err=%v",
		threadID, limit, time.Since(fetchStart).Milliseconds(), err)
	if err != nil || len(messages) == 0 {
		return ""
	}

	var contextParts []string
	// Tomar solo los últimos 2-3 intercambios relevantes para contexto compacto
	count := 0
	maxContext := 3
	for i := len(messages) - 1; i >= 0 && count < maxContext; i-- {
		msg := messages[i]
		// Solo incluir mensajes de usuario para contexto temático
		if msg.Role == "user" {
			content := strings.TrimSpace(msg.Content)

			// Validación básica: ignorar mensajes vacíos o anormalmente largos (errores)
			// Con StreamAssistantWithInstructions, los mensajes deben ser preguntas cortas (<500 chars típicamente)
			if content == "" || len(content) > 500 {
				log.Printf("[conv][buildContext][skip] thread=%s msg_len=%d (too_long_or_empty)", threadID, len(content))
				continue
			}

			// Truncar a 150 caracteres para mantener prompt compacto
			if len(content) > 150 {
				content = content[:150] + "..."
			}
			contextParts = append([]string{fmt.Sprintf("- %s", content)}, contextParts...)
			count++
		}
	}

	if len(contextParts) == 0 {
		return ""
	}

	context := fmt.Sprintf("═══ CONTEXTO CONVERSACIONAL PREVIO ═══\n"+
		"El usuario ha hablado previamente sobre:\n%s\n\n"+
		"🔴 REGLA CRÍTICA DE COHERENCIA TEMÁTICA:\n"+
		"Si la nueva pregunta parece genérica o de seguimiento (ej: 'qué pacientes están exentos?', 'cuál es el tratamiento?', 'cuáles son las indicaciones?'), "+
		"DEBES ASUMIR que se refiere al ÚLTIMO TEMA específico mencionado arriba.\n\n"+
		"Ejemplos:\n"+
		"- Contexto previo: 'tumor de Frantz' → Nueva pregunta: 'qué pacientes están exentos de quimioterapia?' → Responde sobre FRANTZ, no sobre oncología general\n"+
		"- Contexto previo: 'enfermedad de Crohn' → Nueva pregunta: 'cuál es el tratamiento?' → Responde sobre CROHN, no sobre tratamientos generales\n"+
		"- Contexto previo: 'apendicitis' → Nueva pregunta: 'qué complicaciones puede tener?' → Responde sobre APENDICITIS\n\n"+
		"⚠️ SOLO responde de forma genérica si el contexto previo está VACÍO o si la nueva pregunta menciona explícitamente un tema NUEVO Y DIFERENTE.\n\n",
		strings.Join(contextParts, "\n"))

	log.Printf("[conv][buildContext][ok] thread=%s msgs=%d chars=%d", threadID, len(contextParts), len(context))
	return context
}

// SmartMessage implementa el flujo mejorado: 1) RAG específico, 2) PubMed fallback, 3) citar fuente
func (h *Handler) SmartMessage(ctx context.Context, threadID, prompt, focusDocID string, snap TopicSnapshot) (*SmartResponse, error) {
	resp := &SmartResponse{
		Topic:  snap,
		Prompt: prompt,
	}
	targetVectorID := booksVectorID()

	if focusDocID != "" {
		docOnlyPrompt := fmt.Sprintf(`Responde a la consulta usando EXCLUSIVAMENTE la información contenida en el documento con ID: %s

Pregunta del usuario: %s

Instrucciones:
- No utilices conocimiento externo ni otras fuentes.
- Si el documento no contiene información suficiente para responder, di claramente: "El documento no contiene información para responder esta pregunta".
- Cita el nombre del archivo específico si está disponible.
- Responde de forma clara y natural, sin etiquetas ni secciones marcadas.
- Al final incluye: "Fuente: [Nombre del documento]"`, focusDocID, prompt)

		vsID, err := h.ensureVectorStoreID(ctx, threadID)
		if err != nil {
			return nil, err
		}
		stream, err := h.AI.StreamAssistantWithSpecificVectorStore(ctx, threadID, docOnlyPrompt, vsID)
		if err != nil {
			return nil, err
		}
		resp.Stream = stream
		resp.Source = "focus_doc"
		trimmed := strings.TrimSpace(focusDocID)
		if trimmed != "" {
			resp.AllowedSources = append(resp.AllowedSources, trimmed)
		}
		resp.AllowedSources = append(resp.AllowedSources, "documentos adjuntos del hilo")
		if len(resp.AllowedSources) > 1 {
			sort.Strings(resp.AllowedSources)
		}
		resp.HasVectorContext = true
		return resp, nil
	}

	hasDocs := h.threadHasDocuments(ctx, threadID)
	refersToDoc := h.questionRefersToDocument(prompt)
	log.Printf("[conv][SmartMessage][routing] thread=%s has_docs=%v refers_to_doc=%v prompt_preview=\"%s\"", threadID, hasDocs, refersToDoc, sanitizePreview(prompt))

	// FIX: Si el thread tiene documentos, SIEMPRE usar el vector store del thread
	// a menos que sea small talk. Esto cubre casos como "Capitulo 1 que dice?"
	// donde no se detecta referencia explícita pero claramente habla del PDF cargado.
	if hasDocs {
		vsID, err := h.ensureVectorStoreID(ctx, threadID)
		if err != nil {
			return nil, err
		}
		log.Printf("[conv][SmartMessage][doc_only.auto] thread=%s using_thread_vs=%s reason=thread_has_docs", threadID, vsID)

		// Obtener nombres de documentos para enriquecer el prompt
		docNames := h.getThreadDocumentNames(ctx, threadID)

		// Usar prompt mejorado con fallbacks inteligentes
		docOnlyPrompt := h.buildDocOnlyPromptEnhanced(prompt, docNames)

		stream, err := h.AI.StreamAssistantWithSpecificVectorStore(ctx, threadID, docOnlyPrompt, vsID)
		if err != nil {
			return nil, err
		}
		resp.Stream = stream
		resp.Source = "doc_only"

		// Añadir nombres de documentos si los tenemos
		if len(docNames) > 0 {
			resp.AllowedSources = docNames
		} else {
			resp.AllowedSources = []string{"documentos adjuntos del hilo"}
		}
		resp.HasVectorContext = true
		return resp, nil
	}

	if isSmallTalk(prompt) {
		reply := smallTalkReply(prompt)
		ch := make(chan string, 1)
		ch <- reply
		close(ch)
		resp.Stream = ch
		resp.Source = "smalltalk"
		resp.AllowedSources = nil
		return resp, nil
	}

	log.Printf("[conv][SmartMessage][hybrid.start] thread=%s target_vector=%s reason=general_question", threadID, targetVectorID)

	// CRÍTICO: NO enriquecer la búsqueda vectorial con contexto conversacional.
	// El vector search funciona mejor con queries limpias y específicas.
	// El contexto conversacional se añadirá DESPUÉS en el prompt al Assistant.
	// Usar el prompt original tal cual para búsquedas precisas.

	// Timeout aumentado para permitir búsquedas en PubMed que pueden tardar
	searchCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	searchStart := time.Now()
	vdocs := h.buscarVector(searchCtx, targetVectorID, prompt) // Usar prompt ORIGINAL sin enriquecer
	vectorTime := time.Since(searchStart)

	pubmedStart := time.Now()
	pdocs := h.buscarPubMed(searchCtx, prompt) // Usar prompt ORIGINAL sin enriquecer
	pubmedTime := time.Since(pubmedStart)

	log.Printf("[conv][SmartMessage][search.timing] thread=%s vector_ms=%d pubmed_ms=%d total_ms=%d",
		threadID, vectorTime.Milliseconds(), pubmedTime.Milliseconds(), time.Since(searchStart).Milliseconds())

	ctxVec, ctxPub := fusionarResultados(vdocs, pdocs)

	// Generar instrucciones APA basadas en metadatos disponibles
	apaInstructions := buildAPAInstructions(vdocs)

	log.Printf("[conv][SmartMessage][debug] thread=%s vdocs_len=%d pdocs_len=%d ctxVec_len=%d ctxPub_len=%d has_apa_instr=%v",
		threadID, len(vdocs), len(pdocs), len(ctxVec), len(ctxPub), apaInstructions != "")
	if len(vdocs) > 0 {
		for i, d := range vdocs {
			log.Printf("[conv][SmartMessage][debug.vdoc.%d] titulo=\"%s\" contenido_len=%d fuente=%s",
				i, d.Titulo, len(d.Contenido), d.Fuente)
		}
	}
	if ctxVec != "" {
		preview := ctxVec
		if len(ctxVec) > 200 {
			preview = ctxVec[:200]
		}
		log.Printf("[conv][SmartMessage][debug.ctxVec] first_200_chars=\"%s\"", preview)
	}

	allowedSourcesSet := map[string]string{}
	for _, d := range vdocs {
		name := strings.TrimSpace(d.Titulo)
		if name == "" {
			continue
		}
		allowedSourcesSet[strings.ToLower(name)] = name
	}

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
	resp.PubMedReferences = pubRefs
	refsBlock := joinRefsForPrompt(pubRefs)

	vecHas := strings.TrimSpace(ctxVec) != ""
	pubHas := strings.TrimSpace(ctxPub) != ""
	resp.HasVectorContext = vecHas
	resp.HasPubMedContext = pubHas

	// DIAGNÓSTICO DE RELEVANCIA: detectar si el contenido vectorizado es irrelevante
	if vecHas && len(vdocs) > 0 {
		lowerPrompt := strings.ToLower(prompt)
		lowerContent := strings.ToLower(ctxVec)

		// Casos conocidos de baja relevancia
		if (strings.Contains(lowerPrompt, "frantz") || strings.Contains(lowerPrompt, "pseudopapilar")) &&
			!strings.Contains(lowerContent, "frantz") && !strings.Contains(lowerContent, "pseudopapilar") &&
			!strings.Contains(lowerContent, "páncreas") {
			log.Printf("[conv][SmartMessage][WARNING] thread=%s vector_irrelevant: query='%s' got_title='%s' - vector store may lack pancreatic oncology content",
				threadID, sanitizePreview(prompt), vdocs[0].Titulo)
		}
	}

	hasValidSourceBook := false
	for _, d := range vdocs {
		if strings.TrimSpace(d.Titulo) != "" {
			hasValidSourceBook = true
			break
		}
	}

	log.Printf("[conv][SmartMessage][decision] thread=%s vecHas=%v pubHas=%v hasValidSourceBook=%v",
		threadID, vecHas, pubHas, hasValidSourceBook)

	if !vecHas && !pubHas && !hasValidSourceBook {
		msg := "No encontré información verificada en la biblioteca médica ni en PubMed para responder esta pregunta. " +
			"Por favor, reformula tu pregunta o proporciona más detalles para que pueda buscar en las fuentes médicas especializadas."
		ch := make(chan string, 1)
		ch <- msg
		close(ch)
		resp.Stream = ch
		resp.Source = "no_source"
		resp.FallbackReason = "no_results"
		return resp, nil
	}

	if !vecHas && hasValidSourceBook {
		var b strings.Builder
		for _, d := range vdocs {
			if strings.TrimSpace(d.Titulo) != "" {
				fmt.Fprintf(&b, "- %s:\nDocumento médico disponible\n\n", d.Titulo)
			}
		}
		ctxVec = strings.TrimSpace(b.String())
		vecHas = true
		resp.HasVectorContext = true
		log.Printf("[conv][SmartMessage][force_context] thread=%s generated_ctxVec_len=%d", threadID, len(ctxVec))
	}

	// Obtener contexto conversacional reciente para mantener coherencia temática
	// Esto ayuda al Assistant a entender de qué se está hablando sin contaminar las búsquedas
	// CRÍTICO: Usar contexto independiente con timeout generoso para evitar que el contexto HTTP
	// (que puede tener timeouts más cortos) cancele esta operación
	contextStart := time.Now()
	contextFetchCtx, contextCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer contextCancel()
	conversationContext := h.buildConversationContext(contextFetchCtx, threadID, 4) // Últimos 2 intercambios
	log.Printf("[conv][SmartMessage][context_timing] thread=%s elapsed_ms=%d", threadID, time.Since(contextStart).Milliseconds())

	// Determinar modo de integración: solo vector, solo PubMed, o híbrido
	var integrationMode string
	if vecHas && pubHas {
		integrationMode = "hybrid" // Integrar ambas fuentes
		resp.Source = "hybrid"
	} else if vecHas {
		integrationMode = "vector_only"
		resp.Source = "rag"
	} else {
		integrationMode = "pubmed_only"
		resp.Source = "pubmed"
	}
	log.Printf("[conv][SmartMessage][integration] thread=%s mode=%s vecHas=%v pubHas=%v has_context=%v", threadID, integrationMode, vecHas, pubHas, conversationContext != "")

	// Construir prompt adaptado al modo de integración
	var input string
	if integrationMode == "hybrid" {
		// MODO HÍBRIDO: Integrar vector store y PubMed (VERSIÓN OPTIMIZADA)
		input = fmt.Sprintf(
			"⚠️ FORMATO MARKDOWN OBLIGATORIO ⚠️\n"+
				"- ## Headers con \\n\\n antes\n"+
				"- Listas (-, 1.), **negritas**, ## Fuentes AL FINAL\n\n"+
				"Asistente médico experto. Respuestas SOLO de fuentes verificadas.\n\n"+
				"%s"+ // Contexto conversacional
				"TIPO DE CONSULTA:\n"+
				"A) CLÍNICA: síntomas, edad, 'Tengo X' → Razonamiento interno + Hipótesis\n"+
				"B) TEÓRICA: 'Qué es X' → Definición + Fisio + Dx + Tx\n\n"+
				"SI CLÍNICA (A):\n"+
				"INTERNO (no muestres): Demografía, Síntomas TODOS, Signos alarma, 3 Hipótesis (dx/probabilidad/criterios)\n"+
				"Reglas: ACUMULA datos mensajes previos, NO resetees, mantén coherencia temática\n"+
				"RESPUESTA MÉDICO-A-MÉDICO: Lenguaje técnico, 300-500 palabras\n"+
				"Estructura: Análisis → Diferenciales → Recomendaciones\n"+
				"NO uses '[STATE]'. Fluye como colegas.\n\n"+
				"SI TEÓRICA (B):\n"+
				"🔴 Si contexto previo habla tema específico (ej: 'Frantz') y pregunta genérica ('tratamiento?'),\n"+
				"contextualiza al tema previo (Tx Frantz, NO Tx general).\n"+
				"MÉDICO-A-MÉDICO: técnico, preciso, sin simplificar\n"+
				"Estructura: Definición + Fisio + Manifestaciones + Dx + Tx (300-500 palabras)\n"+
				"Usa nomenclatura internacional, guías clínicas (ACC/AHA, NCCN, ESMO)\n\n"+
				"FUENTES:\n"+
				"Biblioteca:\n%s\n\n"+
				"PubMed:\n%s\n\n"+
				"Referencias:\n%s\n\n"+
				"Pregunta: %s\n\n"+
				"REGLAS:\n"+
				"1. USA SOLO información arriba\n"+
				"2. PRIORIZA libros + COMPLEMENTA PubMed\n"+
				"3. Si pide PMIDs → incluye PMID: ######\n"+
				"4. NO inventes fuentes\n\n"+
				"## Fuentes (OBLIGATORIO)\n\n"+
				"### 📚 Libros\n"+
				"**Título.** (año). [Libro texto médico].\n\n"+
				"### 🔬 PubMed\n"+
				"**Título artículo.** — *Revista* (PMID: ######, año).\n\n"+
				"✓ Lista TODAS las fuentes usadas\n"+
				"✓ Formato: **negritas** títulos, *itálicas* revistas\n"+
				"%s\n",
			conversationContext, ctxVec, ctxPub, refsBlock, prompt, apaInstructions,
		)
	} else if integrationMode == "vector_only" {
		input = fmt.Sprintf(
			"⚠️⚠️⚠️ INSTRUCCIÓN CRÍTICA OBLIGATORIA - FORMATO MARKDOWN ⚠️⚠️⚠️\n"+
				"TODAS tus respuestas DEBEN usar formato Markdown estructurado con:\n"+
				"- ## Título principal (usa SIEMPRE ## no #, tamaño moderado)\n"+
				"- ## Secciones (Resumen, Análisis, Recomendaciones)\n"+
				"- CRÍTICO: Agrega DOBLE salto de línea (\\n\\n) ANTES de cada ##\n"+
				"- Listas con viñetas (-) o numeradas (1.)\n"+
				"- **Negritas** para términos importantes\n"+
				"- Sección ## Fuentes AL FINAL (OBLIGATORIA)\n"+
				"NO escribas texto plano sin estructura. SIEMPRE usa Markdown.\n\n"+
				"Eres un asistente médico experto. Debes basar tus respuestas ÚNICAMENTE en fuentes verificadas.\n\n"+
				"%s"+ // Contexto conversacional si existe
				"═══ DETECCIÓN DE TIPO DE CONSULTA ═══\n"+
				"A) CONSULTA CLÍNICA: edad, síntomas, signos, o primera persona ('Tengo X', 'Me duele Y')\n"+
				"B) CONSULTA TEÓRICA: qué es X, tratamiento de Y, fisiopatología de Z\n\n"+
				"═══ SI ES CONSULTA CLÍNICA (tipo A) ═══\n"+
				"RAZONAMIENTO INTERNO (NO MUESTRES AL USUARIO):\n"+
				"Mentalmente construye: Demografía, Síntomas (TODOS), Duración, Signos alarma, 3 Hipótesis con probabilidad, Decisiones previas\n"+
				"Reglas: ACUMULA datos de todos los mensajes, NO resetees, 'Ahora supón X empeora' = mantén previos + añade cambios\n"+
				"MANTÉN coherencia temática: si contexto habla de tema específico, probablemente esta pregunta se refiere al MISMO tema\n\n"+
				"RESPUESTA AL USUARIO (MÉDICO-A-MÉDICO):\n"+
				"Lenguaje técnico, terminología médica precisa, tono profesional dirigido a médicos.\n"+
				"Estructura: Análisis clínico + Diagnósticos diferenciales + Recomendaciones (300-500 palabras).\n"+
				"NO incluyas marcadores '[STATE]', 'Demografía:', etc. Fluye como discusión entre colegas.\n\n"+
				"═══ SI ES CONSULTA TEÓRICA (tipo B) ═══\n"+
				"TONO: MÉDICO-A-MÉDICO (técnico, preciso, sin simplificaciones).\n"+
				"Estructura: Definición + Fisiopatología + Manifestaciones + Diagnóstico + Tratamiento (300-500 palabras).\n"+
				"Usa nomenclatura médica internacional, menciona guías clínicas relevantes.\n\n"+
				"Contexto (Biblioteca Médica):\n%s\n\n"+
				"Pregunta del usuario:\n%s\n\n"+
				"═══ REGLAS GENERALES ═══\n"+
				"1. Respuestas completas y técnicamente precisas\n"+
				"2. SOLO información de la Biblioteca Médica\n"+
				"3. PROHIBIDO conocimiento general\n"+
				"4. Tono: PROFESIONAL MÉDICO (no para pacientes)\n"+
				"5. Si pide N hipótesis/signos → da EXACTAMENTE ese número\n\n"+
				"═══ FORMATO DE SALIDA — MARKDOWN ESTRUCTURADO ═══\n"+
				"OBLIGATORIO usar encabezados Markdown (#, ##, ###), listas (-, 1.), negritas **...**, itálicas *...*, y citas con >.\n"+
				"PROHIBIDO usar bloques de código (```), XML/HTML o JSON en la salida visible.\n"+
				"NO incluyas etiquetas como [STATE], [INTERNAL], ni preámbulos del tipo 'A continuación...'.\n"+
				"Extensión: clara y suficiente; evita párrafos kilométricos (máx. 6–8 líneas por párrafo).\n\n"+
				"Estructura sugerida (adapta nombres según el tema):\n"+
				"# Título breve y específico\n"+
				"## Resumen\n"+
				"- Punto clave 1\n"+
				"- Punto clave 2\n"+
				"## Desarrollo/Análisis\n"+
				"- Hallazgo o razonamiento 1\n"+
				"- Hallazgo o razonamiento 2\n"+
				"> Alerta/nota crítica (si aplica)\n"+
				"## Recomendaciones / Pasos siguientes\n"+
				"- Acción 1\n"+
				"- Acción 2\n\n"+
				"═══ BIBLIOGRAFÍA — NUNCA OMITIR ═══\n"+
				"⚠️ REGLA CRÍTICA: Tu respuesta DEBE terminar con la sección ## Fuentes.\n"+
				"NO es opcional. SIEMPRE incluye esta sección, aunque uses una sola fuente.\n\n"+
				"## Fuentes\n\n"+
				"### 📚 Libros de Texto Médico\n"+
				"- Formato: **Título del libro.** (año). [Libro de texto médico].\n"+
				"- Lista TODOS los libros que usaste con viñetas (-).\n"+
				"- USA **negritas** para títulos.\n"+
				"%s\n",
			conversationContext, ctxVec, prompt, apaInstructions,
		)
	} else {
		// MODO PUBMED ONLY (OPTIMIZADO)
		input = fmt.Sprintf(
			"⚠️ FORMATO MARKDOWN OBLIGATORIO ⚠️\n"+
				"## Headers (\\n\\n antes), listas (-, 1.), **negritas**, ## Fuentes AL FINAL\n\n"+
				"Asistente médico experto. SOLO evidencia de PubMed.\n\n"+
				"%s"+ // Contexto conversacional
				"TIPO:\n"+
				"A) CLÍNICA: síntomas/edad/'Tengo X' → Interno: Demografía, Síntomas TODOS, 3 Hipótesis. Respuesta MÉDICO-A-MÉDICO técnica\n"+
				"B) TEÓRICA: 'Qué es X' → Definición + Fisio + Manifestaciones + Dx + Tx\n"+
				"Mantén coherencia temática con contexto previo.\n\n"+
				"PubMed:\n%s\n\n"+
				"Referencias:\n%s\n\n"+
				"Pregunta: %s\n\n"+
				"REGLAS:\n"+
				"1. SOLO PubMed arriba\n"+
				"2. Si pide PMIDs → incluye PMID: ######\n"+
				"3. Si pide N → da EXACTAMENTE N\n"+
				"4. Tono: MÉDICO-A-MÉDICO (300-500 palabras)\n"+
				"5. NO uses [STATE], bloques código, XML/HTML\n\n"+
				"## Fuentes\n"+
				"### 🔬 PubMed\n"+
				"**Título.** — *Revista* (PMID: ######, año).\n"+
				"Lista TODOS los artículos usados.\n",
			conversationContext, ctxPub, refsBlock, prompt,
		)
	}

	if err := ctx.Err(); err != nil {
		log.Printf("[conv][SmartMessage][context.cancelled] thread=%s err=%v", threadID, err)
		msg := "La consulta tardó demasiado tiempo. Por favor, intenta con una pregunta más específica."
		ch := make(chan string, 1)
		ch <- msg + "\n\n*(Tiempo de espera agotado)*"
		close(ch)
		resp.Stream = ch
		resp.Source = "timeout"
		resp.FallbackReason = "search_timeout"
		return resp, nil
	}

	// NUEVO: Separar userMessage (pregunta original) de instructions (contexto completo)
	// Solo el userMessage se guarda en el thread, instructions se usa solo para el run
	stream, err := h.AI.StreamAssistantWithInstructions(ctx, threadID, prompt, input, targetVectorID)
	if err != nil {
		return nil, err
	}
	resp.Stream = stream
	// resp.Source ya fue asignado según integrationMode, no sobrescribir aquí

	if len(allowedSourcesSet) > 0 {
		for _, v := range allowedSourcesSet {
			resp.AllowedSources = append(resp.AllowedSources, v)
		}
		if len(resp.AllowedSources) > 1 {
			sort.Strings(resp.AllowedSources)
		}
	}
	return resp, nil
}

// appendMissingBookSources añade automáticamente libros que se usaron pero no se citaron
func appendMissingBookSources(response string, vdocs []Documento) string {
	if len(vdocs) == 0 {
		return response
	}

	// Verificar si hay sección "## Fuentes:"
	if !strings.Contains(response, "## Fuentes:") {
		return response
	}

	// Extraer qué libros se citaron
	citedBooks := make(map[string]bool)
	lines := strings.Split(response, "\n")
	inSourcesSection := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## Fuentes:") {
			inSourcesSection = true
			continue
		}
		if inSourcesSection {
			lower := strings.ToLower(line)
			for _, doc := range vdocs {
				titleLower := strings.ToLower(doc.Titulo)
				if titleLower != "" && strings.Contains(lower, titleLower) {
					citedBooks[doc.Titulo] = true
				}
			}
		}
	}

	// Construir lista de libros no citados
	missingBooks := []string{}
	for _, doc := range vdocs {
		if strings.TrimSpace(doc.Titulo) == "" {
			continue
		}
		if !citedBooks[doc.Titulo] {
			missingBooks = append(missingBooks, doc.Titulo)
		}
	}

	// Si hay libros faltantes, añadirlos
	if len(missingBooks) > 0 {
		log.Printf("[conv][appendMissing] thread detected %d uncited books: %v", len(missingBooks), missingBooks)

		// Buscar dónde insertar (después de "## Fuentes:" o "Libros:")
		parts := strings.Split(response, "## Fuentes:")
		if len(parts) == 2 {
			sourcesSection := parts[1]

			// Insertar libros al inicio de la sección de fuentes
			addition := "\nLibros:\n"
			for _, title := range missingBooks {
				addition += fmt.Sprintf("- %s. [Libro de texto médico].\n", title)
			}

			return parts[0] + "## Fuentes:" + addition + sourcesSection
		}
	}

	return response
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

func NewHandler(ai AIClient) *Handler {
	return &Handler{AI: ai, threadTopics: make(map[string]*topicState)}
}
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

	snap := h.snapshotTopic(req.ThreadID)
	resp, err := h.SmartMessage(c.Request.Context(), req.ThreadID, req.Prompt, req.FocusDocID, snap)
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
	if resp == nil || resp.Stream == nil {
		log.Printf("[conv][Message][json][smart.error] thread=%s nil_response", req.ThreadID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "respuesta vacía"})
		return
	}
	stream := resp.Stream
	source := resp.Source
	h.recordTopicInteraction(req.ThreadID, req.Prompt, resp)
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
	if len(resp.AllowedSources) > 0 {
		c.Header("X-Allowed-Sources", strings.Join(resp.AllowedSources, ","))
	}
	if len(resp.PubMedReferences) > 0 {
		c.Header("X-PubMed-References", strings.Join(resp.PubMedReferences, " | "))
	}
	// Enviar títulos de libros consultados como fallback si Assistant no los cita
	if resp.HasVectorContext && len(resp.AllowedSources) > 0 {
		c.Header("X-Vector-Books-Used", strings.Join(resp.AllowedSources, " | "))
	}
	if len(resp.Topic.Keywords) > 0 {
		c.Header("X-Topic-Keywords", strings.Join(resp.Topic.Keywords, ","))
	}
	if resp.FallbackReason != "" {
		c.Header("X-Fallback-Reason", resp.FallbackReason)
	}
	if resp.Topic.IsFirstMessage {
		c.Header("X-Topic-Is-First", "1")
	} else {
		c.Header("X-Topic-Is-First", "0")
	}
	c.Header("X-Topic-Message-Count", strconv.Itoa(resp.Topic.MessageCount))
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
		snap := h.snapshotTopic(threadID)
		resp, err := h.SmartMessage(c.Request.Context(), threadID, prompt, focusDocID, snap)
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
		if resp == nil || resp.Stream == nil {
			log.Printf("[conv][Message][multipart][smart.error] thread=%s nil_response", threadID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "respuesta vacía"})
			return
		}
		h.recordTopicInteraction(threadID, prompt, resp)
		stream := resp.Stream
		source := resp.Source
		if v, ok := c.Get("quota_remaining"); ok {
			c.Header("X-Quota-Remaining", toString(v))
		}
		c.Header("X-Assistant-Start-Ms", time.Since(start).String())
		c.Header("X-Thread-ID", threadID)
		c.Header("X-Strict-Threads", "1")
		c.Header("X-Source-Used", source) // Indicar qué fuente se usó
		if len(resp.AllowedSources) > 0 {
			c.Header("X-Allowed-Sources", strings.Join(resp.AllowedSources, ","))
		}
		if len(resp.PubMedReferences) > 0 {
			c.Header("X-PubMed-References", strings.Join(resp.PubMedReferences, " | "))
		}
		if len(resp.Topic.Keywords) > 0 {
			c.Header("X-Topic-Keywords", strings.Join(resp.Topic.Keywords, ","))
		}
		if resp.FallbackReason != "" {
			c.Header("X-Fallback-Reason", resp.FallbackReason)
		}
		if resp.Topic.IsFirstMessage {
			c.Header("X-Topic-Is-First", "1")
		} else {
			c.Header("X-Topic-Is-First", "0")
		}
		c.Header("X-Topic-Message-Count", strconv.Itoa(resp.Topic.MessageCount))
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
		snap := h.snapshotTopic(threadID)
		resp, err := h.SmartMessage(c.Request.Context(), threadID, prompt, "", snap)
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
		if resp == nil || resp.Stream == nil {
			log.Printf("[conv][Message][multipart][audio.error] thread=%s nil_response", threadID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "respuesta vacía"})
			return
		}
		h.recordTopicInteraction(threadID, prompt, resp)
		stream := resp.Stream
		source := resp.Source
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
		if len(resp.AllowedSources) > 0 {
			c.Header("X-Allowed-Sources", strings.Join(resp.AllowedSources, ","))
		}
		if len(resp.PubMedReferences) > 0 {
			c.Header("X-PubMed-References", strings.Join(resp.PubMedReferences, " | "))
		}
		if len(resp.Topic.Keywords) > 0 {
			c.Header("X-Topic-Keywords", strings.Join(resp.Topic.Keywords, ","))
		}
		if resp.FallbackReason != "" {
			c.Header("X-Fallback-Reason", resp.FallbackReason)
		}
		if resp.Topic.IsFirstMessage {
			c.Header("X-Topic-Is-First", "1")
		} else {
			c.Header("X-Topic-Is-First", "0")
		}
		c.Header("X-Topic-Message-Count", strconv.Itoa(resp.Topic.MessageCount))
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

	// Imagen -> análisis con vision (GPT-4o)
	if isImageExt(ext) {
		h.handleImage(c, threadID, prompt, upFile, tmp, start)
		return
	}

	if ext == ".pdf" {
		h.handlePDF(c, threadID, prompt, upFile, tmp, start)
		return
	}
	// Otros archivos: solo manda prompt (ignora archivo) - usar flujo inteligente
	snap := h.snapshotTopic(threadID)
	resp, err := h.SmartMessage(c.Request.Context(), threadID, prompt, "", snap)
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
	if resp == nil || resp.Stream == nil {
		log.Printf("[conv][Message][multipart][other.error] thread=%s nil_response", threadID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "respuesta vacía"})
		return
	}
	h.recordTopicInteraction(threadID, prompt, resp)
	stream := resp.Stream
	source := resp.Source
	if v, ok := c.Get("quota_remaining"); ok {
		c.Header("X-Quota-Remaining", toString(v))
	}
	c.Header("X-Assistant-Start-Ms", time.Since(start).String())
	c.Header("X-Thread-ID", threadID)
	c.Header("X-Strict-Threads", "1")
	c.Header("X-Source-Used", source) // Indicar qué fuente se usó
	if len(resp.AllowedSources) > 0 {
		c.Header("X-Allowed-Sources", strings.Join(resp.AllowedSources, ","))
	}
	if len(resp.PubMedReferences) > 0 {
		c.Header("X-PubMed-References", strings.Join(resp.PubMedReferences, " | "))
	}
	if len(resp.Topic.Keywords) > 0 {
		c.Header("X-Topic-Keywords", strings.Join(resp.Topic.Keywords, ","))
	}
	if resp.FallbackReason != "" {
		c.Header("X-Fallback-Reason", resp.FallbackReason)
	}
	if resp.Topic.IsFirstMessage {
		c.Header("X-Topic-Is-First", "1")
	} else {
		c.Header("X-Topic-Is-First", "0")
	}
	c.Header("X-Topic-Message-Count", strconv.Itoa(resp.Topic.MessageCount))
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

	// CRÍTICO: Validar si el PDF tiene texto extraíble ANTES de subirlo a OpenAI
	// Esto evita desperdiciar tiempo/recursos indexando PDFs escaneados (solo imágenes)
	if metadata := h.AI.ExtractPDFMetadataFromPath(tmp); metadata != nil {
		if !metadata.HasExtractableText {
			fname := filepath.Base(upFile.Filename)
			log.Printf("[conv][PDF][scanned_rejected] thread=%s file=%s coverage=%.1f%% pages=%d",
				threadID, fname, metadata.TextCoveragePercent, metadata.PageCount)

			msg := fmt.Sprintf(`Este documento (**%s**) parece estar compuesto principalmente por imágenes o escaneos (%.1f%% de cobertura de texto en %d páginas), por lo que no se puede leer texto directamente.

**Opciones disponibles:**
- Sube una versión del documento con texto digital (no escaneado)
- Si el documento es un escaneo, necesitarás aplicar OCR (Reconocimiento Óptico de Caracteres) antes de subirlo
- Puedes usar herramientas como Adobe Acrobat, Google Drive, o servicios online de OCR

No puedo buscar ni citar contenido de documentos que solo contienen imágenes.`,
				fname, metadata.TextCoveragePercent, metadata.PageCount)

			if v, ok := c.Get("quota_remaining"); ok {
				c.Header("X-Quota-Remaining", toString(v))
			}
			c.Header("X-PDF-Scanned", "1")
			c.Header("X-PDF-Text-Coverage", fmt.Sprintf("%.1f", metadata.TextCoveragePercent))
			c.Header("X-PDF-Pages", strconv.Itoa(metadata.PageCount))
			c.Header("X-Assistant-Start-Ms", time.Since(start).String())
			c.Header("X-Thread-ID", threadID)
			c.Header("X-Source-Used", "pdf_scanned_error")
			log.Printf("[conv][PDF][scanned_response] thread=%s file=%s", threadID, fname)

			one := make(chan string, 1)
			one <- msg
			close(one)
			stages := []string{"__STAGE__:start", "__STAGE__:pdf_validation_failed", "__STAGE__:streaming_answer"}
			sseMaybeCapture(c, wrapWithStages(stages, one), threadID)
			return
		}
		log.Printf("[conv][PDF][text_validated] thread=%s file=%s coverage=%.1f%% pages=%d has_text=true",
			threadID, upFile.Filename, metadata.TextCoveragePercent, metadata.PageCount)
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

	// CRÍTICO: Forzar creación de nuevo vector store en cada upload.
	// Esto garantiza que OpenAI no mezcle contenido de PDFs antiguos debido a:
	// 1. Cache/propagación de OpenAI donde ClearVectorStoreFiles reporta "vacío" pero archivos siguen indexados
	// 2. Archivos residuales de sesiones anteriores que no fueron eliminados correctamente
	// ForceNewVectorStore elimina el vector store anterior y crea uno completamente limpio.
	log.Printf("[conv][PDF][forcing_new_vs] thread=%s reason=prevent_file_mixing", threadID)
	vsID, err := h.AI.ForceNewVectorStore(c.Request.Context(), threadID)
	if err != nil {
		log.Printf("[conv][PDF][error] force_new_vector err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)})
		return
	}
	log.Printf("[conv][PDF][vs.ready] thread=%s vs=%s new=true", threadID, vsID)

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

	// CRÍTICO: Esperar a que el vector store termine de INDEXAR el archivo.
	// AddFileToVectorStore solo inicia el proceso; la indexación es asíncrona.
	// Hacemos polling del estado hasta que status=completed.
	// IMPORTANTE: Esto se hace SIEMPRE, incluso si no hay prompt inicial,
	// para que las preguntas subsecuentes encuentren el contenido indexado.

	// Timeout dinámico basado en tamaño del archivo
	// ~2-3 segundos por MB es una estimación conservadora para PDFs grandes
	indexTimeout := 30 * time.Second // Default mínimo
	fileSizeMB := float64(upFile.Size) / (1024 * 1024)
	if fileSizeMB > 5 {
		// Para PDFs grandes, calcular timeout dinámico: 2s por MB (reducido de 3s)
		// Máximo 60s para evitar que exceda timeouts de cliente
		calculatedTimeout := time.Duration(fileSizeMB*2) * time.Second
		if calculatedTimeout > indexTimeout {
			indexTimeout = calculatedTimeout
		}
		// Cap máximo de 60s para prevenir timeouts del cliente
		if indexTimeout > 60*time.Second {
			indexTimeout = 60 * time.Second
		}
		log.Printf("[conv][PDF][large_file] size_mb=%.2f calculated_timeout=%s", fileSizeMB, indexTimeout)
	}

	// Permitir override manual por variable de entorno
	if v := os.Getenv("VS_INDEX_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			indexTimeout = time.Duration(n) * time.Second
			log.Printf("[conv][PDF][timeout_override] manual_timeout=%s", indexTimeout)
		}
	}

	indexStart := time.Now()
	log.Printf("[conv][PDF][indexing.poll] thread=%s vs=%s file_id=%s timeout=%s size_mb=%.2f",
		threadID, vsID, fileID, indexTimeout, fileSizeMB)

	// Para PDFs muy grandes (>10MB), NO esperar a que termine la indexación completa.
	// Iniciar indexación en background y confirmar inmediatamente.
	// Esto previene timeouts del cliente mientras OpenAI procesa el archivo.
	if fileSizeMB > 10 {
		log.Printf("[conv][PDF][large_file_async] thread=%s vs=%s file_id=%s size_mb=%.2f indexing_in_background",
			threadID, vsID, fileID, fileSizeMB)
		// Espera corta (5s) para verificar que la indexación arrancó sin errores
		shortTimeout := 5 * time.Second
		if err := h.AI.PollVectorStoreFileIndexed(c.Request.Context(), vsID, fileID, shortTimeout); err != nil {
			errMsg := err.Error()
			// Solo fallar si es error permanente, no si es timeout
			if strings.Contains(errMsg, "status=failed") || strings.Contains(errMsg, "status=cancelled") {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "PDF indexing failed",
					"details": "OpenAI failed to start processing the PDF. Please try with a different file.",
				})
				return
			}
			// Timeout esperado en 5s - continuar con indexación en background
			log.Printf("[conv][PDF][large_file_async] thread=%s vs=%s indexing_started, continuing_async",
				threadID, vsID)
		} else {
			log.Printf("[conv][PDF][large_file_async] thread=%s vs=%s indexed_quickly elapsed=%s",
				threadID, vsID, time.Since(indexStart))
		}
	} else {
		// PDFs pequeños/medianos (<10MB): esperar a que termine la indexación
		if err := h.AI.PollVectorStoreFileIndexed(c.Request.Context(), vsID, fileID, indexTimeout); err != nil {
			log.Printf("[conv][PDF][indexing.error] thread=%s vs=%s file_id=%s err=%v elapsed=%s",
				threadID, vsID, fileID, err, time.Since(indexStart))

			// Diferenciar entre timeout (puede completarse después) y failed (error permanente)
			errMsg := err.Error()
			if strings.Contains(errMsg, "status=failed") || strings.Contains(errMsg, "status=cancelled") {
				// Error PERMANENTE: el archivo NO se indexará nunca
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "PDF indexing failed",
					"details": "OpenAI failed to process the PDF. This may be due to file corruption, unsupported format, or size limits. Please try with a different file.",
				})
				return
			}

			// Si es timeout, continuar (puede completarse en background)
			log.Printf("[conv][PDF][indexing.timeout.continuing] thread=%s vs=%s file_id=%s", threadID, vsID, fileID)
		} else {
			log.Printf("[conv][PDF][indexing.ready] thread=%s vs=%s file_id=%s elapsed=%s",
				threadID, vsID, fileID, time.Since(indexStart))
		}
	}

	// CRÍTICO: Esperar propagación del índice en la red de OpenAI
	// Ajustar según tamaño: PDFs grandes en background no esperan, otros esperan proporcionalmente
	var postIndexWait time.Duration
	if fileSizeMB > 10 {
		// PDFs grandes: espera mínima (2s) porque indexación sigue en background
		postIndexWait = 2 * time.Second
		log.Printf("[conv][PDF][post_index_wait] thread=%s waiting_%ds reason=minimal_wait_for_large_file",
			threadID, int(postIndexWait.Seconds()))
	} else if fileSizeMB > 1 {
		// PDFs medianos: espera moderada (5-10s)
		postIndexWait = time.Duration(5+fileSizeMB) * time.Second
		if postIndexWait > 10*time.Second {
			postIndexWait = 10 * time.Second
		}
		log.Printf("[conv][PDF][post_index_wait] thread=%s waiting_%ds reason=openai_search_propagation",
			threadID, int(postIndexWait.Seconds()))
	} else {
		// PDFs pequeños: espera corta (3s)
		postIndexWait = 3 * time.Second
		log.Printf("[conv][PDF][post_index_wait] thread=%s waiting_%ds reason=small_file_propagation",
			threadID, int(postIndexWait.Seconds()))
	}
	time.Sleep(postIndexWait)
	log.Printf("[conv][PDF][post_index_wait] thread=%s wait_complete", threadID)

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

		// Mensaje diferente según si indexación está completa o en background
		var msg string
		if fileSizeMB > 10 {
			msg = fmt.Sprintf("✅ Documento '%s' (%.1f MB, %d páginas) cargado correctamente.\n\n"+
				"⏳ La indexación está finalizando en segundo plano. "+
				"Puedes hacer preguntas ahora mismo, y el documento estará completamente disponible en 1-2 minutos.\n\n"+
				"Fuente: %s", fname, fileSizeMB, int(fileSizeMB*100), fname)
		} else {
			msg = fmt.Sprintf("✅ Documento '%s' cargado y procesado correctamente.\n\n"+
				"Puedes hacer preguntas específicas sobre este PDF.\n\n"+
				"Fuente: %s", fname, fname)
		}

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
		c.Header("X-PDF-Size-MB", fmt.Sprintf("%.2f", fileSizeMB))
		if fileSizeMB > 10 {
			c.Header("X-PDF-Indexing-Status", "background")
		} else {
			c.Header("X-PDF-Indexing-Status", "complete")
		}
		log.Printf("[conv][PDF][confirm] thread=%s file=%s size_mb=%.2f doc_only=1 elapsed_ms=%d",
			threadID, upFile.Filename, fileSizeMB, time.Since(start).Milliseconds())
		one := make(chan string, 1)
		one <- msg
		close(one)
		stages := []string{"__STAGE__:start", "__STAGE__:doc_only", "__STAGE__:streaming_answer"}
		sseMaybeCapture(c, wrapWithStages(stages, one), threadID)
		return
	}

	// Si viene prompt junto al PDF, usar el prompt mejorado con fallbacks inteligentes
	fname := filepath.Base(upFile.Filename)
	docNames := []string{fname}
	p := h.buildDocOnlyPromptEnhanced(base, docNames)

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

// handleImage procesa imágenes médicas con GPT-4o Vision (sin vector stores).
// Valida tamaño (20MB max OpenAI), sube la imagen, y hace streaming de análisis vision.
func (h *Handler) handleImage(c *gin.Context, threadID, prompt string, upFile *multipart.FileHeader, tmp string, start time.Time) {
	if upFile.Size <= 0 {
		log.Printf("[conv][IMAGE][error] empty_file thread=%s", threadID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "archivo vacío"})
		return
	}

	// Verificar tamaño (OpenAI Vision: max 20MB)
	maxFileSizeBytes := int64(20 * 1024 * 1024) // 20MB
	if upFile.Size > maxFileSizeBytes {
		sizeMB := float64(upFile.Size) / (1024 * 1024)
		log.Printf("[conv][IMAGE][error] file_too_large thread=%s size_mb=%.1f max_mb=20", threadID, sizeMB)
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error":       "imagen demasiado grande",
			"code":        "file_too_large",
			"detail":      fmt.Sprintf("La imagen pesa %.1f MB. El límite máximo para análisis de imágenes es 20 MB.", sizeMB),
			"max_size_mb": 20,
		})
		return
	}

	// Consumir cuota antes de subir la imagen
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "file_upload"); err != nil {
			field, _ := c.Get("quota_error_field")
			reason, _ := c.Get("quota_error_reason")
			resp := gin.H{"error": "image quota exceeded"}
			if f, ok := field.(string); ok && f != "" {
				resp["field"] = f
			}
			if r, ok := reason.(string); ok && r != "" {
				resp["reason"] = r
			}
			log.Printf("[conv][IMAGE][quota][denied] field=%v reason=%v", field, reason)
			c.JSON(http.StatusForbidden, resp)
			return
		}
	}

	// Validar que haya prompt (las imágenes requieren descripción/pregunta)
	base := strings.TrimSpace(prompt)
	if base == "" {
		log.Printf("[conv][IMAGE][error] empty_prompt thread=%s", threadID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "prompt requerido",
			"detail": "Para analizar una imagen debes incluir una pregunta o descripción de qué necesitas.",
		})
		return
	}

	fname := filepath.Base(upFile.Filename)
	sizeMB := float64(upFile.Size) / (1024 * 1024)
	log.Printf("[conv][IMAGE][start] thread=%s file=%s size_mb=%.2f prompt_len=%d",
		threadID, fname, sizeMB, len(base))

	// Llamar al método de vision streaming (sube imagen + crea mensaje + hace stream)
	stream, err := h.AI.StreamAssistantMessageWithImage(c.Request.Context(), threadID, base, tmp)
	if err != nil {
		log.Printf("[conv][IMAGE][error] stream err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg(err)})
		return
	}

	// Headers para tracking
	if v, ok := c.Get("quota_remaining"); ok {
		c.Header("X-Quota-Remaining", toString(v))
	}
	c.Header("X-File-Type", "image")
	c.Header("X-Image-Size-MB", fmt.Sprintf("%.2f", sizeMB))
	c.Header("X-Image-Name", fname)
	c.Header("X-Vision-Model", "gpt-4o")
	c.Header("X-Assistant-Start-Ms", time.Since(start).String())
	c.Header("X-Thread-ID", threadID)
	c.Header("X-Source-Used", "vision_analysis")

	log.Printf("[conv][IMAGE][streaming] thread=%s file=%s elapsed_ms=%d",
		threadID, fname, time.Since(start).Milliseconds())

	// Stages para el frontend
	stages := []string{"__STAGE__:start", "__STAGE__:image_upload", "__STAGE__:vision_analysis", "__STAGE__:streaming_answer"}
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

func isImageExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
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

func (h *Handler) ensureVectorStoreID(ctx context.Context, threadID string) (string, error) {
	if strings.TrimSpace(threadID) == "" {
		return "", fmt.Errorf("thread_id vacío")
	}
	if vsID := strings.TrimSpace(h.AI.GetVectorStoreID(threadID)); vsID != "" {
		return vsID, nil
	}
	return h.AI.EnsureVectorStore(ctx, threadID)
}

// threadHasDocuments determina si el hilo tiene archivos en su vector store (para activar modo doc-only)
func (h *Handler) threadHasDocuments(ctx context.Context, threadID string) bool {
	if strings.TrimSpace(threadID) == "" {
		return false
	}

	// Primero verificar contador local (actualizado inmediatamente al añadir archivo)
	// Esto evita el delay de propagación de OpenAI API (10-30s)
	localCount := h.AI.CountThreadFiles(threadID)
	if localCount > 0 {
		fmt.Printf("[threadHasDocuments] threadID=%s localCount=%d using_cache=true\n", threadID, localCount)
		return true
	}

	// Fallback a API si contador local es 0 (ej: thread creado en sesión anterior)
	files, err := h.AI.ListVectorStoreFiles(ctx, threadID)
	if err != nil {
		fmt.Printf("[threadHasDocuments] threadID=%s api_error=%v\n", threadID, err)
		return false
	}
	hasFiles := len(files) > 0
	fmt.Printf("[threadHasDocuments] threadID=%s api_files=%d using_cache=false\n", threadID, len(files))
	return hasFiles
}

// getThreadDocumentNames obtiene los nombres reales de los archivos en el vector store del thread
func (h *Handler) getThreadDocumentNames(ctx context.Context, threadID string) []string {
	vsID := h.AI.GetVectorStoreID(threadID)
	if vsID == "" {
		return []string{}
	}

	// Intentar obtener info del último archivo cargado (tiene metadata)
	// Esto es más eficiente que iterar todos los archivos
	fileIDs, err := h.AI.ListVectorStoreFiles(ctx, threadID)
	if err != nil || len(fileIDs) == 0 {
		return []string{}
	}

	names := make([]string, 0, len(fileIDs))
	// Por ahora retornamos indicador genérico; en futuro podemos iterar fileIDs
	// para obtener nombres reales usando getFileName del cliente openai
	if len(fileIDs) > 0 {
		names = append(names, fmt.Sprintf("%d documento(s) PDF", len(fileIDs)))
	}
	return names
}

// getPDFMetadata obtiene los metadatos del último PDF cargado en el thread
// Esto incluye si el PDF tiene texto extraíble o es solo imágenes
func (h *Handler) getPDFMetadata(ctx context.Context, threadID string) (*openai.PDFMetadata, error) {
	// Usar type assertion para acceder a métodos internos del cliente
	type metadataGetter interface {
		GetLastFileMetadata(threadID string) *openai.PDFMetadata
	}

	if getter, ok := h.AI.(metadataGetter); ok {
		metadata := getter.GetLastFileMetadata(threadID)
		return metadata, nil
	}

	return nil, fmt.Errorf("client does not support metadata retrieval")
}

// questionRefersToDocument detecta si la pregunta se refiere explícitamente a un documento
func (h *Handler) questionRefersToDocument(prompt string) bool {
	prompt = strings.ToLower(strings.TrimSpace(prompt))
	if prompt == "" {
		return false
	}

	// Palabras clave que indican referencia a documentos
	docKeywords := []string{
		"documento", "pdf", "archivo", "adjunto", "subido", "subí", "cargado", "cargué",
		"en el documento", "en el pdf", "en el archivo", "según el documento",
		"que dice el documento", "que dice el pdf", "en este documento",
		"del documento", "del pdf", "del archivo",
		"resumen", "resume", "resumir", "qué contiene", "contenido del",
		"explica el documento", "analiza el documento", "análisis del documento",
		// Nuevas: preguntas vagas que probablemente se refieren al documento cargado
		"qué es", "que es", "historia clínica", "capitulo", "capítulo", "sección", "seccion",
		"página", "pagina", "índice", "indice", "tabla de contenido",
	}

	for _, keyword := range docKeywords {
		if strings.Contains(prompt, keyword) {
			return true
		}
	}

	// Detectar preguntas muy cortas (≤ 5 palabras) como potencialmente referidas al documento
	// cuando el thread tiene documentos cargados
	words := strings.Fields(prompt)
	if len(words) <= 5 {
		// Preguntas cortas como "qué es el pdf?" o "historia clínica?"
		// probablemente se refieren al documento, no a conocimiento general
		return true
	}

	return false
}

// enrichQueryWithContext enriquece consultas ambiguas con contexto del documento
func (h *Handler) enrichQueryWithContext(prompt string, docNames []string) string {
	prompt = strings.TrimSpace(prompt)
	lower := strings.ToLower(prompt)

	// Si la consulta es muy vaga o corta, enriquecerla
	if len(prompt) < 20 || strings.Contains(lower, "qué es") || strings.Contains(lower, "que dice") {
		if len(docNames) > 0 {
			return fmt.Sprintf("Busca en el documento '%s' información sobre: %s. Si la pregunta es vaga, proporciona un índice o estructura del documento.", docNames[0], prompt)
		}
		return fmt.Sprintf("Busca en los PDFs adjuntos información sobre: %s. Si no encuentras contenido exacto, proporciona un índice aproximado (títulos, secciones) del documento.", prompt)
	}

	return prompt
}

// buildDocOnlyPromptEnhanced construye prompt mejorado con fallbacks inteligentes
func (h *Handler) buildDocOnlyPromptEnhanced(userPrompt string, docNames []string) string {
	enrichedPrompt := h.enrichQueryWithContext(userPrompt, docNames)

	docContext := "los PDFs adjuntos al thread"
	if len(docNames) > 0 {
		docContext = fmt.Sprintf("el documento: %s", strings.Join(docNames, ", "))
	}

	return fmt.Sprintf(`⚠️ INSTRUCCIÓN CRÍTICA: MODO DOCUMENTO PDF ⚠️

CONTEXTO: Este thread tiene documentos PDF cargados que debes consultar OBLIGATORIAMENTE.

═══ TU TAREA ═══
1. USA el tool "file_search" INMEDIATAMENTE para buscar en %s
2. Lee ÚNICAMENTE el contenido que file_search te devuelva
3. NO uses conocimiento médico general externo

═══ REGLAS DE RESPUESTA ═══

🔹 SI ENCUENTRAS INFORMACIÓN RELEVANTE:
- Responde con el contenido encontrado
- Cita textualmente fragmentos relevantes
- Termina con: "## Fuentes\n- [Nombre del archivo PDF], p. X-Y"

🔹 SI LA CONSULTA ES VAGA O AMBIGUA (ej: "¿qué es el pdf?", "historia clínica?"):
PROHIBIDO responder "no hay información". En su lugar:
A) ÍNDICE: Lista la estructura del documento (títulos, capítulos, secciones detectables)
   Formato: "El documento contiene:\n- Capítulo 1: ...\n- Capítulo 2: ...\n¿Qué sección te interesa?"
   
B) FRAGMENTOS REPRESENTATIVOS: Si no hay índice claro, muestra 3-5 fragmentos importantes con sus páginas
   Formato: "Fragmentos relevantes:\n- p. 5: [fragmento]\n- p. 12: [fragmento]\n..."
   
C) SINÓNIMOS: Si buscaste un término y no lo encontraste, sugiere términos alternativos
   Ejemplo: "No encontré 'historia clínica' exactamente. ¿Buscas: historial médico, expediente clínico, anamnesis?"

🔹 SI EL DOCUMENTO NO TIENE TEXTO EXTRAÍBLE (escaneado sin OCR):
"El documento parece ser un escaneo sin texto extraíble (OCR). Sugerencias:
- Sube una versión con OCR aplicado
- Usa herramientas de conversión como Adobe Acrobat
- Indica manualmente qué sección te interesa si ves el documento"

═══ FORMATO DE SALIDA — MARKDOWN ESTRUCTURADO ═══
OBLIGATORIO usar encabezados Markdown (#, ##, ###), listas (-, 1.), negritas **...**, itálicas *...*, y citas con >.
PROHIBIDO usar bloques de código con fences, XML/HTML o JSON en la salida visible.
NO incluyas etiquetas como [STATE], [INTERNAL], ni preámbulos del tipo 'A continuación...'.
Extensión: clara y suficiente; evita párrafos kilométricos (máx. 6–8 líneas por párrafo).

Estructura sugerida (adapta nombres según el tema):
# Título breve y específico
## Resumen
- Punto clave 1
- Punto clave 2
## Contenido del Documento
- Hallazgo 1
- Hallazgo 2
> Nota importante (si aplica)
## Fuentes
OBLIGATORIO: Cita el archivo Y páginas específicas. PROHIBIDO citar "Documentos PDF cargados" (usa nombre real del archivo).

═══ IMPORTANTE ═══
Tu objetivo NO es "opinar" ni dar teoría externa: es navegar, citar y explicar lo que está en %s,
devolviendo SIEMPRE algo útil (índice/fragmentos/citas) incluso cuando la consulta sea ambigua.

Consulta del usuario:
%s`, docContext, docContext, enrichedPrompt)
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

// normalizeMarkdownToken añade saltos de línea donde OpenAI los omite en el streaming.
// Problema: OpenAI puede enviar "# Título## Resumen- Item" sin \n entre elementos Markdown.
// Solución: Detectar patrones y forzar separación con \n\n para legibilidad.
//
// Ejemplos de transformación:
// - "# Título## Resumen" -> "# Título\n\n## Resumen"
// - "texto- Item 1- Item 2" -> "texto\n- Item 1\n- Item 2"
// - "palabra1. Primer punto" -> "palabra\n1. Primer punto"
// - "frase> Cita importante" -> "frase\n> Cita importante"
func normalizeMarkdownToken(tok string) string {
	if tok == "" {
		return ""
	}

	// 1. Headers H2/H3 pegados a texto: "texto## Header" -> "texto\n\n## Header"
	// Busca ## o ### precedidos por carácter no-blanco
	tok = regexp.MustCompile(`([^\s\n])\s*(#{2,3})\s+`).ReplaceAllString(tok, "$1\n\n$2 ")

	// 2. Headers H1 seguidos inmediatamente de H2/H3: "# Título## Resumen" -> "# Título\n\n## Resumen"
	tok = regexp.MustCompile(`(^#\s+[^\n#]+)(#{2,3})\s+`).ReplaceAllString(tok, "$1\n\n$2 ")

	// 3. Bullets/listas pegadas: "texto- Item" -> "texto\n- Item"
	// Solo si NO es una palabra con guion (ej: "médico-quirúrgico")
	// Busca espacio + guion + espacio pegado a texto anterior
	tok = regexp.MustCompile(`([a-záéíóúñA-ZÁÉÍÓÚÑ0-9.!?])\s*-\s+([A-ZÁÉÍÓÚÑ])`).ReplaceAllString(tok, "$1\n- $2")

	// 4. Listas numeradas pegadas: "texto1. Item" -> "texto\n1. Item"
	// Solo cuando el número está precedido por letra/puntuación (no otro número)
	tok = regexp.MustCompile(`([a-záéíóúñA-ZÁÉÍÓÚÑ.!?])\s*(\d+\.\s+[A-ZÁÉÍÓÚÑ])`).ReplaceAllString(tok, "$1\n$2")

	// 5. Citas pegadas: "texto> Cita" -> "texto\n> Cita"
	tok = regexp.MustCompile(`([a-záéíóúñA-ZÁÉÍÓÚÑ.!?])\s*>\s+`).ReplaceAllString(tok, "$1\n> ")

	// 6. Espaciado alrededor de "## Fuentes" - sección crítica que debe estar bien separada
	tok = regexp.MustCompile(`([^\n])(##\s+Fuentes)`).ReplaceAllString(tok, "$1\n\n$2")

	// 7. Si el token termina con un header sin contenido (ej: "## "), añade salto
	// para que el siguiente token empiece limpio
	trimmed := strings.TrimRight(tok, " \t")
	if strings.HasSuffix(trimmed, "#") || strings.HasSuffix(trimmed, "##") || strings.HasSuffix(trimmed, "###") {
		tok = trimmed + "\n\n"
	}

	return tok
}

// Casos de prueba esperados (ejecutar con go test si se crea handler_test.go):
// normalizeMarkdownToken("# Diagnósticos## Resumen") → "# Diagnósticos\n\n## Resumen"
// normalizeMarkdown Token("texto- Punto 1- Punto 2") → "texto\n- Punto 1\n- Punto 2"
// normalizeMarkdownToken("Análisis1. Primer diagnóstico") → "Análisis\n1. Primer diagnóstico"
// normalizeMarkdownToken("Nota> Importante") → "Nota\n> Importante"
// normalizeMarkdownToken("médico-quirúrgico- Punto") → "médico-quirúrgico\n- Punto" (preserva guion interno)

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
		// CRÍTICO: Normalizar markdown porque OpenAI envía tokens sin saltos de línea
		// Ejemplo: "## DefiniciónLa gastritis..." → "## Definición\n\nLa gastritis..."
		normalized := normalizeMarkdownToken(tok)
		_, _ = c.Writer.Write([]byte("data: " + normalized + "\n\n"))
		c.Writer.Flush()
	}
	// CRÍTICO: Enviar marcador [DONE] al final del stream
	// El frontend espera este marcador para cerrar correctamente el stream
	_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
	c.Writer.Flush()
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

// SourceExtracted contiene fuentes extraídas de respuestas del Assistant
type SourceExtracted struct {
	SourceBooks []string // Nombres de libros/documentos específicos encontrados
	HasSources  bool     // Indica si se encontraron fuentes válidas
}

// extractSourcesFromAssistantResponse extrae fuentes de la respuesta JSON del Assistant
func extractSourcesFromAssistantResponse(response string) *SourceExtracted {
	result := &SourceExtracted{
		SourceBooks: []string{},
		HasSources:  false,
	}

	if response == "" {
		return result
	}

	// Buscar patrones de source_book en la respuesta
	sourceBookRegex := regexp.MustCompile(`"source_book"\s*:\s*"([^"]+)"`)
	matches := sourceBookRegex.FindAllStringSubmatch(response, -1)

	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			sourceBook := strings.TrimSpace(match[1])
			if sourceBook != "" && !seen[sourceBook] {
				// Limpiar nombre del archivo (quitar extensión si es necesaria)
				cleanName := sourceBook
				if strings.HasSuffix(strings.ToLower(cleanName), ".pdf") {
					cleanName = strings.TrimSuffix(cleanName, ".pdf")
					cleanName = strings.TrimSuffix(cleanName, ".PDF")
				}
				result.SourceBooks = append(result.SourceBooks, cleanName)
				seen[sourceBook] = true
				result.HasSources = true
			}
		}
	}

	return result
}

// buscarVectorConFuentes usa fallback directo por ahora - el problema de fuentes se resolverá a nivel de respuesta
func (h *Handler) buscarVectorConFuentes(ctx context.Context, vectorID, query string) []Documento {
	if metaClient, ok := h.AI.(AIClientWithMetadata); ok && strings.TrimSpace(vectorID) != "" {
		res, err := metaClient.SearchInVectorStoreWithMetadata(ctx, vectorID, query)
		if err != nil {
			log.Printf("[conv][buscarVectorConFuentes][metadata_error] vector=%s err=%v", vectorID, err)
		} else if res != nil && res.HasResult {
			title := strings.TrimSpace(res.Source)
			if title == "" {
				title = "Documento médico"
			}
			content := strings.TrimSpace(res.Content)
			if content == "" && strings.TrimSpace(res.Section) != "" {
				content = fmt.Sprintf("Sección relevante: %s", strings.TrimSpace(res.Section))
			}
			return []Documento{{
				Titulo:    title,
				Contenido: content,
				Fuente:    "vector",
				Metadata:  res.Metadata, // Incluir metadatos del PDF
			}}
		}
		log.Printf("[conv][buscarVectorConFuentes][metadata_empty] vector=%s", vectorID)
	}
	log.Printf("[conv][buscarVectorConFuentes][fallback] vector=%s", vectorID)
	return h.buscarVectorFallback(ctx, vectorID, query)
}

// buscarVectorFallback usa quickVectorSearch que retorna el nombre REAL del archivo desde el vector store
func (h *Handler) buscarVectorFallback(ctx context.Context, vectorID, query string) []Documento {
	log.Printf("[conv][buscarVectorFallback][start] vector=%s", vectorID)

	out := []Documento{}

	// Usar quickVectorSearch en lugar de SearchInVectorStore para obtener el FileID y nombre real
	if result, err := h.AI.QuickVectorSearch(ctx, vectorID, query); err == nil && result.HasResult && strings.TrimSpace(result.Content) != "" {
		trimmed := strings.TrimSpace(result.Content)
		if isLikelyNoDataResponse(trimmed) {
			log.Printf("[conv][buscarVectorFallback][skip_no_data] vector=%s msg=%s", vectorID, sanitizePreview(trimmed))
			return out
		}

		// Usar el título REAL del archivo obtenido del vector store (no adivinado)
		bookTitle := result.Source
		if strings.TrimSpace(bookTitle) == "" {
			bookTitle = "Libro de Texto Médico Especializado"
		}

		// Log mejorado con preview del contenido y título REAL extraído
		contentPreview := strings.ReplaceAll(trimmed, "\n", " ")
		if len(contentPreview) > 200 {
			contentPreview = contentPreview[:200] + "..."
		}
		log.Printf("[conv][buscarVectorFallback][ok] vector=%s result_len=%d real_source=%q content_preview=%q",
			vectorID, len(trimmed), bookTitle, contentPreview)

		out = append(out, Documento{
			Titulo:    bookTitle,
			Contenido: trimmed,
			Fuente:    "vector",
			Metadata:  result.Metadata,
		})
	} else if err != nil {
		log.Printf("[conv][buscarVectorFallback][error] vector=%s err=%v", vectorID, err)
	}
	return out
}

// expandMedicalQuery expande queries médicas con sinónimos y variantes conocidas
// para mejorar el recall en búsquedas vectoriales
func expandMedicalQuery(query string) string {
	lower := strings.ToLower(query)

	// Diccionario de expansiones médicas conocidas
	// Formato: "término principal" → [sinónimos, variantes, nombres alternativos]
	expansions := map[string][]string{
		// Tumores pancreáticos
		"frantz":          {"tumor sólido pseudopapilar", "neoplasia de Frantz", "Gruber-Frantz", "solid pseudopapillary neoplasm", "SPN pancreas"},
		"tumor de frantz": {"tumor sólido pseudopapilar", "neoplasia de Frantz", "Gruber-Frantz", "solid pseudopapillary neoplasm"},
		"pseudopapilar":   {"Frantz", "solid pseudopapillary", "SPN"},

		// Enfermedades inflamatorias intestinales
		"crohn":            {"enfermedad de Crohn", "ileítis regional", "enteritis regional", "Crohn disease"},
		"colitis ulcerosa": {"colitis ulcerativa", "proctocolitis ulcerosa", "ulcerative colitis"},
		"eii":              {"enfermedad inflamatoria intestinal", "IBD", "Crohn", "colitis ulcerosa"},

		// Síndromes y condiciones con epónimos
		"whipple":   {"lipodistrofia intestinal", "Whipple disease", "enfermedad de Whipple"},
		"zollinger": {"síndrome de Zollinger-Ellison", "ZES", "gastrinoma"},
		"barrett":   {"esófago de Barrett", "metaplasia de Barrett", "Barrett esophagus"},
		"cushing":   {"síndrome de Cushing", "hipercortisolismo", "Cushing syndrome"},

		// Tumores con nombres alternativos
		"gist":       {"tumor estromal gastrointestinal", "gastrointestinal stromal tumor", "sarcoma estromal"},
		"carcinoide": {"tumor neuroendocrino", "NET", "neuroendocrine tumor"},

		// Procedimientos quirúrgicos con epónimos
		"billroth": {"gastrectomía de Billroth", "Billroth I", "Billroth II", "gastrojejunostomía"},
		"roux":     {"Roux-en-Y", "anastomosis en Y de Roux", "derivación Roux"},
		"hartmann": {"procedimiento de Hartmann", "colostomía de Hartmann", "Hartmann procedure"},
	}

	// Buscar si algún término clave está presente
	for key, synonyms := range expansions {
		if strings.Contains(lower, key) {
			// Expandir con sinónimos para búsqueda más amplia
			expanded := query + " OR " + strings.Join(synonyms, " OR ")
			log.Printf("[conv][expandQuery] original=\"%s\" expanded_with=%d_synonyms", query, len(synonyms))
			return expanded
		}
	}

	return query
}

// buscarVector mantiene compatibilidad - ahora usa buscarVectorConFuentes con expansión automática
func (h *Handler) buscarVector(ctx context.Context, vectorID, query string) []Documento {
	// Expandir query con sinónimos médicos antes de buscar
	expandedQuery := expandMedicalQuery(query)
	return h.buscarVectorConFuentes(ctx, vectorID, expandedQuery)
}

// truncatePreview helper para logs
func truncatePreview(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
} // buscarPubMed consulta PubMed y normaliza el contenido para evitar ruido en el prompt
func (h *Handler) buscarPubMed(ctx context.Context, query string) []Documento {
	out := []Documento{}
	raw, err := h.AI.SearchPubMed(ctx, query)
	if err != nil {
		log.Printf("[conv][buscarPubMed][error] query_len=%d err=%v", len(query), err)
		return out
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	if isLikelyNoDataResponse(raw) {
		log.Printf("[conv][buscarPubMed][skip_error] message=%s", sanitizePreview(raw))
		return out
	}

	// Intentar parsear respuesta estructurada en JSON
	type pubmedStudy struct {
		Title     string   `json:"title"`
		PMID      string   `json:"pmid"`
		Year      int      `json:"year"`
		Journal   string   `json:"journal"`
		KeyPoints []string `json:"key_points"`
		Summary   string   `json:"summary"`
	}
	type pubmedPayload struct {
		Summary string        `json:"summary"`
		Studies []pubmedStudy `json:"studies"`
	}
	var payload pubmedPayload
	if err := json.Unmarshal([]byte(raw), &payload); err == nil && len(payload.Studies) > 0 {
		refs := make([]string, 0, len(payload.Studies))
		bodyParts := make([]string, 0, len(payload.Studies)+1)
		if summary := strings.TrimSpace(payload.Summary); summary != "" {
			bodyParts = append(bodyParts, summary)
		}
		for _, st := range payload.Studies {
			title := strings.TrimSpace(st.Title)
			pmid := strings.TrimSpace(st.PMID)
			if title == "" || pmid == "" {
				continue
			}
			year := st.Year
			if year > 0 && year < 2018 {
				continue
			}
			keyPoints := make([]string, 0, len(st.KeyPoints))
			for _, kp := range st.KeyPoints {
				if trimmed := strings.TrimSpace(kp); trimmed != "" {
					keyPoints = append(keyPoints, trimmed)
				}
			}
			if len(keyPoints) == 0 {
				if s := strings.TrimSpace(st.Summary); s != "" {
					keyPoints = append(keyPoints, s)
				}
			}
			info := strings.Join(keyPoints, "; ")
			if info != "" {
				if year > 0 {
					bodyParts = append(bodyParts, fmt.Sprintf("- %s (%d, PMID %s): %s", title, year, pmid, info))
				} else {
					bodyParts = append(bodyParts, fmt.Sprintf("- %s (PMID %s): %s", title, pmid, info))
				}
			} else {
				if year > 0 {
					bodyParts = append(bodyParts, fmt.Sprintf("- %s (%d, PMID %s)", title, year, pmid))
				} else {
					bodyParts = append(bodyParts, fmt.Sprintf("- %s (PMID %s)", title, pmid))
				}
			}
			refLabel := title
			if journal := strings.TrimSpace(st.Journal); journal != "" {
				refLabel = fmt.Sprintf("%s — %s", refLabel, journal)
			}
			if year > 0 {
				refs = append(refs, fmt.Sprintf("%s (PMID: %s, %d)", refLabel, pmid, year))
			} else {
				refs = append(refs, fmt.Sprintf("%s (PMID: %s)", refLabel, pmid))
			}
			if len(refs) >= 4 {
				break
			}
		}
		if len(refs) > 0 {
			body := strings.TrimSpace(strings.Join(bodyParts, "\n\n"))
			if len(body) > 2500 {
				body = body[:2500] + "..."
			}
			out = append(out, Documento{Titulo: "PubMed", Contenido: body, Fuente: "pubmed", Referencias: refs})
			return out
		}
	}
	clean := stripModelPreambles(raw)
	if isLikelyNoDataResponse(clean) {
		log.Printf("[conv][buscarPubMed][skip_error_clean] message=%s", sanitizePreview(clean))
		return out
	}
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
			// Si tiene título específico (source_book), incluirlo aunque el contenido esté vacío
			if d.Titulo != "" {
				if strings.TrimSpace(d.Contenido) != "" {
					fmt.Fprintf(&b, "- %s:\n%s\n\n", d.Titulo, d.Contenido)
				} else {
					fmt.Fprintf(&b, "- %s:\nInformación disponible en el documento\n\n", d.Titulo)
				}
			} else if strings.TrimSpace(d.Contenido) != "" {
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

// buildAPAInstructions genera instrucciones de formato APA basadas en los metadatos disponibles
func buildAPAInstructions(vectorDocs []Documento) string {
	if len(vectorDocs) == 0 {
		return ""
	}

	var instructions strings.Builder
	instructions.WriteString("\n\nMETADATOS DISPONIBLES PARA CITAS APA:\n")

	hasMetadata := false
	hasTitles := false

	for i, doc := range vectorDocs {
		// Verificar si tiene metadatos reales
		if doc.Metadata != nil {
			hasMetadata = true
			// Intentar acceder a metadatos como map
			if metaMap, ok := doc.Metadata.(map[string]interface{}); ok {
				title := getStringFromMap(metaMap, "title")
				author := getStringFromMap(metaMap, "author")
				created := getStringFromMap(metaMap, "created")

				if title != "" || author != "" {
					fmt.Fprintf(&instructions, "Documento %d (%s):\n", i+1, doc.Titulo)
					if title != "" {
						fmt.Fprintf(&instructions, "  - Título: %s\n", title)
					}
					if author != "" {
						fmt.Fprintf(&instructions, "  - Autor: %s\n", author)
					}
					if created != "" {
						fmt.Fprintf(&instructions, "  - Año: %s\n", created)
					}
				}
			}
		}

		// Verificar si al menos tiene título
		if strings.TrimSpace(doc.Titulo) != "" {
			hasTitles = true
			if !hasMetadata {
				// Solo listar títulos si no hay metadatos
				fmt.Fprintf(&instructions, "Documento %d: %s (sin metadatos adicionales)\n", i+1, doc.Titulo)
			}
		}
	}

	if hasMetadata {
		instructions.WriteString("\nFORMATO APA 7 PARA ## Fuentes (usa metadatos arriba):\n")
		instructions.WriteString("- Con todos los datos: Autor. (Año). Título. [PDF].\n")
		instructions.WriteString("- Sin autor: Título. (Año). [PDF].\n")
		instructions.WriteString("- Sin año: Autor. (s.f.). Título. [PDF].\n")
		instructions.WriteString("- Mínimo: Título. (s.f.). [PDF/Libro de texto médico].\n")
		instructions.WriteString("- IMPORTANTE: USA los metadatos proporcionados arriba\n")
	} else if hasTitles {
		instructions.WriteString("\nFORMATO APA MÍNIMO para ## Fuentes:\n")
		instructions.WriteString("- Usa el título exacto listado arriba\n")
		instructions.WriteString("- Formato: Título exacto. (s.f.). [Libro de texto médico].\n")
		instructions.WriteString("- Ejemplo: Robbins y Cotran. Patología Estructural y Funcional. (s.f.). [Libro de texto médico].\n")
	} else {
		instructions.WriteString("\nNo hay metadatos ni títulos disponibles.\n")
		instructions.WriteString("FORMATO GENÉRICO para ## Fuentes:\n")
		instructions.WriteString("- Libro de Texto Médico Especializado. (s.f.). [Libro de texto médico].\n")
	}

	return instructions.String()
}

// getStringFromMap helper para extraer strings de maps de metadatos
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return strings.TrimSpace(str)
		}
	}
	return ""
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

func isLikelyErrorMessage(s string) bool {
	t := strings.ToLower(strings.TrimSpace(s))
	if t == "" {
		return false
	}
	keywords := []string{
		"hubo un error",
		"ocurrió un error",
		"ocurrio un error",
		"error al procesar",
		"error al ejecutar",
		"intentaré de nuevo",
		"intentare de nuevo",
		"intentalo de nuevo",
		"inténtalo de nuevo",
		"por favor, espera",
		"por favor espera",
		"please wait",
		"please try again",
		"no encontré información",
		"no encontre informacion",
		"no se encontró información",
		"no se encontro informacion",
		"sin información disponible",
		"sin informacion disponible",
	}
	for _, k := range keywords {
		if strings.Contains(t, k) {
			return true
		}
	}
	if len(t) <= 180 && strings.Contains(t, "error") && (strings.Contains(t, "intenta") || strings.Contains(t, "espera") || strings.Contains(t, "proces")) {
		return true
	}
	return false
}

func isLikelyNoDataResponse(s string) bool {
	if isLikelyErrorMessage(s) {
		return true
	}
	t := strings.ToLower(strings.TrimSpace(s))
	if t == "" {
		return true
	}
	phrases := []string{
		"no se encontraron resultados",
		"no se encontraron coincidencias",
		"no hay información relevante",
		"no hay informacion relevante",
		"no se dispone de información",
		"no se dispone de informacion",
	}
	for _, p := range phrases {
		if strings.Contains(t, p) {
			return true
		}
	}
	return false
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

// improveConversationalPrompt mejora preguntas vagas o conversacionales
func improveConversationalPrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	lowerPrompt := strings.ToLower(prompt)

	// Manejar preguntas conversacionales comunes
	if strings.Contains(lowerPrompt, "profundizar") || strings.Contains(lowerPrompt, "más sobre") ||
		strings.Contains(lowerPrompt, "ampliar") || strings.Contains(lowerPrompt, "explicar más") {
		// Si es muy vago, asumir que quiere profundizar en el último tema médico contextual
		if len(prompt) < 50 && (strings.Contains(lowerPrompt, "eso") || strings.Contains(lowerPrompt, "esto")) {
			return "Proporciona información médica académica detallada y profunda sobre el último tema mencionado, incluyendo fisiopatología, manifestaciones clínicas, diagnóstico y tratamiento"
		}
	}

	// Si es muy corto y vago, expandir
	if len(prompt) < 20 && (strings.Contains(lowerPrompt, "eso") || strings.Contains(lowerPrompt, "esto") ||
		strings.Contains(lowerPrompt, "más") || strings.Contains(lowerPrompt, "continúa")) {
		return "Proporciona información médica académica detallada sobre el tema médico en contexto, con enfoque clínico y científico"
	}

	return prompt
}
