package casos_interactivos

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockAI struct{}

func (m *mockAI) CreateThread(ctx context.Context) (string, error) { return "thread-1", nil }
func (m *mockAI) StreamAssistantJSON(ctx context.Context, threadID, userPrompt, jsonInstructions string) (<-chan string, error) {
	ch := make(chan string, 1)
	// Return a minimal valid interactive turn with correct_index to exercise local evaluation
	ch <- `{"feedback":"Evaluación: CORRECTO\nTexto previo.","next":{"hallazgos":{},"pregunta":{"tipo":"single-choice","texto":"¿Paso?","opciones":["Op1","Op2","Op3","Op4"],"correct_index":2}},"finish":0}`
	return ch, nil
}

// mock sin correct_index para probar evaluation_pending y recuperación
type mockMissingCI struct{}
func (m *mockMissingCI) CreateThread(ctx context.Context) (string, error) { return "thread-miss", nil }
func (m *mockMissingCI) StreamAssistantJSON(ctx context.Context, threadID, userPrompt, jsonInstructions string) (<-chan string, error) {
	ch := make(chan string,1)
	// Si el prompt pide correct_index (recuperación) respondemos sólo con correct_index JSON
	if strings.Contains(jsonInstructions, "correct_index") || strings.Contains(userPrompt, "correct_index") {
		ch <- `{"correct_index":1}`
		return ch, nil
	}
	// primera llamada: pregunta sin correct_index
	ch <- `{"feedback":"Evaluación: CORRECTO\nTexto previo.","next":{"hallazgos":{},"pregunta":{"tipo":"single-choice","texto":"Q missing","opciones":["O1","O2","O3","O4"]}},"finish":0}`
	return ch, nil
}

func setup() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	h := &Handler{ai: &mockAI{}, maxQuestions: 3, turnCount: make(map[string]int), threadMaxQuestions: make(map[string]int), askedQuestions: make(map[string][]string), evalCorrect: make(map[string]int), evalAnswers: make(map[string]int), lastCorrectIndex: make(map[string]int), lastOptions: make(map[string][]string)}
	h.RegisterRoutes(r)
	return r
}

func TestStartCase_OK(t *testing.T) {
	r := setup()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/casos-interactivos/iniciar", strings.NewReader(`{"age":"25","sex":"female","type":"interactive","pregnant":false}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["thread_id"] == nil {
		t.Fatalf("missing thread_id")
	}
	data, _ := body["data"].(map[string]any)
	if data == nil { t.Fatalf("missing data map") }
	if _, ok := data["evaluation"].(map[string]any); !ok { t.Fatalf("missing evaluation object on start") }
}

func TestMessage_OK(t *testing.T) {
	r := setup()
	// First start to create thread and initial question
	ws := httptest.NewRecorder()
	reqStart := httptest.NewRequest(http.MethodPost, "/casos-interactivos/iniciar", strings.NewReader(`{"age":"25","sex":"f","type":"interactive","pregnant":false}`))
	reqStart.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ws, reqStart)
	if ws.Code != 200 { t.Fatalf("start expected 200 got %d", ws.Code) }
	var start map[string]any; _ = json.Unmarshal(ws.Body.Bytes(), &start)
	threadID, _ := start["thread_id"].(string)
	if threadID == "" { t.Fatalf("missing thread id") }
	// Now answer with Op3 to trigger local eval (correct_index=2 from mockAI)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/casos-interactivos/mensaje", strings.NewReader(`{"thread_id":"`+threadID+`","mensaje":"Op3"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("expected 200, got %d", w.Code) }
	var body map[string]any; _ = json.Unmarshal(w.Body.Bytes(), &body)
	data, ok := body["data"].(map[string]any); if !ok { t.Fatalf("missing data") }
	fb, _ := data["feedback"].(string)
	if !strings.Contains(fb, "Evaluación: CORRECTO") {
		t.Fatalf("expected evaluation line with CORRECTO, got: %s", fb)
	}
	if _, ok := data["evaluation"].(map[string]any); !ok { t.Fatalf("missing evaluation object on message") }
}

// Additional mock to simulate multiple sequential messages and closing behavior
type countingAI struct{ calls int }

func (m *countingAI) CreateThread(ctx context.Context) (string, error) { return "thread-limit", nil }
func (m *countingAI) StreamAssistantJSON(ctx context.Context, threadID, userPrompt, jsonInstructions string) (<-chan string, error) {
	ch := make(chan string, 1)
	// Always return a valid turn; finish always 0 to let handler enforce limit
	ch <- `{"feedback":"Turn","next":{"hallazgos":{},"pregunta":{"tipo":"single-choice","texto":"Q?","opciones":["A","B","C","D"]}},"finish":0}`
	return ch, nil
}

// TestInteractionLimit ahora valida cierre DIFERIDO: cuando se alcanza el máximo de preguntas todavía finish=0
// y sólo al enviar un mensaje adicional (sin generar nueva pregunta) se obtiene finish=1 con pregunta vacía.
func TestInteractionLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	h := &Handler{ai: &countingAI{}, maxQuestions: 3, turnCount: make(map[string]int), threadMaxQuestions: make(map[string]int), askedQuestions: make(map[string][]string)}
	h.RegisterRoutes(r)

	// Start case (consume 1)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/casos-interactivos/iniciar", strings.NewReader(`{"age":"30","sex":"male","type":"interactive","pregnant":false,"max_interactions":3}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("start expected 200") }
	var start map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &start)
	threadID, _ := start["thread_id"].(string)
	if threadID == "" { t.Fatalf("missing thread id") }

	send := func() map[string]any {
		w := httptest.NewRecorder()
		body := `{"thread_id":"` + threadID + `","mensaje":"A"}`
		req := httptest.NewRequest(http.MethodPost, "/casos-interactivos/mensaje", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != 200 { t.Fatalf("message expected 200") }
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		return resp
	}
	// 2nd question (count=2) aún abierto
	resp1 := send(); data1 := resp1["data"].(map[string]any)
	if finish, _ := data1["finish"].(float64); finish != 0 { t.Fatalf("expected finish=0 after second question") }
	// 3ra generación (count=3 == max) NO debe cerrar todavía: finish=0 y aún muestra pregunta (ya no se usará)
	resp2 := send(); data2 := resp2["data"].(map[string]any)
	if finish, _ := data2["finish"].(float64); finish != 0 { t.Fatalf("expected finish=0 when reaching max (deferred closure)") }
	// Enviar un turno extra para forzar cierre final (sin nueva pregunta)
	respClose := send(); dataClose := respClose["data"].(map[string]any)
	if finish, _ := dataClose["finish"].(float64); finish != 1 { t.Fatalf("expected finish=1 on extra closure turn") }
	next := dataClose["next"].(map[string]any); pregunta := next["pregunta"].(map[string]any)
	if len(pregunta) != 3 { t.Fatalf("expected pregunta object with keys even if empty") }
	if texto, _ := pregunta["texto"].(string); texto != "" { t.Fatalf("expected empty texto in final pregunta") }
}
// (removed stray closing brace)

// TestForceCloseMax2 (cierre diferido): con maxQuestions=2 el primer mensaje tras inicio NO cierra aún (finish=0),
// y el segundo mensaje produce el cierre (finish=1).
func TestForceCloseMax2(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	h := &Handler{ai: &countingAI{}, maxQuestions: 2, turnCount: make(map[string]int), threadMaxQuestions: make(map[string]int), askedQuestions: make(map[string][]string)}
	h.RegisterRoutes(r)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/casos-interactivos/iniciar", strings.NewReader(`{"age":"25","sex":"f","type":"interactive","pregnant":false,"max_interactions":2}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("start expected 200") }
	var start map[string]any; _ = json.Unmarshal(w.Body.Bytes(), &start)
	threadID := start["thread_id"].(string)
	if threadID == "" { t.Fatalf("missing thread id") }

	// Primer mensaje tras inicio: alcanza max pero aún finish=0 (cierre diferido)
	w2 := httptest.NewRecorder()
	body := `{"thread_id":"` + threadID + `","mensaje":"Respuesta"}`
	req2 := httptest.NewRequest(http.MethodPost, "/casos-interactivos/mensaje", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 { t.Fatalf("message expected 200") }
	var resp map[string]any; _ = json.Unmarshal(w2.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if finish, _ := data["finish"].(float64); finish != 0 { t.Fatalf("expected finish=0 (deferred) with max=2 on first message") }
	// Segundo mensaje provoca cierre
	w3 := httptest.NewRecorder()
	body2 := `{"thread_id":"` + threadID + `","mensaje":"Extra"}`
	req3 := httptest.NewRequest(http.MethodPost, "/casos-interactivos/mensaje", strings.NewReader(body2))
	req3.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w3, req3)
	if w3.Code != 200 { t.Fatalf("second message expected 200") }
	var resp2 map[string]any; _ = json.Unmarshal(w3.Body.Bytes(), &resp2)
	data2 := resp2["data"].(map[string]any)
	if finish, _ := data2["finish"].(float64); finish != 1 { t.Fatalf("expected finish=1 on second message (closure)") }
}

// TestImmediateClosureMax1 valida que si max=1 ya se considera cerrado tras la primera pregunta inicial
func TestImmediateClosureMax1(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	h := &Handler{ai: &countingAI{}, maxQuestions: 1, turnCount: make(map[string]int), threadMaxQuestions: make(map[string]int), askedQuestions: make(map[string][]string)}
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/casos-interactivos/iniciar", strings.NewReader(`{"age":"20","sex":"m","type":"interactive","pregnant":false,"max_interactions":1}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("start expected 200") }
	var start map[string]any; _ = json.Unmarshal(w.Body.Bytes(), &start)
	threadID := start["thread_id"].(string)
	if threadID == "" { t.Fatalf("missing thread id") }
	// Enviar mensaje debería retornar finish=1 inmediatamente
	w2 := httptest.NewRecorder()
	body := `{"thread_id":"` + threadID + `","mensaje":"A"}`
	req2 := httptest.NewRequest(http.MethodPost, "/casos-interactivos/mensaje", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 { t.Fatalf("message expected 200") }
	var resp map[string]any; _ = json.Unmarshal(w2.Body.Bytes(), &resp)
	if finish, _ := resp["data"].(map[string]any)["finish"].(float64); finish != 1 { t.Fatalf("expected finish=1 for max=1 case") }
}

// TestScoreOverride ensures that any existing Puntaje lines provided by the model
// are removed and replaced by the authoritative computation when forcing finish.
func TestScoreOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{ai: &countingAI{}, maxQuestions: 2, turnCount: make(map[string]int), threadMaxQuestions: make(map[string]int), askedQuestions: make(map[string][]string), evalCorrect: make(map[string]int), evalAnswers: make(map[string]int)}
	// Simulate one correct and one incorrect answer tracked internally
	threadID := "thread-score"
	h.evalCorrect[threadID] = 1
	h.evalAnswers[threadID] = 2
	data := map[string]any{
		"feedback": "Resumen Final:\nPuntaje: 99/99 (999%)\nReferencias: X",
		"next": map[string]any{"hallazgos": map[string]any{}, "pregunta": map[string]any{"tipo":"","texto":"","opciones":[]string{}}},
		"finish": float64(0),
	}
	forceFinishInteractive(data, threadID, h)
	fb, _ := data["feedback"].(string)
	if !strings.Contains(fb, "Puntaje: 1/2") {
		t.Fatalf("expected computed score 1/2 in feedback, got: %s", fb)
	}
	if strings.Contains(fb, "99/99") {
		t.Fatalf("expected old model-provided score to be removed, got: %s", fb)
	}
}

// TestFinalEvaluationFormat ensures the new structured final summary fields exist.
func TestFinalEvaluationFormat(t *testing.T) {
	h := &Handler{ai: &countingAI{}, maxQuestions: 1, turnCount: make(map[string]int), threadMaxQuestions: make(map[string]int), askedQuestions: make(map[string][]string), evalCorrect: make(map[string]int), evalAnswers: make(map[string]int), lastCorrectIndex: make(map[string]int), lastOptions: make(map[string][]string)}
	threadID := "thread-final"
	h.evalCorrect[threadID] = 0
	h.evalAnswers[threadID] = 0
	data := map[string]any{
		"feedback": "Evaluación: CORRECTO\nTexto explicación previa",
		"next": map[string]any{"hallazgos": map[string]any{}, "pregunta": map[string]any{"tipo":"","texto":"","opciones":[]string{}}},
		"finish": float64(0),
	}
	forceFinishInteractive(data, threadID, h)
	fb, _ := data["feedback"].(string)
	if !strings.Contains(fb, "Resumen Final:") { t.Fatalf("missing 'Resumen Final:' line") }
	if !strings.Contains(fb, "Puntaje:") { t.Fatalf("missing 'Puntaje:' line") }
	fe, ok := data["final_evaluation"].(map[string]any); if !ok { t.Fatalf("missing final_evaluation object") }
	if _, ok := fe["score_correct"]; !ok { t.Fatalf("missing score_correct") }
	if _, ok := fe["tier"]; !ok { t.Fatalf("missing tier") }
	if _, ok := fe["strengths"]; !ok { t.Fatalf("missing strengths") }
	if _, ok := fe["improvements"]; !ok { t.Fatalf("missing improvements") }
	if _, ok := fe["summary"]; !ok { t.Fatalf("missing summary") }
}

// TestEvaluationPending verifica que cuando falta correct_index se marca pending y se intenta recuperación
func TestEvaluationPendingAndRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	h := &Handler{ai: &mockMissingCI{}, maxQuestions: 3, turnCount: make(map[string]int), threadMaxQuestions: make(map[string]int), askedQuestions: make(map[string][]string), evalCorrect: make(map[string]int), evalAnswers: make(map[string]int), lastCorrectIndex: make(map[string]int), lastOptions: make(map[string][]string), lastQuestionText: make(map[string]string), missingCorrectIdx: make(map[string]int)}
	h.RegisterRoutes(r)
	// start
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/casos-interactivos/iniciar", strings.NewReader(`{"age":"22","sex":"f","type":"interactive","pregnant":false}`))
	req.Header.Set("Content-Type","application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("start expected 200") }
	var start map[string]any; _ = json.Unmarshal(w.Body.Bytes(), &start)
	threadID, _ := start["thread_id"].(string)
	if threadID == "" { t.Fatalf("missing thread id") }
	// message: respond to question (which lacks correct_index initially)
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/casos-interactivos/mensaje", strings.NewReader(`{"thread_id":"`+threadID+`","mensaje":"O2"}`))
	req2.Header.Set("Content-Type","application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 { t.Fatalf("message expected 200") }
	var resp map[string]any; _ = json.Unmarshal(w2.Body.Bytes(), &resp)
	data, _ := resp["data"].(map[string]any)
	if data == nil { t.Fatalf("missing data") }
	// evaluation object debe existir
	evalObj, _ := data["evaluation"].(map[string]any)
	if evalObj == nil { t.Fatalf("missing evaluation object") }
	// como se recuperó el índice, is_correct debería ser boolean (true/false) y no pending
	if _, hasPending := evalObj["pending"]; hasPending { t.Fatalf("pending should be cleared after recovery") }
	if _, ok := evalObj["is_correct"].(bool); !ok { t.Fatalf("expected boolean is_correct after recovery") }
	// métrica de missing_correct_index_events debe ser >=1 al cierre
	// forzamos cierre manual para inspeccionar final_evaluation
	dataForce := map[string]any{"feedback":"","next": map[string]any{"hallazgos": map[string]any{}, "pregunta": map[string]any{"tipo":"","texto":"","opciones":[]string{}}}, "finish": float64(0)}
	forceFinishInteractive(dataForce, threadID, h)
	fe := dataForce["final_evaluation"].(map[string]any)
	if v, _ := fe["missing_correct_index_events"].(int); v < 1 { t.Fatalf("expected missing_correct_index_events >=1, got %v", fe["missing_correct_index_events"]) }
}
