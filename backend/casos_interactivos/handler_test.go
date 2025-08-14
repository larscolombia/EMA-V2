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

func TestInteractionLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	h := &Handler{ai: &countingAI{}, maxQuestions: 3, turnCount: make(map[string]int), threadMaxQuestions: make(map[string]int)}
	h.RegisterRoutes(r)

	// Start case
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/casos-interactivos/iniciar", strings.NewReader(`{"age":"30","sex":"male","type":"interactive","pregnant":false,"max_interactions":3}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("start expected 200")
	}
	var start map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &start)
	threadID, _ := start["thread_id"].(string)
	if threadID == "" {
		t.Fatalf("missing thread id")
	}

	// We configured maxQuestions=3. After 3 questions, next message must close.
	// We already consumed 1 question at start. Send 2 more normal messages, then a 3rd that should close immediately before generating another question.
	send := func() map[string]any {
		w := httptest.NewRecorder()
		body := `{"thread_id":"` + threadID + `","mensaje":"A"}`
		req := httptest.NewRequest(http.MethodPost, "/casos-interactivos/mensaje", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("message expected 200")
		}
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		return resp
	}
	resp1 := send()
	data1 := resp1["data"].(map[string]any)
	if finish, _ := data1["finish"].(float64); finish != 0 {
		t.Fatalf("unexpected finish after 2nd question")
	}
	resp2 := send()
	data2 := resp2["data"].(map[string]any)
	// After 3rd generated question the handler increments count and should close on next call
	if finish, _ := data2["finish"].(float64); finish != 0 {
		t.Fatalf("unexpected finish after 3rd question generation")
	}
	// This call should now return finish=1 with empty pregunta
	respClose := send()
	dataClose := respClose["data"].(map[string]any)
	if finish, _ := dataClose["finish"].(float64); finish != 1 {
		t.Fatalf("expected finish=1 on closure turn")
	}
	next := dataClose["next"].(map[string]any)
	pregunta := next["pregunta"].(map[string]any)
	if len(pregunta) != 0 {
		t.Fatalf("expected empty pregunta on closure turn")
	}
}
