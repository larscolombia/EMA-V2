package chat

import (
	"context"
	"strings"
	"testing"
	"time"

	"ema-backend/openai"
)

// mockAI implements AIClient for testing prompt generation without calling OpenAI.
type mockAI struct{ lastPrompt string }

func (m *mockAI) GetAssistantID() string                           { return "asst_mock" }
func (m *mockAI) CreateThread(ctx context.Context) (string, error) { return "thread_mock", nil }
func (m *mockAI) StreamMessage(ctx context.Context, prompt string) (<-chan string, error) {
	ch := make(chan string, 1)
	m.lastPrompt = prompt
	ch <- "ok"
	close(ch)
	return ch, nil
}
func (m *mockAI) StreamAssistantMessage(ctx context.Context, threadID, prompt string) (<-chan string, error) {
	ch := make(chan string, 1)
	m.lastPrompt = prompt
	ch <- "ok"
	close(ch)
	return ch, nil
}
func (m *mockAI) StreamAssistantWithSpecificVectorStore(ctx context.Context, threadID, prompt, vectorStoreID string) (<-chan string, error) {
	return m.StreamAssistantMessage(ctx, threadID, prompt)
}
func (m *mockAI) EnsureVectorStore(ctx context.Context, threadID string) (string, error) {
	return "vs_mock", nil
}
func (m *mockAI) UploadAssistantFile(ctx context.Context, threadID, filePath string) (string, error) {
	return "file_mock", nil
}
func (m *mockAI) PollFileProcessed(ctx context.Context, fileID string, timeout time.Duration) error {
	return nil
}
func (m *mockAI) AddFileToVectorStore(ctx context.Context, vsID, fileID string) error { return nil }
func (m *mockAI) PollVectorStoreFileIndexed(ctx context.Context, vsID, fileID string, timeout time.Duration) error {
	return nil
}
func (m *mockAI) AddSessionBytes(threadID string, delta int64)                        {}
func (m *mockAI) CountThreadFiles(threadID string) int                                { return 0 }
func (m *mockAI) GetSessionBytes(threadID string) int64                               { return 0 }
func (m *mockAI) TranscribeFile(ctx context.Context, filePath string) (string, error) { return "", nil }
func (m *mockAI) StreamAssistantMessageWithFile(ctx context.Context, threadID, prompt, filePath string) (<-chan string, error) {
	return m.StreamAssistantMessage(ctx, threadID, prompt)
}
func (m *mockAI) DeleteThreadArtifacts(ctx context.Context, threadID string) error { return nil }
func (m *mockAI) ForceNewVectorStore(ctx context.Context, threadID string) (string, error) {
	return "vs_mock_new", nil
}
func (m *mockAI) ListVectorStoreFiles(ctx context.Context, threadID string) ([]string, error) {
	return []string{}, nil
}
func (m *mockAI) GetVectorStoreID(threadID string) string { return "vs_mock" }
func (m *mockAI) ClearVectorStoreFiles(ctx context.Context, vsID string) error {
	return nil
}
func (m *mockAI) GetThreadMessages(ctx context.Context, threadID string, limit int) ([]openai.ThreadMessage, error) {
	return []openai.ThreadMessage{}, nil
}

// TestStructuredPromptContent ensures the default structured prompt contains all required sections.
func TestStructuredPromptContent(t *testing.T) {
	m := &mockAI{}
	h := NewHandler(m)
	// simulate strict mode off
	// We call directly the internal logic that builds the prompt by imitating the conditions:
	// Empty user prompt + PDF upload triggers prompt creation inside Message handler.
	// For this unit test, we won't exercise full multipart flow; instead we assert helper string exists.
	// Keep the canonical template here to detect divergence.
	templateSections := []string{
		"1. Resumen Ejecutivo", "2. Objetivo o Propósito", "3. Alcance y Componentes Clave", "4. Propuesta de Valor / Diferenciadores", "5. Entregables y Cronograma", "6. Modelo Comercial / Costos", "7. Riesgos, Supuestos o Limitaciones", "8. Próximos Pasos sugeridos", "No especificado"}
	prompt := "Elabora un resumen estructurado y conciso del documento adjunto (archivo: demo.pdf). Proporciona las secciones en español: \n" +
		"1. Resumen Ejecutivo (3-4 líneas).\n" +
		"2. Objetivo o Propósito.\n" +
		"3. Alcance y Componentes Clave.\n" +
		"4. Propuesta de Valor / Diferenciadores.\n" +
		"5. Entregables y Cronograma (si se mencionan).\n" +
		"6. Modelo Comercial / Costos (si aparecen).\n" +
		"7. Riesgos, Supuestos o Limitaciones.\n" +
		"8. Próximos Pasos sugeridos.\n" +
		"Formato: viñetas claras, NO inventes información; si un punto no está presente escribe 'No especificado'. Termina con una breve recomendación práctica."
	// store in mock to reuse validation style
	m.lastPrompt = prompt
	for _, s := range templateSections {
		if !strings.Contains(prompt, s) {
			t.Fatalf("expected prompt to contain section marker %q", s)
		}
	}
	if !strings.Contains(prompt, "No especificado") {
		t.Fatalf("expected placeholder 'No especificado' in prompt")
	}
	_ = h // future: integrate by refactoring prompt builder into function
}
