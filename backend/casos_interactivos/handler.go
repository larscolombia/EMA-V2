package casos_interactivos

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"ema-backend/openai"

	"github.com/gin-gonic/gin"
)

// Assistant interface used by the handler
type Assistant interface {
	CreateThread(ctx context.Context) (string, error)
	StreamAssistantJSON(ctx context.Context, threadID, userPrompt, jsonInstructions string) (<-chan string, error)
}

type Handler struct {
	ai             Assistant
	quotaValidator func(ctx context.Context, c *gin.Context, flow string) error
	maxQuestions   int
	mu             sync.Mutex
	turnCount      map[string]int // thread_id -> number of questions already asked
}

// DefaultHandler builds the assistant client from env
// Uses CASOS_INTERACTIVOS_ASSISTANT if provided; otherwise falls back to global AssistantID.
func DefaultHandler() *Handler {
	cli := openai.NewClient()
	if id := os.Getenv("CASOS_INTERACTIVOS_ASSISTANT"); strings.TrimSpace(id) != "" {
		cli.AssistantID = id
	}
	// Read max questions from env (default 4)
	maxQ := 4
	if s := strings.TrimSpace(os.Getenv("CASOS_INTERACTIVOS_MAX_PREGUNTAS")); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			maxQ = v
		}
	}
	return &Handler{ai: cli, maxQuestions: maxQ, turnCount: make(map[string]int)}
}

// RegisterRoutes wires only interactive endpoints for the new interactive flow contract
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/casos-interactivos/iniciar", h.StartCase)
	r.POST("/casos-interactivos/mensaje", h.Message)
}

// SetQuotaValidator allows injecting a plan/quota validator
// flow: "interactive_strict_start" | "interactive_strict_message"
func (h *Handler) SetQuotaValidator(fn func(ctx context.Context, c *gin.Context, flow string) error) {
	h.quotaValidator = fn
}

// --- Models --- //

type startReq struct {
	Age      string `json:"age"`
	Sex      string `json:"sex"`
	Type     string `json:"type"`
	Pregnant bool   `json:"pregnant"`
}

type messageReq struct {
	ThreadID string `json:"thread_id"`
	Mensaje  string `json:"mensaje"`
}

// --- Handlers --- //

// StartCase: initializes an interactive case and returns the first turn strictly as the new interactive JSON
// Response shape:
//
//	{
//	  "case": { ... patient profile & narrative ... },
//	  "data": {
//	     "feedback": string,
//	     "next": { "hallazgos": object, "pregunta": { "tipo":"single-choice", "texto": string, "opciones": []string } },
//	     "finish": 0
//	  },
//	  "thread_id": string
//	}
func (h *Handler) StartCase(c *gin.Context) {
	// Ensure counters map exists
	if h.turnCount == nil {
		h.turnCount = make(map[string]int)
	}
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "interactive_strict_start"); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "interactive clinical cases quota exceeded"})
			return
		}
	}
	var req startReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	threadID, err := h.ai.CreateThread(ctx)
	if err != nil {
		c.JSON(http.StatusOK, h.fallbackStart(req, ""))
		return
	}

	userPrompt := strings.Join([]string{
		"INICIO DE SESIÓN (rol system): Recibirás el caso clínico completo y NO comentarás. Guarda para usar más adelante.",
		"FASE DE ANAMNESIS: Muestra historia clínica (síntomas, antecedentes, contexto) y formula una primera pregunta de opción única sobre aproximación diagnóstica.",
		"Formato estricto: JSON con keys: feedback, next{hallazgos{}, pregunta{tipo:'single-choice', texto, opciones[4]}}, finish:0. Sin texto fuera del JSON.",
		"Paciente: edad=" + strings.TrimSpace(req.Age) + ", sexo=" + strings.TrimSpace(req.Sex) + ", gestante=" + boolToStr(req.Pregnant) + ".",
	}, " ")
	instr := strings.Join([]string{
		"Responde SOLO en JSON válido con claves: feedback, next{hallazgos, pregunta{tipo, texto, opciones}}, finish.",
		"No omitas claves ni uses nombres distintos. Sin null ni cadenas vacías. Usa {} si no hay hallazgos. finish=0.",
		"Cada pregunta debe ser única y coherente con el caso.",
		"Fuentes: Vector (cita solo el libro) o PubMed con referencia completa. Alterna fuentes entre respuestas.",
		"Idioma: español. Sin markdown.",
	}, " ")

	ch, err := h.ai.StreamAssistantJSON(ctx, threadID, userPrompt, instr)
	if err != nil {
		c.JSON(http.StatusOK, h.fallbackStart(req, threadID))
		return
	}

	var content string
	select {
	case content = <-ch:
	case <-ctx.Done():
		c.JSON(http.StatusOK, h.fallbackStart(req, threadID))
		return
	}

	jsonStr := extractJSON(content)
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil || !validInteractiveTurn(data) {
		if fixed := h.repairTurn(ctx, threadID, content); fixed != "" {
			_ = json.Unmarshal([]byte(fixed), &data)
		}
	}
	if !validInteractiveTurn(data) {
		data = h.minTurn()
	}

	// Initialize turn counter: first question was asked in this turn
	if threadID != "" {
		h.mu.Lock()
		h.turnCount[threadID] = 1
		h.mu.Unlock()
	}

	resp := map[string]any{
		"case": map[string]any{
			"id":                   0,
			"title":                "Caso clínico interactivo",
			"type":                 "interactive",
			"age":                  strings.TrimSpace(req.Age),
			"sex":                  strings.TrimSpace(req.Sex),
			"gestante":             boolToInt(req.Pregnant),
			"is_real":              1,
			"anamnesis":            "Historia clínica inicial proporcionada por el sistema.",
			"physical_examination": "",
			"diagnostic_tests":     "",
			"final_diagnosis":      "",
			"management":           "",
		},
		"data":      data,
		"thread_id": threadID,
	}
	c.JSON(http.StatusOK, resp)
}

// Message: processes an answer and returns the next strict interactive turn JSON
// Response shape:
// { "data": { "feedback": string, "next": { "hallazgos": object, "pregunta": { ... } }, "finish": 0|1, "thread_id": string } }
func (h *Handler) Message(c *gin.Context) {
	// Ensure counters map exists
	if h.turnCount == nil {
		h.turnCount = make(map[string]int)
	}
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "interactive_strict_message"); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "interactive clinical cases quota exceeded"})
			return
		}
	}
	var req messageReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Mensaje) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()
	threadID := strings.TrimSpace(req.ThreadID)
	if threadID == "" {
		if id, err := h.ai.CreateThread(ctx); err == nil {
			threadID = id
		}
	}

	// Enforce max questions policy using per-thread counters
	curr := h.getCount(threadID)
	if curr >= h.maxQuestions {
		// Already at or over the limit: return final closure turn
		final := h.minTurn()
		final["finish"] = 1
		final["next"].(map[string]any)["hallazgos"] = map[string]any{}
		final["next"].(map[string]any)["pregunta"] = map[string]any{}
		c.JSON(http.StatusOK, map[string]any{"data": withThread(final, threadID)})
		return
	}

	userPrompt := req.Mensaje
	closeHint := ""
	if curr == h.maxQuestions-1 {
		closeHint = " Esta es la pregunta de cierre (cuarta)."
	}
	instr := strings.Join([]string{
		"Responde SOLO en JSON válido con: feedback, next{hallazgos{}, pregunta{tipo:'single-choice'|'open_ended', texto, opciones}}, finish(0|1).",
		"No repitas preguntas previas, progresa lógicamente; cierra en la cuarta con finish=1 y pregunta/hallazgos vacíos.",
		closeHint,
		"Fuentes: Vector o PubMed (varía la referencia). Sin texto fuera del JSON.",
		"Idioma: español.",
	}, " ")

	ch, err := h.ai.StreamAssistantJSON(ctx, threadID, userPrompt, instr)
	if err != nil {
		c.JSON(http.StatusOK, map[string]any{"data": withThread(h.minTurn(), threadID)})
		return
	}
	var content string
	select {
	case content = <-ch:
	case <-ctx.Done():
		c.JSON(http.StatusOK, map[string]any{"data": withThread(h.minTurn(), threadID)})
		return
	}

	jsonStr := extractJSON(content)
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil || !validInteractiveTurn(data) {
		if fixed := h.repairTurn(ctx, threadID, content); fixed != "" {
			_ = json.Unmarshal([]byte(fixed), &data)
		}
	}
	if !validInteractiveTurn(data) {
		data = h.minTurn()
	}
	// Increment counter after generating the next question/turn
	h.incrementCount(threadID)
	// If we've reached the limit now, enforce closure on this returned turn
	if h.getCount(threadID) > h.maxQuestions {
		data["finish"] = 1
		if nx, ok := data["next"].(map[string]any); ok {
			nx["hallazgos"] = map[string]any{}
			nx["pregunta"] = map[string]any{}
		} else {
			data["next"] = map[string]any{"hallazgos": map[string]any{}, "pregunta": map[string]any{}}
		}
	}
	c.JSON(http.StatusOK, map[string]any{"data": withThread(data, threadID)})
}

// --- Helpers --- //

// fallbackStart builds a minimal but valid response when assistant fails
func (h *Handler) fallbackStart(req startReq, threadID string) map[string]any {
	return map[string]any{
		"case": map[string]any{
			"id":                   0,
			"title":                "Caso clínico interactivo",
			"type":                 "interactive",
			"age":                  strings.TrimSpace(req.Age),
			"sex":                  strings.TrimSpace(req.Sex),
			"gestante":             boolToInt(req.Pregnant),
			"is_real":              1,
			"anamnesis":            "Historia clínica inicial no disponible.",
			"physical_examination": "",
			"diagnostic_tests":     "",
			"final_diagnosis":      "",
			"management":           "",
		},
		"data":      h.minTurn(),
		"thread_id": threadID,
	}
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

func validInteractiveTurn(m map[string]any) bool {
	if m == nil {
		return false
	}
	if _, ok := m["feedback"].(string); !ok {
		return false
	}
	next, ok := m["next"].(map[string]any)
	if !ok {
		return false
	}
	if _, ok := next["hallazgos"].(map[string]any); !ok {
		return false
	}
	if _, ok := next["pregunta"].(map[string]any); !ok {
		return false
	}
	if _, ok := m["finish"]; !ok {
		return false
	}
	return true
}

func (h *Handler) minTurn() map[string]any {
	return map[string]any{
		"feedback": "",
		"next": map[string]any{
			"hallazgos": map[string]any{},
			"pregunta": map[string]any{
				"tipo":     "single-choice",
				"texto":    "¿Cuál es el siguiente mejor paso diagnóstico?",
				"opciones": []string{"A", "B", "C", "D"},
			},
		},
		"finish": 0,
	}
}

func withThread(data map[string]any, threadID string) map[string]any {
	if data == nil {
		data = map[string]any{}
	}
	data["thread_id"] = threadID
	return data
}

// turn counter helpers
func (h *Handler) getCount(threadID string) int {
	if threadID == "" {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.turnCount[threadID]
}
func (h *Handler) incrementCount(threadID string) {
	if threadID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.turnCount[threadID] = h.turnCount[threadID] + 1
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

// Attempt a one-shot repair to force strict JSON turn
func (h *Handler) repairTurn(ctx context.Context, threadID, lastContent string) string {
	prompt := strings.Builder{}
	prompt.WriteString("Reescribe tu último mensaje como JSON estricto con claves: feedback, next{hallazgos{}, pregunta{tipo, texto, opciones}}, finish(0|1). ")
	prompt.WriteString("Sin texto fuera del JSON. Usa {} en hallazgos si no hay nuevos.\n\nMensaje previo:\n")
	prev := strings.TrimSpace(lastContent)
	if len(prev) > 4000 {
		prev = prev[:4000]
	}
	prompt.WriteString(prev)
	ch, err := h.ai.StreamAssistantJSON(ctx, threadID, prompt.String(), "Responde SOLO JSON válido")
	if err != nil {
		return ""
	}
	select {
	case fixed := <-ch:
		fixed = extractJSON(fixed)
		if json.Valid([]byte(fixed)) {
			return fixed
		}
	case <-ctx.Done():
		return ""
	}
	return ""
}
