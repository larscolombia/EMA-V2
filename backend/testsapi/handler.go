package testsapi

import (
	"context"
	"encoding/json"
	"log"
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
}

type Handler struct {
	ai          Assistant
	assistantID string
	// Optional: resolver to map category IDs to names
	resolveCategoryNames func(ctx context.Context, ids []int) ([]string, error)
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

func (h *Handler) generate(c *gin.Context) {
	var req generateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	// Prepare prompt to the assistant
	// Keep it concise; the Assistant itself holds domain rules. We only pass variables.
	sb := strings.Builder{}
	sb.WriteString("Genera un JSON con {\"questions\": [...] } de tamaño ")
	sb.WriteString(intToStr(req.NumQuestions))
	sb.WriteString(", usando el conocimiento recuperado por el assistant de sus fuentes configuradas. Si no se especifica categoría, usa medicina interna. ")
	// Resolve category names for clarity if provided
	var catNames []string
	if len(req.IdCategoria) > 0 && h.resolveCategoryNames != nil {
		if names, err := h.resolveCategoryNames(c.Request.Context(), req.IdCategoria); err == nil && len(names) > 0 {
			catNames = names
		}
	}
	if len(catNames) > 0 {
		sb.WriteString("categorías: ")
		sb.WriteString(strings.Join(catNames, ", "))
		sb.WriteString(". ")
	}
	if strings.TrimSpace(req.Nivel) != "" {
		sb.WriteString("nivel: ")
		sb.WriteString(req.Nivel)
		sb.WriteString(". ")
	}
	sb.WriteString("Tipos de pregunta aleatorios entre: true_false, open_ended, single_choice. \n")
	sb.WriteString("Para single_choice incluye 4 o 5 opciones y una única respuesta correcta. \n")
	sb.WriteString("Responde estrictamente en formato JSON, sin bloque de código ni texto adicional.")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	threadID, err := h.ai.CreateThread(ctx)
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
		"Usa exclusivamente información contenida en las fuentes configuradas del assistant (vector privado) y en https://pubmed.ncbi.nlm.nih.gov/.",
		"No utilices otras fuentes ni inventes información.",
		"No menciones nunca el vector en el contenido.",
		"Mantén coherencia, claridad y nivel académico acorde al nivel solicitado.",
		"Responde en español.",
		"No incluyas texto fuera del JSON.",
	)
	jsonInstr := strings.Join(instr, " ")
	ch, err := h.ai.StreamAssistantJSON(ctx, threadID, sb.String(), jsonInstr)
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
		repaired, repOK := h.repairGenerateJSON(ctx, threadID, content, req.NumQuestions, jsonInstr)
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
	// Build prompt for evaluation
	sb := strings.Builder{}
	sb.WriteString("Evalúa las respuestas del usuario. Devuelve JSON con las claves: \n")
	sb.WriteString("{ \"evaluation\": [ { \"question_id\": <int>, \"is_correct\": 0|1, \"fit\": <string> }... ], \"correct_answers\": <int>, \"fit_global\": <string> }\n")
	sb.WriteString("No incluyas texto fuera del JSON.\n")
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	threadID := req.Thread
	if strings.TrimSpace(threadID) == "" {
		var err error
		threadID, err = h.ai.CreateThread(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "assistant thread error"})
			return
		}
	}
	evalInstr := strings.Join([]string{
		"Responde estrictamente en JSON válido con las claves:",
		"evaluation: array de {question_id:int, is_correct:0|1, fit:string} (fit = feedback clínico breve por pregunta).",
		"correct_answers:int,",
		"fit_global:string = TEXTO MULTIPÁRRAFO estructurado (no HTML) con las siguientes secciones claramente separadas por doble salto de línea:\n" +
			"1) 'Puntaje y Calificación:' en la primera línea incluye: 'Total de respuestas correctas: X de N.' en la siguiente línea 'Puntaje: <porcentaje con 2 decimales>%.' y en la tercera línea 'Clasificación: <desempeño alto|desempeño adecuado|desempeño moderado|desempeño bajo>' según porcentaje (>=85 alto, 70-84 adecuado, 50-69 moderado, <50 bajo).\n" +
			"2) 'Retroalimentación:' párrafos (mínimo 2, máximo 5) que: a) Feliciten el esfuerzo. b) Destaquen fortalezas específicas detectadas. c) Señalen brevemente los patrones de error o conceptos a reforzar (sin listas extensas). d) Incluyan 1-2 recomendaciones prácticas de estudio. e) Inviten a solicitar explicación detallada de preguntas falladas.\n" +
			"3) 'Referencias:' una línea con 1–3 citas abreviadas de fuentes médicas de alto valor (ej: 'Harrison. Principios de Medicina Interna.' o 'PMID: 12345678').\n" +
			"Mantén formato de texto plano, sin viñetas, sin JSON dentro. Usa saltos de línea dobles entre secciones y simples entre oraciones dentro de cada párrafo.",
		// Instrucciones adicionales
		"Limita el contenido estrictamente a medicina interna.",
		"Usa exclusivamente información contenida en el vector vs_680fc484cef081918b2b9588b701e2f4 y en https://pubmed.ncbi.nlm.nih.gov/.",
		"No utilices otras fuentes ni inventes información.",
		"Incluye las citas en fit_global (por ejemplo, 'Harrison. Principios de Medicina Interna' o 'PMID: <número>'), sin mencionar el vector.",
		"Mantén coherencia, claridad y nivel académico acorde al nivel solicitado.",
		"Responde en español.",
		"No incluyas texto fuera del JSON.",
	}, " ")
	ch, err := h.ai.StreamAssistantJSON(ctx, threadID, sb.String(), evalInstr)
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
		// Happy path: return assistant result
		c.JSON(http.StatusOK, parsed)
		return
	}
	// Try a repair cycle on evaluation
	if repaired, ok := h.repairEvaluationJSON(ctx, threadID, content, evalInstr); ok {
		var parsed2 map[string]any
		if err := json.Unmarshal([]byte(repaired), &parsed2); err == nil {
			log.Printf("[testsapi.evaluate] repair succeeded; returning fixed JSON (uid=%s, test_id=%d)", req.UID, req.TestID)
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
	c.JSON(http.StatusOK, map[string]any{
		"evaluation":      eval,
		"correct_answers": 0,
		"fit_global":      "Evaluación automática sin modelo",
	})
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

// --- Repair helpers --- //

// repairGenerateJSON asks the assistant in the same thread to rewrite its last message
// as strict JSON with exactly n questions. Returns the repaired JSON string and whether it succeeded.
func (h *Handler) repairGenerateJSON(ctx context.Context, threadID, lastContent string, n int, baseInstr string) (string, bool) {
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
	ch, err := h.ai.StreamAssistantJSON(ctx, threadID, prompt.String(), baseInstr)
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
func (h *Handler) repairEvaluationJSON(ctx context.Context, threadID, lastContent, baseInstr string) (string, bool) {
	prompt := strings.Builder{}
	prompt.WriteString("Tu último mensaje no es JSON válido. Reescribe la evaluación como un único objeto JSON válido con las claves ")
	prompt.WriteString("evaluation (array de {question_id:int, is_correct:0|1, fit:string}), correct_answers:int y fit_global:string. ")
	prompt.WriteString("No incluyas texto fuera del JSON.\n\nMensaje previo:\n")
	prev := strings.TrimSpace(lastContent)
	if len(prev) > 4000 {
		prev = prev[:4000]
	}
	prompt.WriteString(prev)
	ch, err := h.ai.StreamAssistantJSON(ctx, threadID, prompt.String(), baseInstr)
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
