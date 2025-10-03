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

	// Si el prompt es para modo básico (contiene "FORMATO DE RESPUESTA OBLIGATORIO"), devolver formato estructurado
	if strings.Contains(prompt, "FORMATO DE RESPUESTA OBLIGATORIO") {
		response := `Las bradiarritmias constituyen un grupo heterogéneo de trastornos del ritmo cardíaco caracterizados por una frecuencia cardíaca inferior a 60 latidos por minuto. Estos trastornos pueden originarse por alteraciones en la formación del impulso eléctrico en el nodo sinusal, defectos en la conducción auriculoventricular, o una combinación de ambos mecanismos fisiopatológicos.

La fisiopatología de las bradiarritmias involucra disfunción del sistema de conducción eléctrica cardíaca, que puede ser congénita o adquirida. Las manifestaciones clínicas varían desde pacientes completamente asintomáticos hasta presentaciones con fatiga, mareos, síncope, e incluso insuficiencia cardíaca en casos severos. El diagnóstico se basa principalmente en el electrocardiograma y la correlación clínica, mientras que el tratamiento puede requerir desde observación hasta implante de marcapasos permanente según la severidad y repercusión hemodinámica.

## Fuentes:
- Conocimiento médico especializado integrado
- Literatura médica estándar en medicina interna y especialidades`
		ch <- response
	} else {
		ch <- "test response"
	}

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
	mockClient := &MockAIClientWithMetadata{
		MockAIClient: &MockAIClient{
			AssistantID:     "asst_test",
			MockRAGResponse: "La diabetes mellitus es una enfermedad metabólica caracterizada por...",
			MockSource:      "Harrison's Principles of Internal Medicine",
		},
	}

	handler := NewHandler(mockClient)
	snap := handler.snapshotTopic("thread_test")
	resp, err := handler.SmartMessage(context.Background(), "thread_test", "¿Qué es la diabetes?", "", snap)

	if err != nil {
		t.Errorf("No se esperaba error: %v", err)
	}

	if resp.Source != "rag" {
		t.Errorf("Se esperaba source 'rag', se obtuvo: %s", resp.Source)
	}

	response := <-resp.Stream
	if !strings.Contains(response, "## Fuentes:") {
		t.Errorf("La respuesta debe incluir sección '## Fuentes:'")
	}
	if strings.Contains(response, "- Base de conocimiento médico") {
		t.Errorf("No debe usarse 'Base de conocimiento médico' como fuente genérica en el contexto recuperado")
	}
	if !strings.Contains(response, "Harrison's Principles of Internal Medicine") {
		t.Errorf("El contexto debe incluir el título real del documento")
	}
	if len(resp.AllowedSources) == 0 {
		t.Errorf("Se esperaban fuentes permitidas para validación")
	}
}

func TestSmartMessage_NoRAGResultsPubMedFallback(t *testing.T) {
	// Test sin resultados del RAG, con fallback a PubMed
	mockClient := &MockAIClient{
		AssistantID:        "asst_test",
		MockPubMedResponse: `{"summary":"Hallazgos recientes sobre diabetes tipo 2","studies":[{"title":"Management of type 2 diabetes","pmid":"12345678","year":2022,"journal":"Cardiology Review","key_points":["Optimización del control glucémico reduce eventos cardiovasculares","Enfoques combinados de farmacoterapia mejoran resultados"]}]}`,
		ShouldFailRAG:      true,
	}

	handler := NewHandler(mockClient)
	snap := handler.snapshotTopic("thread_test")
	resp, err := handler.SmartMessage(context.Background(), "thread_test", "¿Qué es la diabetes?", "", snap)

	if err != nil {
		t.Errorf("No se esperaba error: %v", err)
	}

	if resp.Source != "pubmed" {
		t.Errorf("Se esperaba source 'pubmed', se obtuvo: %s", resp.Source)
	}

	response := <-resp.Stream
	if !strings.Contains(response, "PubMed") {
		t.Errorf("La respuesta debe indicar que proviene de PubMed")
	}
	if len(resp.PubMedReferences) == 0 {
		t.Errorf("Se esperaban referencias de PubMed en la respuesta")
	}
}

func TestSmartMessage_NoResultsAnywhere(t *testing.T) {
	// Test sin resultados en ninguna fuente
	mockClient := &MockAIClient{
		AssistantID:      "asst_test",
		ShouldFailRAG:    true,
		ShouldFailPubMed: true,
	}

	handler := NewHandler(mockClient)
	snap := handler.snapshotTopic("thread_test")
	resp, err := handler.SmartMessage(context.Background(), "thread_test", "término inexistente xyz123", "", snap)

	if err != nil {
		t.Errorf("No se esperaba error: %v", err)
	}

	if resp.Source != "no_source" {
		t.Errorf("Se esperaba source 'no_source', se obtuvo: %s", resp.Source)
	}

	response := <-resp.Stream
	if !strings.Contains(response, "No encontré una referencia") {
		t.Errorf("El mensaje debe informar la ausencia de referencias, got: %s", response)
	}
	if resp.FallbackReason != "no_results" {
		t.Errorf("Se esperaba fallback_reason 'no_results', got=%s", resp.FallbackReason)
	}
}

func TestSmartMessage_BackwardCompatibility(t *testing.T) {
	// Test de compatibilidad con clientes que no soportan metadatos
	mockClient := &MockAIClient{
		AssistantID:     "asst_test",
		MockRAGResponse: "Información médica encontrada...",
	}

	handler := NewHandler(mockClient)
	snap := handler.snapshotTopic("thread_test")
	resp, err := handler.SmartMessage(context.Background(), "thread_test", "pregunta médica", "", snap)

	if err != nil {
		t.Errorf("No se esperaba error: %v", err)
	}

	if resp.Source != "rag" {
		t.Errorf("Se esperaba source 'rag', se obtuvo: %s", resp.Source)
	}

	response := <-resp.Stream
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

	handler := NewHandler(mockClient)

	// Pregunta general que NO menciona documentos - debe usar flujo híbrido, no doc-only
	snap := handler.snapshotTopic("thread_uVISzHAqHBpSDuR8L79OeuDr")
	resp, err := handler.SmartMessage(context.Background(), "thread_uVISzHAqHBpSDuR8L79OeuDr", "Que es la gastritis?", "", snap)

	if err != nil {
		t.Errorf("No se esperaba error: %v", err)
	}

	// DEBE usar flujo híbrido (rag/pubmed), NO doc_only
	if resp.Source == "doc_only" {
		t.Errorf("No debería usar doc_only para pregunta general, se obtuvo: %s", resp.Source)
	}

	if resp.Source != "rag" && resp.Source != "pubmed" {
		t.Errorf("Se esperaba source 'rag' o 'pubmed' para pregunta general, se obtuvo: %s", resp.Source)
	}

	response := <-resp.Stream
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

	handler := NewHandler(mockClient)

	// Pregunta que SÍ menciona documentos - debe usar doc-only
	snap := handler.snapshotTopic("thread_uVISzHAqHBpSDuR8L79OeuDr")
	resp, err := handler.SmartMessage(context.Background(), "thread_uVISzHAqHBpSDuR8L79OeuDr", "que dice el documento sobre gastritis?", "", snap)

	if err != nil {
		t.Errorf("No se esperaba error: %v", err)
	}

	// DEBE usar doc_only cuando se menciona explícitamente el documento
	if resp.Source != "doc_only" {
		t.Errorf("Se esperaba source 'doc_only' cuando se pregunta sobre documento, se obtuvo: %s", resp.Source)
	}

	response := <-resp.Stream
	if !strings.Contains(response, "documento") {
		t.Errorf("La respuesta debe mencionar los documentos")
	}
}

func TestSmartMessage_BasicModeWithSources(t *testing.T) {
	// Test: modo básico debe incluir sección de fuentes
	mockClient := &MockAIClient{
		AssistantID:      "asst_test",
		ShouldFailRAG:    true, // Falla búsqueda en vector
		ShouldFailPubMed: true, // Falla búsqueda en PubMed
	}

	handler := NewHandler(mockClient)
	snap := handler.snapshotTopic("thread_test")
	resp, err := handler.SmartMessage(context.Background(), "thread_test", "Que es un tumor de frantz?", "", snap)

	if err != nil {
		t.Errorf("No se esperaba error: %v", err)
	}

	if resp.Source != "no_source" {
		t.Errorf("Se esperaba source 'no_source' cuando fallan las búsquedas, se obtuvo: %s", resp.Source)
	}

	response := <-resp.Stream
	if !strings.Contains(response, "No encontré una referencia") {
		t.Errorf("La respuesta debe comunicar falta de referencias, got=%s", response)
	}
}

func TestSmartMessage_HybridMode(t *testing.T) {
	// Test: modo híbrido cuando hay resultados de vector Y PubMed
	mockClient := &MockAIClientWithMetadata{
		MockAIClient: &MockAIClient{
			AssistantID:        "asst_test",
			MockRAGResponse:    "La insuficiencia cardíaca con fracción de eyección reducida (IC-FEr) se caracteriza por...",
			MockSource:         "Braunwald's Heart Disease",
			MockPubMedResponse: `{"summary":"Nuevas terapias para IC-FEr han mostrado beneficio clínico significativo","studies":[{"title":"SGLT2 inhibitors in heart failure","pmid":"34567890","year":2022,"journal":"NEJM","key_points":["Reducción del 25% en hospitalizaciones cardiovasculares","Mejora en clase funcional NYHA"]}]}`,
		},
	}

	handler := NewHandler(mockClient)
	snap := handler.snapshotTopic("thread_test")
	resp, err := handler.SmartMessage(context.Background(), "thread_test", "¿Qué tratamientos existen para insuficiencia cardíaca?", "", snap)

	if err != nil {
		t.Errorf("No se esperaba error: %v", err)
	}

	// En modo híbrido, debe tener source="hybrid"
	if resp.Source != "hybrid" {
		t.Errorf("Se esperaba source 'hybrid' cuando hay vector Y PubMed, se obtuvo: %s", resp.Source)
	}

	if !resp.HasVectorContext {
		t.Errorf("En modo híbrido debe tener HasVectorContext=true")
	}

	if !resp.HasPubMedContext {
		t.Errorf("En modo híbrido debe tener HasPubMedContext=true")
	}

	if len(resp.AllowedSources) == 0 {
		t.Errorf("Se esperaban fuentes permitidas del vector store")
	}

	if len(resp.PubMedReferences) == 0 {
		t.Errorf("Se esperaban referencias de PubMed")
	}

	response := <-resp.Stream
	if !strings.Contains(response, "## Fuentes:") {
		t.Errorf("La respuesta debe incluir sección '## Fuentes:'")
	}

	// Verificar que menciona integración híbrida en el prompt
	// (esto lo verificamos indirectamente viendo que tiene ambos contextos)
	if len(response) < 100 {
		t.Errorf("La respuesta híbrida debe ser sustancial, got=%d chars", len(response))
	}
}
