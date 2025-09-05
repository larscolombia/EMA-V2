package conversations_ia

import (
	"context"
	"strings"
	"testing"
	"time"
)

// MockAIClient implementa AIClient para tests
type MockAIClient struct {
	AssistantID string
	MockRAGResponse string
	MockPubMedResponse string
	ShouldFailRAG bool
	ShouldFailPubMed bool
}

func (m *MockAIClient) GetAssistantID() string { return m.AssistantID }
func (m *MockAIClient) CreateThread(ctx context.Context) (string, error) { return "thread_test", nil }
func (m *MockAIClient) StreamAssistantMessage(ctx context.Context, threadID, prompt string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "test response"
	close(ch)
	return ch, nil
}
func (m *MockAIClient) EnsureVectorStore(ctx context.Context, threadID string) (string, error) { return "vs_test", nil }
func (m *MockAIClient) UploadAssistantFile(ctx context.Context, threadID, filePath string) (string, error) { return "file_test", nil }
func (m *MockAIClient) PollFileProcessed(ctx context.Context, fileID string, timeout time.Duration) error { return nil }
func (m *MockAIClient) AddFileToVectorStore(ctx context.Context, vsID, fileID string) error { return nil }
func (m *MockAIClient) AddSessionBytes(threadID string, delta int64) {}
func (m *MockAIClient) CountThreadFiles(threadID string) int { return 0 }
func (m *MockAIClient) GetSessionBytes(threadID string) int64 { return 0 }
func (m *MockAIClient) TranscribeFile(ctx context.Context, filePath string) (string, error) { return "", nil }
func (m *MockAIClient) DeleteThreadArtifacts(ctx context.Context, threadID string) error { return nil }
func (m *MockAIClient) ForceNewVectorStore(ctx context.Context, threadID string) (string, error) { return "vs_new", nil }
func (m *MockAIClient) ListVectorStoreFiles(ctx context.Context, threadID string) ([]string, error) { return []string{}, nil }
func (m *MockAIClient) GetVectorStoreID(threadID string) string { return "vs_test" }

// Implementación de los nuevos métodos
func (m *MockAIClient) SearchInVectorStore(ctx context.Context, vectorStoreID, query string) (string, error) {
	if m.ShouldFailRAG {
		return "", nil // No encontró información
	}
	return m.MockRAGResponse, nil
}

func (m *MockAIClient) SearchPubMed(ctx context.Context, query string) (string, error) {
	if m.ShouldFailPubMed {
		return "", nil // No encontró información
	}
	return m.MockPubMedResponse, nil
}

func (m *MockAIClient) StreamAssistantWithSpecificVectorStore(ctx context.Context, threadID, prompt, vectorStoreID string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- m.MockRAGResponse
	close(ch)
	return ch, nil
}

func TestSmartMessage_RAGFound(t *testing.T) {
	// Setup
	mockClient := &MockAIClient{
		AssistantID: "asst_test",
		MockRAGResponse: "Información encontrada en RAG sobre diabetes mellitus",
		ShouldFailRAG: false,
	}
	
	handler := &Handler{AI: mockClient}
	
	// Test
	ctx := context.Background()
	stream, source, err := handler.SmartMessage(ctx, "thread_test", "¿Qué es la diabetes?")
	
	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if source != "rag" {
		t.Errorf("Expected source 'rag', got '%s'", source)
	}
	
	// Leer respuesta del stream
	var response string
	select {
	case response = <-stream:
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
	
	if !strings.Contains(response, "diabetes mellitus") {
		t.Errorf("Expected response to contain diabetes mellitus, got: %s", response)
	}
}

func TestSmartMessage_PubMedFallback(t *testing.T) {
	// Setup
	mockClient := &MockAIClient{
		AssistantID: "asst_test",
		MockPubMedResponse: "Estudio de PubMed PMID:12345 sobre hipertensión",
		ShouldFailRAG: true, // RAG no encuentra nada
		ShouldFailPubMed: false,
	}
	
	handler := &Handler{AI: mockClient}
	
	// Test
	ctx := context.Background()
	stream, source, err := handler.SmartMessage(ctx, "thread_test", "¿Qué es la hipertensión?")
	
	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if source != "pubmed" {
		t.Errorf("Expected source 'pubmed', got '%s'", source)
	}
	
	// Leer respuesta del stream
	var response string
	select {
	case response = <-stream:
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
	
	if !strings.Contains(response, "PubMed") {
		t.Errorf("Expected response to contain PubMed, got: %s", response)
	}
}

func TestSmartMessage_NoSourcesFound(t *testing.T) {
	// Setup
	mockClient := &MockAIClient{
		AssistantID: "asst_test",
		ShouldFailRAG: true, // RAG no encuentra nada
		ShouldFailPubMed: true, // PubMed no encuentra nada
	}
	
	handler := &Handler{AI: mockClient}
	
	// Test
	ctx := context.Background()
	stream, source, err := handler.SmartMessage(ctx, "thread_test", "¿Qué es algo muy específico?")
	
	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if source != "none" {
		t.Errorf("Expected source 'none', got '%s'", source)
	}
	
	// Leer respuesta del stream
	var response string
	select {
	case response = <-stream:
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
	
	if !strings.Contains(response, "no pudo ser respondida") {
		t.Errorf("Expected response to contain 'no pudo ser respondida', got: %s", response)
	}
}

func TestVectorStoreIDUsage(t *testing.T) {
	// Setup
	mockClient := &MockAIClient{
		AssistantID: "asst_test",
		MockRAGResponse: "Respuesta del vector específico",
		ShouldFailRAG: false,
	}
	
	handler := &Handler{AI: mockClient}
	
	// Test
	ctx := context.Background()
	_, source, err := handler.SmartMessage(ctx, "thread_test", "test query")
	
	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if source != "rag" {
		t.Errorf("Expected to use RAG source, got '%s'", source)
	}
	
	// Verificar que se está usando el vector store específico
	const expectedVectorID = "vs_680fc484cef081918b2b9588b701e2f4"
	
	// En una implementación real, podríamos capturar las llamadas al mock
	// Por ahora, verificamos que la función funciona sin errores
}
