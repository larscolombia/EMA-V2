package casos_interactivos

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"ema-backend/openai"

	"github.com/gin-gonic/gin"
)

// Assistant interface used by the handler
type Assistant interface {
	CreateThread(ctx context.Context) (string, error)
	StreamAssistantJSON(ctx context.Context, threadID, userPrompt, jsonInstructions string) (<-chan string, error)
	// Búsqueda de evidencia (RAG + PubMed)
	SearchInVectorStore(ctx context.Context, vectorStoreID, query string) (string, error)
	SearchInVectorStoreWithMetadata(ctx context.Context, vectorStoreID, query string) (*openai.VectorSearchResult, error)
	SearchPubMed(ctx context.Context, query string) (string, error)
}

type Handler struct {
	ai                 Assistant
	quotaValidator     func(ctx context.Context, c *gin.Context, flow string) error
	maxQuestions       int
	mu                 sync.Mutex
	turnCount          map[string]int      // thread_id -> number of questions already asked
	threadMaxQuestions map[string]int      // thread_id -> max questions for this specific thread
	askedQuestions     map[string][]string // thread_id -> list of question texts already asked (to reduce repetition)
	evalCorrect        map[string]int      // thread_id -> count correct answers
	evalAnswers        map[string]int      // thread_id -> total evaluated answers
	vectorID           string              // knowledge vector id for references
	// local evaluation support
	lastCorrectIndex  map[string]int      // thread_id -> correct index of last question
	lastOptions       map[string][]string // thread_id -> slice of option texts of last question
	lastQuestionText  map[string]string   // thread_id -> texto de la última pregunta
	missingCorrectIdx map[string]int      // thread_id -> veces que faltó correct_index (para métricas)
	closureDue        map[string]bool     // thread_id -> se alcanzó max y el próximo turno debe ser cierre
}

// evaluateLastAnswer realiza evaluación local determinista usando índice explícito (si provisto),
// letra, número o similitud. Retorna (isCorrect, evaluated) donde evaluated=false si quedó pendiente.
func (h *Handler) evaluateLastAnswer(threadID, userAnswer string, explicit *int, data map[string]any) (bool, bool) {
	if threadID == "" {
		return false, false
	}
	userAns := strings.TrimSpace(userAnswer)
	h.mu.Lock()
	ci, okCI := h.lastCorrectIndex[threadID]
	opts := h.lastOptions[threadID]
	h.mu.Unlock()
	if !okCI || ci < 0 || ci >= len(opts) || len(opts) == 0 {
		data["evaluation_pending"] = true
		return false, false
	}
	idx, okIdx := mapUserAnswerToIndex(userAns, explicit, opts)
	isCorrect := okIdx && idx == ci
	fb, _ := data["feedback"].(string)
	data["feedback"] = rebuildFeedbackWithEvaluation(fb, isCorrect)
	h.mu.Lock()
	if _, pending := data["evaluation_pending"].(bool); !pending {
		h.evalAnswers[threadID] = h.evalAnswers[threadID] + 1
		if isCorrect {
			h.evalCorrect[threadID] = h.evalCorrect[threadID] + 1
		}
	}
	h.mu.Unlock()
	data["last_is_correct"] = isCorrect
	return isCorrect, true
}

const interactiveSchemaVersion = "interactive_v2"

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
	if vector == "" {
		vector = "vs_680fc484cef081918b2b9588b701e2f4"
	}
	return &Handler{
		ai:                 cli,
		maxQuestions:       maxQ,
		turnCount:          make(map[string]int),
		threadMaxQuestions: make(map[string]int),
		askedQuestions:     make(map[string][]string),
		evalCorrect:        make(map[string]int),
		evalAnswers:        make(map[string]int),
		vectorID:           vector,
		lastCorrectIndex:   make(map[string]int),
		lastOptions:        make(map[string][]string),
		lastQuestionText:   make(map[string]string),
		missingCorrectIdx:  make(map[string]int),
		closureDue:         make(map[string]bool),
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
	ThreadID    string `json:"thread_id"`
	Mensaje     string `json:"mensaje"`
	AnswerIndex *int   `json:"answer_index,omitempty"`
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
	if h.lastCorrectIndex == nil {
		h.lastCorrectIndex = make(map[string]int)
	}
	if h.lastOptions == nil {
		h.lastOptions = make(map[string][]string)
	}
	if h.lastQuestionText == nil {
		h.lastQuestionText = make(map[string]string)
	}
	if h.missingCorrectIdx == nil {
		h.missingCorrectIdx = make(map[string]int)
	}
	if h.closureDue == nil {
		h.closureDue = make(map[string]bool)
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
	// Timeouts configurables para evitar 504 a través de Nginx/proxy
	startTout := 25 // segundos por defecto
	if s := strings.TrimSpace(os.Getenv("INTERACTIVE_START_TIMEOUT_SEC")); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 5 && v <= 90 {
			startTout = v
		}
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(startTout)*time.Second)
	defer cancel()
	// Soft-timeout para responder rápido si el asistente demora demasiado (evita 504 en proxy)
	startSoft := 8 // segundos por defecto
	if s := strings.TrimSpace(os.Getenv("INTERACTIVE_START_SOFT_TIMEOUT_SEC")); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 3 && v <= 30 {
			startSoft = v
		}
	}
	c.Header("X-Interactive-Start-Soft-Timeout", strconv.Itoa(startSoft))

	// Use user's max interactions preference if provided, otherwise use default
	currentMaxQuestions := h.maxQuestions
	if req.MaxInteractions > 0 && req.MaxInteractions >= 3 && req.MaxInteractions <= 10 {
		currentMaxQuestions = req.MaxInteractions
	}

	// Fast path opcional: no llamar al asistente externo si INTERACTIVE_FAKE=1
	if os.Getenv("INTERACTIVE_FAKE") == "1" {
		// Construir thread id sintético
		threadID := fmt.Sprintf("thread_%d", time.Now().UnixNano())
		// Store the max questions for this specific thread
		if threadID != "" {
			h.mu.Lock()
			h.threadMaxQuestions[threadID] = currentMaxQuestions
			if currentMaxQuestions == 1 {
				h.closureDue[threadID] = true
			}
			h.mu.Unlock()
		}
		// Construir turno inicial determinista
		data := map[string]any{
			"feedback": "Anamnesis inicial de prueba.",
			"next": map[string]any{
				"hallazgos": map[string]any{},
				"pregunta": map[string]any{
					"tipo":          "single-choice",
					"texto":         "¿Cuál es el siguiente mejor paso diagnóstico?",
					"opciones":      []string{"Opción A", "Opción B", "Opción C", "Opción D"},
					"correct_index": 0,
				},
			},
			"finish": 0.0,
		}
		// Inicializar contadores y estado
		h.mu.Lock()
		h.turnCount[threadID] = 1
		h.lastCorrectIndex[threadID] = 0
		h.lastOptions[threadID] = []string{"Opción A", "Opción B", "Opción C", "Opción D"}
		h.lastQuestionText[threadID] = "¿Cuál es el siguiente mejor paso diagnóstico?"
		h.askedQuestions[threadID] = append(h.askedQuestions[threadID], h.lastQuestionText[threadID])
		h.mu.Unlock()
		clinicalHistory := "Historia clínica inicial proporcionada por el sistema."
		if s, ok := data["feedback"].(string); ok {
			clinicalHistory = s
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
			"data": func() map[string]any {
				if _, exists := data["evaluation"]; !exists {
					data["evaluation"] = map[string]any{
						"user_answer":    "",
						"correct_answer": "",
						"correct_index":  0,
						"is_correct":     nil,
						"total_correct":  0,
						"total_answered": 0,
					}
				}
				return data
			}(),
			"thread_id":      threadID,
			"schema_version": interactiveSchemaVersion,
		}
		log.Printf("[InteractiveCase][Start][TEST] thread=%s max=%d turn=%d", threadID, currentMaxQuestions, 1)
		c.JSON(http.StatusOK, resp)
		return
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
		// Si solo hay 1 pregunta permitida, el siguiente mensaje debe cerrar inmediatamente
		if currentMaxQuestions == 1 {
			h.closureDue[threadID] = true
		}
		h.mu.Unlock()
	}

	userPrompt := strings.Join([]string{
		"INICIO DE SESIÓN (rol system): Recibirás el caso clínico completo y NO comentarás. Guarda para usar más adelante.",
		"FASE DE ANAMNESIS: En 'feedback' muestra la historia clínica completa (síntomas, antecedentes, contexto social/familiar, evolución) con 3-4 párrafos detallados.",
		"Incluye motivo de consulta, historia de la enfermedad actual, antecedentes, examen físico inicial. Al final del feedback añade una línea 'Referencias:' seguida de UNA cita abreviada basada en el vector " + h.vectorID + " (no menciones el id) o un PMID.",
		"Formato estricto: JSON con keys: feedback, next{hallazgos{}, pregunta{tipo:'single-choice', texto, opciones[4], correct_index:int}}, finish:0. 'opciones' deben ser CUATRO textos clínicos completos (no solo letras A/B/C/D, sin prefijos 'A -'). 'correct_index' (0-3) indica cuál opción es correcta y NO se debe mencionar en el texto. Sin texto fuera del JSON.",
		"Paciente: edad=" + strings.TrimSpace(req.Age) + ", sexo=" + strings.TrimSpace(req.Sex) + ", gestante=" + boolToStr(req.Pregnant) + ".",
	}, " ")
	instr := strings.Join([]string{
		"Responde SOLO en JSON válido con claves: feedback, next{hallazgos, pregunta{tipo, texto, opciones, correct_index}}, finish.",
		"'feedback' (200-300 palabras) termina con una línea 'Referencias:' y 1 cita (libro guía o PMID) sin mencionar vectores ni IDs internos.",
		"Cada valor de 'opciones' debe ser un enunciado clínico completo; NO uses únicamente letras (A,B,C,D) ni las repitas. No añadas prefijos de letra, solo el texto de la opción.",
		"IMPORTANTE: Las opciones se randomizarán automáticamente, NO pongas siempre la correcta en posición A. Crea 4 opciones balanceadas y plausibles.",
		"No omitas claves ni uses nombres distintos. Sin null ni cadenas vacías. Usa {} si no hay hallazgos. finish=0.",
		"Cada pregunta debe ser única y coherente con el caso. Idioma: español. Sin markdown.",
	}, " ")

	ch, err := h.ai.StreamAssistantJSON(ctx, threadID, userPrompt, instr)
	if err != nil {
		c.JSON(http.StatusOK, h.fallbackStart(req, threadID))
		return
	}

	var content string
	softTimer := time.NewTimer(time.Duration(startSoft) * time.Second)
	defer softTimer.Stop()
	select {
	case content = <-ch:
		// ok
	case <-softTimer.C:
		// Cancelar operación lenta y devolver fallback inmediato
		cancel()
		log.Printf("[InteractiveCase][Start][SoftTimeout] thread=%s soft=%ds", threadID, startSoft)
		c.JSON(http.StatusOK, h.fallbackStart(req, threadID))
		return
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

	// Aplicar randomización de opciones ANTES de almacenar índices
	applyOptionShuffle(data)

	// Initialize turn counter: first question was asked in this turn
	if threadID != "" {
		h.mu.Lock()
		h.turnCount[threadID] = 1
		// store initial question to help reduce repetition later
		if q := extractQuestionText(data); q != "" {
			h.askedQuestions[threadID] = append(h.askedQuestions[threadID], q)
		}
		// capture initial correct_index & options (optional structure)
		if nx, ok := data["next"].(map[string]any); ok {
			if pq, ok := nx["pregunta"].(map[string]any); ok {
				if ci, ok := pq["correct_index"].(float64); ok {
					h.lastCorrectIndex[threadID] = int(ci)
				}
				if txt, ok := pq["texto"].(string); ok {
					h.lastQuestionText[threadID] = strings.TrimSpace(txt)
				}
				// extract options to support answer matching
				if rawOpts, ok := pq["opciones"].([]any); ok {
					var opts []string
					for _, v := range rawOpts {
						if s, ok := v.(string); ok {
							opts = append(opts, strings.TrimSpace(s))
						}
					}
					if len(opts) > 0 {
						h.lastOptions[threadID] = opts
					}
				} else if rawStr, ok := pq["opciones"].([]string); ok {
					var opts []string
					for _, s := range rawStr {
						opts = append(opts, strings.TrimSpace(s))
					}
					if len(opts) > 0 {
						h.lastOptions[threadID] = opts
					}
				}
			}
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
		"data": func() map[string]any {
			// Seed evaluation object (sin respuesta aún)
			if data == nil {
				data = map[string]any{}
			}
			if _, exists := data["evaluation"]; !exists {
				data["evaluation"] = map[string]any{
					"user_answer":    "",
					"correct_answer": "",
					"correct_index":  -1,
					"is_correct":     nil,
					"total_correct":  0,
					"total_answered": 0,
				}
			}
			return data
		}(),
		"thread_id":      threadID,
		"schema_version": interactiveSchemaVersion,
	}
	// Anexar referencias al inicio en el campo anamnesis (o management si prefieres)
	func() {
		defer func() { _ = recover() }()
		if cs, ok := resp["case"].(map[string]any); ok {
			// construir query usando título/diagnóstico si existe; fallback a primeros renglones de la anamnesis
			q := buildInteractiveCaseQuery(cs)
			if strings.TrimSpace(q) == "" {
				q = extractFirstLine(clinicalHistory)
			}
			if strings.TrimSpace(q) != "" {
				refs := h.collectInteractiveEvidence(ctx, q)
				if strings.TrimSpace(refs) != "" {
					an := strings.TrimSpace(fmt.Sprint(cs["anamnesis"]))
					cs["anamnesis"] = appendRefs(an, refs)
				}
			}
		}
	}()
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
	if h.lastCorrectIndex == nil {
		h.lastCorrectIndex = make(map[string]int)
	}
	if h.lastOptions == nil {
		h.lastOptions = make(map[string][]string)
	}
	if h.lastQuestionText == nil {
		h.lastQuestionText = make(map[string]string)
	}
	if h.missingCorrectIdx == nil {
		h.missingCorrectIdx = make(map[string]int)
	}
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "interactive_strict_message"); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "interactive clinical cases quota exceeded"})
			return
		}
	}
	if v, ok := c.Get("quota_remaining"); ok {
		c.Header("X-Quota-Remaining", toString(v))
	}
	var req messageReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Mensaje) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	// Timeout configurable para mensajes para evitar 504
	msgTout := 25 // segundos por defecto
	if s := strings.TrimSpace(os.Getenv("INTERACTIVE_MESSAGE_TIMEOUT_SEC")); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 5 && v <= 90 {
			msgTout = v
		}
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(msgTout)*time.Second)
	defer cancel()
	// Soft-timeout para respuestas rápidas (mitiga 504)
	msgSoft := 8 // segundos por defecto
	if s := strings.TrimSpace(os.Getenv("INTERACTIVE_MESSAGE_SOFT_TIMEOUT_SEC")); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 3 && v <= 30 {
			msgSoft = v
		}
	}
	c.Header("X-Interactive-Message-Soft-Timeout", strconv.Itoa(msgSoft))
	threadID := strings.TrimSpace(req.ThreadID)
	if threadID == "" {
		if id, err := h.ai.CreateThread(ctx); err == nil {
			threadID = id
		}
	}

	// Fast path opcional: no llamar al asistente externo si INTERACTIVE_FAKE=1
	if os.Getenv("INTERACTIVE_FAKE") == "1" {
		curr := h.getCount(threadID)
		maxQuestions := h.getMaxQuestions(threadID)
		closing := h.closureDue[threadID]
		// Si aún no hay configuración de max para este thread, fijar por defecto
		if threadID != "" {
			h.mu.Lock()
			if _, ok := h.threadMaxQuestions[threadID]; !ok {
				h.threadMaxQuestions[threadID] = h.maxQuestions
			}
			h.mu.Unlock()
		}
		// Cierre si ya marcado o se alcanzó el máximo
		if closing || curr >= maxQuestions {
			data := h.minTurn()
			forceFinishInteractive(data, threadID, h)
			// Asegurar pregunta final vacía
			if nx, ok := data["next"].(map[string]any); ok {
				nx["pregunta"] = finalQuestion()
			}
			log.Printf("[InteractiveCase][Message][TEST Return] thread=%s count=%d max=%d finish=1 closing=true", threadID, curr, maxQuestions)
			c.JSON(http.StatusOK, map[string]any{"data": withThread(data, threadID)})
			return
		}
		// Siguiente pregunta determinista
		qNum := curr + 1
		data := map[string]any{
			"feedback": "Evaluación: CORRECTO\nExplicación breve de prueba.",
			"next": map[string]any{
				"hallazgos": map[string]any{},
				"pregunta": map[string]any{
					"tipo":          "single-choice",
					"texto":         fmt.Sprintf("Pregunta %d de prueba", qNum+1),
					"opciones":      []string{"Opción A", "Opción B", "Opción C", "Opción D"},
					"correct_index": 0,
				},
			},
			"finish": 0.0,
		}
		// Registrar y avanzar contador
		if threadID != "" {
			h.mu.Lock()
			h.lastCorrectIndex[threadID] = 0
			h.lastOptions[threadID] = []string{"Opción A", "Opción B", "Opción C", "Opción D"}
			h.lastQuestionText[threadID] = fmt.Sprintf("Pregunta %d de prueba", qNum+1)
			h.askedQuestions[threadID] = append(h.askedQuestions[threadID], h.lastQuestionText[threadID])
			h.mu.Unlock()
		}
		// incrementa y marca cierre si corresponde
		h.incrementCount(threadID)
		cnt := h.getCount(threadID)
		if cnt >= maxQuestions {
			h.mu.Lock()
			h.closureDue[threadID] = true
			h.mu.Unlock()
		}
		// Construir evaluation object similar al camino normal
		h.mu.Lock()
		corr := h.evalCorrect[threadID]
		ans := h.evalAnswers[threadID]
		h.mu.Unlock()
		data["evaluation"] = map[string]any{
			"user_answer":    strings.TrimSpace(req.Mensaje),
			"correct_answer": "Opción A",
			"correct_index":  0,
			"is_correct":     nil,
			"total_correct":  corr,
			"total_answered": ans,
		}
		data["schema_version"] = interactiveSchemaVersion
		log.Printf("[InteractiveCase][Message][TEST Return] thread=%s count=%d max=%d finish=0 closing=false", threadID, cnt, maxQuestions)
		c.JSON(http.StatusOK, map[string]any{"data": withThread(data, threadID)})
		return
	}

	// Enforce max questions policy using per-thread counters
	curr := h.getCount(threadID) // questions already asked
	maxQuestions := h.getMaxQuestions(threadID)
	// cierre diferido: si closureDue ya estaba marcado este turno es de cierre
	closing := h.closureDue[threadID]
	log.Printf("[InteractiveCase][Message][Begin] thread=%s curr=%d max=%d closing=%v", threadID, curr, maxQuestions, closing)

	userPrompt := req.Mensaje
	var instr string
	if closing {
		instr = strings.Join([]string{
			"Responde SOLO en JSON válido con: feedback, next{hallazgos{}, pregunta{}}, finish(1).",
			"Formato 'feedback': primera línea 'Resumen Final:'; luego síntesis (≤80 palabras) con diagnóstico probable, diferenciales clave, manejo inicial.",
			"NO incluyas línea 'Puntaje:' (el sistema la añadirá).",
			"Finalmente línea 'Referencias:' con 1-2 citas (primera de libro/guía, opcional PMID). NO menciones vectores ni IDs. Sin nueva pregunta. finish=1.",
			"Idioma: español. Sin texto fuera del JSON.",
		}, " ")
	} else {
		// Build a short memory of prior questions to discourage repetition
		var prevQs []string
		if threadID != "" {
			h.mu.Lock()
			prevQs = append(prevQs, h.askedQuestions[threadID]...)
			h.mu.Unlock()
		}
		if len(prevQs) > 5 {
			prevQs = prevQs[len(prevQs)-5:]
		}
		prevList := ""
		if len(prevQs) > 0 {
			for i, q := range prevQs {
				prevQs[i] = strings.TrimSpace(q)
			}
			prevList = "PREGUNTAS YA HECHAS (JAMÁS repetir estos temas ni variantes): " + strings.Join(prevQs, " | ") + ". PROGRESA a temas NUEVOS completamente diferentes."
		}

		// Obtener información diagnóstica progresiva
		currentTurn := h.getCount(threadID) + 1
		diagnosticInfo := generateProgressiveDiagnostics(currentTurn, threadID)

		instr = strings.Join([]string{
			"Responde SOLO en JSON válido con: feedback, next{hallazgos{}, pregunta{tipo:'single-choice'|'open_ended', texto, opciones, correct_index}}, finish(0).",
			"Formato 'feedback': primera línea 'Evaluación: CORRECTO' o 'Evaluación: INCORRECTO' (en mayúsculas).",
			"Luego una explicación académica y más profunda (120-220 palabras) que explique el razonamiento clínico: por qué la opción es correcta/incorrecta, qué hallazgos la sustentan, y referencias concisas al final." + diagnosticInfo,
			"La última línea debe empezar con 'Fuente:' y contener 1-2 citas (libro/guía o PMID).",
			"Cada elemento de 'opciones' debe ser un texto descriptivo clínico; NO uses solo 'A','B','C','D'. El sistema asignará letras externamente.",
			"IMPORTANTE: Las opciones de respuesta se randomizarán automáticamente, NO pongas siempre la correcta en posición A. Crea 4 opciones balanceadas.",
			"PROHIBIDO repetir historia clínica inicial. OBLIGATORIO progresar hacia nuevos aspectos diagnósticos o terapéuticos.",
			prevList,
			"VARIEDAD TEMÁTICA OBLIGATORIA: cada pregunta debe abordar aspecto COMPLETAMENTE diferente (diagnóstico→manejo→pronóstico→complicaciones→seguimiento).",
			"No cierres todavía: siempre finish=0 hasta que el sistema solicite el resumen final.",
			"Evita repetir preguntas ya hechas. Sin texto fuera del JSON. Idioma: español. No menciones vectores ni IDs internos.",
		}, " ")
	}

	ch, err := h.ai.StreamAssistantJSON(ctx, threadID, userPrompt, instr)
	if err != nil {
		d := withThread(h.minTurn(), threadID)
		d["schema_version"] = interactiveSchemaVersion
		c.JSON(http.StatusOK, map[string]any{"data": d})
		return
	}
	var content string
	softTimer := time.NewTimer(time.Duration(msgSoft) * time.Second)
	defer softTimer.Stop()
	select {
	case content = <-ch:
		// ok
	case <-softTimer.C:
		cancel()
		log.Printf("[InteractiveCase][Message][SoftTimeout] thread=%s soft=%ds", threadID, msgSoft)
		d := withThread(h.minTurn(), threadID)
		d["schema_version"] = interactiveSchemaVersion
		c.JSON(http.StatusOK, map[string]any{"data": d})
		return
	case <-ctx.Done():
		d := withThread(h.minTurn(), threadID)
		d["schema_version"] = interactiveSchemaVersion
		c.JSON(http.StatusOK, map[string]any{"data": d})
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
		// Aplicar randomización de opciones ANTES de almacenar índices
		applyOptionShuffle(data)

		// Evaluación local determinista usando correct_index almacenado (recuperando si falta)
		if threadID != "" {
			h.mu.Lock()
			ci, okCI := h.lastCorrectIndex[threadID]
			opts := h.lastOptions[threadID]
			qText := h.lastQuestionText[threadID]
			h.mu.Unlock()
			// attempt recovery if missing
			if (!okCI || ci < 0 || ci >= len(opts)) && len(opts) > 0 {
				// metric: detection of missing correct_index
				h.mu.Lock()
				h.missingCorrectIdx[threadID] = h.missingCorrectIdx[threadID] + 1
				h.mu.Unlock()
				// 1) Intentar con evidencia (libros + PubMed) para asegurar base científica
				if recIdx, ok := h.recoverCorrectIndexEvidence(ctx, qText, opts); ok {
					h.mu.Lock()
					h.lastCorrectIndex[threadID] = recIdx
					ci = recIdx
					okCI = true
					h.mu.Unlock()
				} else if recIdx, ok := h.recoverCorrectIndex(ctx, threadID, qText, opts); ok { // 2) fallback al asistente
					h.mu.Lock()
					h.lastCorrectIndex[threadID] = recIdx
					ci = recIdx
					okCI = true
					h.mu.Unlock()
				}
			}
			if okCI && len(opts) > 0 && ci >= 0 && ci < len(opts) {
				_, _ = h.evaluateLastAnswer(threadID, req.Mensaje, req.AnswerIndex, data)
			} else {
				data["evaluation_pending"] = true
			}
		}
		rebuildIfEmbedded := func(preg map[string]any) {
			// Si opciones inválidas (vacías, solo letras sueltas) intentar extraer de feedback o pregunta.texto
			rawOpts, _ := preg["opciones"].([]any)
			var existing []string
			for _, v := range rawOpts {
				if s, ok := v.(string); ok {
					existing = append(existing, strings.TrimSpace(s))
				}
			}
			needsParse := len(existing) < 2 || allShortLetters(existing)
			if !needsParse {
				return
			}
			texto, _ := preg["texto"].(string)
			combined := texto + "\n" + toStringSafe(data["feedback"]) // buscar en ambos
			extracted := extractEmbeddedOptions(combined)
			if len(extracted) >= 2 {
				arr := make([]any, 0, len(extracted))
				for _, o := range extracted {
					arr = append(arr, o)
				}
				preg["opciones"] = arr
			}
		}
		preguntaValid := func() bool {
			next, ok := data["next"].(map[string]any)
			if !ok {
				return false
			}
			preg, ok := next["pregunta"].(map[string]any)
			if !ok {
				return false
			}
			rebuildIfEmbedded(preg)
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
			// Anular siempre cualquier intento de cierre anticipado antes del turno de resumen
			data["finish"] = 0.0
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
		// (parseEvalCorrect eliminado; evaluación es 100% local determinista)
		// record asked question & capture next correct_index/options for subsequent answer evaluation
		if q := extractQuestionText(data); q != "" && threadID != "" {
			h.mu.Lock()
			h.askedQuestions[threadID] = append(h.askedQuestions[threadID], q)
			// IMPORTANTE: capturar correct_index y opciones DESPUÉS del shuffle
			if nx, ok := data["next"].(map[string]any); ok {
				if pq, ok := nx["pregunta"].(map[string]any); ok {
					if ci, ok := pq["correct_index"].(float64); ok {
						h.lastCorrectIndex[threadID] = int(ci)
					}
					if txt, ok := pq["texto"].(string); ok {
						h.lastQuestionText[threadID] = strings.TrimSpace(txt)
					}
					if rawOpts, ok := pq["opciones"].([]any); ok {
						var opts []string
						for _, v := range rawOpts {
							if s, ok := v.(string); ok {
								opts = append(opts, strings.TrimSpace(s))
							}
						}
						if len(opts) > 0 {
							h.lastOptions[threadID] = opts
						}
					} else if rawStr, ok := pq["opciones"].([]string); ok {
						var opts []string
						for _, s := range rawStr {
							opts = append(opts, strings.TrimSpace(s))
						}
						if len(opts) > 0 {
							h.lastOptions[threadID] = opts
						}
					}
				}
			}
			h.mu.Unlock()
		}
		// Incrementar sólo si NO estamos en un turno marcado previamente para cierre diferido
		if !h.closureDue[threadID] {
			h.incrementCount(threadID)
		}
		cnt := h.getCount(threadID)
		// Lógica de progresión: nunca establecer finish=1 aquí; sólo marcar closureDue cuando cnt == maxQuestions
		switch {
		case cnt < maxQuestions:
			data["finish"] = 0.0
		case cnt == maxQuestions:
			data["finish"] = 0.0
			h.mu.Lock()
			h.closureDue[threadID] = true
			h.mu.Unlock()
		default:
			data["finish"] = 0.0
		}
	} else {
		// Evaluar también la última respuesta antes de cerrar si procede
		_, _ = h.evaluateLastAnswer(threadID, req.Mensaje, req.AnswerIndex, data)
		forceFinishInteractive(data, threadID, h)
		// Añadir referencias en el cierre (feedback final)
		func() {
			defer func() { _ = recover() }()
			fb := toStringSafe(data["feedback"])
			q := fb
			if strings.TrimSpace(q) == "" {
				q = extractQuestionText(data)
			}
			if strings.TrimSpace(q) == "" {
				q = req.Mensaje
			}
			if strings.TrimSpace(q) == "" {
				return
			}
			refs := h.collectInteractiveEvidence(ctx, q)
			if strings.TrimSpace(refs) == "" {
				return
			}
			data["feedback"] = appendRefs(fb, refs)
		}()
		// limpiar flag
		h.mu.Lock()
		delete(h.closureDue, threadID)
		h.mu.Unlock()
	}
	// Attach structured evaluation object (latest answer + cumulative)
	if threadID != "" {
		h.mu.Lock()
		corr := h.evalCorrect[threadID]
		ans := h.evalAnswers[threadID]
		ci, okCI := h.lastCorrectIndex[threadID]
		opts := h.lastOptions[threadID]
		h.mu.Unlock()
		userAns := strings.TrimSpace(req.Mensaje)
		correctAns := ""
		if !okCI {
			ci = -1
		} else if ci >= 0 && ci < len(opts) {
			correctAns = opts[ci]
		}
		// Determinista: tomar resultado de evaluateLastAnswer almacenado en data
		isCorrectPtr := interface{}(nil)
		if v, ok := data["last_is_correct"].(bool); ok {
			isCorrectPtr = v
		}
		evalObj := map[string]any{
			"user_answer":    userAns,
			"correct_answer": correctAns,
			"correct_index":  ci,
			"is_correct":     isCorrectPtr,
			"total_correct":  corr,
			"total_answered": ans,
		}
		if v, ok := data["evaluation_pending"].(bool); ok && v {
			evalObj["pending"] = true
			evalObj["is_correct"] = nil
		}
		data["evaluation"] = evalObj
	}
	// Attach schema version
	data["schema_version"] = interactiveSchemaVersion
	// Restaurar pregunta mínima si quedó vacía inesperadamente (defensa extra)
	if !closing {
		if nx, ok := data["next"].(map[string]any); ok {
			if pq, ok := nx["pregunta"].(map[string]any); ok {
				texto, _ := pq["texto"].(string)
				if strings.TrimSpace(texto) == "" {
					mt := h.minTurn()
					if mnext, ok := mt["next"].(map[string]any); ok {
						if mpq, ok := mnext["pregunta"].(map[string]any); ok {
							pq["tipo"] = mpq["tipo"]
							pq["texto"] = mpq["texto"]
							pq["opciones"] = mpq["opciones"]
						}
					}
				}
			}
		}
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
		"data":           h.minTurn(),
		"thread_id":      threadID,
		"schema_version": interactiveSchemaVersion,
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
	if data == nil {
		return ""
	}
	next, ok := data["next"].(map[string]any)
	if !ok {
		return ""
	}
	preg, ok := next["pregunta"].(map[string]any)
	if !ok {
		return ""
	}
	txt, _ := preg["texto"].(string)
	return strings.TrimSpace(txt)
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
	// Build clearer structured summary with heuristic strengths/improvements
	pct := 0.0
	if total > 0 {
		pct = (float64(corr) / float64(total)) * 100.0
	}
	tier := performanceTier(pct)
	fbOriginal, _ := data["feedback"].(string)
	// Eliminar cualquier línea de Evaluación previa para no duplicar semántica en el resumen final
	if fbOriginal != "" {
		lines := strings.Split(fbOriginal, "\n")
		var cleaned []string
		for _, ln := range lines {
			ul := strings.ToUpper(strings.TrimSpace(ln))
			if strings.HasPrefix(ul, "EVALUACIÓN:") || strings.HasPrefix(ul, "EVALUACION:") {
				continue
			}
			cleaned = append(cleaned, ln)
		}
		fbOriginal = strings.Join(cleaned, "\n")
	}
	coreSummary := extractCoreSummary(fbOriginal)
	if coreSummary == "" {
		coreSummary = "No disponible"
	}
	strengths, improvements := deriveStrengthsImprovements(corr, total, pct)
	finalLines := []string{"Resumen Final:"}
	finalLines = append(finalLines, fmt.Sprintf("Puntaje: %d/%d (%.1f%%) - %s", corr, total, pct, tier))
	if total > 0 {
		finalLines = append(finalLines,
			"Desempeño:",
			fmt.Sprintf("- Preguntas respondidas: %d", total),
			fmt.Sprintf("- Respuestas correctas: %d", corr),
		)
	} else {
		finalLines = append(finalLines, "Desempeño: sin preguntas evaluadas")
	}
	finalLines = append(finalLines, "Síntesis: "+coreSummary)
	finalLines = append(finalLines, "Fortalezas: "+strengths)
	finalLines = append(finalLines, "Áreas de mejora: "+improvements)
	if !strings.Contains(strings.ToLower(fbOriginal), "referencias:") {
		finalLines = append(finalLines, "Referencias: Fuente clínica estándar")
	}
	data["feedback"] = strings.Join(finalLines, "\n")
	data["status"] = "finished"
	// include metric of missing correct_index events if present
	h.mu.Lock()
	miss := h.missingCorrectIdx[threadID]
	h.mu.Unlock()
	data["final_evaluation"] = map[string]any{
		"score_correct":                corr,
		"score_total":                  total,
		"score_percent":                pct,
		"tier":                         tier,
		"strengths":                    strengths,
		"improvements":                 improvements,
		"summary":                      coreSummary,
		"missing_correct_index_events": miss,
	}
}

// performanceTier returns a qualitative tier label for a percent score
func performanceTier(pct float64) string {
	switch {
	case pct >= 85:
		return "Excelente"
	case pct >= 70:
		return "Bueno"
	case pct >= 50:
		return "Aceptable"
	case pct > 0:
		return "Necesita refuerzo"
	default:
		return "Sin aciertos"
	}
}

// deriveStrengthsImprovements returns single-line strengths & improvements heuristics.
func deriveStrengthsImprovements(corr, total int, pct float64) (string, string) {
	if total == 0 {
		return "Inicio del caso completado.", "Responder preguntas para generar retroalimentación específica."
	}
	switch {
	case pct >= 85:
		return "Excelente identificación de hallazgos y razonamiento clínico integrador.", "Profundizar en diagnósticos diferenciales secundarios y seguimiento a largo plazo."
	case pct >= 70:
		return "Buen razonamiento y selección de conductas apropiadas.", "Refinar priorización de pruebas complementarias específicas."
	case pct >= 50:
		return "Reconoces parte de los hallazgos clave y estructuras un plan básico.", "Consolidar criterios diagnósticos y justificar secuencia de manejo."
	case pct > 0:
		return "Participación activa y formulación inicial de hipótesis.", "Reforzar correlación clínico-patológica y selección de pruebas iniciales."
	default:
		return "Participación inicial registrada.", "Repasar fundamentos diagnósticos básicos antes de avanzar."
	}
}

// mapUserAnswerToIndex intenta resolver el índice elegido por el usuario.
// Prioridad: answer_index explícito > letra única > número > similitud con opción.
func mapUserAnswerToIndex(userRaw string, explicit *int, options []string) (int, bool) {
	if explicit != nil && options != nil && *explicit >= 0 && *explicit < len(options) {
		return *explicit, true
	}
	u := strings.TrimSpace(userRaw)
	if u == "" {
		return -1, false
	}
	if len(u) == 1 {
		c := u[0]
		switch {
		case c >= 'A' && c <= 'Z':
			idx := int(c - 'A')
			if idx < len(options) {
				return idx, true
			}
		case c >= 'a' && c <= 'z':
			idx := int(c - 'a')
			if idx < len(options) {
				return idx, true
			}
		case c >= '0' && c <= '9':
			idx := int(c - '0')
			if idx < len(options) {
				return idx, true
			}
		}
	}
	// similitud textual: escoger la opción con mayor jaccard sobre tokens normalizados
	bestIdx := -1
	bestScore := 0.0
	nu := tokenize(normalizeAnswer(u))
	for i, opt := range options {
		score := jaccard(nu, tokenize(normalizeAnswer(opt)))
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	if bestIdx >= 0 && bestScore >= 0.4 {
		return bestIdx, true
	}
	return -1, false
}

// extractCoreSummary tries to salvage a concise summary from original feedback (excluding Evaluación/Puntaje lines)
func extractCoreSummary(fb string) string {
	if fb == "" {
		return ""
	}
	lines := strings.Split(fb, "\n")
	var kept []string
	for _, ln := range lines {
		ltrim := strings.TrimSpace(ln)
		u := strings.ToUpper(ltrim)
		if ltrim == "" {
			continue
		}
		if strings.HasPrefix(u, "EVALUACIÓN:") || strings.HasPrefix(u, "EVALUACION:") || strings.HasPrefix(u, "PUNTAJE:") {
			continue
		}
		kept = append(kept, ltrim)
		if len(kept) >= 2 {
			break
		} // keep it brief
	}
	return strings.Join(kept, " ")
}

// rebuildFeedbackWithEvaluation injects a first line 'Evaluación: CORRECTO/INCORRECTO'
// overriding any existing evaluation marker.
func rebuildFeedbackWithEvaluation(original string, isCorrect bool) string {
	lines := strings.Split(strings.TrimSpace(original), "\n")
	var cleaned []string
	for _, ln := range lines {
		ul := strings.ToUpper(strings.TrimSpace(ln))
		if strings.HasPrefix(ul, "EVALUACIÓN:") || strings.HasPrefix(ul, "EVALUACION:") {
			continue
		}
		cleaned = append(cleaned, ln)
	}
	eval := "Evaluación: INCORRECTO"
	if isCorrect {
		eval = "Evaluación: CORRECTO"
	}
	return eval + "\n" + strings.Join(cleaned, "\n")
}

// --- Answer normalization & matching --- //

// normalizeAnswer converts to lowercase, strips diacritics & punctuation, collapses spaces.
func normalizeAnswer(s string) string {
	runes := []rune(strings.TrimSpace(s))
	var b strings.Builder
	lastSpace := false
	for _, r := range runes {
		// strip accents manually for common vowels
		switch r {
		case 'á', 'à', 'ä', 'â', 'Á', 'À', 'Ä', 'Â':
			r = 'a'
		case 'é', 'è', 'ë', 'ê', 'É', 'È', 'Ë', 'Ê':
			r = 'e'
		case 'í', 'ì', 'ï', 'î', 'Í', 'Ì', 'Ï', 'Î':
			r = 'i'
		case 'ó', 'ò', 'ö', 'ô', 'Ó', 'Ò', 'Ö', 'Ô':
			r = 'o'
		case 'ú', 'ù', 'ü', 'û', 'Ú', 'Ù', 'Ü', 'Û':
			r = 'u'
		case 'ñ', 'Ñ':
			r = 'n'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			lastSpace = false
			continue
		}
		if unicode.IsSpace(r) {
			if !lastSpace {
				b.WriteRune(' ')
				lastSpace = true
			}
		}
		// ignore other chars
	}
	return strings.TrimSpace(b.String())
}

func tokenize(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Fields(s)
	return parts
}

// jaccard similarity of token sets
func jaccard(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	setA := map[string]struct{}{}
	setB := map[string]struct{}{}
	for _, w := range a {
		setA[w] = struct{}{}
	}
	for _, w := range b {
		setB[w] = struct{}{}
	}
	var inter int
	for w := range setA {
		if _, ok := setB[w]; ok {
			inter++
		}
	}
	un := len(setA) + len(setB) - inter
	if un == 0 {
		return 0
	}
	return float64(inter) / float64(un)
}

// stripLeadingLetterPrefix remove patterns like "A - " or "A) " at start.
func stripLeadingLetterPrefix(s string) string {
	trim := strings.TrimSpace(s)
	if len(trim) >= 3 {
		c := trim[0]
		if (c >= 'A' && c <= 'D') || (c >= 'a' && c <= 'd') {
			rest := strings.TrimSpace(trim[1:])
			if strings.HasPrefix(rest, "-") || strings.HasPrefix(rest, ")") {
				return strings.TrimSpace(rest[1:])
			}
		}
	}
	return trim
}

// answerIsCorrect applies flexible matching heuristics.
func answerIsCorrect(userRaw, correctRaw string) bool {
	userTrim := strings.TrimSpace(userRaw)
	// Intento 1: mapping directo letra -> índice (A-D) si usuario sólo da letra
	if len(userTrim) == 1 {
		c := userTrim[0]
		if c >= 'A' && c <= 'D' { // se evaluará externamente comparando índices, handled fuera si se pasa opciones
			// devolver false aquí y dejar heurística textual podría fallar; se maneja en capa superior con índice
		}
	}
	userClean := normalizeAnswer(stripLeadingLetterPrefix(userRaw))
	correctClean := normalizeAnswer(stripLeadingLetterPrefix(correctRaw))
	if userClean == "" || correctClean == "" {
		return false
	}
	if userClean == correctClean {
		return true
	}
	ut := tokenize(userClean)
	ct := tokenize(correctClean)
	if len(ut) >= 2 {
		jac := jaccard(ut, ct)
		if jac >= 0.8 {
			return true
		}
		// subset allowance
		var subset = true
		for _, w := range ut {
			if !containsToken(ct, w) {
				subset = false
				break
			}
		}
		if subset && float64(len(ut))/float64(len(ct)) >= 0.5 {
			return true
		}
	}
	// containment for long phrases
	if len(userClean) >= 15 && strings.Contains(correctClean, userClean) {
		return true
	}
	return false
}

func containsToken(tokens []string, w string) bool {
	for _, t := range tokens {
		if t == w {
			return true
		}
	}
	return false
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

// recoverCorrectIndex intenta recuperar el correct_index perdido consultando al asistente.
// Retorna (index, true) si pudo recuperarlo.
func (h *Handler) recoverCorrectIndex(ctx context.Context, threadID, question string, options []string) (int, bool) {
	if threadID == "" || len(options) == 0 {
		return -1, false
	}
	// construir prompt compacto enumerando opciones 0..n
	var sb strings.Builder
	sb.WriteString("Pregunta previa: \"")
	sb.WriteString(question)
	sb.WriteString("\". Opciones:")
	for i, opt := range options {
		sb.WriteString(" ")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString(") ")
		sb.WriteString(opt)
		sb.WriteString(";")
	}
	sb.WriteString(" Responde SOLO JSON {\"correct_index\":X} indicando el índice (0-")
	sb.WriteString(strconv.Itoa(len(options) - 1))
	sb.WriteString(") de la opción más correcta. No añadas explicación.")
	// usamos mismas instrucciones genéricas
	ch, err := h.ai.StreamAssistantJSON(ctx, threadID, sb.String(), "Devuelve solo JSON con correct_index")
	if err != nil {
		return -1, false
	}
	select {
	case content := <-ch:
		js := extractJSON(content)
		var tmp map[string]any
		if json.Unmarshal([]byte(js), &tmp) == nil {
			if v, ok := tmp["correct_index"].(float64); ok {
				idx := int(v)
				if idx >= 0 && idx < len(options) {
					return idx, true
				}
			}
		}
	case <-ctx.Done():
		return -1, false
	}
	return -1, false
}

// recoverCorrectIndexEvidence intenta deducir el índice correcto usando evidencia externa
// (vector de libros y PubMed) cuando el asistente omitió correct_index.
// Devuelve (index, true) si hay un candidato único con soporte; en empates o sin señal, false.
func (h *Handler) recoverCorrectIndexEvidence(ctx context.Context, question string, options []string) (int, bool) {
	if len(options) == 0 {
		return -1, false
	}
	// Limitar presupuesto de tiempo para no bloquear el turno completo
	tctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	type score struct {
		s       int    // puntaje discreto
		details string // opcional, no usado de momento
	}
	best := -1
	bestScore := -1
	tie := false
	// Explorar cada opción con consultas breves
	for i, opt := range options {
		// Construir consulta compacta combinando pregunta y opción
		q := strings.TrimSpace(question)
		o := strings.TrimSpace(opt)
		if len(q) > 180 {
			q = q[:180]
		}
		if len(o) > 160 {
			o = o[:160]
		}
		query := q
		if query != "" && o != "" {
			query = q + " — Candidato: " + o
		} else if o != "" {
			query = o
		}
		sc := 0
		// 1) Vector de libros
		if strings.HasPrefix(strings.TrimSpace(h.vectorID), "vs_") {
			if res, err := h.ai.SearchInVectorStoreWithMetadata(tctx, h.vectorID, query); err == nil && res != nil {
				if res.HasResult {
					sc += 2 // resultado con metadatos vale más
				} else if txt, err2 := h.ai.SearchInVectorStore(tctx, h.vectorID, query); err2 == nil && strings.TrimSpace(txt) != "" {
					sc += 1
				}
			}
		}
		// 2) PubMed
		if pm, err := h.ai.SearchPubMed(tctx, query); err == nil && strings.TrimSpace(pm) != "" {
			sc += 1
		}
		// Selección con manejo de empates
		if sc > bestScore {
			bestScore = sc
			best = i
			tie = false
		} else if sc == bestScore {
			tie = true
		}
	}
	if bestScore <= 0 || tie || best < 0 {
		return -1, false
	}
	return best, true
}

// --- Embedded options parsing helpers --- //

var embeddedOptRe = regexp.MustCompile(`(?mi)^[ \t]*[\-*•]?\s*([A-Da-d0-9])\s*[-\)\.]*\s+([^\n]{3,})$`)

func allShortLetters(opts []string) bool {
	if len(opts) == 0 {
		return true
	}
	for _, o := range opts {
		oc := strings.TrimSpace(o)
		if len(oc) != 1 {
			return false
		}
		if !((oc[0] >= 'A' && oc[0] <= 'D') || (oc[0] >= 'a' && oc[0] <= 'd')) {
			return false
		}
	}
	return true
}

func extractEmbeddedOptions(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	var res []string
	seen := map[string]struct{}{}
	for _, ln := range lines {
		m := embeddedOptRe.FindStringSubmatch(ln)
		if len(m) == 3 {
			body := strings.TrimSpace(m[2])
			if body == "" {
				continue
			}
			if _, ok := seen[body]; ok {
				continue
			}
			seen[body] = struct{}{}
			res = append(res, body)
		}
	}
	if len(res) >= 2 && len(res) <= 6 {
		return res
	}
	return nil
}

func toStringSafe(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}

// --- Evidence helpers (RAG + PubMed) for interactive cases --- //

// appendRefs añade un bloque de referencias al final del texto, si hay referencias.
func appendRefs(s, refs string) string {
	if strings.TrimSpace(refs) == "" {
		return s
	}
	if strings.TrimSpace(s) == "" {
		return strings.TrimSpace(refs)
	}
	return strings.TrimRight(s, "\n ") + refs
}

// extractFirstLine obtiene la primera línea no vacía de un bloque de texto
func extractFirstLine(s string) string {
	for _, ln := range strings.Split(strings.TrimSpace(s), "\n") {
		t := strings.TrimSpace(ln)
		if t != "" {
			return t
		}
	}
	return ""
}

// buildInteractiveCaseQuery arma una consulta breve a partir del caso para buscar evidencia
func buildInteractiveCaseQuery(caseMap map[string]any) string {
	title := strings.TrimSpace(toStringSafe(caseMap["title"]))
	diag := strings.TrimSpace(toStringSafe(caseMap["final_diagnosis"]))
	if title != "" && diag != "" {
		q := title + " — " + diag
		if len(q) > 220 {
			q = q[:220]
		}
		return q
	}
	an := strings.TrimSpace(toStringSafe(caseMap["anamnesis"]))
	if an != "" {
		q := extractFirstLine(an)
		if len(q) > 240 {
			q = q[:240]
		}
		return q
	}
	return title
}

// collectInteractiveEvidence consulta primero el vector de libros fijo (handler.vectorID) y luego PubMed
func (h *Handler) collectInteractiveEvidence(ctx context.Context, query string) string {
	// En modo pruebas, evitar llamadas externas que alargan la ejecución
	if os.Getenv("TESTING") == "1" {
		return ""
	}
	refs := make([]string, 0, 3)
	// 1) Libros (vector fijo configurado en handler)
	if strings.HasPrefix(strings.TrimSpace(h.vectorID), "vs_") {
		if res, err := h.ai.SearchInVectorStoreWithMetadata(ctx, h.vectorID, query); err == nil && res != nil && res.HasResult {
			src := strings.TrimSpace(res.Source)
			sec := strings.TrimSpace(res.Section)
			snip := strings.TrimSpace(res.Content)
			if src == "" {
				src = "Base de conocimiento médico"
			}
			if len(snip) > 420 {
				snip = snip[:420] + "…"
			}
			line := src
			if sec != "" {
				line += " — " + sec
			}
			if snip != "" {
				line += ": \"" + snip + "\""
			}
			refs = append(refs, line)
		} else if txt, err2 := h.ai.SearchInVectorStore(ctx, h.vectorID, query); err2 == nil && strings.TrimSpace(txt) != "" {
			t := strings.TrimSpace(txt)
			if len(t) > 420 {
				t = t[:420] + "…"
			}
			refs = append(refs, "Base de conocimiento médico: \""+t+"\"")
		}
	}
	// 2) PubMed
	if pm, err := h.ai.SearchPubMed(ctx, query); err == nil && strings.TrimSpace(pm) != "" {
		p := strings.TrimSpace(pm)
		if len(p) > 600 {
			p = p[:600] + "…"
		}
		refs = append(refs, "PubMed: "+p)
	}
	if len(refs) == 0 {
		return ""
	}
	b := &strings.Builder{}
	b.WriteString("\n\nReferencias:\n")
	for i, r := range refs {
		if i >= 3 {
			break
		}
		b.WriteString("- ")
		b.WriteString(r)
		b.WriteString("\n")
	}
	return b.String()
}

// shuffleOptionsWithCorrectIndex randomiza las opciones y actualiza el índice correcto
func shuffleOptionsWithCorrectIndex(options []string, correctIndex int) ([]string, int) {
	if len(options) <= 1 || correctIndex < 0 || correctIndex >= len(options) {
		return options, correctIndex
	}

	// Crear copia para no modificar el original
	shuffled := make([]string, len(options))
	copy(shuffled, options)

	// Recordar cuál es la opción correcta
	correctOption := options[correctIndex]

	// Usar seed basado en tiempo para randomización
	rand.Seed(time.Now().UnixNano())

	// Algoritmo Fisher-Yates para shuffle
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	// Encontrar nueva posición de la opción correcta
	newCorrectIndex := -1
	for i, opt := range shuffled {
		if opt == correctOption {
			newCorrectIndex = i
			break
		}
	}

	return shuffled, newCorrectIndex
}

// applyOptionShuffle aplica randomización a la estructura de pregunta
func applyOptionShuffle(data map[string]any) {
	// Deshabilitar randomización en tests
	if os.Getenv("TESTING") == "1" {
		return
	}

	nx, ok := data["next"].(map[string]any)
	if !ok {
		return
	}

	pregunta, ok := nx["pregunta"].(map[string]any)
	if !ok {
		return
	}

	// Extraer opciones y correct_index
	rawOpts, hasOpts := pregunta["opciones"]
	correctIdx, hasIdx := pregunta["correct_index"]

	if !hasOpts || !hasIdx {
		return
	}

	// Convertir opciones a slice de strings
	var options []string
	switch opts := rawOpts.(type) {
	case []any:
		for _, v := range opts {
			if s, ok := v.(string); ok {
				options = append(options, s)
			}
		}
	case []string:
		options = opts
	default:
		return
	}

	// Convertir correct_index a int
	var correctIndex int
	switch idx := correctIdx.(type) {
	case float64:
		correctIndex = int(idx)
	case int:
		correctIndex = idx
	default:
		return
	}

	// Aplicar randomización
	shuffledOptions, newCorrectIndex := shuffleOptionsWithCorrectIndex(options, correctIndex)

	// Actualizar la estructura
	pregunta["opciones"] = shuffledOptions
	pregunta["correct_index"] = newCorrectIndex
}

// generateProgressiveDiagnostics genera contenido diagnóstico basado en el turno
func generateProgressiveDiagnostics(turnNumber int, threadID string) string {
	var diagnostics []string

	switch {
	case turnNumber == 1:
		// Primer turno: solo anamnesis y examen físico
		return ""
	case turnNumber == 2:
		// Segundo turno: laboratorios básicos
		diagnostics = append(diagnostics,
			"LABORATORIOS DISPONIBLES:",
			"- Hemograma completo, química sanguínea básica",
			"- Gases arteriales, electrolitos",
			"- Marcadores inflamatorios (PCR, VSG)")
	case turnNumber == 3:
		// Tercer turno: laboratorios específicos
		diagnostics = append(diagnostics,
			"LABORATORIOS ESPECÍFICOS:",
			"- Marcadores cardíacos (troponinas, CK-MB)",
			"- Función renal y hepática completa",
			"- Coagulación (PT, PTT, INR)")
	case turnNumber == 4:
		// Cuarto turno: imágenes básicas
		diagnostics = append(diagnostics,
			"IMÁGENES DIAGNÓSTICAS:",
			"- Radiografía de tórax",
			"- Electrocardiograma",
			"- Ecografía abdominal")
	case turnNumber >= 5:
		// Turnos avanzados: estudios especializados
		diagnostics = append(diagnostics,
			"ESTUDIOS ESPECIALIZADOS:",
			"- TAC o RM según indicación clínica",
			"- Estudios funcionales específicos",
			"- Interconsultas especializadas")
	}

	if len(diagnostics) > 0 {
		return "\n\n" + strings.Join(diagnostics, "\n")
	}
	return ""
}
