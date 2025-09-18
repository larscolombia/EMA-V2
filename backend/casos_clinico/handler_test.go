package casos_clinico

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ema-backend/openai"

	"github.com/gin-gonic/gin"
)

type mockAI struct{}

func (m *mockAI) CreateThread(ctx context.Context) (string, error) { return "thread_x", nil }
func (m *mockAI) StreamAssistantMessage(ctx context.Context, threadID, prompt string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "{\"respuesta\":{\"text\":\"Hola\"}}"
	close(ch)
	return ch, nil
}
func (m *mockAI) StreamAssistantJSON(ctx context.Context, threadID, userPrompt, jsonInstructions string) (<-chan string, error) {
	ch := make(chan string, 1)
	if strings.Contains(userPrompt, "interactivo") || strings.Contains(jsonInstructions, "interactive") {
		ch <- "{\"case\":{\"id\":1,\"title\":\"Titulo\",\"type\":\"interactive\",\"age\":\"25\",\"sex\":\"M\",\"gestante\":0,\"is_real\":1,\"anamnesis\":\"...\",\"physical_examination\":\"...\",\"diagnostic_tests\":\"...\",\"final_diagnosis\":\"...\",\"management\":\"...\"},\"data\":{\"questions\":{\"texto\":\"Q?\",\"tipo\":\"open_ended\",\"opciones\":[]}}}"
	} else if strings.Contains(jsonInstructions, "'case' debe contener") {
		ch <- "{\"case\":{\"id\":1,\"title\":\"Titulo\",\"type\":\"static\",\"age\":\"25\",\"sex\":\"M\",\"gestante\":0,\"is_real\":1,\"anamnesis\":\"...\",\"physical_examination\":\"...\",\"diagnostic_tests\":\"...\",\"final_diagnosis\":\"...\",\"management\":\"...\"}}"
	} else {
		ch <- "{\"respuesta\":{\"text\":\"ok\"}}"
	}
	close(ch)
	return ch, nil
}

// New methods to satisfy extended Assistant interface
func (m *mockAI) SearchInVectorStore(ctx context.Context, vectorStoreID, query string) (string, error) {
	return "", nil
}
func (m *mockAI) SearchInVectorStoreWithMetadata(ctx context.Context, vectorStoreID, query string) (*openai.VectorSearchResult, error) {
	return &openai.VectorSearchResult{HasResult: false}, nil
}
func (m *mockAI) SearchPubMed(ctx context.Context, query string) (string, error) {
	return "", nil
}

func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r)
	return r
}

func TestGenerateAnalytical_OK(t *testing.T) {
	h := NewHandler(&mockAI{}, &mockAI{})
	r := setupRouter(h)

	body := map[string]any{"age": "30", "sex": "F", "type": "static", "pregnant": false}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/caso-clinico", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGenerateInteractive_OK(t *testing.T) {
	h := NewHandler(&mockAI{}, &mockAI{})
	r := setupRouter(h)

	body := map[string]any{"age": "30", "sex": "M", "type": "interactive", "pregnant": false}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/casos-clinicos/interactivo", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChatAnalytical_OK(t *testing.T) {
	h := NewHandler(&mockAI{}, &mockAI{})
	r := setupRouter(h)

	body := map[string]any{"thread_id": "", "mensaje": "Hola"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/casos-clinicos/conversar", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChatInteractive_OK(t *testing.T) {
	h := NewHandler(&mockAI{}, &mockAI{})
	r := setupRouter(h)

	body := map[string]any{"thread_id": "", "mensaje": "Siguiente"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/casos-clinicos/interactivo/conversar", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
