package casos_interactivos

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// init attempts to load .env manually so OPENAI_API_KEY & assistant IDs are available when running `go test`.
func init() {
	if os.Getenv("OPENAI_API_KEY") != "" { // already set
		return
	}
	// Look for .env in current working directory or parent
	candidates := []string{".env", filepath.Join("..", ".env")}
	for _, p := range candidates {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if i := strings.Index(line, "="); i > 0 {
				key := strings.TrimSpace(line[:i])
				val := strings.TrimSpace(line[i+1:])
				val = strings.Trim(val, `"`) // remove surrounding quotes
				if os.Getenv(key) == "" {
					_ = os.Setenv(key, val)
				}
			}
		}
		_ = f.Close()
		break
	}
}

// This test hits the real OpenAI assistant (through ema-backend/openai) if OPENAI_API_KEY is present.
// It executes 3 full interactive runs from scratch and ensures that exactly max_interactions
// non-empty questions are produced before the handler returns a closure turn (finish=1 with empty pregunta).
// Skipped automatically when the key isn't set to avoid CI/network usage.
func TestRealInteractiveCase_MultipleRuns(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set; skipping real integration test")
	}
	gin.SetMode(gin.TestMode)
	const maxInteractions = 5
	for run := 1; run <= 3; run++ {
		r := gin.Default()
		h := DefaultHandler()            // respects env vars
		h.maxQuestions = maxInteractions // ensure default (overridden by request body too)
		h.RegisterRoutes(r)

		// --- StartCase --- //
		wStart := httptest.NewRecorder()
		startBody := `{"age":"35","sex":"female","type":"interactive","pregnant":false,"max_interactions":5}`
		reqStart := httptest.NewRequest(http.MethodPost, "/casos-interactivos/iniciar", strings.NewReader(startBody))
		reqStart.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(wStart, reqStart)
		if wStart.Code != 200 {
			t.Fatalf("run %d: start status %d", run, wStart.Code)
		}
		var startResp map[string]any
		_ = json.Unmarshal(wStart.Body.Bytes(), &startResp)
		threadID, _ := startResp["thread_id"].(string)
		if threadID == "" {
			t.Fatalf("run %d: missing thread_id", run)
		}
		data, _ := startResp["data"].(map[string]any)
		next, _ := data["next"].(map[string]any)
		pregunta, _ := next["pregunta"].(map[string]any)
		qText, _ := pregunta["texto"].(string)
		questions := 0
		if strings.TrimSpace(qText) != "" {
			questions++
		}

		// Loop answering until closure
		attempts := 0
		for {
			attempts++
			if attempts > maxInteractions+3 { // safety to avoid infinite loops
				t.Fatalf("run %d: exceeded attempts without closure", run)
			}
			// If we've already reached the target number of questions, send one more answer to trigger closure turn
			if questions >= maxInteractions {
				wClose := httptest.NewRecorder()
				msgBody := `{"thread_id":"` + threadID + `","mensaje":"Respuesta final"}`
				reqMsg := httptest.NewRequest(http.MethodPost, "/casos-interactivos/mensaje", strings.NewReader(msgBody))
				reqMsg.Header.Set("Content-Type", "application/json")
				r.ServeHTTP(wClose, reqMsg)
				if wClose.Code != 200 {
					t.Fatalf("run %d: closure status %d", run, wClose.Code)
				}
				var closeResp map[string]any
				_ = json.Unmarshal(wClose.Body.Bytes(), &closeResp)
				cData, _ := closeResp["data"].(map[string]any)
				finish, _ := cData["finish"].(float64)
				n, _ := cData["next"].(map[string]any)
				pq, _ := n["pregunta"].(map[string]any)
				if finish != 1 {
					t.Fatalf("run %d: expected finish=1 after triggering closure, got %v", run, finish)
				}
				if len(pq) != 0 {
					t.Fatalf("run %d: expected empty pregunta on closure, got %#v", run, pq)
				}
				break
			}
			// Send answer to get next question
			wMsg := httptest.NewRecorder()
			msgBody := `{"thread_id":"` + threadID + `","mensaje":"Respuesta"}`
			reqMsg := httptest.NewRequest(http.MethodPost, "/casos-interactivos/mensaje", strings.NewReader(msgBody))
			reqMsg.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(wMsg, reqMsg)
			if wMsg.Code != 200 {
				t.Fatalf("run %d: message status %d", run, wMsg.Code)
			}
			var msgResp map[string]any
			_ = json.Unmarshal(wMsg.Body.Bytes(), &msgResp)
			mData, _ := msgResp["data"].(map[string]any)
			finish, _ := mData["finish"].(float64)
			n, _ := mData["next"].(map[string]any)
			pq, _ := n["pregunta"].(map[string]any)
			qText, _ := pq["texto"].(string)
			if strings.TrimSpace(qText) != "" {
				questions++
			}
			if finish == 1 && questions < maxInteractions {
				t.Fatalf("run %d: assistant closed early at %d < %d questions", run, questions, maxInteractions)
			}
			time.Sleep(1500 * time.Millisecond)
		}
		if questions != maxInteractions {
			t.Fatalf("run %d: expected %d questions, got %d", run, maxInteractions, questions)
		}
	}
}
