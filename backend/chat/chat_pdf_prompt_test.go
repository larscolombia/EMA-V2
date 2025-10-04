package chat

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ema-backend/openai"

	"github.com/gin-gonic/gin"
)

// mockAI minimal para capturar prompt usado tras procesamiento de archivo
type mockAIPrompt struct {
	AssistantID string
	lastPrompt  string
}

func (m *mockAIPrompt) GetAssistantID() string                           { return m.AssistantID }
func (m *mockAIPrompt) CreateThread(ctx context.Context) (string, error) { return "thread_mock", nil }
func (m *mockAIPrompt) StreamMessage(ctx context.Context, prompt string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- prompt
	close(ch)
	m.lastPrompt = prompt
	return ch, nil
}
func (m *mockAIPrompt) StreamAssistantMessage(ctx context.Context, threadID, prompt string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- prompt
	close(ch)
	m.lastPrompt = prompt
	return ch, nil
}
func (m *mockAIPrompt) StreamAssistantWithSpecificVectorStore(ctx context.Context, threadID, prompt, vectorStoreID string) (<-chan string, error) {
	return m.StreamAssistantMessage(ctx, threadID, prompt)
}
func (m *mockAIPrompt) EnsureVectorStore(ctx context.Context, threadID string) (string, error) {
	return "vs1", nil
}
func (m *mockAIPrompt) UploadAssistantFile(ctx context.Context, threadID, filePath string) (string, error) {
	return "file1", nil
}
func (m *mockAIPrompt) PollFileProcessed(ctx context.Context, fileID string, timeout time.Duration) error {
	return nil
}
func (m *mockAIPrompt) AddFileToVectorStore(ctx context.Context, vsID, fileID string) error {
	return nil
}
func (m *mockAIPrompt) PollVectorStoreFileIndexed(ctx context.Context, vsID, fileID string, timeout time.Duration) error {
	return nil
}
func (m *mockAIPrompt) AddSessionBytes(threadID string, delta int64) {}
func (m *mockAIPrompt) CountThreadFiles(threadID string) int         { return 0 }
func (m *mockAIPrompt) GetSessionBytes(threadID string) int64        { return 0 }
func (m *mockAIPrompt) TranscribeFile(ctx context.Context, filePath string) (string, error) {
	return "", nil
}
func (m *mockAIPrompt) StreamAssistantMessageWithFile(ctx context.Context, threadID, prompt, filePath string) (<-chan string, error) {
	return m.StreamAssistantMessage(ctx, threadID, prompt)
}
func (m *mockAIPrompt) DeleteThreadArtifacts(ctx context.Context, threadID string) error { return nil }
func (m *mockAIPrompt) ForceNewVectorStore(ctx context.Context, threadID string) (string, error) {
	return "vs1_new", nil
}
func (m *mockAIPrompt) ListVectorStoreFiles(ctx context.Context, threadID string) ([]string, error) {
	return []string{}, nil
}
func (m *mockAIPrompt) GetVectorStoreID(threadID string) string { return "vs1" }
func (m *mockAIPrompt) GetThreadMessages(ctx context.Context, threadID string, limit int) ([]openai.ThreadMessage, error) {
	return []openai.ThreadMessage{}, nil
}

// Test que verifica selección de prompt según STRUCTURED_PDF_SUMMARY
func TestPDFStructuredPromptToggle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pdfPath := filepath.Join("files", "Propuesta Comerccial LARS - Inforcid.pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		t.Skip("PDF no disponible en entorno CI")
	}
	mk := &mockAIPrompt{AssistantID: "asst_dummy"}
	h := NewHandler(mk)

	// helper para ejecutar upload
	run := func() string {
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		fw, _ := w.CreateFormFile("file", filepath.Base(pdfPath))
		data, _ := os.ReadFile(pdfPath)
		fw.Write(data[:min(2048, len(data))]) // no necesitamos todo el archivo para testear prompt
		w.WriteField("thread_id", "thread_local")
		w.Close()
		req := httptest.NewRequest("POST", "/asistente/message", body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rr := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rr)
		c.Request = req
		h.Message(c)
		return mk.lastPrompt
	}

	os.Setenv("STRUCTURED_PDF_SUMMARY", "1")
	p1 := run()
	if !strings.Contains(p1, "1. Resumen Ejecutivo") {
		t.Errorf("se esperaba prompt estructurado, got: %s", p1)
	}
	os.Setenv("STRUCTURED_PDF_SUMMARY", "")
	p2 := run()
	if strings.Contains(p2, "Recomendación Breve") && strings.Contains(p2, "1. Resumen Ejecutivo") {
		t.Errorf("se esperaba prompt genérico sin secciones numeradas fijas, got: %s", p2)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
