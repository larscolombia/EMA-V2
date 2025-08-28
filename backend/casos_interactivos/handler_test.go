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
	// Return a minimal valid interactive turn
	ch <- `{"feedback":"Correcto.","next":{"hallazgos":{},"pregunta":{"tipo":"single-choice","texto":"¿Paso?","opciones":["A","B","C","D"]}},"finish":0}`
	return ch, nil
}

func setup() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	h := &Handler{ai: &mockAI{}}
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
}

func TestMessage_OK(t *testing.T) {
	r := setup()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/casos-interactivos/mensaje", strings.NewReader(`{"thread_id":"t","mensaje":"Mi respuesta"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if _, ok := body["data"].(map[string]any); !ok {
		t.Fatalf("missing data")
	}
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

// TestInteractionLimit verifica cierre en el mismo turno que alcanza el máximo (force close post-increment)
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
	// 3rd generación debería cerrar inmediatamente (count pasa a 3 == max)
	resp2 := send(); data2 := resp2["data"].(map[string]any)
	if finish, _ := data2["finish"].(float64); finish != 1 { t.Fatalf("expected finish=1 when reaching max") }
	next := data2["next"].(map[string]any); pregunta := next["pregunta"].(map[string]any)
	// Esperamos claves presentes pero vacías
	if texto, _ := pregunta["texto"].(string); texto != "" { t.Fatalf("expected empty texto in final pregunta") }
	if tipo, _ := pregunta["tipo"].(string); tipo != "" { t.Fatalf("expected empty tipo in final pregunta") }
	if opts, ok := pregunta["opciones"].([]any); ok && len(opts) != 0 { t.Fatalf("expected empty opciones slice") }
}

// TestForceCloseMax2 asegura que con maxQuestions=2 se cierra en el primer mensaje después del inicio
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

	// Primer mensaje tras inicio debe forzar cierre (count pasa de 1 a 2 == max)
	w2 := httptest.NewRecorder()
	body := `{"thread_id":"` + threadID + `","mensaje":"Respuesta"}`
	req2 := httptest.NewRequest(http.MethodPost, "/casos-interactivos/mensaje", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 { t.Fatalf("message expected 200") }
	var resp map[string]any; _ = json.Unmarshal(w2.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if finish, _ := data["finish"].(float64); finish != 1 { t.Fatalf("expected finish=1 with max=2 on first message") }
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
