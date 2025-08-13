package testsapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockAssistant struct{ forEval bool }

func (m *mockAssistant) CreateThread(ctx context.Context) (string, error) { return "thread_test", nil }
func (m *mockAssistant) StreamAssistantMessage(ctx context.Context, threadID, prompt string) (<-chan string, error) {
	ch := make(chan string, 1)
	if m.forEval {
		ch <- `{"evaluation":[{"question_id":1,"is_correct":1,"fit":"Bien"}],"correct_answers":1,"fit_global":"Correcto"}`
	} else {
		// Minimal valid JSON expected by generate handler
		ch <- `{"questions":[{"id":1,"question":"Q?","answer":"A","type":"single_choice","options":["A","B","C"],"category":null}]}`
	}
	close(ch)
	return ch, nil
}

func (m *mockAssistant) StreamAssistantJSON(ctx context.Context, threadID, userPrompt, jsonInstructions string) (<-chan string, error) {
	// For generate: return minimal valid JSON
	ch := make(chan string, 1)
	if m.forEval {
		ch <- `{"evaluation":[{"question_id":1,"is_correct":1,"fit":"Bien"}],"correct_answers":1,"fit_global":"Correcto"}`
	} else {
		ch <- `{"questions":[{"id":1,"question":"Q?","answer":"A","type":"single_choice","options":["A","B","C"],"category":null}]}`
	}
	close(ch)
	return ch, nil
}

func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r)
	return r
}

func TestGenerate_ok(t *testing.T) {
	h := NewHandler(&mockAssistant{}, "asst_mock")
	r := setupRouter(h)

	body := map[string]any{"id_categoria": []int{1}, "num_questions": 3, "nivel": "basico"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/tests/generate/1", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if !resp.Success {
		t.Fatalf("success=false")
	}
	if resp.Data["thread_id"] == "" {
		t.Fatalf("missing thread_id")
	}
	if _, ok := resp.Data["questions"].([]any); !ok {
		t.Fatalf("missing questions array")
	}
}

func TestEvaluate_ok(t *testing.T) {
	h := NewHandler(&mockAssistant{forEval: true}, "asst_mock")
	r := setupRouter(h)

	body := map[string]any{
		"uid":       "quiz-1",
		"thread":    "thread_test",
		"userId":    1,
		"test_id":   123,
		"questions": []map[string]any{{"question_id": 1, "answer": "A", "type": "single_choice"}},
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/tests/responder-test/submit", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
