package openai_test

import (
	"bytes"
	"context"
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

// TestStreamMessage_RealEnv exercises the OpenAI client with real .env values.
// It passes if:
//   - OPENAI_API_KEY and CHAT_PRINCIPAL_ASSISTANT are set and we receive some non-empty output, or
//   - they are not set and the client returns a benign empty string via the placeholder channel.
func TestStreamMessage_RealEnv(t *testing.T) {
	// Load .env from common locations
	tried := []string{
		".env",
		// When running from module root
		"backend/.env",
		// When running from package directory (backend/openai)
		"../.env",
		// CI or different layouts
		"../backend/.env",
		"../../.env",
		"../../backend/.env",
	}
	loaded := false
	for _, p := range tried {
		if err := godotenv.Load(p); err == nil {
			t.Logf("loaded env from %s", p)
			loaded = true
			break
		}
	}
	if !loaded {
		t.Log("no .env file loaded; relying on process env")
	}

	key := os.Getenv("OPENAI_API_KEY")
	asst := os.Getenv("CHAT_PRINCIPAL_ASSISTANT")
	// CHAT_MODEL is optional

	c := openaipkg.NewClient()
	ctx := context.Background()

	question := "hola puedes responderme?"
	start := time.Now()
	ch, err := c.StreamMessage(ctx, question)
	if err != nil {
		t.Fatalf("StreamMessage returned error: %v", err)
	}

	// Collect up to first 10 chunks or until channel closes
	got := ""
	count := 0
	for tok := range ch {
		got += tok
		count++
		if count >= 10 || len(got) > 0 {
			break
		}
	}

	elapsed := time.Since(start)
	t.Logf("time to first token/request completion: %s", elapsed)

	if key == "" || asst == "" {
		// No credentials configured: we expect the tolerant empty output path
		if got != "" {
			t.Fatalf("expected empty output when API not configured; got: %q", got)
		}
		t.Log("no OPENAI_API_KEY/CHAT_PRINCIPAL_ASSISTANT configured; placeholder behavior validated")
		return
	}

	if len(got) == 0 {
		t.Fatalf("expected some output from OpenAI, got empty string (question=%s)", question)
	}
	t.Logf("assistant replied: %q", got)
	// Optional visibility in CI logs
	fmt.Println("assistant replied:", got)
}

// TestChatHandlerMultipart simulates a multipart upload with optional PDF/audio and prompt.
// If Assistants is configured, it first creates a real thread and sends the message with thread_id
// so that file attachments can be associated with the thread.
func TestChatHandlerMultipart(t *testing.T) {
	// Load env
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("backend/.env")
	_ = godotenv.Load("../backend/.env")

	ai := openaipkg.NewClient()
	h := chatpkg.NewHandler(ai)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/asistente/message", h.Message)
	r.POST("/asistente/start", h.Start)

	// Build multipart body
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	// Attach a PDF or an audio file if one exists; otherwise send without file
	audioCandidates := []string{
		"testdata/sample.mp3", "testdata/sample.m4a", "../testdata/sample.mp3",
	}
	pdfCandidates := []string{
		"testdata/sample.pdf",
		"../testdata/sample.pdf",
		"temporal/doc.pdf",
		"backend/temporal/doc.pdf",
		"../temporal/doc.pdf",
		"../backend/temporal/doc.pdf",
	}
	var audioFound string
	var pdfFound string
	for _, p := range audioCandidates {
		if _, err := os.Stat(p); err == nil {
			audioFound = p
			break
		}
	}
	if audioFound == "" {
		for _, p := range pdfCandidates {
			if _, err := os.Stat(p); err == nil {
				pdfFound = p
				break
			}
		}
	}
	if audioFound != "" {
		fw, err := w.CreateFormFile("file", filepath.Base(audioFound))
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		f, err := os.Open(audioFound)
		if err != nil {
			t.Fatalf("open audio: %v", err)
		}
		defer f.Close()
		if _, err := io.Copy(fw, f); err != nil {
			t.Fatalf("copy audio: %v", err)
		}
	} else if pdfFound != "" {
		fw, err := w.CreateFormFile("file", filepath.Base(pdfFound))
		if err != nil {
			t.Fatalf("CreateFormFile pdf: %v", err)
		}
		f, err := os.Open(pdfFound)
		if err != nil {
			t.Fatalf("open pdf: %v", err)
		}
		defer f.Close()
		if _, err := io.Copy(fw, f); err != nil {
			t.Fatalf("copy pdf: %v", err)
		}
	} else {
		// No fixture found: create a tiny temporary PDF so we always cover the file-upload path
		tmp, err := os.CreateTemp("", "sample-*.pdf")
		if err != nil {
			t.Fatalf("CreateTemp pdf: %v", err)
		}
		// Minimal PDF header/footer; server only checks extension and size > 0
		_, _ = tmp.WriteString("%PDF-1.4\n%EOF\n")
		_ = tmp.Close()
		defer os.Remove(tmp.Name())

		fw, err := w.CreateFormFile("file", filepath.Base(tmp.Name()))
		if err != nil {
			t.Fatalf("CreateFormFile tmp pdf: %v", err)
		}
		f, err := os.Open(tmp.Name())
		if err != nil {
			t.Fatalf("open tmp pdf: %v", err)
		}
		defer f.Close()
		if _, err := io.Copy(fw, f); err != nil {
			t.Fatalf("copy tmp pdf: %v", err)
		}
	}
	// Add prompt
	_ = w.WriteField("prompt", "Este es un mensaje de prueba para el asistente. Si hay audio, úsalo como contexto.")
	threadID := "test-thread"
	// If Assistants is configured, create a real thread via the handler
	if os.Getenv("OPENAI_API_KEY") != "" && strings.HasPrefix(os.Getenv("CHAT_PRINCIPAL_ASSISTANT"), "asst_") {
		reqStart := httptest.NewRequest(http.MethodPost, "/asistente/start", strings.NewReader(`{"prompt":""}`))
		reqStart.Header.Set("Content-Type", "application/json")
		recStart := httptest.NewRecorder()
		r.ServeHTTP(recStart, reqStart)
		if recStart.Code != http.StatusOK {
			t.Fatalf("start thread failed: code=%d body=%s", recStart.Code, recStart.Body.String())
		}
		bodyStart := recStart.Body.String()
		// very small parse: find thread_id field
		if idx := strings.Index(bodyStart, "\"thread_id\":"); idx >= 0 {
			// naive parse
			rest := bodyStart[idx+len("\"thread_id\":"):]
			q1 := strings.Index(rest, "\"")
			q2 := strings.Index(rest[q1+1:], "\"")
			if q1 >= 0 && q2 > 0 {
				threadID = rest[q1+1 : q1+1+q2]
			}
		}
	}
	_ = w.WriteField("thread_id", threadID)
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/asistente/message", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()

	// Execute request
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}
	// Body should contain SSE lines and [DONE]
	out := rec.Body.String()
	t.Logf("SSE response:\n%s", out)
	if !strings.Contains(out, "[DONE]") {
		t.Fatalf("expected SSE to end with [DONE], got: %q", out)
	}
}
