package testsapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func setupRealRouter(t *testing.T) *gin.Engine {
	t.Helper()
	_ = godotenv.Load(".env")
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set; skipping real assistant integration test")
	}
	if os.Getenv("CUESTIONARIOS_MEDICOS_GENERALES") == "" {
		t.Skip("CUESTIONARIOS_MEDICOS_GENERALES not set; skipping real assistant integration test")
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := DefaultHandler()
	h.RegisterRoutes(r)
	return r
}

func TestGenerate_Integration(t *testing.T) {
	r := setupRealRouter(t)
	body := map[string]any{"id_categoria": []int{1}, "num_questions": 3, "nivel": "basico"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/tests/generate/1", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("/tests/generate expected 200, got %d: %s", w.Code, w.Body.String())
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
	if _, ok := resp.Data["questions"].([]any); !ok {
		t.Fatalf("missing questions array in response: %v", resp.Data)
	}
}

func TestEvaluate_Integration(t *testing.T) {
	r := setupRealRouter(t)
	body := map[string]any{
		"uid":       "quiz-it",
		"thread":    "",
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
		t.Fatalf("/tests/responder-test/submit expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Basic shape check
	var parsed map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if _, ok := parsed["evaluation"]; !ok {
		t.Logf("warning: response lacks 'evaluation' key: %v", parsed)
	}
}
