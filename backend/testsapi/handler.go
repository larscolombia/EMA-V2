package testsapi

import (
	"context"
	"encoding/json"
	"log"
	"math"
	randpkg "math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ema-backend/openai"

	"github.com/gin-gonic/gin"
)

// Assistant is a minimal interface implemented by openai.Client
type Assistant interface {
	CreateThread(ctx context.Context) (string, error)
	StreamAssistantMessage(ctx context.Context, threadID, prompt string) (<-chan string, error)
	StreamAssistantJSON(ctx context.Context, threadID, userPrompt, jsonInstructions string) (<-chan string, error)

	// Responses API compatible methods
	CreateThreadOrConversation(ctx context.Context) (string, error)
	StreamResponseWithInstructionsCompatible(
		ctx context.Context,
		conversationID, userInput, instructions, vectorStoreID string,
	) (<-chan string, error)
	StreamAssistantJSONCompatible(
		ctx context.Context,
		threadID, userPrompt, jsonInstructions, vectorStoreID string,
	) (<-chan string, error)
}

type Handler struct {
	ai          Assistant
	assistantID string
	// Optional: resolver to map category IDs to names
	resolveCategoryNames func(ctx context.Context, ids []int) ([]string, error)
	quotaValidator       func(ctx context.Context, c *gin.Context, flow string) error
}

func NewHandler(ai Assistant, assistantID string) *Handler {
	return &Handler{ai: ai, assistantID: assistantID}
}

func DefaultHandler() *Handler {
	cli := openai.NewClient()
	// Override assistant with the medical questionnaires assistant if provided
	if id := os.Getenv("CUESTIONARIOS_MEDICOS_GENERALES"); strings.TrimSpace(id) != "" {
		cli.AssistantID = id
	}
	return &Handler{ai: cli, assistantID: cli.AssistantID}
}

// SetCategoryResolver injects a resolver that maps category IDs to readable names.
func (h *Handler) SetCategoryResolver(fn func(ctx context.Context, ids []int) ([]string, error)) {
	h.resolveCategoryNames = fn
}

type generateRequest struct {
	IdCategoria  []int  `json:"id_categoria"`
	NumQuestions int    `json:"num_questions"`
	Nivel        string `json:"nivel"`
}

// GenerateResponse matches what Flutter expects: { success: true, data: { test_id, thread_id, questions: [...] } }
type generateResponse struct {
	Success bool           `json:"success"`
	Data    map[string]any `json:"data"`
}

type evalQuestion struct {
	QuestionID int    `json:"question_id"`
	Answer     string `json:"answer"`
	Type       string `json:"type"`
}

type evaluateRequest struct {
	UID    string         `json:"uid"`
	Thread string         `json:"thread"`
	UserID int            `json:"userId"`
	TestID int            `json:"test_id"`
	Items  []evalQuestion `json:"questions"`
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/tests/generate/:userId", h.generate)
	r.POST("/tests/responder-test/submit", h.evaluate)
}

// SetQuotaValidator allows main to inject quota validator (quiz generation counts)
func (h *Handler) SetQuotaValidator(fn func(ctx context.Context, c *gin.Context, flow string) error) {
	h.quotaValidator = fn
}

func (h *Handler) generate(c *gin.Context) {
	// Quota: quiz generation consumes questionnaires bucket
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "quiz_generate"); err != nil {
			field, _ := c.Get("quota_error_field")
			reason, _ := c.Get("quota_error_reason")
			resp := gin.H{"error": "quiz quota exceeded"}
			if f, ok := field.(string); ok && f != "" {
				resp["field"] = f
			}
			if r, ok := reason.(string); ok && r != "" {
				resp["reason"] = r
			}
			c.JSON(http.StatusForbidden, resp)
			return
		}
		if v, ok := c.Get("quota_remaining"); ok {
			c.Header("X-Quota-Questionnaires-Remaining", toString(v)) // legacy
			c.Header("X-Quota-Field", "questionnaires")
			c.Header("X-Quota-Remaining", toString(v))
		}
	}
	var req generateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	// Resolve category names for clarity if provided
	var catNames []string
	if len(req.IdCategoria) > 0 && h.resolveCategoryNames != nil {
		if names, err := h.resolveCategoryNames(c.Request.Context(), req.IdCategoria); err == nil && len(names) > 0 {
			catNames = names
		}
	}

	// PASO 1: Realizar búsquedas en libros y PubMed para basar las preguntas en fuentes confiables
	// Vector store específico para banco de preguntas médicas
	vectorID := "vs_691deb92da488191aaeefba2b80406d7"
	log.Printf("[testsapi.generate] VECTOR_CONFIGURED: using vector_store_id=%s (Banco de Preguntas)", vectorID)

	// Construir query de búsqueda basada en categorías o medicina interna
	searchQuery := "medicina interna"
	if len(catNames) > 0 {
		searchQuery = strings.Join(catNames, " ")
	}
	log.Printf("[testsapi.generate] searching sources: categories=%v query=%s vector_id=%s", catNames, searchQuery, vectorID)

	// Buscar en libros (vector store)
	vectorContext := ""
	vectorSource := ""
	if extClient, ok := h.ai.(interface {
		QuickVectorSearch(ctx context.Context, vectorStoreID, query string) (*openai.VectorSearchResult, error)
	}); ok {
		if result, err := extClient.QuickVectorSearch(ctx, vectorID, searchQuery); err == nil && result != nil && result.HasResult {
			vectorContext = strings.TrimSpace(result.Content)
			vectorSource = strings.TrimSpace(result.Source)
			if vectorSource == "" {
				vectorSource = "Libro de Medicina"
			}
			log.Printf("[testsapi.generate] vector search OK: source=%s content_len=%d", vectorSource, len(vectorContext))
		} else {
			log.Printf("[testsapi.generate] vector search failed or empty: err=%v", err)
		}
	}

	// Buscar en PubMed
	pubmedContext := ""
	if extClient, ok := h.ai.(interface {
		SearchPubMed(ctx context.Context, query string) (string, error)
	}); ok {
		if result, err := extClient.SearchPubMed(ctx, searchQuery); err == nil {
			pubmedContext = strings.TrimSpace(result)
			log.Printf("[testsapi.generate] pubmed search OK: content_len=%d", len(pubmedContext))
		} else {
			log.Printf("[testsapi.generate] pubmed search failed: err=%v", err)
		}
	}

	// PASO 2: Preparar el prompt con contexto recuperado
	sb := strings.Builder{}
	sb.WriteString("Genera un JSON con {\"questions\": [...] } de tamaño ")
	sb.WriteString(intToStr(req.NumQuestions))
	sb.WriteString(".\n\n")

	// Añadir contexto de fuentes si está disponible
	if vectorContext != "" {
		sb.WriteString("=== CONOCIMIENTO MÉDICO DE REFERENCIA ===\n")
		sb.WriteString("(Extraído de fuente académica confiable)\n\n")
		// Limitar a 3000 caracteres para no saturar el prompt
		if len(vectorContext) > 3000 {
			vectorContext = vectorContext[:3000]
		}
		sb.WriteString(vectorContext)
		sb.WriteString("\n\n")
	}

	if pubmedContext != "" {
		sb.WriteString("=== EVIDENCIA CIENTÍFICA ADICIONAL ===\n")
		sb.WriteString("(Estudios y literatura médica)\n\n")
		// Limitar a 2000 caracteres
		if len(pubmedContext) > 2000 {
			pubmedContext = pubmedContext[:2000]
		}
		sb.WriteString(pubmedContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString("=== INSTRUCCIONES PARA GENERAR PREGUNTAS ===\n")
	sb.WriteString("Usa el conocimiento médico proporcionado arriba como base científica para crear preguntas clínicas prácticas.\n")
	sb.WriteString("IMPORTANTE: Las preguntas deben ser sobre CONCEPTOS MÉDICOS, diagnósticos, tratamientos, factores de riesgo, fisiopatología, etc.\n")
	sb.WriteString("NO preguntes sobre los libros o fuentes (ej: NO '¿Qué dice el libro sobre...?' o '¿Según qué autor...?').\n")
	sb.WriteString("SÍ pregunta sobre conocimiento médico (ej: '¿Cuáles son los factores de riesgo cardiovascular?' o '¿Cuál es el tratamiento de primera línea para...?').\n\n")

	if len(catNames) > 0 {
		sb.WriteString("Categoría médica: ")
		sb.WriteString(strings.Join(catNames, ", "))
		sb.WriteString(".\n")
	} else {
		sb.WriteString("Categoría médica: Medicina Interna General.\n")
	}
	if strings.TrimSpace(req.Nivel) != "" {
		sb.WriteString("Nivel de dificultad: ")
		sb.WriteString(req.Nivel)
		sb.WriteString(".\n")
	}
	sb.WriteString("Tipos de pregunta: true_false, open_ended, single_choice (aleatorios).\n")
	sb.WriteString("Para single_choice incluye 4-5 opciones con una única respuesta correcta.\n")
	sb.WriteString("Formato de respuesta: JSON estricto, sin texto adicional.")

	threadID, err := h.ai.CreateThreadOrConversation(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "assistant thread error"})
		return
	}
	// JSON-only instructions to maximize adherence while letting the assistant use its pre-configured RAG
	// Build JSON-only + domain instructions (conditional domain restriction)
	instr := []string{
		"Responde estrictamente en formato JSON válido.",
		"Devuelve un objeto con la clave 'questions' como array de tamaño solicitado.",
		"Cada pregunta debe tener: id(int), question(string), answer(string), type in [true_false, open_ended, single_choice],",
		"options(array<string>) SOLO si type=single_choice (4 o 5 opciones, única correcta).",
		"Incluye preguntas con 'excepto' o 'sin excepto' cuando corresponda.",
	}
	if len(catNames) > 0 {
		instr = append(instr, "Limita el contenido estrictamente a la(s) categoría(s) seleccionada(s) y su subdominio clínico: "+strings.Join(catNames, ", ")+".")
	} else {
		instr = append(instr, "Limita el contenido estrictamente a medicina interna.")
	}
	instr = append(instr,
		"CRÍTICO: Las preguntas deben ser sobre CONOCIMIENTO MÉDICO (diagnósticos, tratamientos, fisiopatología, factores de riesgo, manejo clínico).",
		"PROHIBIDO: NO preguntes sobre fuentes, libros o autores (ej: '¿Qué menciona el libro...?', '¿Según qué autor...?').",
		"CORRECTO: Pregunta directamente sobre conceptos (ej: '¿Cuáles son los factores de riesgo de IAM?', '¿Cuál es el tratamiento de primera línea para HTA?').",
		"Basa el contenido médico en el conocimiento proporcionado en el contexto, pero formula preguntas clínicas directas.",
		"NO inventes información: usa SOLO el conocimiento del contexto proporcionado.",
		"Mantén coherencia, claridad y rigor clínico.",
		"Responde en español.",
		"No incluyas texto fuera del JSON.",
	)
	jsonInstr := strings.Join(instr, " ")
	ch, err := h.ai.StreamAssistantJSONCompatible(ctx, threadID, sb.String(), jsonInstr, vectorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "assistant run error"})
		return
	}
	var content string
	// Expect a single final message from StreamAssistantMessage
	select {
	case content = <-ch:
	case <-ctx.Done():
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "assistant timeout"})
		return
	}
	jsonStr := extractJSON(content)
	var parsed map[string]any
	_ = json.Unmarshal([]byte(jsonStr), &parsed)
	// Try to coerce to []any from various shapes
	var rawQuestions []any
	if v, ok := parsed["questions"]; ok {
		switch vv := v.(type) {
		case []any:
			rawQuestions = vv
		case map[string]any:
			rawQuestions = []any{vv}
		}
	}
	if len(rawQuestions) == 0 {
		log.Printf("[testsapi.generate] missing/invalid questions; attempting JSON repair (userId=%s)", c.Param("userId"))
		// Try one repair cycle in the same thread to rewrite as strict JSON
		repaired, repOK := h.repairGenerateJSON(ctx, threadID, content, req.NumQuestions, jsonInstr, vectorID)
		if repOK {
			parsed = map[string]any{}
			_ = json.Unmarshal([]byte(repaired), &parsed)
			if v, ok := parsed["questions"]; ok {
				switch vv := v.(type) {
				case []any:
					rawQuestions = vv
				case map[string]any:
					rawQuestions = []any{vv}
				}
			}
		}
		if len(rawQuestions) == 0 {
			log.Printf("[testsapi.generate] repair failed; synthesizing safe defaults (userId=%s)", c.Param("userId"))
		} else {
			log.Printf("[testsapi.generate] repair succeeded; got %d questions (userId=%s)", len(rawQuestions), c.Param("userId"))
		}
	} else {
		log.Printf("[testsapi.generate] assistant returned %d questions before normalization (userId=%s)", len(rawQuestions), c.Param("userId"))
	}
	// Normalize questions; if empty, synthesize safe defaults
	normalized := normalizeQuestions(rawQuestions, req.NumQuestions)
	// Randomize order of questions and options to avoid positional bias
	normalized = randomizeQuestions(normalized)
	log.Printf("[testsapi.generate] normalized questions count=%d (requested=%d)", len(normalized), req.NumQuestions)
	// Build response data compatible with Flutter mapper
	data := map[string]any{
		"test_id":   time.Now().Nanosecond(),
		"thread_id": threadID,
		"questions": normalized,
	}
	c.JSON(http.StatusOK, generateResponse{Success: true, Data: data})
}

func (h *Handler) evaluate(c *gin.Context) {
	var req evaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	// PASO 1: Buscar en fuentes médicas para fundamentar la evaluación
	// Vector store específico para banco de preguntas médicas
	vectorID := "vs_691deb92da488191aaeefba2b80406d7"
	log.Printf("[testsapi.evaluate] VECTOR_CONFIGURED: using vector_store_id=%s (Banco de Preguntas)", vectorID)

	// Construir query basada en las respuestas del usuario para buscar contexto relevante
	searchQuery := "evaluación médica"
	if len(req.Items) > 0 {
		// Usar la primera pregunta como contexto de búsqueda
		searchQuery = req.Items[0].Answer
	}
	log.Printf("[testsapi.evaluate] searching sources for evaluation: query=%s vector_id=%s", searchQuery, vectorID)

	// Buscar contexto en libros
	vectorContext := ""
	vectorSource := ""
	if extClient, ok := h.ai.(interface {
		QuickVectorSearch(ctx context.Context, vectorStoreID, query string) (*openai.VectorSearchResult, error)
	}); ok {
		if result, err := extClient.QuickVectorSearch(ctx, vectorID, searchQuery); err == nil && result != nil && result.HasResult {
			vectorContext = strings.TrimSpace(result.Content)
			vectorSource = strings.TrimSpace(result.Source)
			if vectorSource == "" {
				vectorSource = "Libro de Medicina"
			}
			log.Printf("[testsapi.evaluate] vector search OK: source=%s content_len=%d", vectorSource, len(vectorContext))
		} else {
			log.Printf("[testsapi.evaluate] vector search failed or empty: err=%v", err)
		}
	}

	// Buscar en PubMed
	pubmedContext := ""
	if extClient, ok := h.ai.(interface {
		SearchPubMed(ctx context.Context, query string) (string, error)
	}); ok {
		if result, err := extClient.SearchPubMed(ctx, searchQuery); err == nil {
			pubmedContext = strings.TrimSpace(result)
			log.Printf("[testsapi.evaluate] pubmed search OK: content_len=%d", len(pubmedContext))
		} else {
			log.Printf("[testsapi.evaluate] pubmed search failed: err=%v", err)
		}
	}

	// PASO 2: Build prompt for evaluation con contexto
	sb := strings.Builder{}

	// Añadir contexto de fuentes si está disponible
	if vectorContext != "" {
		sb.WriteString("=== CONOCIMIENTO MÉDICO DE REFERENCIA ===\n")
		sb.WriteString("(Para validar respuestas clínicas)\n\n")
		if len(vectorContext) > 2000 {
			vectorContext = vectorContext[:2000]
		}
		sb.WriteString(vectorContext)
		sb.WriteString("\n\n")
	}

	if pubmedContext != "" {
		sb.WriteString("=== EVIDENCIA CIENTÍFICA ===\n")
		sb.WriteString("(Literatura médica actual)\n\n")
		if len(pubmedContext) > 1500 {
			pubmedContext = pubmedContext[:1500]
		}
		sb.WriteString(pubmedContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString("=== TAREA DE EVALUACIÓN ===\n")
	sb.WriteString("Usa el conocimiento médico proporcionado arriba para evaluar las respuestas del usuario.\n")
	sb.WriteString("Proporciona feedback clínico educativo sobre los conceptos médicos.\n")
	sb.WriteString("IMPORTANTE: El feedback debe explicar conceptos médicos, NO mencionar libros o fuentes en el texto de retroalimentación.\n\n")
	sb.WriteString("Formato de respuesta:\n")
	sb.WriteString("{ \"evaluation\": [ { \"question_id\": <int>, \"is_correct\": 0|1, \"fit\": <string> }... ], \"correct_answers\": <int>, \"fit_global\": <string> }\n\n")
	sb.WriteString("Respuestas del usuario: ")
	// Compact user answers into JSON-like text for the model
	items := make([]map[string]any, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, map[string]any{
			"question_id": it.QuestionID,
			"answer":      it.Answer,
			"type":        it.Type,
		})
	}
	b, _ := json.Marshal(items)
	sb.Write(b)

	threadID := req.Thread
	if strings.TrimSpace(threadID) == "" {
		var err error
		threadID, err = h.ai.CreateThreadOrConversation(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "assistant thread error"})
			return
		}
	}
	// Construir string de fuentes claras para incluir en las instrucciones
	sourcesInfo := ""
	if vectorSource != "" {
		sourcesInfo += "Libro consultado: " + vectorSource + "\n"
	}
	if pubmedContext != "" {
		sourcesInfo += "Artículos de PubMed consultados (incluye PMIDs específicos en las referencias)\n"
	}

	evalInstr := strings.Join([]string{
		"Responde estrictamente en JSON válido con las claves:",
		"evaluation: array de {question_id:int, is_correct:0|1, fit:string} (fit = explicación clínica breve, SIN mencionar libros o fuentes).",
		"correct_answers:int,",
		"fit_global:string = TEXTO MULTIPÁRRAFO estructurado (no HTML) con las siguientes secciones claramente separadas por doble salto de línea:\n" +
			"1) 'Puntaje y Calificación:' en la primera línea incluye: 'Total de respuestas correctas: X de N.' en la siguiente línea 'Puntaje: <porcentaje con 2 decimales>%.' y en la tercera línea 'Clasificación: <desempeño alto|desempeño adecuado|desempeño moderado|desempeño bajo>' según porcentaje (>=85 alto, 70-84 adecuado, 50-69 moderado, <50 bajo).\n" +
			"2) 'Retroalimentación:' párrafos (mínimo 2, máximo 5) que: a) Feliciten el esfuerzo. b) Destaquen fortalezas específicas detectadas. c) Expliquen conceptos médicos relevantes (fisiopatología, tratamientos, diagnósticos) SIN mencionar libros. d) Incluyan 1-2 recomendaciones prácticas de estudio sobre temas médicos. e) Inviten a profundizar en conceptos específicos.\n" +
			"3) 'Referencias:' DEBE incluir citas COMPLETAS y ESPECÍFICAS de las fuentes consultadas:\n" +
			"   - Si usaste libro: nombre COMPLETO del libro (ej: 'Harrison. Principios de Medicina Interna, 21ª edición' o 'Robbins y Cotran. Patología Estructural y Funcional').\n" +
			"   - Si usaste PubMed: incluye PMID específico y título del estudio (ej: 'PMID: 12345678 - Cardiovascular risk factors in hypertensive patients').\n" +
			"   - Lista 1-3 referencias separadas por punto y coma.\n" +
			"   - FUENTES DISPONIBLES: " + sourcesInfo + "\n" +
			"Mantén formato de texto plano, sin viñetas, sin JSON dentro. Usa saltos de línea dobles entre secciones y simples entre oraciones dentro de cada párrafo.",
		// Instrucciones adicionales
		"CRÍTICO: La retroalimentación debe explicar CONCEPTOS MÉDICOS (fisiopatología, tratamientos, diagnósticos).",
		"PROHIBIDO: NO menciones libros, autores o fuentes en el texto de retroalimentación (solo en la sección Referencias al final).",
		"CORRECTO: Explica directamente los conceptos (ej: 'La hipertensión arterial se define como...', 'Los factores de riesgo cardiovascular incluyen...').",
		"Verifica cada respuesta usando el conocimiento médico del contexto proporcionado.",
		"CRÍTICO REFERENCIAS: Las citas en 'Referencias' deben ser COMPLETAS, ESPECÍFICAS y VERIFICABLES (nombre completo de libros, PMIDs con títulos de artículos).",
		"Extrae los PMIDs y títulos exactos del contexto de PubMed proporcionado arriba.",
		"Mantén coherencia, claridad y rigor clínico.",
		"Responde en español.",
		"No incluyas texto fuera del JSON.",
	}, " ")
	ch, err := h.ai.StreamAssistantJSONCompatible(ctx, threadID, sb.String(), evalInstr, vectorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "assistant run error"})
		return
	}
	var content string
	select {
	case content = <-ch:
	case <-ctx.Done():
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "assistant timeout"})
		return
	}
	jsonStr := extractJSON(content)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
		log.Printf("[testsapi.evaluate] assistant evaluation parsed OK (uid=%s, test_id=%d)", req.UID, req.TestID)
		// Augment with summary if evaluation array present
		parsed = augmentEvaluationSummary(parsed, req.Items)
		c.JSON(http.StatusOK, parsed)
		return
	}
	// Try a repair cycle on evaluation
	if repaired, ok := h.repairEvaluationJSON(ctx, threadID, content, evalInstr, vectorID); ok {
		var parsed2 map[string]any
		if err := json.Unmarshal([]byte(repaired), &parsed2); err == nil {
			log.Printf("[testsapi.evaluate] repair succeeded; returning fixed JSON (uid=%s, test_id=%d)", req.UID, req.TestID)
			parsed2 = augmentEvaluationSummary(parsed2, req.Items)
			c.JSON(http.StatusOK, parsed2)
			return
		}
	}
	// Fallback: build a minimal evaluation so app doesn't fail
	log.Printf("[testsapi.evaluate] assistant evaluation parse failed, returning fallback (uid=%s, test_id=%d)", req.UID, req.TestID)
	eval := make([]map[string]any, 0, len(req.Items))
	for _, it := range req.Items {
		eval = append(eval, map[string]any{
			"question_id": it.QuestionID,
			"is_correct":  0,
			"fit":         "Respuesta registrada",
		})
	}
	fallback := map[string]any{
		"evaluation":      eval,
		"correct_answers": 0,
		"fit_global":      "Evaluación automática sin modelo",
	}
	fallback = augmentEvaluationSummary(fallback, req.Items)
	c.JSON(http.StatusOK, fallback)
}

var jsonRe = regexp.MustCompile(`(?s)\{.*\}`)

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		return s
	}
	m := jsonRe.FindString(s)
	if m != "" {
		return m
	}
	return "{}"
}

func intToStr(n int) string { return strconv.Itoa(n) }

// --- Normalization helpers --- //

func normalizeQuestions(raw []any, n int) []map[string]any {
	out := make([]map[string]any, 0, max(1, n))
	// Try to normalize assistant-provided questions first
	for i, item := range raw {
		q, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, normalizeQuestion(q, i+1))
	}
	// If empty, synthesize placeholders
	if len(out) == 0 {
		if n <= 0 {
			n = 3
		}
		for i := 1; i <= n; i++ {
			out = append(out, map[string]any{
				"id":       i,
				"question": "Pregunta " + strconv.Itoa(i) + ": seleccione la opción correcta.",
				"answer":   "Opción A",
				"type":     "single_choice",
				"options":  []string{"Opción A", "Opción B", "Opción C", "Opción D"},
				"category": nil,
			})
		}
	}
	// Ensure sequential ids and valid shape
	for i := range out {
		out[i] = normalizeQuestion(out[i], i+1)
	}
	return out
}

func normalizeQuestion(q map[string]any, seq int) map[string]any {
	// id
	id := seq
	switch v := q["id"].(type) {
	case float64:
		id = int(v)
	case int:
		id = v
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			id = n
		}
	}
	// type
	t := "single_choice"
	if tv, ok := q["type"]; ok {
		t = normalizeType(toStr(tv))
	}
	// question
	question := strings.TrimSpace(toStr(q["question"]))
	if question == "" {
		question = "Pregunta " + strconv.Itoa(seq) + ""
	}
	// options
	opts := toStringSlice(q["options"])
	// answer
	answer := strings.TrimSpace(toStr(q["answer"]))

	// Ensure options and answer coherence
	switch t {
	case "true_false":
		opts = []string{"true", "false"}
		if answer != "true" && answer != "false" {
			answer = "true"
		}
	case "single_choice":
		if len(opts) < 2 {
			// fabricate with answer present
			if answer == "" {
				answer = "Opción A"
			}
			opts = uniqueStrings([]string{answer, "Opción B", "Opción C", "Opción D"})
		}
		// Ensure answer is one of options
		if answer == "" || !containsIgnoreCase(opts, answer) {
			answer = opts[0]
		}
		// Trim to max 5 options
		if len(opts) > 5 {
			opts = opts[:5]
		}
	case "open_ended":
		// options should be empty; answer can be empty
		opts = []string{}
	}

	q["id"] = id
	q["type"] = t
	q["question"] = question
	q["options"] = opts
	q["answer"] = answer
	if _, ok := q["category"]; !ok {
		q["category"] = nil
	}
	return q
}

func normalizeType(s string) string {
	s = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(s, "-", "_")))
	switch s {
	case "true_false", "truefalse", "verdadero_falso", "tf":
		return "true_false"
	case "open_ended", "open", "abierta":
		return "open_ended"
	case "single_choice", "single", "opcion_unica", "multiple_choice", "multiple":
		return "single_choice"
	default:
		return "single_choice"
	}
}

func toStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case json.Number:
		return x.String()
	default:
		return ""
	}
}

// toString replica helper simple usado en otros handlers para cabeceras de cuota
func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return ""
	}
}
func toStringSlice(v any) []string {
	switch x := v.(type) {
	case []any:
		out := make([]string, 0, len(x))
		for _, e := range x {
			out = append(out, toStr(e))
		}
		return out
	case []string:
		return append([]string(nil), x...)
	case string:
		// try JSON decode
		var arr []any
		if err := json.Unmarshal([]byte(x), &arr); err == nil {
			return toStringSlice(arr)
		}
		// comma-separated
		parts := strings.Split(x, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	default:
		return []string{}
	}
}

func containsIgnoreCase(a []string, s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, x := range a {
		if strings.ToLower(strings.TrimSpace(x)) == s {
			return true
		}
	}
	return false
}

func uniqueStrings(a []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(a))
	for _, s := range a {
		k := strings.ToLower(strings.TrimSpace(s))
		if !seen[k] {
			seen[k] = true
			out = append(out, s)
		}
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// --- Randomization helpers --- //

// randomizeQuestions shuffles the order of questions and for single_choice questions
// it shuffles the options while keeping the stored answer consistent.
// For true_false, we keep canonical order most of the time but occasionally flip to reduce bias.
func randomizeQuestions(qs []map[string]any) []map[string]any {
	if len(qs) == 0 {
		return qs
	}
	// Seed RNG; time-based seed is fine per-request
	randpkg.Seed(time.Now().UnixNano())
	// Shuffle questions (do NOT change IDs; assistant remembers original IDs for evaluation)
	randpkg.Shuffle(len(qs), func(i, j int) { qs[i], qs[j] = qs[j], qs[i] })
	// Shuffle options per question when applicable
	for _, q := range qs {
		t := toStr(q["type"])
		switch t {
		case "single_choice":
			opts := toStringSlice(q["options"])
			if len(opts) >= 2 {
				ans := toStr(q["answer"])
				// build indices, shuffle, then rebuild opts
				idx := make([]int, len(opts))
				for i := range idx {
					idx[i] = i
				}
				randpkg.Shuffle(len(idx), func(i, j int) { idx[i], idx[j] = idx[j], idx[i] })
				newOpts := make([]string, len(opts))
				for pos, oldIdx := range idx {
					newOpts[pos] = opts[oldIdx]
				}
				q["options"] = newOpts
				// Map answer to shuffled options without bias
				if ans == "" {
					ans = newOpts[randpkg.Intn(len(newOpts))]
				} else if !containsIgnoreCase(newOpts, ans) {
					best := bestSimilarityOption(ans, newOpts)
					if best != "" {
						ans = best
					} else {
						for _, o := range newOpts {
							if strings.EqualFold(strings.TrimSpace(o), strings.TrimSpace(ans)) {
								ans = o
								break
							}
						}
						if !containsIgnoreCase(newOpts, ans) {
							ans = newOpts[randpkg.Intn(len(newOpts))]
						}
					}
				}
				q["answer"] = ans
			}
		case "true_false":
			// 25% chance to flip order to avoid always-true-first bias
			if randpkg.Intn(4) == 0 {
				q["options"] = []string{"false", "true"}
			} else {
				q["options"] = []string{"true", "false"}
			}
			ans := strings.ToLower(strings.TrimSpace(toStr(q["answer"])))
			if ans != "true" && ans != "false" {
				ans = "true"
			}
			q["answer"] = ans
		}
	}
	return qs

}

// bestSimilarityOption picks the option with highest Jaccard similarity on token sets.
// Returns empty string if no option shares tokens with the answer.
func bestSimilarityOption(answer string, options []string) string {
	ansTok := tokenizeSimple(answer)
	best := ""
	bestScore := -1.0
	for _, o := range options {
		score := jaccardSimple(ansTok, tokenizeSimple(o))
		if score > bestScore {
			bestScore = score
			best = o
		}
	}
	if bestScore <= 0 {
		return ""
	}
	return best
}

// tokenizeSimple lowercases and extracts alphanumeric tokens as a set.
func tokenizeSimple(s string) map[string]struct{} {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return map[string]struct{}{}
	}
	// Replace non-alphanumeric with spaces
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == 'á' || r == 'é' || r == 'í' || r == 'ó' || r == 'ú' || r == 'ñ' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	parts := strings.Fields(b.String())
	out := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		out[p] = struct{}{}
	}
	return out
}

// jaccardSimple computes Jaccard similarity between two token sets.
func jaccardSimple(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// repairGenerateJSON asks the assistant in the same thread to rewrite its last message
// as strict JSON with exactly n questions. Returns the repaired JSON string and whether it succeeded.
func (h *Handler) repairGenerateJSON(ctx context.Context, threadID, lastContent string, n int, baseInstr, vectorID string) (string, bool) {
	prompt := strings.Builder{}
	prompt.WriteString("Tu último mensaje no cumple el formato requerido. Reescríbelo como un único objeto JSON válido. ")
	prompt.WriteString("Debe contener la clave 'questions' con exactamente ")
	prompt.WriteString(intToStr(n))
	prompt.WriteString(" elementos. No incluyas texto fuera del JSON.\n\nMensaje previo:\n")
	// Include only a trimmed snippet to avoid token bloat
	prev := strings.TrimSpace(lastContent)
	if len(prev) > 4000 {
		prev = prev[:4000]
	}
	prompt.WriteString(prev)
	ch, err := h.ai.StreamAssistantJSONCompatible(ctx, threadID, prompt.String(), baseInstr, vectorID)
	if err != nil {
		return "", false
	}
	select {
	case fixed := <-ch:
		fixed = extractJSON(fixed)
		if strings.TrimSpace(fixed) != "{}" && json.Valid([]byte(fixed)) {
			return fixed, true
		}
	case <-ctx.Done():
		return "", false
	}
	return "", false
}

// repairEvaluationJSON asks the assistant to rewrite evaluation as strict JSON using the same schema.
func (h *Handler) repairEvaluationJSON(ctx context.Context, threadID, lastContent, baseInstr, vectorID string) (string, bool) {
	prompt := strings.Builder{}
	prompt.WriteString("Tu último mensaje no es JSON válido. Reescribe la evaluación como un único objeto JSON válido con las claves ")
	prompt.WriteString("evaluation (array de {question_id:int, is_correct:0|1, fit:string}), correct_answers:int y fit_global:string. ")
	prompt.WriteString("No incluyas texto fuera del JSON.\n\nMensaje previo:\n")
	prev := strings.TrimSpace(lastContent)
	if len(prev) > 4000 {
		prev = prev[:4000]
	}
	prompt.WriteString(prev)
	ch, err := h.ai.StreamAssistantJSONCompatible(ctx, threadID, prompt.String(), baseInstr, vectorID)
	if err != nil {
		return "", false
	}
	select {
	case fixed := <-ch:
		fixed = extractJSON(fixed)
		if strings.TrimSpace(fixed) != "{}" && json.Valid([]byte(fixed)) {
			return fixed, true
		}
	case <-ctx.Done():
		return "", false
	}
	return "", false
}

// augmentEvaluationSummary computes a lightweight summary with counts and missed questions.
// Expects map possibly containing 'evaluation' and 'correct_answers'.
// Adds field 'summary': { total:int, correct:int, incorrect:int, correct_percentage:float, missed_ids:[], correct_answers_map:{question_id:answer} }
// correct_answers_map derived from evaluation where is_correct==1 if answer present in input items; limited to 50 entries.
func augmentEvaluationSummary(parsed map[string]any, reqItems []evalQuestion) map[string]any {
	if parsed == nil {
		return parsed
	}
	// Extract evaluation array
	evalAny, ok := parsed["evaluation"]
	var evalArr []map[string]any
	if ok {
		switch v := evalAny.(type) {
		case []any:
			for _, e := range v {
				if m, ok := e.(map[string]any); ok {
					evalArr = append(evalArr, m)
				}
			}
		case []map[string]any:
			evalArr = v
		}
	}
	total := len(evalArr)
	correct := 0
	missedIDs := []int{}
	// map questionID -> answer from original request for correct ones
	answerLookup := map[int]string{}
	for _, it := range reqItems {
		answerLookup[it.QuestionID] = it.Answer
	}
	correctMap := map[string]string{}
	for _, e := range evalArr {
		qid := -1
		if idf, ok := e["question_id"].(float64); ok {
			qid = int(idf)
		} else if idi, ok := e["question_id"].(int); ok {
			qid = idi
		}
		isC := 0
		if icf, ok := e["is_correct"].(float64); ok {
			isC = int(icf)
		} else if ici, ok := e["is_correct"].(int); ok {
			isC = ici
		}
		if isC == 1 {
			correct++
			if ans, ok2 := answerLookup[qid]; ok2 && len(correctMap) < 50 {
				correctMap[strconv.Itoa(qid)] = ans
			}
		} else if qid >= 0 {
			missedIDs = append(missedIDs, qid)
		}
	}
	incorrect := total - correct
	pct := 0.0
	if total > 0 {
		pct = (float64(correct) / float64(total)) * 100.0
	}
	parsed["summary"] = map[string]any{
		"total":               total,
		"correct":             correct,
		"incorrect":           incorrect,
		"correct_percentage":  fmtFloat(pct),
		"missed_ids":          missedIDs,
		"correct_answers_map": correctMap,
	}
	return parsed
}

func fmtFloat(f float64) float64 { // round to 2 decimals
	return math.Round(f*100) / 100
}
