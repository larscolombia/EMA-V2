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
