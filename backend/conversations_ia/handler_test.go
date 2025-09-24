package conversations_ia

import (
	"context"
	"strings"
	"testing"
	"time"
)

// MockAIClient implementa AIClient para tests
type MockAIClient struct {
	AssistantID        string
	MockRAGResponse    string
	MockPubMedResponse string
	MockSource         string
	ShouldFailRAG      bool
	ShouldFailPubMed   bool
	ThreadHasFiles     bool     // Simula si el hilo tiene archivos
	VectorStoreFiles   []string // Lista de archivos en el vector store
}

func (m *MockAIClient) GetAssistantID() string                           { return m.AssistantID }
func (m *MockAIClient) CreateThread(ctx context.Context) (string, error) { return "thread_test", nil }
func (m *MockAIClient) StreamAssistantMessage(ctx context.Context, threadID, prompt string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "test response"
	close(ch)
	return ch, nil
}
func (m *MockAIClient) EnsureVectorStore(ctx context.Context, threadID string) (string, error) {
	return "vs_test", nil
}
func (m *MockAIClient) UploadAssistantFile(ctx context.Context, threadID, filePath string) (string, error) {
	return "file_test", nil
}
func (m *MockAIClient) PollFileProcessed(ctx context.Context, fileID string, timeout time.Duration) error {
	return nil
}
func (m *MockAIClient) AddFileToVectorStore(ctx context.Context, vsID, fileID string) error {
	return nil
}
func (m *MockAIClient) AddSessionBytes(threadID string, delta int64) {}
func (m *MockAIClient) CountThreadFiles(threadID string) int         { return 0 }
func (m *MockAIClient) GetSessionBytes(threadID string) int64        { return 0 }
func (m *MockAIClient) TranscribeFile(ctx context.Context, filePath string) (string, error) {
	return "", nil
}
func (m *MockAIClient) DeleteThreadArtifacts(ctx context.Context, threadID string) error { return nil }
func (m *MockAIClient) ForceNewVectorStore(ctx context.Context, threadID string) (string, error) {
	return "vs_new", nil
}
func (m *MockAIClient) ListVectorStoreFiles(ctx context.Context, threadID string) ([]string, error) {
	if m.ThreadHasFiles && len(m.VectorStoreFiles) > 0 {
		return m.VectorStoreFiles, nil
	}
	return []string{}, nil
}
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
	ch <- prompt // Devuelve el prompt para verificar en tests
	close(ch)
	return ch, nil
}

// MockAIClientWithMetadata implementa AIClientWithMetadata para tests avanzados
type MockAIClientWithMetadata struct {
	*MockAIClient
}

func (m *MockAIClientWithMetadata) SearchInVectorStoreWithMetadata(ctx context.Context, vectorStoreID, query string) (*VectorSearchResult, error) {
	if m.ShouldFailRAG {
		return &VectorSearchResult{
			Content:   "",
			Source:    "",
			VectorID:  vectorStoreID,
			HasResult: false,
		}, nil
	}

	source := m.MockSource
	if source == "" {
		source = "Manual de Medicina Interna Harrison"
	}

	return &VectorSearchResult{
		Content:   m.MockRAGResponse,
		Source:    source,
		VectorID:  vectorStoreID,
		HasResult: true,
	}, nil
}

// Tests para el flujo SmartMessage

func TestSmartMessage_RAGResults(t *testing.T) {
	// Test con resultados del RAG (vector store)
	mockClient := &MockAIClient{
		AssistantID:     "asst_test",
		MockRAGResponse: "La diabetes mellitus es una enfermedad metabólica caracterizada por...",
		MockSource:      "Harrison's Principles of Internal Medicine",
	}

	handler := &Handler{AI: mockClient}

	stream, source, err := handler.SmartMessage(context.Background(), "thread_test", "¿Qué es la diabetes?", "")

	if err != nil {
		t.Errorf("No se esperaba error: %v", err)
	}

	if source != "rag" {
		t.Errorf("Se esperaba source 'rag', se obtuvo: %s", source)
	}

	// Verificar que el stream contiene la información esperada
	response := <-stream
	if !strings.Contains(response, "Referencias:") {
		t.Errorf("La respuesta debe incluir sección de Referencias")
	}
}

func TestSmartMessage_NoRAGResultsPubMedFallback(t *testing.T) {
	// Test sin resultados del RAG, con fallback a PubMed
	mockClient := &MockAIClient{
		AssistantID:        "asst_test",
		MockPubMedResponse: "Estudio de PubMed sobre diabetes...",
		ShouldFailRAG:      true,
	}

	handler := &Handler{AI: mockClient}

	stream, source, err := handler.SmartMessage(context.Background(), "thread_test", "¿Qué es la diabetes?", "")

	if err != nil {
		t.Errorf("No se esperaba error: %v", err)
	}

	if source != "pubmed" {
		t.Errorf("Se esperaba source 'pubmed', se obtuvo: %s", source)
	}

	// Verificar que el stream contiene la información de PubMed
	response := <-stream
	if !strings.Contains(response, "PubMed") {
		t.Errorf("La respuesta debe indicar que proviene de PubMed")
	}
}

func TestSmartMessage_NoResultsAnywhere(t *testing.T) {
	// Test sin resultados en ninguna fuente
	mockClient := &MockAIClient{
		AssistantID:      "asst_test",
		ShouldFailRAG:    true,
		ShouldFailPubMed: true,
	}

	handler := &Handler{AI: mockClient}

	stream, source, err := handler.SmartMessage(context.Background(), "thread_test", "término inexistente xyz123", "")

	if err != nil {
		t.Errorf("No se esperaba error: %v", err)
	}

	if source != "none" {
		t.Errorf("Se esperaba source 'none', se obtuvo: %s", source)
	}

	// Verificar que el stream contiene mensaje de no encontrado
	response := <-stream
	if !strings.Contains(response, "No se encontró información relevante") {
		t.Errorf("La respuesta debe indicar que no se encontró información")
	}
}

func TestSmartMessage_BackwardCompatibility(t *testing.T) {
	// Test de compatibilidad con clientes que no soportan metadatos
	mockClient := &MockAIClient{
		AssistantID:     "asst_test",
		MockRAGResponse: "Información médica encontrada...",
	}

	handler := &Handler{AI: mockClient}

	stream, source, err := handler.SmartMessage(context.Background(), "thread_test", "pregunta médica", "")

	if err != nil {
		t.Errorf("No se esperaba error: %v", err)
	}

	if source != "rag" {
		t.Errorf("Se esperaba source 'rag', se obtuvo: %s", source)
	}

	// Verificar que el stream contiene información
	response := <-stream
	if len(response) == 0 {
		t.Errorf("La respuesta no debe estar vacía")
	}
}

func TestSmartMessage_ThreadWithDocsGeneralQuestion(t *testing.T) {
	// Test del caso del usuario: hilo con documentos pero pregunta general (no debe usar doc-only)
	mockClient := &MockAIClient{
		AssistantID:      "asst_test",
		MockRAGResponse:  "La gastritis es una inflamación de la mucosa gástrica...",
		MockSource:       "Harrison's Principles of Internal Medicine",
		ThreadHasFiles:   true,                    // Simula que el hilo tiene documentos
		VectorStoreFiles: []string{"file_abc123"}, // Simula archivo existente
	}

	handler := &Handler{AI: mockClient}

	// Pregunta general que NO menciona documentos - debe usar flujo híbrido, no doc-only
	stream, source, err := handler.SmartMessage(context.Background(), "thread_uVISzHAqHBpSDuR8L79OeuDr", "Que es la gastritis?", "")

	if err != nil {
		t.Errorf("No se esperaba error: %v", err)
	}

	// DEBE usar flujo híbrido (rag/pubmed), NO doc_only
	if source == "doc_only" {
		t.Errorf("No debería usar doc_only para pregunta general, se obtuvo: %s", source)
	}

	if source != "rag" && source != "pubmed" {
		t.Errorf("Se esperaba source 'rag' o 'pubmed' para pregunta general, se obtuvo: %s", source)
	}

	// Verificar que el stream contiene información del conocimiento general
	response := <-stream
	if !strings.Contains(response, "gastritis") {
		t.Errorf("La respuesta debe contener información sobre gastritis")
	}
}

func TestSmartMessage_ThreadWithDocsDocumentQuestion(t *testing.T) {
	// Test: hilo con documentos Y pregunta que menciona documentos - SÍ debe usar doc-only
	mockClient := &MockAIClient{
		AssistantID:      "asst_test",
		MockRAGResponse:  "Los documentos no contienen información para responder esta pregunta.",
		ThreadHasFiles:   true,
		VectorStoreFiles: []string{"file_abc123"},
	}

	handler := &Handler{AI: mockClient}

	// Pregunta que SÍ menciona documentos - debe usar doc-only
	stream, source, err := handler.SmartMessage(context.Background(), "thread_uVISzHAqHBpSDuR8L79OeuDr", "que dice el documento sobre gastritis?", "")

	if err != nil {
		t.Errorf("No se esperaba error: %v", err)
	}

	// DEBE usar doc_only cuando se menciona explícitamente el documento
	if source != "doc_only" {
		t.Errorf("Se esperaba source 'doc_only' cuando se pregunta sobre documento, se obtuvo: %s", source)
	}

	response := <-stream
	if !strings.Contains(response, "documento") {
		t.Errorf("La respuesta debe mencionar los documentos")
	}
}
