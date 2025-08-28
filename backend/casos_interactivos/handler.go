package casos_interactivos

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"fmt"

	"ema-backend/openai"

	"github.com/gin-gonic/gin"
)

// Assistant interface used by the handler
type Assistant interface {
	CreateThread(ctx context.Context) (string, error)
	StreamAssistantJSON(ctx context.Context, threadID, userPrompt, jsonInstructions string) (<-chan string, error)
}

type Handler struct {
	ai                 Assistant
	quotaValidator     func(ctx context.Context, c *gin.Context, flow string) error
	maxQuestions       int
	mu                 sync.Mutex
	turnCount          map[string]int // thread_id -> number of questions already asked
	threadMaxQuestions map[string]int // thread_id -> max questions for this specific thread
	askedQuestions     map[string][]string // thread_id -> list of question texts already asked (to reduce repetition)
	evalCorrect        map[string]int // thread_id -> count correct answers
	evalAnswers        map[string]int // thread_id -> total evaluated answers
	vectorID           string // knowledge vector id for references
}

// finalQuestion builds the canonical empty question object used when the case is finished.
// Mantiene las claves para que el frontend detecte cierre sin romper parsing.
func finalQuestion() map[string]any {
	return map[string]any{
		"tipo":     "",
		"texto":    "",
		"opciones": []string{},
	}
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
	vector := strings.TrimSpace(os.Getenv("INTERACTIVE_VECTOR_ID"))
	if vector == "" { vector = "vs_680fc484cef081918b2b9588b701e2f4" }
	return &Handler{
		ai:                 cli,
		maxQuestions:       maxQ,
		turnCount:          make(map[string]int),
		threadMaxQuestions: make(map[string]int),
		askedQuestions:     make(map[string][]string),
		evalCorrect:        make(map[string]int),
		evalAnswers:        make(map[string]int),
		vectorID:           vector,
	}
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
	Age             string `json:"age"`
	Sex             string `json:"sex"`
	Type            string `json:"type"`
	Pregnant        bool   `json:"pregnant"`
	MaxInteractions int    `json:"max_interactions,omitempty"`
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
	if h.threadMaxQuestions == nil {
		h.threadMaxQuestions = make(map[string]int)
	}
	if h.askedQuestions == nil { // lazy init in case of nil
		h.askedQuestions = make(map[string][]string)
	}
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "interactive_strict_start"); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "interactive clinical cases quota exceeded"})
			return
		}
	}
	if v, ok := c.Get("quota_remaining"); ok {
		c.Header("X-Quota-Remaining", toString(v))
		c.Header("X-Quota-Field", "clinical_cases")
	}
	var req startReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	// Use user's max interactions preference if provided, otherwise use default
	currentMaxQuestions := h.maxQuestions
	if req.MaxInteractions > 0 && req.MaxInteractions >= 3 && req.MaxInteractions <= 10 {
		currentMaxQuestions = req.MaxInteractions
	}

	threadID, err := h.ai.CreateThread(ctx)
	if err != nil {
		c.JSON(http.StatusOK, h.fallbackStart(req, ""))
		return
	}

	// Store the max questions for this specific thread
	if threadID != "" {
		h.mu.Lock()
		h.threadMaxQuestions[threadID] = currentMaxQuestions
		h.mu.Unlock()
	}

	userPrompt := strings.Join([]string{
		"INICIO DE SESIÓN (rol system): Recibirás el caso clínico completo y NO comentarás. Guarda para usar más adelante.",
		"FASE DE ANAMNESIS: En 'feedback' muestra la historia clínica completa (síntomas, antecedentes, contexto social/familiar, evolución) con 3-4 párrafos detallados.",
		"Incluye motivo de consulta, historia de la enfermedad actual, antecedentes, examen físico inicial. Al final del feedback añade una línea 'Referencias:' seguida de UNA cita abreviada basada en el vector " + h.vectorID + " (no menciones el id) o un PMID.",
		"Formato estricto: JSON con keys: feedback, next{hallazgos{}, pregunta{tipo:'single-choice', texto, opciones[4]}}, finish:0. Sin texto fuera del JSON.",
		"Paciente: edad=" + strings.TrimSpace(req.Age) + ", sexo=" + strings.TrimSpace(req.Sex) + ", gestante=" + boolToStr(req.Pregnant) + ".",
	}, " ")
	instr := strings.Join([]string{
		"Responde SOLO en JSON válido con claves: feedback, next{hallazgos, pregunta{tipo, texto, opciones}}, finish.",
		"'feedback' (200-300 palabras) termina con una línea 'Referencias:' y 1 cita (libro guía o PMID) sin mencionar vectores ni IDs internos.",
		"No omitas claves ni uses nombres distintos. Sin null ni cadenas vacías. Usa {} si no hay hallazgos. finish=0.",
		"Cada pregunta debe ser única y coherente con el caso. Idioma: español. Sin markdown.",
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
		// store initial question to help reduce repetition later
		if q := extractQuestionText(data); q != "" {
			h.askedQuestions[threadID] = append(h.askedQuestions[threadID], q)
		}
		h.mu.Unlock()
	}

	// Extract clinical history from the AI feedback
	clinicalHistory := "Historia clínica inicial proporcionada por el sistema."
	if feedback, ok := data["feedback"].(string); ok && strings.TrimSpace(feedback) != "" {
		clinicalHistory = feedback
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
			"anamnesis":            clinicalHistory,
			"physical_examination": "",
			"diagnostic_tests":     "",
			"final_diagnosis":      "",
			"management":           "",
		},
		"data":      data,
		"thread_id": threadID,
	}
	// Count the very first question delivered in StartCase as one interaction
	if threadID != "" {
		h.mu.Lock()
		if h.turnCount == nil { // safety
			h.turnCount = make(map[string]int)
		}
		h.turnCount[threadID] = 1
		h.mu.Unlock()
	}
	log.Printf("[InteractiveCase][Start] thread=%s max=%d turn=%d", threadID, currentMaxQuestions, 1)
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
	if h.threadMaxQuestions == nil {
		h.threadMaxQuestions = make(map[string]int)
	}
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "interactive_strict_message"); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "interactive clinical cases quota exceeded"})
			return
		}
	}
	if v, ok := c.Get("quota_remaining"); ok { c.Header("X-Quota-Remaining", toString(v)) }
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
	curr := h.getCount(threadID)              // questions already asked
	maxQuestions := h.getMaxQuestions(threadID)
	closing := curr >= maxQuestions           // we've already asked the max; now we only need final feedback
	log.Printf("[InteractiveCase][Message][Begin] thread=%s curr=%d max=%d closing=%v", threadID, curr, maxQuestions, closing)

	userPrompt := req.Mensaje
	var instr string
	if closing {
		instr = strings.Join([]string{
			"Responde SOLO en JSON válido con: feedback, next{hallazgos{}, pregunta{}}, finish(1).",
			"Formato 'feedback': primera línea 'Resumen Final:'; luego síntesis (≤80 palabras) con diagnóstico probable, diferenciales clave, manejo inicial.",
			"Después añade línea 'Puntaje:' con 'X/Y (<pct>%)' usando las veces etiquetadas como CORRECTO/INCORRECTO en evaluaciones previas (si no conoces datos coloca X=Y=0).",
			"Finalmente línea 'Referencias:' con 1-2 citas (primera de libro/guía, opcional PMID). NO menciones vectores ni IDs. Sin nueva pregunta. finish=1.",
			"Idioma: español. Sin texto fuera del JSON.",
		}, " ")
	} else {
		// Build a short memory of prior questions to discourage repetition
		var prevQs []string
		if threadID != "" {
			h.mu.Lock(); prevQs = append(prevQs, h.askedQuestions[threadID]...); h.mu.Unlock()
		}
		if len(prevQs) > 5 { // limit context length
			prevQs = prevQs[len(prevQs)-5:]
		}
		prevList := ""
		if len(prevQs) > 0 {
			for i, q := range prevQs { prevQs[i] = strings.TrimSpace(q) }
			prevList = "Preguntas previas (NO repetir ni variantes triviales): " + strings.Join(prevQs, " | ") + "."
		}
		instr = strings.Join([]string{
			"Responde SOLO en JSON válido con: feedback, next{hallazgos{}, pregunta{tipo:'single-choice'|'open_ended', texto, opciones}}, finish(0|1).",
			"Formato 'feedback': primera línea 'Evaluación: CORRECTO' o 'Evaluación: INCORRECTO' (en mayúsculas).",
			"Luego explicación breve (≤45 palabras) y última línea 'Fuente:' con UNA cita (libro/guía o PMID).",
			"NO repitas historia clínica inicial ni preguntas previas. Progresa lógicamente.",
			prevList,
			"Si decides cerrar por coherencia clínica pon finish=1 y NO generes pregunta (pregunta vacía).",
			"Evita repetir preguntas ya hechas. Sin texto fuera del JSON. Idioma: español. No menciones vectores ni IDs internos.",
		}, " ")
	}

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
	// If not closing: handle validation / repair of premature closure & ensure a question exists
	if !closing {
		preguntaValid := func() bool {
			next, ok := data["next"].(map[string]any)
			if !ok {
				return false
			}
			preg, ok := next["pregunta"].(map[string]any)
			if !ok {
				return false
			}
			texto, _ := preg["texto"].(string)
			opcionesAny, okAny := preg["opciones"].([]any)
			if !okAny {
				// try []string
				if arr, ok := preg["opciones"].([]string); ok {
					for _, v := range arr {
						_ = v
					}
					// convert to []any for length check
					opcionesAny = make([]any, len(arr))
					for i, v := range arr {
						opcionesAny[i] = v
					}
				}
			}
			tipo, _ := preg["tipo"].(string)
			if strings.TrimSpace(texto) == "" {
				return false
			}
			if strings.Contains(strings.ToLower(tipo), "single") && len(opcionesAny) < 2 {
				return false
			}
			return true
		}()
		if finF, ok := data["finish"].(float64); ok && finF == 1 {
			// prevent premature finish before reaching max unless assistant intentionally closed logically *and* we accept it
			// We relax: keep finish=1 only if there is no pregunta or it's empty; else reset to 0
			keep := false
			if nx, ok := data["next"].(map[string]any); ok {
				if pq, ok := nx["pregunta"].(map[string]any); ok {
					texto, _ := pq["texto"].(string)
					if strings.TrimSpace(texto) == "" { keep = true }
				} else { keep = true }
			}
			if !keep { data["finish"] = 0.0 }
		}
		if !preguntaValid {
			mt := h.minTurn()
			if nx, ok := data["next"].(map[string]any); ok {
				nx["pregunta"] = mt["next"].(map[string]any)["pregunta"]
			} else {
				data["next"] = mt["next"]
			}
			log.Printf("[InteractiveCase][Message][RepairedQuestion] thread=%s curr=%d max=%d", threadID, curr, maxQuestions)
		}
		// parse evaluation correctness (solo si no es cierre y feedback formateado)
		if threadID != "" {
			fb, _ := data["feedback"].(string)
			correct := parseEvalCorrect(fb)
			if correct >= 0 { // -1 indica no parseado
				h.mu.Lock()
				h.evalAnswers[threadID] = h.evalAnswers[threadID] + 1
				if correct == 1 { h.evalCorrect[threadID] = h.evalCorrect[threadID] + 1 }
				h.mu.Unlock()
			}
		}
		// record asked question to reduce repetition
		if q := extractQuestionText(data); q != "" && threadID != "" {
			h.mu.Lock(); h.askedQuestions[threadID] = append(h.askedQuestions[threadID], q); h.mu.Unlock()
		}
		// Increment counter only when a new question is produced
		h.incrementCount(threadID)

		// Safety: si tras incrementar alcanzamos o superamos el máximo, forzamos cierre coherente
		if h.getCount(threadID) >= maxQuestions {
			forceFinishInteractive(data, threadID, h)
		}
	} else {
		// Force closure normalization: usar estructura vacía esperada por frontend (claves presentes, valores vacíos)
		forceFinishInteractive(data, threadID, h)
	}
	log.Printf("[InteractiveCase][Message][Return] thread=%s count=%d max=%d finish=%v closing=%v", threadID, h.getCount(threadID), maxQuestions, data["finish"], closing)
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
			"anamnesis":            "Caso clínico básico generado por el sistema de respaldo.",
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

func (h *Handler) getMaxQuestions(threadID string) int {
	if threadID == "" {
		return h.maxQuestions
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if max, exists := h.threadMaxQuestions[threadID]; exists {
		return max
	}
	return h.maxQuestions
}

// extractQuestionText pulls pregunta.texto from a data turn
func extractQuestionText(data map[string]any) string {
	if data == nil { return "" }
	next, ok := data["next"].(map[string]any); if !ok { return "" }
	preg, ok := next["pregunta"].(map[string]any); if !ok { return "" }
	txt, _ := preg["texto"].(string)
	return strings.TrimSpace(txt)
}

// parseEvalCorrect analiza la primera línea del feedback en busca de 'Evaluación: CORRECTO' o 'Evaluación: INCORRECTO'
// Devuelve 1 correcto, 0 incorrecto, -1 si no se pudo parsear.
func parseEvalCorrect(feedback string) int {
	line := strings.Split(strings.TrimSpace(feedback), "\n")[0]
	l := strings.ToUpper(strings.TrimSpace(line))
	if strings.Contains(l, "EVALUACIÓN:") {
		if strings.Contains(l, "CORRECTO") { return 1 }
		if strings.Contains(l, "INCORRECTO") { return 0 }
	}
	return -1
}

// forceFinishInteractive normaliza estructura de cierre y agrega puntaje si hay datos.
func forceFinishInteractive(data map[string]any, threadID string, h *Handler) {
	fq := finalQuestion()
	if nx, ok := data["next"].(map[string]any); ok {
		nx["hallazgos"] = map[string]any{}
		nx["pregunta"] = fq
	} else {
		data["next"] = map[string]any{"hallazgos": map[string]any{}, "pregunta": fq}
	}
	data["finish"] = 1
	h.mu.Lock()
	corr := h.evalCorrect[threadID]
	total := h.evalAnswers[threadID]
	h.mu.Unlock()
	if fb, ok := data["feedback"].(string); ok {
		if !strings.Contains(fb, "Puntaje:") { // evitar duplicar si el assistant ya lo puso
			pct := 0.0
			if total > 0 { pct = (float64(corr) / float64(total)) * 100.0 }
			resumen := fb
			if !strings.Contains(strings.ToLower(fb), "resumen final") {
				resumen = "Resumen Final:\n" + fb
			}
			summaryLine := fmt.Sprintf("Puntaje: %d/%d (%.1f%%)", corr, total, pct)
			if !strings.Contains(resumen, summaryLine) {
				resumen = resumen + "\n" + summaryLine
			}
            if !strings.Contains(strings.ToLower(resumen), "referencias:") {
                resumen = resumen + "\nReferencias: Fuente clínica estándar"
            }
            data["feedback"] = resumen
        }
    }
    data["status"] = "finished"
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

// helper for quota header (duplicate small util to avoid cross-package import tangle)
func toString(v interface{}) string {
	switch t := v.(type) {
	case string: return t
	case int: return strconv.Itoa(t)
	case int64: return strconv.FormatInt(t,10)
	default: return ""
	}
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
