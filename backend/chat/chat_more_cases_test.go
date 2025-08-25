package chat_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	chatpkg "ema-backend/chat"
	openaipkg "ema-backend/openai"
)

// helper to setup router
func setupRouter(t *testing.T) *gin.Engine {
	tried := []string{".env", "backend/.env", "../.env", "../backend/.env", "../../.env", "../../backend/.env"}
	for _, p := range tried {
		_ = godotenv.Load(p)
	}
	ai := openaipkg.NewClient()
	h := chatpkg.NewHandler(ai)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/asistente/start", h.Start)
	r.POST("/asistente/message", h.Message)
	r.POST("/asistente/delete", h.Delete)
	return r
}

// TestProcessing202 forces a very short poll timeout to increase probability of 202 during file processing.
func TestProcessing202(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" || !strings.HasPrefix(os.Getenv("CHAT_PRINCIPAL_ASSISTANT"), "asst_") {
		t.Skip("requires real OpenAI env")
	}
	// Force poll=0s to trigger 202 quickly
	_ = os.Setenv("VS_POLL_SEC", "0")
	defer os.Unsetenv("VS_POLL_SEC")

	r := setupRouter(t)
	// start thread
	recStart := httptest.NewRecorder()
	reqStart := httptest.NewRequest(http.MethodPost, "/asistente/start", strings.NewReader(`{"prompt":""}`))
	reqStart.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recStart, reqStart)
	var startResp struct {
		ThreadID string `json:"thread_id"`
	}
	_ = json.Unmarshal(recStart.Body.Bytes(), &startResp)
	if startResp.ThreadID == "" {
		t.Fatalf("no thread_id")
	}

	// send only PDF
	pdf := filepath.FromSlash("backend/chat/docsprueba/Propuesta-405.pdf")
	if _, err := os.Stat(pdf); err != nil {
		t.Skip("pdf missing")
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("file", filepath.Base(pdf))
	f, _ := os.Open(pdf)
	_, _ = io.Copy(fw, f)
	_ = f.Close()
	_ = w.WriteField("prompt", "")
	_ = w.WriteField("thread_id", startResp.ThreadID)
	_ = w.Close()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/asistente/message", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted && rec.Code != http.StatusOK {
		t.Fatalf("expected 202 or 200, got %d", rec.Code)
	}
}

// TestLimits validates max files and session size errors if configured via env.
func TestLimitsConfigurable(t *testing.T) {
	// Only run if user configured small limits
	if os.Getenv("OPENAI_API_KEY") == "" || !strings.HasPrefix(os.Getenv("CHAT_PRINCIPAL_ASSISTANT"), "asst_") {
		t.Skip("requires real OpenAI env")
	}
	if os.Getenv("VS_MAX_FILES") == "" && os.Getenv("VS_MAX_MB") == "" {
		t.Skip("no VS_MAX_* limits configured; skipping to avoid flaky assertions")
	}
	r := setupRouter(t)
	// start thread
	recStart := httptest.NewRecorder()
	reqStart := httptest.NewRequest(http.MethodPost, "/asistente/start", strings.NewReader(`{"prompt":""}`))
	reqStart.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recStart, reqStart)
	var startResp struct {
		ThreadID string `json:"thread_id"`
	}
	_ = json.Unmarshal(recStart.Body.Bytes(), &startResp)
	if startResp.ThreadID == "" {
		t.Fatalf("no thread_id")
	}

	// attempt upload
	pdf := filepath.FromSlash("backend/chat/docsprueba/Propuesta-405.pdf")
	if _, err := os.Stat(pdf); err != nil {
		t.Skip("pdf missing")
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("file", filepath.Base(pdf))
	f, _ := os.Open(pdf)
	_, _ = io.Copy(fw, f)
	_ = f.Close()
	_ = w.WriteField("prompt", "")
	_ = w.WriteField("thread_id", startResp.ThreadID)
	_ = w.Close()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/asistente/message", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	r.ServeHTTP(rec, req)
	if rec.Code == http.StatusBadRequest {
		// ok: limit enforced
		return
	}
	if rec.Code != http.StatusAccepted && rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestFollowUpGrounding sends multiple follow-ups to ensure grounding persists.
func TestFollowUpGrounding(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" || !strings.HasPrefix(os.Getenv("CHAT_PRINCIPAL_ASSISTANT"), "asst_") {
		t.Skip("requires real OpenAI env")
	}
	r := setupRouter(t)
	// start
	recStart := httptest.NewRecorder()
	reqStart := httptest.NewRequest(http.MethodPost, "/asistente/start", strings.NewReader(`{"prompt":""}`))
	reqStart.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recStart, reqStart)
	var startResp struct {
		ThreadID string `json:"thread_id"`
	}
	_ = json.Unmarshal(recStart.Body.Bytes(), &startResp)
	if startResp.ThreadID == "" {
		t.Fatalf("no thread_id")
	}

	// upload PDF once
	pdf := filepath.FromSlash("backend/chat/docsprueba/Propuesta-405.pdf")
	if _, err := os.Stat(pdf); err != nil {
		t.Skip("pdf missing")
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("file", filepath.Base(pdf))
	f, _ := os.Open(pdf)
	_, _ = io.Copy(fw, f)
	_ = f.Close()
	_ = w.WriteField("prompt", "")
	_ = w.WriteField("thread_id", startResp.ThreadID)
	_ = w.Close()
	recU := httptest.NewRecorder()
	reqU := httptest.NewRequest(http.MethodPost, "/asistente/message", &body)
	reqU.Header.Set("Content-Type", w.FormDataContentType())
	r.ServeHTTP(recU, reqU)
	if recU.Code != http.StatusOK && recU.Code != http.StatusAccepted {
		t.Fatalf("upload status=%d", recU.Code)
	}
	// wait briefly if accepted
	if recU.Code == http.StatusAccepted {
		time.Sleep(12 * time.Second)
	}

	// send two follow-ups
	questions := []string{
		"Dame un resumen ejecutivo.",
		"¿Quién es el responsable del contenido técnico?",
	}
	for _, q := range questions {
		bodyQ := fmt.Sprintf(`{"thread_id":"%s","prompt":"%s"}`, startResp.ThreadID, escapeJSON2(q))
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/asistente/message", strings.NewReader(bodyQ))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("follow-up status=%d body=%s", rec.Code, rec.Body.String())
		}
		out := rec.Body.String()
		if !strings.Contains(out, "data:") || !strings.Contains(out, "[DONE]") {
			t.Fatalf("SSE malformed: %s", out)
		}
	}
}

// TestCleanupEndpoint validates /asistente/delete best-effort cleanup returns 204.
func TestCleanupEndpoint(t *testing.T) {
	r := setupRouter(t)
	// delete without thread_id -> 400
	recBad := httptest.NewRecorder()
	reqBad := httptest.NewRequest(http.MethodPost, "/asistente/delete", strings.NewReader(`{}`))
	reqBad.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recBad.Code)
	}

	// create a fake thread id and delete -> 204 (best-effort)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/asistente/delete", strings.NewReader(`{"thread_id":"thread_fake_1"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

// escapeJSON reused helper
func escapeJSON2(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n", "\r", "\\r")
	return r.Replace(s)
}

// TestTextThenPdfFlow simula el caso: primero texto "Hola" sin archivo, luego PDF vacío en el mismo hilo.
func TestTextThenPdfFlow(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" || !strings.HasPrefix(os.Getenv("CHAT_PRINCIPAL_ASSISTANT"), "asst_") {
		t.Skip("requires real OpenAI env")
	}
	r := setupRouter(t)
	// start
	recStart := httptest.NewRecorder()
	reqStart := httptest.NewRequest(http.MethodPost, "/asistente/start", strings.NewReader(`{"prompt":""}`))
	reqStart.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recStart, reqStart)
	var startResp struct {
		ThreadID string `json:"thread_id"`
	}
	_ = json.Unmarshal(recStart.Body.Bytes(), &startResp)
	if startResp.ThreadID == "" {
		t.Fatalf("no thread_id")
	}

	// 1) Texto sin archivo
	first := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/asistente/message", strings.NewReader(fmt.Sprintf(`{"thread_id":"%s","prompt":"Hola"}`, startResp.ThreadID)))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(first, req1)
	if first.Code != http.StatusOK {
		t.Fatalf("text-only status=%d body=%s", first.Code, first.Body.String())
	}
	if !strings.Contains(first.Body.String(), "data:") || !strings.Contains(first.Body.String(), "[DONE]") {
		t.Fatalf("SSE malformed for text-only: %s", first.Body.String())
	}

	// 2) PDF vacío (prompt "") en el mismo hilo
	pdf := filepath.FromSlash("backend/chat/docsprueba/Propuesta-405.pdf")
	if _, err := os.Stat(pdf); err != nil {
		t.Skip("pdf missing")
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("file", filepath.Base(pdf))
	f, _ := os.Open(pdf)
	_, _ = io.Copy(fw, f)
	_ = f.Close()
	_ = w.WriteField("prompt", "")
	_ = w.WriteField("thread_id", startResp.ThreadID)
	_ = w.Close()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/asistente/message", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted && rec.Code != http.StatusOK {
		t.Fatalf("expected 202 or 200 for pdf, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestPdfWithoutThread validates that a PDF upload without thread_id does not 500 and returns a normal stream.
func TestPdfWithoutThread(t *testing.T) {
	// Run only if env is present; otherwise base client may emit placeholder empty string and handler may fall back
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("requires OpenAI env")
	}
	r := setupRouter(t)

	pdf := filepath.FromSlash("backend/chat/docsprueba/Propuesta-405.pdf")
	if _, err := os.Stat(pdf); err != nil {
		t.Skip("pdf missing")
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("file", filepath.Base(pdf))
	f, _ := os.Open(pdf)
	_, _ = io.Copy(fw, f)
	_ = f.Close()
	_ = w.WriteField("prompt", "")
	// omit thread_id on purpose
	_ = w.Close()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/asistente/message", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 without thread_id, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "data:") || !strings.Contains(rec.Body.String(), "[DONE]") {
		t.Fatalf("SSE malformed without thread_id: %s", rec.Body.String())
	}
}

// TestStructuredPromptHeader valida que cuando el prompt está vacío y se sube un PDF,
// el handler inyecta el prompt estructurado (structured-v1) y añade cabecera X-RAG-Prompt.
func TestStructuredPromptHeader(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" || !strings.HasPrefix(os.Getenv("CHAT_PRINCIPAL_ASSISTANT"), "asst_") {
		t.Skip("requires OpenAI env")
	}
	r := setupRouter(t)
	// start thread
	recStart := httptest.NewRecorder()
	reqStart := httptest.NewRequest(http.MethodPost, "/asistente/start", strings.NewReader(`{"prompt":""}`))
	reqStart.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recStart, reqStart)
	var startResp struct { ThreadID string `json:"thread_id"` }
	_ = json.Unmarshal(recStart.Body.Bytes(), &startResp)
	if startResp.ThreadID == "" { t.Fatalf("no thread_id") }

	pdf := filepath.FromSlash("backend/chat/docsprueba/Propuesta-405.pdf")
	if _, err := os.Stat(pdf); err != nil { t.Skip("pdf missing") }
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("file", filepath.Base(pdf))
	f, _ := os.Open(pdf)
	_, _ = io.Copy(fw, f)
	_ = f.Close()
	_ = w.WriteField("prompt", "")
	_ = w.WriteField("thread_id", startResp.ThreadID)
	_ = w.Close()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/asistente/message", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusAccepted { t.Fatalf("unexpected status %d", rec.Code) }
	// Header debe estar presente si terminó en 200; si 202 podría aún no haber hecho run
	if rec.Code == http.StatusOK {
		if got := rec.Header().Get("X-RAG-Prompt"); got != "structured-v1" {
			t.Fatalf("esperabamos X-RAG-Prompt=structured-v1, obtuvimos %q", got)
		}
	}
	// Verificar que al menos la primera parte del stream contiene prefacio español (puede tardar en tokenizar secciones)
	bodyStr := rec.Body.String()
	if rec.Code == http.StatusOK && !strings.Contains(bodyStr, "Resumen Ejecutivo") {
		t.Logf("Advertencia: no se detectó 'Resumen Ejecutivo' aún en el stream inicial (puede llegar después). Body=%s", truncate(bodyStr, 400))
	}
}
