package casos_clinico

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"ema-backend/openai"
	"ema-backend/sse"

	"github.com/gin-gonic/gin"
)

// Assistant matches the minimal interface implemented by openai.Client
type Assistant interface {
	CreateThread(ctx context.Context) (string, error)
	StreamAssistantMessage(ctx context.Context, threadID, prompt string) (<-chan string, error)
	StreamAssistantJSON(ctx context.Context, threadID, userPrompt, jsonInstructions string) (<-chan string, error)
	// Métodos adicionales para búsqueda de evidencia (RAG + PubMed)
	SearchInVectorStore(ctx context.Context, vectorStoreID, query string) (string, error)
	SearchInVectorStoreWithMetadata(ctx context.Context, vectorStoreID, query string) (*openai.VectorSearchResult, error)
	SearchPubMed(ctx context.Context, query string) (string, error)
}

type Handler struct {
	aiAnalytical             Assistant
	aiInteractive            Assistant
	quotaValidator           func(ctx context.Context, c *gin.Context, flow string) error
	mu                       sync.Mutex
	analyticalTurns          map[string]int // thread_id -> number of chat turns served
	lastAnalyticalDiag       []string
	maxAnalyticalDiagHistory int
}

// NewHandler lets you inject different assistants (analytical/interactive). If one is nil, the other will be used for both flows.
func NewHandler(analytical Assistant, interactive Assistant) *Handler {
	if analytical == nil && interactive == nil {
		cli := openai.NewClient()
		return &Handler{aiAnalytical: cli, aiInteractive: cli, analyticalTurns: make(map[string]int), maxAnalyticalDiagHistory: resolveDiagHistoryEnv()}
	}
	if analytical == nil {
		analytical = interactive
	}
	if interactive == nil {
		interactive = analytical
	}
	return &Handler{aiAnalytical: analytical, aiInteractive: interactive, analyticalTurns: make(map[string]int), maxAnalyticalDiagHistory: resolveDiagHistoryEnv()}
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
	return &Handler{aiAnalytical: cliA, aiInteractive: cliI, analyticalTurns: make(map[string]int), maxAnalyticalDiagHistory: resolveDiagHistoryEnv()}
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
			// Enrich 403 with field/reason if present
			field, _ := c.Get("quota_error_field")
			reason, _ := c.Get("quota_error_reason")
			c.JSON(http.StatusForbidden, gin.H{"error": "clinical cases quota exceeded", "field": field, "reason": reason})
			return
		}
	}
	if v, ok := c.Get("quota_remaining"); ok {
		c.Header("X-Quota-Remaining", toString(v))
		c.Header("X-Quota-Field", "clinical_cases")
	}
	var req generateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(getHTTPTimeoutSec())*time.Second)
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
	// Construir cláusula de evitación de diagnósticos recientes
	h.mu.Lock()
	avoidList := append([]string(nil), h.lastAnalyticalDiag...)
	h.mu.Unlock()
	avoidClause := ""
	if len(avoidList) > 0 {
		avoidClause = "Evita que el diagnóstico final sea exactamente alguno de estos diagnósticos usados recientemente: " + strings.Join(avoidList, "; ") + ". Selecciona una patología distinta plausible dadas las características del paciente."
	}
	userPrompt := strings.Join([]string{
		"Genera un único objeto JSON con la clave 'case' describiendo un caso clínico completo (estático).",
		"El paciente: edad=" + strings.TrimSpace(req.Age) + ", sexo=" + strings.TrimSpace(req.Sex) + ", gestante=" + boolToStr(req.Pregnant) + ".",
		"Incluye anamnesis, examen físico, pruebas diagnósticas, diagnóstico final y plan de manejo.",
		avoidClause,
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
		// Fallback: entregar caso mínimo para evitar 504 en el cliente/proxy
		c.JSON(http.StatusOK, map[string]any{
			"case": map[string]any{
				"id":                   0,
				"title":                "Caso clínico analítico",
				"type":                 "static",
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
			"thread_id": threadID,
			"note":      "timeout",
		})
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

	// Registrar diagnóstico final para reducir repeticiones en futuros casos
	if caseMap, ok := parsed["case"].(map[string]any); ok {
		if diagRaw, ok2 := caseMap["final_diagnosis"]; ok2 {
			diag := strings.TrimSpace(fmt.Sprint(diagRaw))
			if diag != "" {
				h.mu.Lock()
				// Evitar duplicados exactos
				already := false
				for _, d := range h.lastAnalyticalDiag {
					if strings.EqualFold(d, diag) {
						already = true
						break
					}
				}
				if !already {
					h.lastAnalyticalDiag = append(h.lastAnalyticalDiag, diag)
					if len(h.lastAnalyticalDiag) > h.maxAnalyticalDiagHistory && h.maxAnalyticalDiagHistory > 0 {
						// recortar al tamaño
						start := len(h.lastAnalyticalDiag) - h.maxAnalyticalDiagHistory
						h.lastAnalyticalDiag = append([]string(nil), h.lastAnalyticalDiag[start:]...)
					}
				}
				h.mu.Unlock()
			}
		}
	}
	// Anexar referencias (RAG + PubMed) de forma no disruptiva: al final de management
	// Por defecto habilitado (deshabilitar con CLINICAL_APPEND_REFS=false)
	if isRAGEnabled() {
		func() {
			// proteger contra pánicos por tipos inesperados
			defer func() { _ = recover() }()
			// Timeout más agresivo para búsquedas de evidencia (10s máximo)
			refCtx, refCancel := context.WithTimeout(ctx, 10*time.Second)
			defer refCancel()

			if caseMap, ok := parsed["case"].(map[string]any); ok {
				q := buildCaseQuery(caseMap)
				if strings.TrimSpace(q) != "" {
					refs := collectEvidence(refCtx, h.aiAnalytical, q)
					if strings.TrimSpace(refs) != "" {
						mg := strings.TrimSpace(fmt.Sprint(caseMap["management"]))
						caseMap["management"] = appendRefs(mg, refs)
					}
				}
			}
		}()
	}
	c.JSON(http.StatusOK, parsed)
}

// ChatAnalytical continues the analytical chat and returns { respuesta: { text: string } }
func (h *Handler) ChatAnalytical(c *gin.Context) {
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "analytical_chat"); err != nil {
			field, _ := c.Get("quota_error_field")
			reason, _ := c.Get("quota_error_reason")
			c.JSON(http.StatusForbidden, gin.H{"error": "clinical cases quota exceeded", "field": field, "reason": reason})
			return
		}
	}
	if v, ok := c.Get("quota_remaining"); ok {
		c.Header("X-Quota-Remaining", toString(v))
		c.Header("X-Quota-Field", "clinical_cases")
	}
	var req chatReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Mensaje) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(getHTTPTimeoutSec())*time.Second)
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
	// Determinar turno lógico (1-based). El mapa almacena el último turno completado.
	h.mu.Lock()
	turn := h.analyticalTurns[threadID] + 1
	h.mu.Unlock()

	// Construimos instrucciones dinámicas para fomentar profundidad y continuidad.
	var phaseInstr string
	switch {
	case turn < 3:
		phaseInstr = "Extiende el razonamiento clínico inicial: analiza síntomas cardinales, factores de riesgo y plantea hipótesis diagnósticas preliminares. Integra resultados de ayudas diagnósticas (laboratorio, imagen, etc.) si ya existen en el hilo/caso; si no existen, sugiere las más pertinentes y explica qué hallazgos esperarías y cómo cambiarían la probabilidad diagnóstica."
	case turn < 6:
		phaseInstr = "Profundiza correlación fisiopatológica. Integra explícitamente resultados de ayudas diagnósticas disponibles y su impacto en hipótesis; si faltan, propone exámenes concretos y justifica su utilidad."
	case turn < 9:
		phaseInstr = "Integra hallazgos (incluyendo resultados de pruebas diagnósticas) y prioriza diagnósticos diferenciales con justificación comparativa (por qué uno es más probable que otro)."
	default:
		phaseInstr = "Prepara el cierre: sintetiza los datos clave, incluyendo ayudas diagnósticas y su interpretación, y guía al usuario hacia diagnóstico final y plan terapéutico."
	}

	closingInstr := ""
	// Tras el turno 9 (> =10), permite conclusión y referencias PERO siempre con pregunta final breve.
	if turn >= 10 {
		closingInstr = "Si la información es suficiente para cerrar, entrega: Conclusión final + Plan de manejo + 'Referencias:' con 2-3 citas (formato narrativo abreviado). AÚN ASÍ, SIEMPRE añade al final una pregunta breve relacionada con el caso (p. ej., siguiente paso, diagnóstico diferencial a descartar, o seguimiento)."
	} else {
		closingInstr = "No cierres todavía ni des conclusiones definitivas. No incluyas bibliografía aún."
	}

	// Búsqueda de evidencia ANTES de generar respuesta para fundamentar evaluación crítica
	var ragContext string
	if isRAGEnabled() {
		func() {
			defer func() { _ = recover() }()
			refCtx, refCancel := context.WithTimeout(ctx, 5*time.Second)
			defer refCancel()
			refs := collectEvidence(refCtx, h.aiAnalytical, userPrompt)
			if strings.TrimSpace(refs) != "" {
				ragContext = "EVIDENCIA CIENTÍFICA DISPONIBLE para fundamentar tu evaluación:\n" + refs + "\n\n"
			}
		}()
	}

	instr := strings.Join([]string{
		"Responde estrictamente en JSON válido con la clave 'respuesta': { 'text': <string> }.",
		"Estructura del texto: Razonamiento clínico progresivo (2–3 párrafos, 150–220 palabras totales) evaluando la respuesta del usuario, seguido de pregunta final.",
		phaseInstr,
		closingInstr,
		"EVALUACIÓN CRÍTICA DENTRO DEL CONTEXTO CLÍNICO:",
		"- Evalúa la respuesta del usuario según el contexto ESPECÍFICO del caso (edad, diagnóstico, hallazgos clínicos presentados).",
		"- Si la respuesta es INCORRECTA o inapropiada: explica claramente por qué no está indicada EN ESTE CASO PARTICULAR. Indica la conducta correcta.",
		"- Si la respuesta es CORRECTA: refuerza los conceptos clave sin elogios innecesarios.",
		"- NO uses emojis (❌✅) ni encabezados ('Evaluación:', 'Razonamiento clínico:').",
		"- NO uses lenguaje condescendiente que valide errores ('podría ser una opción', 'no sería la primera elección pero...').",
		"- NO introduzcas escenarios o poblaciones ajenas al caso (ej: no menciones lactantes si el paciente es adolescente).",
		"- FUNDAMENTA con evidencia científica disponible en el contexto.",
		"Ejemplo INCORRECTO: 'La radiografía simple de abdomen no confirma apendicitis. En este caso, los exámenes adecuados son hemograma, PCR y ecografía abdominal. Si hay duda diagnóstica, puede considerarse TAC de abdomen.'",
		"Ejemplo CORRECTO: 'Solicitar Monospot es apropiado dado el cuadro clínico. Esta prueba tiene alta especificidad (>95%) en adolescentes con presentación compatible y complementa el diagnóstico cuando hay linfocitosis atípica y hallazgos físicos sugestivos.'",
		"CONTEXTUALIZACIÓN: Relaciona CADA comentario con las características ESPECÍFICAS del paciente presentado (edad, comorbilidades, presentación clínica, hallazgos previos).",
		"COMPARACIÓN: Si es incorrecta, indica la conducta CORRECTA para ESTE paciente específico y explica por qué es superior.",
		"Cada párrafo separado por UNA línea en blanco. Sin viñetas, tablas ni markdown.",
		"La ÚLTIMA línea: SOLO la pregunta, sin texto adicional.",
		"No inventes datos nuevos. Mantente dentro del caso clínico presentado.",
		"Idioma: español.",
	}, " ")

	// Integrar evidencia en el prompt del usuario si está disponible
	if ragContext != "" {
		userPrompt = ragContext + "RESPUESTA DEL USUARIO: " + userPrompt
	}
	// Si el cliente solicita streaming SSE, emitimos eventos con marcadores de etapa
	accept := strings.ToLower(strings.TrimSpace(c.GetHeader("Accept")))
	if strings.Contains(accept, "text/event-stream") {
		// Búsqueda de evidencia para streaming
		var refsSSE string
		if isRAGEnabled() {
			func() {
				defer func() { _ = recover() }()
				refCtx, refCancel := context.WithTimeout(ctx, 5*time.Second)
				defer refCancel()
				refsSSE = collectEvidence(refCtx, h.aiAnalytical, req.Mensaje)
			}()
		}

		// Para SSE, pedimos TEXTO PLANO (no JSON) para que el frontend no reciba envoltorios.
		// Instrucciones textuales equivalentes a la versión JSON:
		textInstr := strings.Join([]string{
			"Responde en TEXTO PLANO en español, sin markdown ni JSON.",
			"Estructura: Razonamiento clínico (2–3 párrafos, 150–220 palabras) evaluando la respuesta del usuario, seguido de pregunta final.",
			phaseInstr,
			closingInstr,
			"EVALUACIÓN CRÍTICA DENTRO DEL CONTEXTO CLÍNICO:",
			"- Evalúa según el contexto ESPECÍFICO del caso presentado (edad, diagnóstico, hallazgos).",
			"- Si es INCORRECTA: explica por qué no está indicada EN ESTE CASO. Indica la conducta correcta.",
			"- Si es CORRECTA: refuerza conceptos clave sin elogios innecesarios.",
			"- NO uses emojis ni encabezados.",
			"- NO uses lenguaje condescendiente que valide errores.",
			"- NO introduzcas escenarios ajenos al caso presentado.",
			"- FUNDAMENTA con evidencia científica disponible.",
			"Ejemplo: 'La radiografía simple de abdomen no confirma apendicitis. En este caso, los exámenes adecuados son hemograma, PCR y ecografía abdominal.'",
			"CONTEXTUALIZACIÓN: Relaciona con características del paciente presentado.",
			"COMPARACIÓN: Si es incorrecta, indica la conducta CORRECTA para este paciente.",
			"Separa párrafos con UNA línea en blanco.",
			"La ÚLTIMA línea: SOLO la pregunta.",
			"Mantente dentro del caso clínico presentado.",
		}, " ")

		// Integrar evidencia si está disponible
		promptWithContext := req.Mensaje
		if strings.TrimSpace(refsSSE) != "" {
			promptWithContext = "EVIDENCIA CIENTÍFICA DISPONIBLE:\n" + refsSSE + "\n\nRESPUESTA DEL USUARIO: " + req.Mensaje
		}

		// Obtener stream de texto plano del assistant
		prompt := strings.Join([]string{
			"Mensaje del usuario:", promptWithContext,
			"\n\nInstrucciones:", textInstr,
		}, " ")
		ch, err := h.aiAnalytical.StreamAssistantMessage(ctx, threadID, prompt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "assistant error"})
			return
		}

		// Marcar etapas de búsqueda de evidencia para el frontend
		stages := []string{"__STAGE__:start", "__STAGE__:rag_search"}
		if strings.TrimSpace(refsSSE) == "" {
			stages = append(stages, "__STAGE__:rag_empty", "__STAGE__:no_source", "__STAGE__:streaming_answer")
		} else {
			hasPub := strings.Contains(refsSSE, "PubMed:")
			hasRag := strings.Contains(refsSSE, "Base de conocimiento médico") || strings.Contains(refsSSE, "Referencias:")
			if hasRag {
				stages = append(stages, "__STAGE__:rag_found")
			} else {
				stages = append(stages, "__STAGE__:rag_empty")
			}
			if hasPub {
				stages = append(stages, "__STAGE__:pubmed_search", "__STAGE__:pubmed_found")
			}
			stages = append(stages, "__STAGE__:streaming_answer")
		}

		// Stream con marcadores de etapa y garantía de pregunta final si el modelo la omite
		sse.Stream(c, wrapWithFinalQuestion(wrapWithStages(stages, ch), userPrompt, turn))
		return
	}

	// Fallback: comportamiento legacy (no streaming) — consumir primer chunk y responder JSON
	ch, err := h.aiAnalytical.StreamAssistantJSON(ctx, threadID, userPrompt, instr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "assistant error"})
		return
	}
	var content string
	select {
	case content = <-ch:
	case <-ctx.Done():
		// Fallback: devolver texto plano en JSON simple para no romper UI
		c.JSON(http.StatusOK, gin.H{"text": "No pude responder a tiempo. Intenta nuevamente.", "thread_id": threadID, "note": "timeout"})
		return
	}
	jsonStr := extractJSON(content)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil || parsed["respuesta"] == nil {
		if fixed, ok := h.repairAnalyticalChatJSON(ctx, threadID, content); ok {
			parsed = map[string]any{}
			_ = json.Unmarshal([]byte(fixed), &parsed)
			// Registrar turno completado
			h.mu.Lock()
			h.analyticalTurns[threadID] = turn
			h.mu.Unlock()
			// Anexar referencias solo en cierre (>= turno 10) para respetar estilo existente
			if turn >= 10 {
				safeAppendRefsToRespuesta(ctx, h.aiAnalytical, &parsed, req.Mensaje)
			}
			// Responder solo texto para UI: {text, thread_id}
			if respMap, ok := parsed["respuesta"].(map[string]any); ok {
				raw := strings.TrimSpace(fmt.Sprint(respMap["text"]))
				q := deriveFinalQuestion(raw, req.Mensaje, turn)
				txt := ensureEndsWithQuestionWithFallback(raw, q)
				c.JSON(http.StatusOK, gin.H{"text": txt, "thread_id": threadID})
			} else {
				raw := strings.TrimSpace(content)
				q := deriveFinalQuestion(raw, req.Mensaje, turn)
				c.JSON(http.StatusOK, gin.H{"text": ensureEndsWithQuestionWithFallback(raw, q), "thread_id": threadID})
			}
			return
		}
	} else {
		// Registrar turno completado
		h.mu.Lock()
		h.analyticalTurns[threadID] = turn
		h.mu.Unlock()
		// Anexar referencias solo en cierre (>= turno 10)
		if turn >= 10 {
			safeAppendRefsToRespuesta(ctx, h.aiAnalytical, &parsed, req.Mensaje)
		}
		// Responder solo texto para UI
		if respMap, ok := parsed["respuesta"].(map[string]any); ok {
			raw := strings.TrimSpace(fmt.Sprint(respMap["text"]))
			q := deriveFinalQuestion(raw, req.Mensaje, turn)
			c.JSON(http.StatusOK, gin.H{"text": ensureEndsWithQuestionWithFallback(raw, q), "thread_id": threadID})
		} else {
			raw := strings.TrimSpace(content)
			q := deriveFinalQuestion(raw, req.Mensaje, turn)
			c.JSON(http.StatusOK, gin.H{"text": ensureEndsWithQuestionWithFallback(raw, q), "thread_id": threadID})
		}
		return
	}
	// Fallback text
	raw := strings.TrimSpace(content)
	q := deriveFinalQuestion(raw, req.Mensaje, turn)
	c.JSON(http.StatusOK, gin.H{"text": ensureEndsWithQuestionWithFallback(raw, q), "thread_id": threadID})
}

// wrapWithFinalQuestion re-emite el stream y, al finalizar, agrega una pregunta final
// si el contenido no terminó en una línea de pregunta.
func wrapWithFinalQuestion(in <-chan string, userPrompt string, turn int) <-chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		var buf strings.Builder
		for msg := range in {
			buf.WriteString(msg)
			out <- msg
		}
		// Si no termina en pregunta, derivar una acorde al contenido y turno
		if !endsWithQuestionMark(buf.String()) {
			q := deriveFinalQuestion(buf.String(), userPrompt, turn)
			if strings.TrimSpace(q) == "" {
				q = "¿Cuál sería el siguiente paso más adecuado?"
			}
			if os.Getenv("CLINICAL_DEBUG_FINALQ") == "true" {
				tail := buf.String()
				if len(tail) > 180 {
					tail = tail[len(tail)-180:]
				}
				log.Printf("[analytical][sse] appended final question | turn=%d | derived='%s' | tail=\n%s", turn, q, tail)
			}
			out <- "\n\n" + q
		}
	}()
	return out
}

// ensureEndsWithQuestion garantiza que el texto termine con una pregunta en la última línea.
// Si ya termina con '?', no hace cambios. Si no, agrega dos saltos de línea y una pregunta genérica breve.
func ensureEndsWithQuestion(text string) string {
	t := strings.TrimRight(text, " \t\n")
	// encontrar última línea no vacía
	lines := strings.Split(t, "\n")
	i := len(lines) - 1
	for ; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			break
		}
	}
	if i >= 0 {
		last := strings.TrimSpace(lines[i])
		if strings.HasSuffix(last, "?") {
			return t
		}
	} else {
		// texto vacío
		t = ""
	}
	// Añadir pregunta breve por defecto
	q := "¿Cuál sería el siguiente paso más adecuado?"
	if t == "" {
		return q
	}
	return t + "\n\n" + q
}

// ensureEndsWithQuestionWithFallback usa una pregunta sugerida si falta el cierre interrogativo
func ensureEndsWithQuestionWithFallback(text, fallbackQuestion string) string {
	if endsWithQuestionMark(text) {
		return strings.TrimRight(text, " \t\n")
	}
	fb := strings.TrimSpace(fallbackQuestion)
	if fb == "" {
		return ensureEndsWithQuestion(text)
	}
	t := strings.TrimRight(text, " \t\n")
	if t == "" {
		return fb
	}
	if os.Getenv("CLINICAL_DEBUG_FINALQ") == "true" {
		tail := t
		if len(tail) > 180 {
			tail = tail[len(tail)-180:]
		}
		log.Printf("[analytical][json] appended final question | derived='%s' | tail=\n%s", fb, tail)
	}
	return t + "\n\n" + fb
}

// deriveFinalQuestion intenta construir una pregunta coherente con el contenido y la fase del turno
func deriveFinalQuestion(text, userPrompt string, turn int) string {
	t := strings.ToLower(text)
	// Reglas por contenido
	if strings.Contains(t, "diagnóstico") || strings.Contains(t, "diagnostico") {
		if strings.Contains(t, "diferencial") || strings.Contains(t, "diferenciales") || strings.Contains(t, "otros diagn") {
			return "¿Qué otros diagnósticos posibles deberían considerarse según los factores de riesgo y la presentación clínica del paciente?"
		}
		if strings.Contains(t, "más probable") || strings.Contains(t, "mas probable") || strings.Contains(t, "probabilidad") {
			return "¿Cuál es tu diagnóstico más probable y por qué?"
		}
	}
	if strings.Contains(t, "ayuda diagn") || strings.Contains(t, "prueba") || strings.Contains(t, "examen") || strings.Contains(t, "estudio") {
		return "¿Qué ayudas diagnósticas solicitarías ahora y qué hallazgos esperarías?"
	}
	if strings.Contains(t, "manejo") || strings.Contains(t, "tratamiento") || strings.Contains(t, "terapéut") || strings.Contains(t, "terapeut") {
		return "¿Cuál sería el siguiente paso más adecuado en el manejo?"
	}
	// Reglas por fase
	switch {
	case turn <= 2:
		return "¿Cuál es tu diagnóstico inicial prioritario y por qué?"
	case turn <= 4:
		return "¿Qué ayudas diagnósticas solicitarías ahora y qué hallazgos esperarías?"
	case turn <= 7:
		return "¿Qué diagnósticos diferenciales considerarías y cuál es su probabilidad relativa?"
	case turn < 10:
		return "¿Cuál sería el plan de manejo inicial más apropiado?"
	default:
		return "¿Cuál es tu diagnóstico final y cuál sería el siguiente paso en el manejo?"
	}
}

// endsWithQuestionMark informa si la última línea no vacía de s termina con '?'
func endsWithQuestionMark(s string) bool {
	t := strings.TrimRight(s, " \t\n")
	if t == "" {
		return false
	}
	lines := strings.Split(t, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		last := strings.TrimSpace(lines[i])
		if last == "" {
			continue
		}
		// Permitir comillas/paréntesis/cierre ASCII después del '?'
		// Recorremos de derecha a izquierda saltando cierres comunes (ASCII seguro).
		r := []rune(last)
		j := len(r) - 1
		for j >= 0 {
			switch r[j] {
			case ' ', '\t', '\n', '\r', '"', '\'', ')', ']', '}', '.':
				j--
				continue
			}
			break
		}
		if j >= 0 && r[j] == '?' {
			return true
		}
		return false
	}
	return false
}

// GenerateInteractive creates the case and an initial question: returns { case: {...}, data: { questions: { texto,tipo,opciones } }, thread_id }
func (h *Handler) GenerateInteractive(c *gin.Context) {
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "interactive_generate"); err != nil {
			field, _ := c.Get("quota_error_field")
			reason, _ := c.Get("quota_error_reason")
			c.JSON(http.StatusForbidden, gin.H{"error": "clinical cases quota exceeded", "field": field, "reason": reason})
			return
		}
	}
	if v, ok := c.Get("quota_remaining"); ok {
		c.Header("X-Quota-Remaining", toString(v))
		c.Header("X-Quota-Field", "clinical_cases")
	}
	var req generateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(getHTTPTimeoutSec())*time.Second)
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
		// Fallback: caso y pregunta mínimos para evitar 504
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
			"thread_id": threadID,
			"note":      "timeout",
		})
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
	// Anexar referencias al management del caso (no altera preguntas)
	// Por defecto habilitado (deshabilitar con CLINICAL_APPEND_REFS=false)
	if isRAGEnabled() {
		func() {
			defer func() { _ = recover() }()
			if caseMap, ok := parsed["case"].(map[string]any); ok {
				q := buildCaseQuery(caseMap)
				if strings.TrimSpace(q) != "" {
					// Timeout agresivo para evidencia en generación interactiva
					refCtx, refCancel := context.WithTimeout(ctx, 8*time.Second)
					defer refCancel()
					refs := collectEvidence(refCtx, h.aiInteractive, q)
					if strings.TrimSpace(refs) != "" {
						mg := strings.TrimSpace(fmt.Sprint(caseMap["management"]))
						caseMap["management"] = appendRefs(mg, refs)
					}
				}
			}
		}()
	}
	c.JSON(http.StatusOK, parsed)
}

// ChatInteractive processes a user message and returns feedback + next question: { data: { feedback, question, thread_id } }
func (h *Handler) ChatInteractive(c *gin.Context) {
	if h.quotaValidator != nil {
		if err := h.quotaValidator(c.Request.Context(), c, "interactive_chat"); err != nil {
			field, _ := c.Get("quota_error_field")
			reason, _ := c.Get("quota_error_reason")
			c.JSON(http.StatusForbidden, gin.H{"error": "clinical cases quota exceeded", "field": field, "reason": reason})
			return
		}
	}
	if v, ok := c.Get("quota_remaining"); ok {
		c.Header("X-Quota-Remaining", toString(v))
		c.Header("X-Quota-Field", "clinical_cases")
	}
	var req chatReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Mensaje) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(getHTTPTimeoutSec())*time.Second)
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
		// Fallback: devolver estructura mínima para mantener el flujo
		c.JSON(http.StatusOK, map[string]any{
			"data": map[string]any{
				"feedback": "No pude responder a tiempo. Intenta nuevamente.",
				"question": map[string]any{
					"texto":    "Propón tu diagnóstico diferencial principal.",
					"tipo":     "open_ended",
					"opciones": []string{},
				},
				"thread_id": threadID,
			},
			"note": "timeout",
		})
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
	// No anexamos referencias en feedback para mantener brevedad (<40 palabras). Las referencias se agregan en GenerateInteractive (management).
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

// resolveDiagHistoryEnv lee variable opcional CLINICAL_ANALYTICAL_DIAG_HISTORY
// para definir cuántos diagnósticos recientes se recuerdan y evita repetir.
// Valor por defecto: 8. Si es 0 o negativo, se desactiva la función.
func resolveDiagHistoryEnv() int {
	v := strings.TrimSpace(os.Getenv("CLINICAL_ANALYTICAL_DIAG_HISTORY"))
	if v == "" {
		return 8
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return 8
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

// helper for quota header string conversion
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

// --- Evidence helpers (RAG + PubMed) --- //

// isRAGEnabled retorna true si RAG está habilitado (por defecto: true, excepto si CLINICAL_APPEND_REFS="false")
func isRAGEnabled() bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv("CLINICAL_APPEND_REFS")))
	// Si está vacío o cualquier valor distinto de "false", está habilitado
	return val != "false"
}

// getFixedBooksVectorID retorna el vector store fijo usado para libros del asistente (si está configurado);
// si no, usa el ID observado en conversations_ia (mantener sincronizado si cambia).
func getFixedBooksVectorID() string {
	if v := strings.TrimSpace(os.Getenv("INTERACTIVE_VECTOR_ID")); v != "" { // permitir override por env
		return v
	}
	// Valor por defecto usado en conversations_ia.SmartMessage
	return "vs_680fc484cef081918b2b9588b701e2f4"
}

// collectEvidence busca primero en vector de libros y luego en PubMed; devuelve bloque de referencias o cadena vacía.
func collectEvidence(ctx context.Context, ai Assistant, query string) string {
	// 1) Libros (vector fijo) con metadatos si están disponibles
	refs := make([]string, 0, 3)
	vectorID := getFixedBooksVectorID()
	if strings.HasPrefix(vectorID, "vs_") {
		if res, err := ai.SearchInVectorStoreWithMetadata(ctx, vectorID, query); err == nil && res != nil && res.HasResult {
			// Formato APA simplificado: Autor/Fuente (año). Título/Sección.
			src := strings.TrimSpace(res.Source)
			sec := strings.TrimSpace(res.Section)
			if src == "" {
				src = "Fuente médica"
			}
			// Intentar extraer año si está en formato "Libro (2020)" o similar
			year := ""
			if idx := strings.LastIndex(src, "("); idx != -1 {
				if idx2 := strings.Index(src[idx:], ")"); idx2 != -1 {
					yearCandidate := strings.TrimSpace(src[idx+1 : idx+idx2])
					if len(yearCandidate) == 4 {
						year = yearCandidate
						src = strings.TrimSpace(src[:idx])
					}
				}
			}
			line := src
			if year != "" {
				line = line + " (" + year + ")"
			}
			if sec != "" {
				line = line + ". " + sec + "."
			} else {
				line = line + "."
			}
			refs = append(refs, line)
		} else if txt, err2 := ai.SearchInVectorStore(ctx, vectorID, query); err2 == nil && strings.TrimSpace(txt) != "" {
			// Fallback sin metadatos
			refs = append(refs, "Base de conocimiento médico.")
		}
	}
	// 2) PubMed - formato simplificado
	if pm, err := ai.SearchPubMed(ctx, query); err == nil && strings.TrimSpace(pm) != "" {
		// Intentar extraer primer autor y año si está en el formato típico de PubMed
		p := strings.TrimSpace(pm)
		// Buscar patrón "Autor et al. (año)" o "Autor (año)"
		pmRef := "PubMed."
		if idx := strings.Index(p, "("); idx != -1 && idx < 100 {
			if idx2 := strings.Index(p[idx:], ")"); idx2 != -1 && idx2 < 10 {
				author := strings.TrimSpace(p[:idx])
				year := strings.TrimSpace(p[idx+1 : idx+idx2])
				if len(year) == 4 && author != "" {
					pmRef = author + " (" + year + ")."
				}
			}
		}
		refs = append(refs, pmRef)
	}
	if len(refs) == 0 {
		return ""
	}
	// Construir bloque "Referencias:" con formato APA simplificado
	b := &strings.Builder{}
	b.WriteString("\n\nReferencias:\n")
	for i, r := range refs {
		if i >= 3 {
			break
		}
		b.WriteString(r)
		b.WriteString("\n")
	}
	return b.String()
}

// appendRefs añade el bloque de referencias al campo de texto si no está vacío.
func appendRefs(s, refs string) string {
	if strings.TrimSpace(refs) == "" {
		return s
	}
	if strings.TrimSpace(s) == "" {
		return strings.TrimSpace(refs)
	}
	return strings.TrimRight(s, "\n ") + refs
}

// buildCaseQuery arma una consulta breve a partir del caso para buscar evidencia
func buildCaseQuery(caseMap map[string]any) string {
	title := strings.TrimSpace(fmt.Sprint(caseMap["title"]))
	diag := strings.TrimSpace(fmt.Sprint(caseMap["final_diagnosis"]))
	if title != "" && diag != "" {
		q := title + " — " + diag
		if len(q) > 220 {
			q = q[:220]
		}
		return q
	}
	// Fallback: usar fragmentos clave del caso
	ana := strings.TrimSpace(fmt.Sprint(caseMap["anamnesis"]))
	exa := strings.TrimSpace(fmt.Sprint(caseMap["physical_examination"]))
	dtx := strings.TrimSpace(fmt.Sprint(caseMap["diagnostic_tests"]))
	parts := make([]string, 0, 4)
	if title != "" {
		parts = append(parts, title)
	}
	if diag != "" {
		parts = append(parts, diag)
	}
	if ana != "" {
		parts = append(parts, ana)
	}
	if exa != "" {
		parts = append(parts, exa)
	}
	if dtx != "" {
		parts = append(parts, dtx)
	}
	q := strings.Join(parts, ". ")
	if len(q) > 240 {
		q = q[:240]
	}
	return q
}

// safeAppendRefsToRespuesta agrega referencias al campo respuesta.text en JSON {respuesta:{text}}
func safeAppendRefsToRespuesta(ctx context.Context, ai Assistant, parsed *map[string]any, userMsg string) {
	defer func() { _ = recover() }()
	if parsed == nil {
		return
	}
	// Solo buscar referencias si está habilitado para evitar timeouts
	if !isRAGEnabled() {
		return
	}
	p := *parsed
	r, ok := p["respuesta"].(map[string]any)
	if !ok {
		return
	}
	txt := strings.TrimSpace(fmt.Sprint(r["text"]))
	// Usar el mensaje del usuario como query principal; fallback al propio texto
	query := strings.TrimSpace(userMsg)
	if query == "" {
		query = txt
	}
	if query == "" {
		return
	}
	// Timeout agresivo para evitar bloqueos
	refCtx, refCancel := context.WithTimeout(ctx, 8*time.Second)
	defer refCancel()
	refs := collectEvidence(refCtx, ai, query)
	if strings.TrimSpace(refs) == "" {
		return
	}
	r["text"] = appendRefs(txt, refs)
}

// wrapWithStages emite marcadores de etapa antes de reenviar el stream principal.
// Cada entrada de `stages` se enviará como un token individual en el canal de salida.
func wrapWithStages(stages []string, ch <-chan string) <-chan string {
	out := make(chan string)
	go func() {
		for _, s := range stages {
			if strings.TrimSpace(s) == "" {
				continue
			}
			out <- s
		}
		for tok := range ch {
			out <- tok
		}
		close(out)
	}()
	return out
}

// getHTTPTimeoutSec lee el timeout HTTP en segundos para las rutas de casos clínicos
// desde CLINICAL_HTTP_TIMEOUT_SEC. Por defecto 45 segundos.
func getHTTPTimeoutSec() int {
	v := strings.TrimSpace(os.Getenv("CLINICAL_HTTP_TIMEOUT_SEC"))
	if v == "" {
		return 45
	}
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return n
	}
	return 45
}
