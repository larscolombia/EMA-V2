package casos_clinico

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"ema-backend/openai"

	"github.com/gin-gonic/gin"
)

// Assistant matches the minimal interface implemented by openai.Client
type Assistant interface {
	CreateThread(ctx context.Context) (string, error)
	StreamAssistantMessage(ctx context.Context, threadID, prompt string) (<-chan string, error)
	StreamAssistantJSON(ctx context.Context, threadID, userPrompt, jsonInstructions string) (<-chan string, error)
}

type Handler struct {
	aiAnalytical   Assistant
	aiInteractive  Assistant
	quotaValidator func(ctx context.Context, c *gin.Context, flow string) error
}

// NewHandler lets you inject different assistants (analytical/interactive). If one is nil, the other will be used for both flows.
func NewHandler(analytical Assistant, interactive Assistant) *Handler {
	if analytical == nil && interactive == nil {
		cli := openai.NewClient()
		return &Handler{aiAnalytical: cli, aiInteractive: cli}
	}
	if analytical == nil {
		analytical = interactive
	}
	if interactive == nil {
		interactive = analytical
	}
	return &Handler{aiAnalytical: analytical, aiInteractive: interactive}
}

// DefaultHandler configures assistants from env:
// - CASOS_CLINICOS_ANALITICO: Assistant ID for analytical flow (static)
// - CASOS_CLINICOS_INTERACTIVO: Assistant ID for interactive flow
func DefaultHandler() *Handler {
	// Analytical client
	cliA := openai.NewClient()
	if id := os.Getenv("CASOS_CLINICOS_ANALITICO"); strings.TrimSpace(id) != "" {
		cliA.AssistantID = id
	}
	// Interactive client (may reuse analytical)
	var cliI *openai.Client
	if id := os.Getenv("CASOS_CLINICOS_INTERACTIVO"); strings.TrimSpace(id) != "" {
		cliI = openai.NewClient()
		cliI.AssistantID = id
	} else {
		cliI = cliA
	}
	return &Handler{aiAnalytical: cliA, aiInteractive: cliI}
}

// RegisterRoutes wires endpoints expected by the Flutter client.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/caso-clinico", h.GenerateAnalytical)
	r.POST("/casos-clinicos/conversar", h.ChatAnalytical)
	r.POST("/casos-clinicos/interactivo", h.GenerateInteractive)
	r.POST("/casos-clinicos/interactivo/conversar", h.ChatInteractive)
}

// SetQuotaValidator allows injecting a plan/quota validator.
// flow is one of: "analytical_generate", "analytical_chat", "interactive_generate", "interactive_chat".
func (h *Handler) SetQuotaValidator(fn func(ctx context.Context, c *gin.Context, flow string) error) {
	h.quotaValidator = fn
}

// --- Request models --- //

type generateReq struct {
	Age      string `json:"age"`
	Sex      string `json:"sex"`
	Type     string `json:"type"`
	Pregnant bool   `json:"pregnant"`
}

type chatReq struct {
	ThreadID string `json:"thread_id"`
	Mensaje  string `json:"mensaje"`
}

// --- Handlers --- //

// GenerateAnalytical creates a static case (analytical) and returns { case: {...}, thread_id: "..." }
func (h *Handler) GenerateAnalytical(c *gin.Context) {
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "analytical_generate"); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "clinical cases quota exceeded"})
			return
		}
	}
	var req generateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	threadID, err := h.aiAnalytical.CreateThread(ctx)
	if err != nil {
		// Fallback: synthesize minimal case
		c.JSON(http.StatusOK, map[string]any{
			"case": map[string]any{
				"id":                   0,
				"title":                "Caso clínico analítico",
				"type":                 "static",
				"age":                  strings.TrimSpace(req.Age),
				"sex":                  strings.TrimSpace(req.Sex),
				"pregnant":             req.Pregnant,
				"gestante":             boolToInt(req.Pregnant),
				"is_real":              1,
				"anamnesis":            "Paciente con motivo de consulta no especificado.",
				"physical_examination": "Examen físico sin datos relevantes.",
				"diagnostic_tests":     "Sin pruebas diagnósticas realizadas.",
				"final_diagnosis":      "Diagnóstico reservado.",
				"management":           "Manejo expectante.",
			},
			"thread_id": "",
		})
		return
	}

	// Build prompt for JSON-only case
	userPrompt := strings.Join([]string{
		"Genera un único objeto JSON con la clave 'case' describiendo un caso clínico completo (estático).",
		"El paciente: edad=" + strings.TrimSpace(req.Age) + ", sexo=" + strings.TrimSpace(req.Sex) + ", gestante=" + boolToStr(req.Pregnant) + ".",
		"Incluye anamnesis, examen físico, pruebas diagnósticas, diagnóstico final y plan de manejo.",
		"No incluyas texto fuera del JSON.",
	}, " ")
	instr := strings.Join([]string{
		"Responde estrictamente en JSON válido con la clave 'case'.",
		"'case' debe contener las claves: id(int), title(string), type('static'), age(string), sex(string), gestante(0|1) o pregnant(true|false), is_real(0|1),",
		"anamnesis(string), physical_examination(string), diagnostic_tests(string), final_diagnosis(string), management(string).",
		"Usa exclusivamente información del assistant y PubMed si es necesario; no menciones fuentes privadas.",
		"Idioma: español. Sin markdown ni texto adicional.",
	}, " ")

	ch, err := h.aiAnalytical.StreamAssistantJSON(ctx, threadID, userPrompt, instr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "assistant error"})
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
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil || parsed["case"] == nil {
		// Try repair once
		if fixed, ok := h.repairAnalyticalJSON(ctx, threadID, content); ok {
			parsed = map[string]any{}
			_ = json.Unmarshal([]byte(fixed), &parsed)
		} else {
			parsed = map[string]any{}
		}
	}
	// Ensure minimal shape
	if _, ok := parsed["case"]; !ok {
		parsed["case"] = map[string]any{
			"id":                   0,
			"title":                "Caso clínico analítico",
			"type":                 "static",
			"age":                  strings.TrimSpace(req.Age),
			"sex":                  strings.TrimSpace(req.Sex),
			"gestante":             boolToInt(req.Pregnant),
			"is_real":              1,
			"anamnesis":            "Paciente sin datos suficientes.",
			"physical_examination": "Examen físico sin hallazgos.",
			"diagnostic_tests":     "No se realizaron pruebas.",
			"final_diagnosis":      "Reservado.",
			"management":           "Sintomático.",
		}
	}
	parsed["thread_id"] = threadID
	c.JSON(http.StatusOK, parsed)
}

// ChatAnalytical continues the analytical chat and returns { respuesta: { text: string } }
func (h *Handler) ChatAnalytical(c *gin.Context) {
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "analytical_chat"); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "clinical cases quota exceeded"})
			return
		}
	}
	var req chatReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Mensaje) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()
	threadID := strings.TrimSpace(req.ThreadID)
	if threadID == "" {
		id, err := h.aiAnalytical.CreateThread(ctx)
		if err == nil {
			threadID = id
		}
	}
	// Ask assistant to respond with JSON-wrapped message to avoid extra formatting
	userPrompt := req.Mensaje
	instr := strings.Join([]string{
		"Responde estrictamente en JSON válido con la clave 'respuesta': { 'text': <string> }.",
		"Mantén un tono docente, breve y conversacional acorde a un caso clínico analítico en 10 turnos máximo.",
		"No incluyas títulos, enumeraciones rígidas ni markdown.",
		"Idioma: español.",
	}, " ")
	ch, err := h.aiAnalytical.StreamAssistantJSON(ctx, threadID, userPrompt, instr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "assistant error"})
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
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil || parsed["respuesta"] == nil {
		if fixed, ok := h.repairAnalyticalChatJSON(ctx, threadID, content); ok {
			parsed = map[string]any{}
			_ = json.Unmarshal([]byte(fixed), &parsed)
			c.JSON(http.StatusOK, parsed)
			return
		}
	} else {
		c.JSON(http.StatusOK, parsed)
		return
	}
	// Fallback text
	c.JSON(http.StatusOK, map[string]any{"respuesta": map[string]any{"text": strings.TrimSpace(content)}})
}

// GenerateInteractive creates the case and an initial question: returns { case: {...}, data: { questions: { texto,tipo,opciones } }, thread_id }
func (h *Handler) GenerateInteractive(c *gin.Context) {
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "interactive_generate"); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "clinical cases quota exceeded"})
			return
		}
	}
	var req generateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	threadID, err := h.aiInteractive.CreateThread(ctx)
	if err != nil {
		c.JSON(http.StatusOK, map[string]any{
			"case": map[string]any{
				"id":                   0,
				"title":                "Caso clínico interactivo",
				"type":                 "interactive",
				"age":                  strings.TrimSpace(req.Age),
				"sex":                  strings.TrimSpace(req.Sex),
				"gestante":             boolToInt(req.Pregnant),
				"is_real":              1,
				"anamnesis":            "Paciente con motivo de consulta no especificado.",
				"physical_examination": "Examen físico sin datos relevantes.",
				"diagnostic_tests":     "Sin pruebas diagnósticas realizadas.",
				"final_diagnosis":      "Diagnóstico reservado.",
				"management":           "Manejo expectante.",
			},
			"data": map[string]any{
				"questions": map[string]any{
					"texto":    "¿Qué síntoma clave ampliarías primero?",
					"tipo":     "open_ended",
					"opciones": []string{},
				},
			},
			"thread_id": "",
		})
		return
	}

	userPrompt := strings.Join([]string{
		"Genera un objeto JSON con 'case' y 'data.questions' (pregunta inicial) para un caso clínico interactivo.",
		"Perfil: edad=" + strings.TrimSpace(req.Age) + ", sexo=" + strings.TrimSpace(req.Sex) + ", gestante=" + boolToStr(req.Pregnant) + ".",
		"La pregunta inicial debe ser breve y abierta (texto), o bien de opción única con 4 opciones.",
		"No incluyas texto fuera del JSON.",
	}, " ")
	instr := strings.Join([]string{
		"Responde en JSON válido con las claves: 'case' y 'data'.",
		"'case' con: id, title, type('interactive'), age, sex, gestante(0|1) o pregnant, is_real, anamnesis, physical_examination, diagnostic_tests, final_diagnosis, management.",
		"'data': { 'questions': { 'texto': string, 'tipo': 'open_ended'|'single_choice', 'opciones': array<string> } }.",
		"Idioma: español. Sin markdown ni texto adicional.",
	}, " ")
	ch, err := h.aiInteractive.StreamAssistantJSON(ctx, threadID, userPrompt, instr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "assistant error"})
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
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil || parsed["case"] == nil {
		if fixed, ok := h.repairInteractiveGenerateJSON(ctx, threadID, content); ok {
			parsed = map[string]any{}
			_ = json.Unmarshal([]byte(fixed), &parsed)
		} else {
			parsed = map[string]any{}
		}
	}
	parsed["thread_id"] = threadID
	// Ensure minimal question present
	ensureInteractiveDefaults(parsed, req)
	c.JSON(http.StatusOK, parsed)
}

// ChatInteractive processes a user message and returns feedback + next question: { data: { feedback, question, thread_id } }
func (h *Handler) ChatInteractive(c *gin.Context) {
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "interactive_chat"); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "clinical cases quota exceeded"})
			return
		}
	}
	var req chatReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Mensaje) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()
	threadID := strings.TrimSpace(req.ThreadID)
	if threadID == "" {
		id, err := h.aiInteractive.CreateThread(ctx)
		if err == nil {
			threadID = id
		}
	}
	userPrompt := req.Mensaje
	instr := strings.Join([]string{
		"Responde estrictamente en JSON válido con la clave 'data' que contenga:",
		"feedback: string (retroalimentación breve < 40 palabras) y",
		"question: { texto: string, tipo: 'open_ended'|'single_choice', opciones: array<string> } para el siguiente paso.",
		"Mantén un flujo de unos 10 turnos desde anamnesis hasta manejo.",
		"Idioma: español. Sin texto fuera del JSON.",
	}, " ")
	ch, err := h.aiInteractive.StreamAssistantJSON(ctx, threadID, userPrompt, instr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "assistant error"})
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
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil || parsed["data"] == nil {
		if fixed, ok := h.repairInteractiveChatJSON(ctx, threadID, content); ok {
			parsed = map[string]any{}
			_ = json.Unmarshal([]byte(fixed), &parsed)
		} else {
			parsed = map[string]any{}
		}
	}
	if _, ok := parsed["data"].(map[string]any); !ok {
		parsed["data"] = map[string]any{}
	}
	parsed["data"].(map[string]any)["thread_id"] = threadID
	// Ensure minimal shape
	if _, ok := parsed["data"].(map[string]any)["question"]; !ok {
		parsed["data"].(map[string]any)["question"] = map[string]any{
			"texto":    "Propón tu diagnóstico diferencial principal.",
			"tipo":     "open_ended",
			"opciones": []string{},
		}
	}
	if _, ok := parsed["data"].(map[string]any)["feedback"]; !ok {
		parsed["data"].(map[string]any)["feedback"] = "Respuesta registrada."
	}
	c.JSON(http.StatusOK, parsed)
}

// --- Helpers --- //

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

func ensureInteractiveDefaults(parsed map[string]any, req generateReq) {
	// Ensure 'case'
	if _, ok := parsed["case"]; !ok {
		parsed["case"] = map[string]any{
			"id":                   0,
			"title":                "Caso clínico interactivo",
			"type":                 "interactive",
			"age":                  strings.TrimSpace(req.Age),
			"sex":                  strings.TrimSpace(req.Sex),
			"gestante":             boolToInt(req.Pregnant),
			"is_real":              1,
			"anamnesis":            "Paciente con motivo de consulta no especificado.",
			"physical_examination": "Examen físico sin datos relevantes.",
			"diagnostic_tests":     "Sin pruebas diagnósticas realizadas.",
			"final_diagnosis":      "Diagnóstico reservado.",
			"management":           "Manejo expectante.",
		}
	}
	// Ensure 'data.questions'
	data, _ := parsed["data"].(map[string]any)
	if data == nil {
		data = map[string]any{}
		parsed["data"] = data
	}
	if _, ok := data["questions"]; !ok {
		data["questions"] = map[string]any{
			"texto":    "¿Qué síntoma clave ampliarías primero?",
			"tipo":     "open_ended",
			"opciones": []string{},
		}
	}
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// --- Repair helpers (JSON) --- //

// repairAnalyticalJSON asks to rewrite as valid JSON with 'case' keys.
func (h *Handler) repairAnalyticalJSON(ctx context.Context, threadID, lastContent string) (string, bool) {
	prompt := strings.Builder{}
	prompt.WriteString("Tu último mensaje debe reescribirse como un único objeto JSON válido con la clave 'case' y las claves requeridas. ")
	prompt.WriteString("Incluye: id, title, type('static'), age, sex, gestante(0|1) o pregnant, is_real, anamnesis, physical_examination, diagnostic_tests, final_diagnosis, management. ")
	prompt.WriteString("No incluyas texto fuera del JSON.\n\nMensaje previo:\n")
	prev := strings.TrimSpace(lastContent)
	if len(prev) > 4000 {
		prev = prev[:4000]
	}
	prompt.WriteString(prev)
	instr := "Responde estrictamente en JSON válido con la clave 'case'."
	ch, err := h.aiAnalytical.StreamAssistantJSON(ctx, threadID, prompt.String(), instr)
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

// repairInteractiveGenerateJSON ensures both 'case' and 'data.questions'.
func (h *Handler) repairInteractiveGenerateJSON(ctx context.Context, threadID, lastContent string) (string, bool) {
	prompt := strings.Builder{}
	prompt.WriteString("Reescribe como JSON válido con 'case' y 'data.questions' {texto,tipo,opciones}. ")
	prompt.WriteString("'case' debe tener id,title,type('interactive'),age,sex,gestante/pregnant,is_real,anamnesis,physical_examination,diagnostic_tests,final_diagnosis,management. ")
	prompt.WriteString("No incluyas texto fuera del JSON.\n\nMensaje previo:\n")
	prev := strings.TrimSpace(lastContent)
	if len(prev) > 4000 {
		prev = prev[:4000]
	}
	prompt.WriteString(prev)
	instr := "Responde estrictamente en JSON válido con las claves 'case' y 'data'."
	ch, err := h.aiInteractive.StreamAssistantJSON(ctx, threadID, prompt.String(), instr)
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

// repairAnalyticalChatJSON ensures {respuesta:{text}}.
func (h *Handler) repairAnalyticalChatJSON(ctx context.Context, threadID, lastContent string) (string, bool) {
	prompt := strings.Builder{}
	prompt.WriteString("Reescribe tu respuesta como JSON válido: {\\\"respuesta\\\":{\\\"text\\\":<string>}}. Sin texto adicional.\n\nMensaje previo:\n")
	prev := strings.TrimSpace(lastContent)
	if len(prev) > 4000 {
		prev = prev[:4000]
	}
	prompt.WriteString(prev)
	instr := "Responde estrictamente en JSON válido con la clave 'respuesta'."
	ch, err := h.aiAnalytical.StreamAssistantJSON(ctx, threadID, prompt.String(), instr)
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

// repairInteractiveChatJSON ensures {data:{feedback,question{...}}}.
func (h *Handler) repairInteractiveChatJSON(ctx context.Context, threadID, lastContent string) (string, bool) {
	prompt := strings.Builder{}
	prompt.WriteString("Reescribe como JSON válido con 'data': { 'feedback': string, 'question': { 'texto': string, 'tipo': 'open_ended'|'single_choice', 'opciones': array<string> } }. ")
	prompt.WriteString("Sin texto fuera del JSON.\n\nMensaje previo:\n")
	prev := strings.TrimSpace(lastContent)
	if len(prev) > 4000 {
		prev = prev[:4000]
	}
	prompt.WriteString(prev)
	instr := "Responde estrictamente en JSON válido con la clave 'data'."
	ch, err := h.aiInteractive.StreamAssistantJSON(ctx, threadID, prompt.String(), instr)
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
