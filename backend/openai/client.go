package openai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	openai "github.com/sashabaranov/go-openai"
	"rsc.io/pdf"
)

type Client struct {
	api         *openai.Client
	AssistantID string
	Model       string
	key         string
	httpClient  *http.Client
	// session vector stores: thread_id -> vector_store_id
	vsMu         sync.RWMutex
	vectorStore  map[string]string
	vsLastAccess map[string]time.Time // thread_id -> last access
	fileMu       sync.RWMutex
	fileCache    map[string]string // key: threadID+"|"+sha256 -> fileID
	// session usage tracking (in-memory)
	sessMu    sync.RWMutex
	sessBytes map[string]int64
	sessFiles map[string]int
	// last uploaded file per thread to bias instructions
	lastMu      sync.RWMutex
	lastFile    map[string]LastFileInfo
	lastCleanup time.Time
	vsTTL       time.Duration
	// Ensure *Client implements the chat.AIClient interface (compile-time check) via a blank identifier assignment.
	// (We inline the minimal subset because importing chat here would create a cycle; so we skip direct assertion.)
}

// GetAssistantID returns the configured Assistant ID (implements chat.AIClient).
func (c *Client) GetAssistantID() string { return c.AssistantID }

type LastFileInfo struct {
	ID       string
	Name     string
	At       time.Time
	Hash     string
	Metadata *PDFMetadata // Metadatos extraídos del PDF (nil si no es PDF o falla extracción)
}

// PDFMetadata contiene información bibliográfica extraída de un PDF
type PDFMetadata struct {
	Title               string  `json:"title,omitempty"`
	Author              string  `json:"author,omitempty"`
	Subject             string  `json:"subject,omitempty"`
	Keywords            string  `json:"keywords,omitempty"`
	Creator             string  `json:"creator,omitempty"`
	Producer            string  `json:"producer,omitempty"`
	Created             string  `json:"created,omitempty"`     // Fecha de creación
	Modified            string  `json:"modified,omitempty"`    // Fecha de modificación
	HasExtractableText  bool    `json:"has_extractable_text"`  // Si el PDF tiene texto extraíble (no es solo imagen)
	TextCoveragePercent float64 `json:"text_coverage_percent"` // Porcentaje estimado de contenido con texto
	PageCount           int     `json:"page_count,omitempty"`  // Número de páginas
}

// sanitizeEnv limpia espacios y elimina comillas simples o dobles rodeando todo el valor.
func sanitizeEnv(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 {
		if (strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"")) || (strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'")) {
			v = strings.TrimSuffix(strings.TrimPrefix(v, string(v[0])), string(v[len(v)-1]))
		}
	}
	return v
}

func NewClient() *Client {
	key := sanitizeEnv(os.Getenv("OPENAI_API_KEY"))
	assistant := sanitizeEnv(os.Getenv("CHAT_PRINCIPAL_ASSISTANT"))
	model := sanitizeEnv(os.Getenv("CHAT_MODEL"))
	var api *openai.Client
	if key != "" {
		api = openai.NewClient(key)
	}
	vsMap, _ := loadVectorStoreFile()
	fcMap, _ := loadFileCache()
	var ttl time.Duration
	if v := strings.TrimSpace(os.Getenv("VS_TTL_MINUTES")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ttl = time.Duration(n) * time.Minute
		}
	}
	return &Client{
		api:          api,
		AssistantID:  assistant,
		Model:        model,
		key:          key,
		httpClient:   &http.Client{Timeout: 180 * time.Second}, // Aumentado a 180s para runs largos y addMessage lentos
		vectorStore:  vsMap,
		vsLastAccess: make(map[string]time.Time),
		fileCache:    fcMap,
		sessBytes:    make(map[string]int64),
		sessFiles:    make(map[string]int),
		lastFile:     make(map[string]LastFileInfo),
		vsTTL:        ttl,
	}
}

func (c *Client) StreamMessage(ctx context.Context, prompt string) (<-chan string, error) {
	// Prefer Assistants API if an AssistantID is configured
	if c.key != "" && c.AssistantID != "" && len(c.AssistantID) >= 5 && c.AssistantID[:5] == "asst_" {
		// Create a transient thread then run once and emit a single chunk
		threadID, err := c.CreateThread(ctx)
		if err == nil && threadID != "" {
			return c.StreamAssistantMessage(ctx, threadID, prompt)
		}
		// If thread creation fails, fall back to chat completions below
	}
	// If API or model is not configured, emit a minimal placeholder stream to avoid server 500s
	if c.api == nil || c.AssistantID == "" {
		log.Printf("[openai][StreamMessage] missing_config api_nil=%v assistant_id=%s", c.api == nil, c.AssistantID)
		ch := make(chan string, 1)
		go func() {
			defer close(ch)
			ch <- ""
		}()
		return ch, nil
	}
	// Resolve model to use: prefer CHAT_MODEL; if AssistantID looks like an assistant, fallback to a sane default
	model := c.Model
	if model == "" {
		if len(c.AssistantID) >= 5 && c.AssistantID[:5] == "asst_" {
			// Assistant IDs are not models; default to gpt-4-turbo (mejor contexto conversacional)
			// gpt-4-turbo tiene ventana de contexto de 128k tokens vs 128k de gpt-4o
			// pero gpt-4-turbo es más estable para conversaciones largas con mejor seguimiento de tema
			model = "gpt-4-turbo"
		} else {
			// If AssistantID is actually a model name, allow using it directly
			model = c.AssistantID
		}
	}

	stream, err := c.api.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		// Some models (e.g., gpt-4.1) may not support ChatCompletionStream. Fallback to non-stream.
		log.Printf("[openai][stream.init.error] %v", err)
		fbModel := c.Model
		if fbModel == "" || (len(c.AssistantID) >= 5 && c.AssistantID[:5] == "asst_") {
			fbModel = "gpt-4o"
		}
		resp, err2 := c.api.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model: fbModel,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
		})
		if err2 != nil {
			log.Printf("[openai][fallback.error] %v", err2)
			return nil, err
		}
		out := make(chan string, 1)
		go func() {
			defer close(out)
			if len(resp.Choices) > 0 {
				msg := resp.Choices[0].Message.Content
				if msg != "" {
					log.Printf("[openai][fallback.msg] len=%d", len(msg))
					out <- msg
				}
			}
		}()
		return out, nil
	}

	ch := make(chan string)

	go func() {
		defer stream.Close()
		defer close(ch)
		anyToken := false
		for {
			resp, err := stream.Recv()
			if err != nil {
				if err.Error() != "EOF" {
					log.Printf("[openai][stream.end] err=%v", err)
				}
				break
			}
			if len(resp.Choices) == 0 {
				log.Printf("[openai][stream.warn] empty choices in delta")
				continue
			}
			token := resp.Choices[0].Delta.Content
			if token != "" {
				anyToken = true
				log.Printf("[openai][stream.token] %s", token)
				ch <- token
			}
		}

		// Fallback: si no llegaron tokens, intentamos una completion no-stream
		if !anyToken {
			fbModel := c.Model
			if fbModel == "" || (len(c.AssistantID) >= 5 && c.AssistantID[:5] == "asst_") {
				fbModel = "gpt-4-turbo"
			}
			resp, err := c.api.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
				Model: fbModel,
				Messages: []openai.ChatCompletionMessage{
					{Role: openai.ChatMessageRoleUser, Content: prompt},
				},
			})
			if err != nil {
				log.Printf("[openai][fallback.error] %v", err)
				return
			}
			if len(resp.Choices) > 0 {
				msg := resp.Choices[0].Message.Content
				if msg != "" {
					log.Printf("[openai][fallback.msg] len=%d", len(msg))
					ch <- msg
				}
			}
		}
	}()

	return ch, nil
}

// TranscribeFile uses OpenAI's audio transcription APIs to convert an audio file into text.
// It tries gpt-4o-mini-transcribe first and falls back to whisper-1 for compatibility.
func (c *Client) TranscribeFile(ctx context.Context, filePath string) (string, error) {
	if c.api == nil {
		return "", errors.New("openai api not configured")
	}
	// Try modern transcribe model first
	req := openai.AudioRequest{
		Model:    "gpt-4o-mini-transcribe",
		FilePath: filePath,
	}
	resp, err := c.api.CreateTranscription(ctx, req)
	if err == nil {
		return resp.Text, nil
	}
	// Fallback to whisper-1
	reqFallback := openai.AudioRequest{
		Model:    "whisper-1",
		FilePath: filePath,
	}
	resp2, err2 := c.api.CreateTranscription(ctx, reqFallback)
	if err2 != nil {
		return "", err
	}
	return resp2.Text, nil
}

// --- Assistants API helpers (HTTP) --- //

func (c *Client) apiURL(path string) string {
	url := "https://api.openai.com/v1" + path
	fmt.Printf("DEBUG: API URL: %s\n", url)
	return url
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any) (*http.Response, error) {
	if c.key == "" {
		return nil, errors.New("openai api key not configured")
	}
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.apiURL(path), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// Required for Assistants v2 endpoints
	req.Header.Set("OpenAI-Beta", "assistants=v2")
	return c.httpClient.Do(req)
}

// Helper to treat 404 as success for idempotent deletes
func okOrNotFound(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	b, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("request failed: %d %s", resp.StatusCode, string(b))
}

// --- Vector Store Helpers --- //

// ensureVectorStore returns the existing vector store id for a thread or creates a new one.
func (c *Client) ensureVectorStore(ctx context.Context, threadID string) (string, error) {
	if c.key == "" {
		return "", fmt.Errorf("OpenAI API key not set")
	}
	c.maybeCleanupVectorStores()

	c.vsMu.RLock()
	if id, ok := c.vectorStore[threadID]; ok {
		c.vsMu.RUnlock()
		c.touchVectorStore(threadID)
		fmt.Printf("DEBUG: Using existing vector store: %s for thread: %s\n", id, threadID)
		return id, nil
	}
	c.vsMu.RUnlock()
	// create new vector store named with threadID
	payload := map[string]any{"name": "vs_session_" + threadID}
	fmt.Printf("DEBUG: Creating new vector store for thread: %s\n", threadID)
	resp, err := c.doJSON(ctx, http.MethodPost, "/vector_stores", payload)
	if err != nil {
		fmt.Printf("ERROR: Failed to create vector store: %v\n", err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create vector store failed: %s", string(b))
	}
	var data struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	c.vsMu.Lock()
	c.vectorStore[threadID] = data.ID
	c.vsLastAccess[threadID] = time.Now()
	// persist
	snapshot := make(map[string]string, len(c.vectorStore))
	for k, v := range c.vectorStore {
		snapshot[k] = v
	}
	c.vsMu.Unlock()
	_ = saveVectorStoreFile(snapshot)
	return data.ID, nil
}

// EnsureVectorStore is an exported wrapper for handlers or tests
func (c *Client) EnsureVectorStore(ctx context.Context, threadID string) (string, error) {
	return c.ensureVectorStore(ctx, threadID)
}

// addFileToVectorStore attaches a file to the vector store.
func (c *Client) addFileToVectorStore(ctx context.Context, vsID, fileID string) error {
	payload := map[string]any{"file_id": fileID}
	resp, err := c.doJSON(ctx, http.MethodPost, "/vector_stores/"+vsID+"/files", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vector store add file failed: %s", string(b))
	}
	return nil
}

// AddFileToVectorStore exported wrapper
func (c *Client) AddFileToVectorStore(ctx context.Context, vsID, fileID string) error {
	fmt.Printf("DEBUG: Adding file %s to vector store %s\n", fileID, vsID)

	if c.key == "" {
		return fmt.Errorf("OpenAI API key not set")
	}

	if vsID == "" {
		return fmt.Errorf("vector store ID is empty")
	}

	if fileID == "" {
		return fmt.Errorf("file ID is empty")
	}

	if err := c.addFileToVectorStore(ctx, vsID, fileID); err != nil {
		fmt.Printf("ERROR: Failed to add file to vector store: %v\n", err)
		return err
	}

	// bump file count for the owning thread if we can map vs->thread
	// Our mapping is threadID -> vsID, so reverse lookup
	c.vsMu.RLock()
	for thread, id := range c.vectorStore {
		if id == vsID {
			c.vsMu.RUnlock()
			c.sessMu.Lock()
			c.sessFiles[thread] = c.sessFiles[thread] + 1
			c.sessMu.Unlock()
			c.touchVectorStore(thread)
			fmt.Printf("DEBUG: Incremented file count for thread %s\n", thread)
			return nil
		}
	}
	c.vsMu.RUnlock()
	return nil
}

// removeFileFromVectorStore removes a file from a vector store (not from OpenAI files, just from the vector store).
func (c *Client) removeFileFromVectorStore(ctx context.Context, vsID, fileID string) error {
	if vsID == "" || fileID == "" {
		return fmt.Errorf("vsID or fileID is empty")
	}
	resp, err := c.doJSON(ctx, http.MethodDelete, "/vector_stores/"+vsID+"/files/"+fileID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// 200-299 significa éxito
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("[vector_store][remove_file] vs=%s file=%s removed", vsID, fileID)
		return nil
	}
	// 404 significa que el archivo ya no está en el vector store (no es un error crítico)
	if resp.StatusCode == 404 {
		log.Printf("[vector_store][remove_file] vs=%s file=%s already_removed", vsID, fileID)
		return nil
	}
	b, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("remove file from vector store failed: %s (status=%d)", string(b), resp.StatusCode)
}

// ClearVectorStoreFiles removes ALL files from a vector store to ensure clean state.
// This is critical to prevent mixing PDFs from different uploads.
// Also invalidates the file cache for the thread to force fresh uploads.
func (c *Client) ClearVectorStoreFiles(ctx context.Context, vsID string) error {
	if vsID == "" {
		return fmt.Errorf("vsID is empty")
	}

	// Buscar el threadID correspondiente a este vector store para limpiar su caché
	var threadID string
	c.vsMu.RLock()
	for tid, vid := range c.vectorStore {
		if vid == vsID {
			threadID = tid
			break
		}
	}
	c.vsMu.RUnlock()

	// Listar archivos actuales en el vector store
	resp, err := c.doJSON(ctx, http.MethodGet, "/vector_stores/"+vsID+"/files?limit=100", nil)
	if err != nil {
		return fmt.Errorf("failed to list vector store files: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("list vector store files failed: %s (status=%d)", string(b), resp.StatusCode)
	}

	var data struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("failed to decode vector store files: %v", err)
	}

	if len(data.Data) == 0 {
		log.Printf("[vector_store][clear] vs=%s already_empty", vsID)
		return nil
	}

	log.Printf("[vector_store][clear] vs=%s removing_files_count=%d", vsID, len(data.Data))

	// Eliminar cada archivo del vector store
	for _, file := range data.Data {
		if err := c.removeFileFromVectorStore(ctx, vsID, file.ID); err != nil {
			// Log error pero continuar con los demás archivos
			log.Printf("[vector_store][clear] vs=%s file=%s error=%v", vsID, file.ID, err)
		}
	}

	// CRÍTICO: Invalidar caché de archivos para este thread
	// Esto fuerza que el próximo upload sea un archivo NUEVO en lugar de reutilizar
	// un file_id anterior que podría estar corrupto o apuntar a un archivo viejo.
	if threadID != "" {
		c.fileMu.Lock()
		clearedCount := 0
		for key := range c.fileCache {
			if strings.HasPrefix(key, threadID+"|") {
				delete(c.fileCache, key)
				clearedCount++
			}
		}
		c.fileMu.Unlock()
		if clearedCount > 0 {
			log.Printf("[vector_store][clear] vs=%s thread=%s invalidated_file_cache_entries=%d", vsID, threadID, clearedCount)
		}

		// Resetear contador de archivos para este thread
		c.sessMu.Lock()
		c.sessFiles[threadID] = 0
		c.sessMu.Unlock()
		log.Printf("[vector_store][clear] vs=%s thread=%s reset_file_count", vsID, threadID)
	}

	log.Printf("[vector_store][clear] vs=%s cleared_successfully", vsID)
	return nil
}

// pollFileProcessed waits until file status=processed or timeout.
func (c *Client) pollFileProcessed(ctx context.Context, fileID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	backoff := 500 * time.Millisecond
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("file %s not processed in time", fileID)
		}
		resp, err := c.doJSON(ctx, http.MethodGet, "/files/"+fileID, nil)
		if err != nil {
			return err
		}
		var data struct {
			Status string `json:"status"`
			ID     string `json:"id"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&data)
		resp.Body.Close()
		if data.Status == "processed" {
			return nil
		}
		if data.Status == "error" {
			return fmt.Errorf("file processing error: %s", fileID)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			if backoff < 5*time.Second {
				backoff *= 2
			}
		}
	}
}

// PollFileProcessed exported wrapper
func (c *Client) PollFileProcessed(ctx context.Context, fileID string, timeout time.Duration) error {
	return c.pollFileProcessed(ctx, fileID, timeout)
}

// pollVectorStoreFileIndexed waits until the file's indexing in the vector store is completed.
// This is CRITICAL because adding a file to a vector store starts an async indexing process.
// Status can be: in_progress, completed, failed, cancelled.
func (c *Client) pollVectorStoreFileIndexed(ctx context.Context, vsID, fileID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	backoff := 500 * time.Millisecond
	pollCount := 0

	for {
		pollCount++
		if time.Now().After(deadline) {
			log.Printf("[indexing][timeout] vs=%s file=%s polls=%d elapsed=%s", vsID, fileID, pollCount, timeout)
			return fmt.Errorf("vector store file %s not indexed in time", fileID)
		}

		resp, err := c.doJSON(ctx, http.MethodGet, "/vector_stores/"+vsID+"/files/"+fileID, nil)
		if err != nil {
			log.Printf("[indexing][api_error] vs=%s file=%s err=%v", vsID, fileID, err)
			return err
		}

		var data struct {
			Status    string `json:"status"`
			ID        string `json:"id"`
			LastError *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"last_error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&data)
		resp.Body.Close()

		// Log estado en cada poll
		if pollCount%5 == 1 { // Log cada 5 polls para no saturar
			log.Printf("[indexing][poll] vs=%s file=%s status=%s poll_count=%d", vsID, fileID, data.Status, pollCount)
		}

		if data.Status == "completed" {
			log.Printf("[indexing][success] vs=%s file=%s polls=%d", vsID, fileID, pollCount)
			return nil
		}

		// Si falla, loggear el error detallado
		if data.Status == "failed" {
			errMsg := "unknown error"
			if data.LastError != nil {
				errMsg = fmt.Sprintf("%s: %s", data.LastError.Code, data.LastError.Message)
			}
			log.Printf("[indexing][FAILED] vs=%s file=%s polls=%d error=%s", vsID, fileID, pollCount, errMsg)

			// Si falló muy rápido (<5s), podría ser transitorio - dar una última oportunidad
			elapsed := time.Since(deadline.Add(-timeout))
			if elapsed < 5*time.Second && pollCount < 10 {
				log.Printf("[indexing][retry_after_fail] vs=%s file=%s reason=too_fast elapsed=%s", vsID, fileID, elapsed)
				time.Sleep(2 * time.Second) // Esperar 2s antes de reintentar
				continue
			}

			return fmt.Errorf("vector store file indexing failed: %s (status=%s, error=%s)", fileID, data.Status, errMsg)
		}

		if data.Status == "cancelled" {
			log.Printf("[indexing][cancelled] vs=%s file=%s polls=%d", vsID, fileID, pollCount)
			return fmt.Errorf("vector store file indexing cancelled: %s", fileID)
		}

		select {
		case <-ctx.Done():
			log.Printf("[indexing][context_cancelled] vs=%s file=%s polls=%d", vsID, fileID, pollCount)
			return ctx.Err()
		case <-time.After(backoff):
			if backoff < 3*time.Second {
				backoff *= 2
			}
		}
	}
}

// PollVectorStoreFileIndexed exported wrapper
func (c *Client) PollVectorStoreFileIndexed(ctx context.Context, vsID, fileID string, timeout time.Duration) error {
	return c.pollVectorStoreFileIndexed(ctx, vsID, fileID, timeout)
}

// emitMarkdownChunks envía el texto dividido en chunks pequeños (~100 chars)
// para simular streaming SSE, permitiendo que normalizeMarkdownToken() añada
// saltos de línea correctamente entre headers y contenido.
// ESTRATEGIA: Headers se envían JUNTO con su párrafo siguiente para que
// normalizeMarkdownToken() pueda detectar y añadir \n\n si falta.
func emitMarkdownChunks(out chan<- string, text string) {
	if text == "" {
		return
	}

	lines := strings.Split(text, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Si es un header markdown (## Título), acumular con líneas siguientes
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			chunk := line
			i++

			// Añadir líneas siguientes hasta completar ~200 chars o encontrar otro header
			for i < len(lines) {
				nextLine := lines[i]
				// Si encontramos otro header, detener
				if strings.HasPrefix(strings.TrimSpace(nextLine), "#") {
					break
				}
				chunk += "\n" + nextLine
				i++
				// Limitar chunk a ~200 chars
				if len(chunk) > 200 {
					break
				}
			}

			out <- chunk
		} else {
			// Línea normal: acumular hasta ~150 chars
			chunk := line
			i++
			for i < len(lines) && len(chunk) < 150 {
				nextLine := lines[i]
				if strings.HasPrefix(strings.TrimSpace(nextLine), "#") {
					break
				}
				chunk += "\n" + nextLine
				i++
			}
			out <- chunk
		}
	}
}

// UploadAssistantFile uploads a file with purpose=assistants; caches per-thread by sha256
func (c *Client) UploadAssistantFile(ctx context.Context, threadID, filePath string) (string, error) {
	if c.key == "" {
		return "", fmt.Errorf("OpenAI API key not set")
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("ERROR: File does not exist: %s\n", filePath)
		return "", fmt.Errorf("file not found: %s", filePath)
	}

	fmt.Printf("DEBUG: Uploading file: %s for thread: %s\n", filePath, threadID)

	// hash content
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("ERROR: Failed to open file: %v\n", err)
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		fmt.Printf("ERROR: Failed to calculate file hash: %v\n", err)
		return "", err
	}
	sum := h.Sum(nil)
	hash := encoding_hex(sum)
	key := threadID + "|" + hash
	c.fileMu.RLock()
	if id, ok := c.fileCache[key]; ok {
		c.fileMu.RUnlock()
		fmt.Printf("DEBUG: Using cached file ID: %s\n", id)

		// Extraer metadatos incluso para archivos cacheados
		metadata := extractPDFMetadata(filePath)

		c.lastMu.Lock()
		c.lastFile[threadID] = LastFileInfo{
			ID:       id,
			Name:     filepath.Base(filePath),
			At:       time.Now(),
			Hash:     hash,
			Metadata: metadata,
		}
		c.lastMu.Unlock()
		return id, nil
	}
	c.fileMu.RUnlock()
	// reopen for upload
	f2, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("ERROR: Failed to reopen file: %v\n", err)
		return "", err
	}
	defer f2.Close()
	resp, err := c.doMultipart(ctx, http.MethodPost, "/files", map[string]io.Reader{
		"file":    f2,
		"purpose": bytes.NewBufferString("assistants"),
	})
	if err != nil {
		fmt.Printf("ERROR: Failed to upload file: %v\n", err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("file upload failed: %s", string(b))
	}
	var data struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	c.fileMu.Lock()
	c.fileCache[key] = data.ID
	snap := make(map[string]string, len(c.fileCache))
	for k, v := range c.fileCache {
		snap[k] = v
	}
	c.fileMu.Unlock()
	_ = saveFileCache(snap)

	// Extraer metadatos del PDF si aplica
	metadata := extractPDFMetadata(filePath)

	// remember last uploaded file for this thread
	c.lastMu.Lock()
	c.lastFile[threadID] = LastFileInfo{
		ID:       data.ID,
		Name:     filepath.Base(filePath),
		At:       time.Now(),
		Hash:     hash,
		Metadata: metadata,
	}
	c.lastMu.Unlock()
	return data.ID, nil
}

func encoding_hex(b []byte) string { return hex.EncodeToString(b) }

// extractPDFMetadata extrae metadatos de un archivo PDF y detecta si tiene texto extraíble
// Retorna nil si no es PDF o si falla la extracción (no crítico)
func extractPDFMetadata(filePath string) (meta *PDFMetadata) {
	// Solo procesar archivos .pdf
	if !strings.HasSuffix(strings.ToLower(filePath), ".pdf") {
		return nil
	}

	// Validar que el archivo existe y es accesible
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Printf("[pdf][metadata][warning] file not accessible %s: %v", filePath, err)
		return nil
	}

	meta = &PDFMetadata{}

	// Extraer título del nombre de archivo
	baseName := filepath.Base(filePath)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// Limpiar y capitalizar el nombre del archivo
	cleaned := strings.ReplaceAll(nameWithoutExt, "_", " ")
	cleaned = strings.ReplaceAll(cleaned, "-", " ")
	meta.Title = cleanPDFString(cleaned)

	// Validar tamaño del archivo para calcular métricas
	fileSizeKB := fileInfo.Size() / 1024

	// Proteger contra panics de la librería rsc.io/pdf
	// Algunos PDFs usan codificaciones no soportadas que causan panic
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[pdf][metadata][panic] recovered from PDF parsing panic %s: %v", filePath, r)
			// Cuando rsc.io/pdf falla, confiamos en OpenAI
			// OpenAI tiene parsers más robustos que pueden manejar encodings raros (ASCII85, etc.)
			log.Printf("[pdf][metadata][parse_panic] file=%s size_kb=%d - trusting OpenAI (has better parsers)",
				filepath.Base(filePath), fileSizeKB)
			meta.HasExtractableText = true
			meta.TextCoveragePercent = 100.0
			meta.PageCount = 0
			log.Printf("[pdf][metadata][extracted] file=%s title=%q size_kb=%d (parse_panic, assumed_text=true)",
				filepath.Base(filePath), meta.Title, fileSizeKB)
		}
	}()

	// Intentar abrir y analizar el PDF para detectar texto
	f, err := os.Open(filePath)
	if err != nil {
		log.Printf("[pdf][metadata][warning] cannot open for text analysis %s: %v", filePath, err)
		// Asumimos que tiene texto si no podemos abrirlo
		meta.HasExtractableText = true
		meta.TextCoveragePercent = 100.0
		log.Printf("[pdf][metadata][extracted] file=%s title=%q (from filename, text_check=failed, assumed_text=true)",
			filepath.Base(filePath), meta.Title)
		return meta
	}
	defer f.Close()

	// Analizar PDF con rsc.io/pdf para detectar texto
	pdfReader, err := pdf.NewReader(f, fileInfo.Size())
	if err != nil {
		log.Printf("[pdf][metadata][warning] cannot parse PDF %s: %v", filePath, err)
		// Asumimos que tiene texto si no podemos parsearlo
		meta.HasExtractableText = true
		meta.TextCoveragePercent = 100.0
		log.Printf("[pdf][metadata][extracted] file=%s title=%q (from filename, parse_failed, assumed_text=true)",
			filepath.Base(filePath), meta.Title)
		return meta
	}

	meta.PageCount = pdfReader.NumPage()

	// Muestrear páginas para detectar texto extraíble
	// Estrategia: revisar primeras 3 páginas, página media y últimas 2
	totalPages := pdfReader.NumPage()
	if totalPages == 0 {
		log.Printf("[pdf][metadata][warning] PDF has 0 pages: %s", filePath)
		meta.HasExtractableText = false
		meta.TextCoveragePercent = 0
		return meta
	}

	samplesToCheck := []int{}
	// Primeras 3 páginas
	for i := 1; i <= min(3, totalPages); i++ {
		samplesToCheck = append(samplesToCheck, i)
	}
	// Página media
	if totalPages > 6 {
		midPage := totalPages / 2
		samplesToCheck = append(samplesToCheck, midPage)
	}
	// Últimas 2 páginas
	if totalPages > 3 {
		for i := max(totalPages-1, 4); i <= totalPages; i++ {
			if !contains(samplesToCheck, i) {
				samplesToCheck = append(samplesToCheck, i)
			}
		}
	}

	pagesWithText := 0
	totalTextLength := 0

	for _, pageNum := range samplesToCheck {
		page := pdfReader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		content := page.Content()

		// Extraer texto de la página
		textContent := extractTextFromContent(&content)
		textLen := len(strings.TrimSpace(textContent))

		if textLen > 50 { // Umbral mínimo: al menos 50 caracteres de texto real
			pagesWithText++
			totalTextLength += textLen
		}
	}

	// Calcular cobertura de texto
	sampledPages := len(samplesToCheck)
	if sampledPages == 0 {
		meta.HasExtractableText = false
		meta.TextCoveragePercent = 0
	} else {
		coveragePercent := (float64(pagesWithText) / float64(sampledPages)) * 100
		meta.TextCoveragePercent = coveragePercent

		// Threshold ajustado a 50%: si la mitad de páginas tienen texto, es usable
		// Esto permite procesar PDFs mixtos (algunas páginas escaneadas, otras digitales)
		meta.HasExtractableText = coveragePercent >= 50.0
	}

	avgTextPerPage := 0
	if pagesWithText > 0 {
		avgTextPerPage = totalTextLength / pagesWithText
	}

	log.Printf("[pdf][metadata][extracted] file=%s title=%q pages=%d sampled=%d with_text=%d coverage=%.1f%% avg_chars=%d has_text=%v",
		filepath.Base(filePath), meta.Title, totalPages, sampledPages, pagesWithText,
		meta.TextCoveragePercent, avgTextPerPage, meta.HasExtractableText)

	return meta
}

// extractTextFromContent extrae texto de un Content de PDF
func extractTextFromContent(content *pdf.Content) string {
	var buf strings.Builder
	for _, text := range content.Text {
		s := cleanPDFString(text.S)
		if s != "" {
			buf.WriteString(s)
			buf.WriteString(" ")
		}
	}
	return buf.String()
}

// min retorna el mínimo de dos enteros
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// UploadImageFile uploads an image file with purpose=vision for GPT-4o vision capabilities
// Supports: jpg, jpeg, png, gif, webp
// Max size: 20MB (OpenAI limit for vision)
// Returns the file_id that can be used in messages with image_file content type
func (c *Client) UploadImageFile(ctx context.Context, imagePath string) (string, error) {
	if c.key == "" {
		return "", fmt.Errorf("OpenAI API key not set")
	}

	// Validar que el archivo existe
	fileInfo, err := os.Stat(imagePath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("image file not found: %s", imagePath)
	}

	// Validar tamaño (OpenAI vision limit es 20MB)
	const maxSize = 20 * 1024 * 1024 // 20MB
	if fileInfo.Size() > maxSize {
		sizeMB := float64(fileInfo.Size()) / (1024 * 1024)
		return "", fmt.Errorf("image too large: %.1fMB (max 20MB)", sizeMB)
	}

	// Validar extensión
	ext := strings.ToLower(filepath.Ext(imagePath))
	validExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true,
		".gif": true, ".webp": true,
	}
	if !validExts[ext] {
		return "", fmt.Errorf("unsupported image format: %s (supported: jpg, jpeg, png, gif, webp)", ext)
	}

	log.Printf("[image][upload] file=%s size_kb=%.1f", filepath.Base(imagePath), float64(fileInfo.Size())/1024)

	// Abrir archivo para upload
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %v", err)
	}
	defer file.Close()

	// Upload a OpenAI con purpose=vision
	resp, err := c.doMultipart(ctx, http.MethodPost, "/files", map[string]io.Reader{
		"file":    file,
		"purpose": bytes.NewBufferString("vision"),
	})
	if err != nil {
		log.Printf("[image][upload][error] file=%s err=%v", filepath.Base(imagePath), err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[image][upload][failed] file=%s status=%d body=%s", filepath.Base(imagePath), resp.StatusCode, string(body))
		return "", fmt.Errorf("image upload failed: %s", string(body))
	}

	var data struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("failed to decode upload response: %v", err)
	}

	log.Printf("[image][upload][ok] file=%s file_id=%s", filepath.Base(imagePath), data.ID)
	return data.ID, nil
}

// max retorna el máximo de dos enteros
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// contains verifica si un slice de enteros contiene un valor
func contains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// cleanPDFString limpia strings extraídos de PDFs (elimina caracteres de control, etc.)
func cleanPDFString(s string) string {
	s = strings.TrimSpace(s)
	// Eliminar caracteres de control y no imprimibles
	var buf strings.Builder
	for _, r := range s {
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			buf.WriteRune(r)
		}
	}
	return strings.TrimSpace(buf.String())
}

// parsePDFDate convierte fechas PDF (formato D:YYYYMMDDHHmmSSOHH'mm') a año simple
// Retorna solo el año para APA (ej: "2023")
func parsePDFDate(pdfDate string) string {
	// Formato típico: D:20230415120000+00'00'
	// Extraer año (posiciones 2-6)
	if len(pdfDate) < 6 {
		return ""
	}
	if pdfDate[0] == 'D' && pdfDate[1] == ':' {
		yearStr := pdfDate[2:6]
		// Validar que sean dígitos
		if _, err := strconv.Atoi(yearStr); err == nil {
			return yearStr
		}
	}
	return ""
}

// Session usage helpers
func (c *Client) AddSessionBytes(threadID string, delta int64) {
	c.sessMu.Lock()
	c.sessBytes[threadID] = c.sessBytes[threadID] + delta
	c.sessMu.Unlock()
}

func (c *Client) GetSessionBytes(threadID string) int64 {
	c.sessMu.RLock()
	v := c.sessBytes[threadID]
	c.sessMu.RUnlock()
	return v
}

func (c *Client) CountThreadFiles(threadID string) int {
	c.sessMu.RLock()
	n := c.sessFiles[threadID]
	c.sessMu.RUnlock()
	return n
}

// doMultipart is a helper to send multipart/form-data requests (used for file uploads).
func (c *Client) doMultipart(ctx context.Context, method, path string, form map[string]io.Reader) (*http.Response, error) {
	if c.key == "" {
		return nil, errors.New("openai api key not configured")
	}
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		for field, r := range form {
			// If the reader is an *os.File, use CreateFormFile to set filename
			if f, ok := r.(*os.File); ok {
				part, err := mw.CreateFormFile(field, f.Name())
				if err != nil {
					_ = mw.Close()
					return
				}
				_, _ = io.Copy(part, f)
				continue
			}
			part, err := mw.CreateFormField(field)
			if err != nil {
				_ = mw.Close()
				return
			}
			_, _ = io.Copy(part, r)
		}
		_ = mw.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, method, c.apiURL(path), pr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("OpenAI-Beta", "assistants=v2")
	return c.httpClient.Do(req)
}

// CreateThread creates a new Assistants thread and returns the id.
func (c *Client) CreateThread(ctx context.Context) (string, error) {
	if c.key == "" {
		log.Printf("[openai][CreateThread][error] missing_api_key")
		return "", fmt.Errorf("openai api key not configured")
	}
	if c.AssistantID == "" {
		log.Printf("[openai][CreateThread][error] missing_assistant_id")
		return "", fmt.Errorf("assistant not configured")
	}
	resp, err := c.doJSON(ctx, http.MethodPost, "/threads", map[string]any{})
	if err != nil {
		log.Printf("[openai][CreateThread][transport.error] %v", err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		log.Printf("[openai][CreateThread][status.error] code=%d body=%s", resp.StatusCode, sanitizeBody(string(b)))
		return "", fmt.Errorf("create thread failed: %s", string(b))
	}
	var data struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.ID, nil
}

// addMessage posts a user message to a thread.
// Implementa retry automático para manejar latencias de OpenAI API
func (c *Client) addMessage(ctx context.Context, threadID, prompt string) error {
	payload := map[string]any{
		"role": "user",
		// Assistants v2 messages expect content items like {type:"text", text:"..."}
		"content": []map[string]any{{"type": "text", "text": prompt}},
	}

	// DEBUG: Log primeros 300 caracteres del mensaje para verificar contenido
	preview := prompt
	if len(preview) > 300 {
		preview = preview[:300] + "..."
	}
	log.Printf("[addMessage][DEBUG] thread=%s msg_preview=%q", threadID, preview)

	// CRÍTICO: OpenAI puede tardar >90s en threads grandes con búsquedas RAG complejas
	// El contexto padre (ctx) ya puede estar cerca de expirar, así que NO lo usamos aquí
	// Usamos un contexto nuevo con timeout generoso para esta operación específica
	addCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Intentar hasta 2 veces si falla por timeout
	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		resp, err := c.doJSON(addCtx, http.MethodPost, "/threads/"+threadID+"/messages", payload)
		if err != nil {
			lastErr = err
			if attempt < 2 && (strings.Contains(err.Error(), "deadline exceeded") || strings.Contains(err.Error(), "timeout")) {
				log.Printf("[addMessage][retry] thread=%s attempt=%d err=%v", threadID, attempt, err)
				time.Sleep(3 * time.Second) // Pausa antes de reintentar
				continue
			}
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("add message failed: %s", string(b))
		}
		return nil
	}
	return lastErr
}

// addMessageWithImage posts a user message with an image attachment using GPT-4o vision
// The image must be already uploaded to OpenAI files with purpose=vision
func (c *Client) addMessageWithImage(ctx context.Context, threadID, prompt, imageFileID string) error {
	// Construir contenido mixto: texto + imagen
	content := []map[string]any{
		{
			"type": "text",
			"text": prompt,
		},
		{
			"type": "image_file",
			"image_file": map[string]string{
				"file_id": imageFileID,
			},
		},
	}

	payload := map[string]any{
		"role":    "user",
		"content": content,
	}

	log.Printf("[addMessageWithImage][DEBUG] thread=%s image_file=%s prompt_preview=%q",
		threadID, imageFileID, truncateString(prompt, 200))

	// Intentar hasta 2 veces si falla por timeout
	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		resp, err := c.doJSON(ctx, http.MethodPost, "/threads/"+threadID+"/messages", payload)
		if err != nil {
			lastErr = err
			if attempt < 2 && (strings.Contains(err.Error(), "deadline exceeded") || strings.Contains(err.Error(), "timeout")) {
				log.Printf("[addMessageWithImage][retry] thread=%s attempt=%d err=%v", threadID, attempt, err)
				time.Sleep(2 * time.Second)
				continue
			}
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(resp.Body)
			log.Printf("[addMessageWithImage][error] thread=%s status=%d body=%s", threadID, resp.StatusCode, string(b))
			return fmt.Errorf("add message with image failed: %s", string(b))
		}
		log.Printf("[addMessageWithImage][ok] thread=%s image_file=%s", threadID, imageFileID)
		return nil
	}
	return lastErr
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// addMessageWithAttachment posts a user message with a file attachment (for file_search tool).

// runAndWait creates a run (optionally with instructions) and polls until completion, then returns the assistant text.
func (c *Client) runAndWait(ctx context.Context, threadID string, instructions string, vectorStoreID string) (string, error) {
	// Máxima duración interna antes de intentar devolver contenido parcial
	// Aumentado a 90s para dar más tiempo a runs complejos con libros grandes
	const maxRunDuration = 90 * time.Second
	start := time.Now()
	// create run
	payload := map[string]any{"assistant_id": c.AssistantID}

	// Las instrucciones ahora vienen completas desde el handler, NO agregamos reglas adicionales
	// para evitar confusión o dilución del mensaje principal
	if strings.TrimSpace(instructions) != "" {
		payload["instructions"] = strings.TrimSpace(instructions)
		// DEBUG: Log instructions enviadas al run
		instrPreview := instructions
		if len(instrPreview) > 300 {
			instrPreview = instrPreview[:300] + "..."
		}
		log.Printf("[runAndWait][DEBUG] thread=%s instructions_preview=%q", threadID, instrPreview)
	}

	// Always include file_search tool + vector store (even if empty) for consistent retrieval context
	// NOTA: max_num_results NO es soportado por OpenAI Assistants API v2 en file_search
	// La optimización principal viene de reducir el prompt (80% menos tokens)
	tools := []map[string]any{{"type": "file_search"}}
	payload["tools"] = tools
	if vectorStoreID != "" {
		payload["tool_resources"] = map[string]any{
			"file_search": map[string]any{
				"vector_store_ids": []string{vectorStoreID},
			},
		}
		log.Printf("[runAndWait][DEBUG] thread=%s OVERRIDE_VECTOR_STORE=%s (optimized_prompt)", threadID, vectorStoreID)
	}

	// DEBUG: Log payload completo para verificar qué estamos enviando
	payloadJSON, _ := json.Marshal(payload)
	log.Printf("[runAndWait][DEBUG] thread=%s run_payload=%s", threadID, string(payloadJSON))

	// TIMING: Medir cuánto tarda OpenAI en crear el run
	createRunStart := time.Now()
	resp, err := c.doJSON(ctx, http.MethodPost, "/threads/"+threadID+"/runs", payload)
	createRunElapsed := time.Since(createRunStart)
	if err != nil {
		log.Printf("[runAndWait][CREATE_RUN_ERROR] thread=%s elapsed=%v err=%v", threadID, createRunElapsed, err)
		return "", err
	}
	defer resp.Body.Close()
	log.Printf("[runAndWait][CREATE_RUN_SUCCESS] thread=%s elapsed=%v status_code=%d", threadID, createRunElapsed, resp.StatusCode)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create run failed: %s", string(b))
	}
	var run struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		ToolResources struct {
			FileSearch struct {
				VectorStoreIDs []string `json:"vector_store_ids"`
			} `json:"file_search"`
		} `json:"tool_resources"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return "", err
	}

	// DEBUG: Verificar qué vector stores está usando realmente el run
	if len(run.ToolResources.FileSearch.VectorStoreIDs) > 0 {
		log.Printf("[runAndWait][DEBUG] thread=%s run_id=%s ACTUAL_VECTOR_STORES=%v", threadID, run.ID, run.ToolResources.FileSearch.VectorStoreIDs)
	}

	// DIAGNÓSTICO: Logging detallado de cada paso del polling
	pollStart := time.Now()
	pollCount := 0
	lastStatus := ""

	// poll
	for {
		select {
		case <-ctx.Done():
			log.Printf("[runAndWait][TIMEOUT] thread=%s run_id=%s elapsed=%v polls=%d last_status=%s",
				threadID, run.ID, time.Since(pollStart), pollCount, lastStatus)
			return "", ctx.Err()
		case <-time.After(400 * time.Millisecond):
		}

		pollCount++

		// Si excedemos la duración máxima, intentamos devolver lo que haya aunque el run no esté completado
		if time.Since(start) > maxRunDuration {
			// Intentar leer mensajes actuales aunque el run siga en progreso
			mresp, merr := c.doJSON(ctx, http.MethodGet, "/threads/"+threadID+"/messages?limit=10&order=desc", nil)
			if merr == nil {
				var ml struct {
					Data []struct {
						Role    string `json:"role"`
						Content []struct {
							Type string `json:"type"`
							Text struct {
								Value string `json:"value"`
							} `json:"text"`
						} `json:"content"`
					} `json:"data"`
				}
				_ = json.NewDecoder(mresp.Body).Decode(&ml)
				mresp.Body.Close()
				for _, m := range ml.Data {
					if m.Role == "assistant" {
						var buf bytes.Buffer
						for _, cpart := range m.Content {
							if cpart.Type == "text" {
								buf.WriteString(cpart.Text.Value)
							}
						}
						text := cleanAssistantAnnotations(buf.String()) // Limpiar anotaciones de OpenAI
						if strings.TrimSpace(text) != "" {
							// Devolver respuesta parcial sin nota de timeout
							// El usuario no necesita saber detalles técnicos internos
							return text, nil
						}
					}
				}
			}
			// Si no pudimos recuperar nada, continuamos polling hasta timeout real del handler
		}
		pollCheckStart := time.Now()
		rresp, rerr := c.doJSON(ctx, http.MethodGet, "/threads/"+threadID+"/runs/"+run.ID, nil)
		if rerr != nil {
			log.Printf("[runAndWait][POLL_ERROR] thread=%s run_id=%s poll#%d err=%v", threadID, run.ID, pollCount, rerr)
			return "", rerr
		}
		var r struct {
			Status    string `json:"status"`
			LastError struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"last_error"`
		}
		_ = json.NewDecoder(rresp.Body).Decode(&r)
		rresp.Body.Close()

		// Log cambios de estado
		if r.Status != lastStatus {
			log.Printf("[runAndWait][STATUS_CHANGE] thread=%s run_id=%s poll#%d elapsed=%v status: %s→%s",
				threadID, run.ID, pollCount, time.Since(pollStart), lastStatus, r.Status)
			lastStatus = r.Status
		}

		// Log cada 30 polls (~12 segundos) si sigue en progreso
		if pollCount%30 == 0 && r.Status != "completed" {
			log.Printf("[runAndWait][STILL_RUNNING] thread=%s run_id=%s poll#%d elapsed=%v status=%s api_latency=%v",
				threadID, run.ID, pollCount, time.Since(pollStart), r.Status, time.Since(pollCheckStart))
		}

		if r.Status == "completed" {
			log.Printf("[runAndWait][COMPLETED] thread=%s run_id=%s total_polls=%d total_elapsed=%v",
				threadID, run.ID, pollCount, time.Since(pollStart))
			break
		}
		if r.Status == "failed" || r.Status == "cancelled" || r.Status == "expired" {
			errorMsg := r.Status
			if r.LastError.Message != "" {
				errorMsg = fmt.Sprintf("%s: %s (code: %s)", r.Status, r.LastError.Message, r.LastError.Code)
			}
			log.Printf("[runAndWait][ERROR] thread=%s run_id=%s status=%s last_error=%+v", threadID, run.ID, r.Status, r.LastError)
			return "", fmt.Errorf("run status: %s", errorMsg)
		}
	}
	// fetch last assistant message
	// CRÍTICO: Usar contexto independiente porque GetMessages puede tardar 60-90s en threads grandes
	// y el contexto del run puede estar cerca de expirar
	fetchStart := time.Now()
	fetchCtx, fetchCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer fetchCancel()

	mresp, merr := c.doJSON(fetchCtx, http.MethodGet, "/threads/"+threadID+"/messages?limit=10&order=desc", nil)
	if merr != nil {
		log.Printf("[runAndWait][GetMessages][ERROR] thread=%s elapsed_ms=%d err=%v", threadID, time.Since(fetchStart).Milliseconds(), merr)
		return "", merr
	}
	log.Printf("[runAndWait][GetMessages][SUCCESS] thread=%s elapsed_ms=%d", threadID, time.Since(fetchStart).Milliseconds())
	defer mresp.Body.Close()
	var ml struct {
		Data []struct {
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text struct {
					Value string `json:"value"`
				} `json:"text"`
			} `json:"content"`
		} `json:"data"`
	}
	if err := json.NewDecoder(mresp.Body).Decode(&ml); err != nil {
		return "", err
	}
	for _, m := range ml.Data {
		if m.Role == "assistant" {
			var buf bytes.Buffer
			log.Printf("DEBUG: Processing assistant message with %d content parts", len(m.Content))
			for i, c := range m.Content {
				log.Printf("DEBUG: Content part %d: type=%s, value_length=%d", i, c.Type, len(c.Text.Value))
				if c.Type == "text" && c.Text.Value != "" {
					buf.WriteString(c.Text.Value)
				}
			}
			finalText := cleanAssistantAnnotations(buf.String()) // Limpiar anotaciones de OpenAI
			log.Printf("DEBUG: Final assistant message length: %d characters", len(finalText))
			if len(finalText) > 200 {
				log.Printf("DEBUG: First 200 chars: %s...", finalText[:200])
			} else {
				log.Printf("DEBUG: Full message: %s", finalText)
			}
			if s := finalText; s != "" {
				return s, nil
			}
		}
	}
	return "", nil
}

// runVisionAndWait uses Chat Completions API directly (not Assistants API) for vision analysis.
// This bypasses assistant configuration issues (reasoning_effort, o3-mini incompatibility with images).
// It encodes the local image as base64 and sends it with the prompt to gpt-4o.
func (c *Client) runVisionAndWait(ctx context.Context, threadID string, systemInstructions string, imagePath string) (string, error) {
	log.Printf("[runVisionAndWait][start] thread=%s image=%s using_chat_completions_api", threadID, filepath.Base(imagePath))

	// 1. Read and encode image as base64
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %v", err)
	}

	// Detect MIME type
	mimeType := "image/jpeg"
	ext := strings.ToLower(filepath.Ext(imagePath))
	switch ext {
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image)

	log.Printf("[runVisionAndWait][DEBUG] thread=%s image_size_kb=%.1f mime=%s",
		threadID, float64(len(imageData))/1024, mimeType)

	// 2. Get latest prompt from thread (the one with the image)
	latestPrompt, err := c.getLatestUserMessage(ctx, threadID)
	if err != nil || latestPrompt == "" {
		latestPrompt = "Analiza esta imagen"
	}

	// 3. Build Chat Completions payload with system + current image
	chatMessages := []map[string]any{
		{
			"role":    "system",
			"content": systemInstructions,
		},
		{
			"role": "user",
			"content": []map[string]any{
				{
					"type": "text",
					"text": latestPrompt,
				},
				{
					"type": "image_url",
					"image_url": map[string]string{
						"url": dataURI,
					},
				},
			},
		},
	}

	payload := map[string]any{
		"model":       "gpt-4o",
		"messages":    chatMessages,
		"max_tokens":  4096,
		"temperature": 0.7,
	}

	// Log payload (truncate base64 for readability)
	log.Printf("[runVisionAndWait][DEBUG] model=gpt-4o messages=2 (system+user_with_image) prompt=%q", latestPrompt)

	// 4. Call /v1/chat/completions
	resp, err := c.doJSON(ctx, http.MethodPost, "/chat/completions", payload)
	if err != nil {
		return "", fmt.Errorf("chat completions request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("chat completions failed: %s", string(b))
	}

	// 5. Parse response
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode chat response: %v", err)
	}

	if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty response from chat completions")
	}

	text := chatResp.Choices[0].Message.Content
	log.Printf("[runVisionAndWait][done] thread=%s chars=%d", threadID, len(text))

	// 6. Add assistant response back to thread for history continuity
	if err := c.addAssistantMessageToThread(ctx, threadID, text); err != nil {
		log.Printf("[runVisionAndWait][warn] failed to add response to thread: %v", err)
		// No fallar - la respuesta ya está generada
	}

	return text, nil
}

// getLatestUserMessage retrieves the most recent user message text from thread
func (c *Client) getLatestUserMessage(ctx context.Context, threadID string) (string, error) {
	path := fmt.Sprintf("/threads/%s/messages?limit=5&order=desc", threadID)
	resp, err := c.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var ml struct {
		Data []struct {
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text *struct {
					Value string `json:"value"`
				} `json:"text,omitempty"`
			} `json:"content"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ml); err != nil {
		return "", err
	}

	// Find latest user message
	for _, msg := range ml.Data {
		if msg.Role == "user" {
			for _, c := range msg.Content {
				if c.Type == "text" && c.Text != nil && c.Text.Value != "" {
					return c.Text.Value, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no user message found")
}

// addAssistantMessageToThread adds an assistant message to the thread for history continuity
func (c *Client) addAssistantMessageToThread(ctx context.Context, threadID, content string) error {
	payload := map[string]any{
		"role":    "assistant",
		"content": content,
	}

	resp, err := c.doJSON(ctx, http.MethodPost, "/threads/"+threadID+"/messages", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add assistant message failed: %s", string(b))
	}

	return nil
}

// getLatestAssistantMessage retrieves the most recent assistant message from thread
func (c *Client) getLatestAssistantMessage(ctx context.Context, threadID string) (string, error) {
	mresp, err := c.doJSON(ctx, http.MethodGet, "/threads/"+threadID+"/messages?limit=1&order=desc", nil)
	if err != nil {
		return "", err
	}
	defer mresp.Body.Close()

	var ml struct {
		Data []struct {
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text struct {
					Value string `json:"value"`
				} `json:"text"`
			} `json:"content"`
		} `json:"data"`
	}

	if err := json.NewDecoder(mresp.Body).Decode(&ml); err != nil {
		return "", err
	}

	if len(ml.Data) == 0 {
		return "", fmt.Errorf("no messages found")
	}

	msg := ml.Data[0]
	if msg.Role != "assistant" {
		return "", fmt.Errorf("latest message is not from assistant")
	}

	var buf bytes.Buffer
	for _, c := range msg.Content {
		if c.Type == "text" && c.Text.Value != "" {
			buf.WriteString(c.Text.Value)
		}
	}

	text := cleanAssistantAnnotations(buf.String()) // Limpiar anotaciones de OpenAI
	if text == "" {
		return "", fmt.Errorf("assistant message is empty")
	}

	log.Printf("[getLatestAssistantMessage] thread=%s chars=%d", threadID, len(text))
	return text, nil
}

// ThreadMessage representa un mensaje del historial del thread
type ThreadMessage struct {
	Role    string // "user" o "assistant"
	Content string
}

// GetThreadMessages obtiene los últimos N mensajes del historial del thread
// para proporcionar contexto conversacional en búsquedas
func (c *Client) GetThreadMessages(ctx context.Context, threadID string, limit int) ([]ThreadMessage, error) {
	if limit <= 0 {
		limit = 10
	}

	path := fmt.Sprintf("/threads/%s/messages?limit=%d&order=desc", threadID, limit)
	resp, err := c.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ml struct {
		Data []struct {
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text struct {
					Value string `json:"value"`
				} `json:"text"`
			} `json:"content"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ml); err != nil {
		return nil, err
	}

	messages := make([]ThreadMessage, 0, len(ml.Data))
	for _, m := range ml.Data {
		var content bytes.Buffer
		for _, c := range m.Content {
			if c.Type == "text" && c.Text.Value != "" {
				content.WriteString(c.Text.Value)
			}
		}
		if text := cleanAssistantAnnotations(content.String()); text != "" { // Limpiar anotaciones de OpenAI
			messages = append(messages, ThreadMessage{
				Role:    m.Role,
				Content: text,
			})
		}
	}

	// Reverse para orden cronológico (los obtuvimos en orden desc)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// StreamAssistantMessage uses Assistants API but emits the final text as a single chunk for simplicity.
func (c *Client) StreamAssistantMessage(ctx context.Context, threadID, prompt string) (<-chan string, error) {
	if c.key == "" || c.AssistantID == "" {
		return nil, errors.New("assistants not configured")
	}
	vsID, _ := c.ensureVectorStore(ctx, threadID) // ignore error; run still works without

	// Crear contexto nuevo con timeout específico para addMessage (desacoplado del request HTTP)
	// Aumentado a 90s + retry automático para manejar latencias extremas de OpenAI API
	addMsgCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if err := c.addMessage(addMsgCtx, threadID, prompt); err != nil {
		return nil, err
	}
	// Debug meta para detectar hilos mezclados: log con hash corto del prompt
	hashPrompt := ""
	if len(prompt) > 0 {
		if len(prompt) > 64 {
			hashPrompt = fmt.Sprintf("%x", sha256.Sum256([]byte(prompt)))[:12]
		} else {
			hashPrompt = fmt.Sprintf("len%d", len(prompt))
		}
	}
	log.Printf("[assist][StreamAssistantMessage][start] thread=%s vs=%s prompt_hash=%s", threadID, vsID, hashPrompt)
	out := make(chan string, 1)
	go func() {
		defer close(out)
		// CRÍTICO: Instrucciones para modo documento (cuando usuario sube PDFs)
		strict := `═══ REGLAS PARA DOCUMENTOS SUBIDOS ═══

DETECCIÓN:
A) CONSULTA CLÍNICA: edad, síntomas, signos, o primera persona ('Tengo X', 'Me duele Y')
B) CONSULTA TEÓRICA: qué es X tumor, tratamiento de Y, capítulo Z del libro

═══ SI ES CONSULTA CLÍNICA (tipo A) ═══
RAZONAMIENTO INTERNO (NO MUESTRES AL USUARIO):
Mentalmente construye: Demografía, Síntomas (TODOS de TODOS los mensajes), Duración, Signos alarma, 3 Hipótesis con probabilidad, Decisiones previas
Reglas: ACUMULA datos, NO resetees, 'Ahora supón X empeora' = mantén previos + añade cambios

RESPUESTA AL USUARIO (ESTO SÍ LO MUESTRA):
Respuesta natural, profesional y completa (250-400 palabras) que incluya:
- Análisis del cuadro clínico basado en datos acumulados
- Hipótesis diferenciales justificadas (número exacto si lo piden)
- Recomendaciones diagnósticas/terapéuticas
- Tono médico profesional pero accesible

CRÍTICO: NO incluyas '[STATE]', 'Demografía:', 'Síntomas clave:', etc. en la respuesta visible.
La respuesta debe fluir naturalmente como un médico hablando.

═══ SI ES CONSULTA TEÓRICA (tipo B) ═══
Respuesta directa: definición, características, tratamiento (200-350 palabras).
NO uses razonamiento interno visible.

═══ REGLAS DE BÚSQUEDA (ambos tipos) ═══
1. Si es pregunta sobre contenido: USA el tool "file_search"
2. SOLO responde con información del documento
3. NO uses conocimiento médico general externo
4. Si es saludo: responde cordialmente sin file_search

═══ FORMATO ═══
- Respuestas completas y bien desarrolladas
- Tono natural y profesional
- NO marcadores artificiales '[Respuesta...]', '[STATE]', etc.
- Si encontraste info: cita + "Fuente: [archivo], pp. X-Y"
- Si NO encontraste: "No encontré información sobre esto en el documento"
- PROHIBIDO inventar

═══ SEGUIMIENTO (para consultas clínicas) ═══
- "Y el tratamiento?" → identifica DE QUÉ condición hablan
- "Qué más dice?" → continúa tema actual
- Si hablaban de cefalea, NO saltes a otro tema sin razón`

		// Bias to the most recently uploaded file if any
		c.lastMu.RLock()
		if lf, ok := c.lastFile[threadID]; ok && strings.TrimSpace(lf.Name) != "" {
			strict = strict + fmt.Sprintf("\n\n6. DOCUMENTO ACTUAL: '%s' - Busca SOLO en este archivo", lf.Name)
			log.Printf("[assist][StreamAssistantMessage] thread=%s bias_last_file=%s age=%s", threadID, lf.Name, time.Since(lf.At))
		}
		c.lastMu.RUnlock()
		text, err := c.runAndWait(ctx, threadID, strict, vsID)
		if err == nil && text != "" {
			outHash := ""
			if len(text) > 120 {
				outHash = fmt.Sprintf("%x", sha256.Sum256([]byte(text)))[:12]
			} else {
				outHash = fmt.Sprintf("len%d", len(text))
			}
			log.Printf("[assist][StreamAssistantMessage][done] thread=%s prompt_hash=%s out_hash=%s chars=%d", threadID, hashPrompt, outHash, len(text))
			// En modo test, guardar respuesta completa en archivo temporal
			if os.Getenv("TEST_CAPTURE_FULL") == "1" {
				tmpFile := "/tmp/assistant_full_" + threadID + ".txt"
				os.WriteFile(tmpFile, []byte(text), 0644)
			}
			// DEBUG: Enviar texto completo sin chunking para verificar formato original de OpenAI
			log.Printf("[DEBUG] Text preview (first 300 chars): %q", truncateString(text, 300))
			out <- text
		}
		if err != nil {
			log.Printf("[assist][StreamAssistantMessage][error] thread=%s err=%v", threadID, err)
		}
	}()
	return out, nil
}

// StreamAssistantMessageWithFile uploads a file and attaches it to the user message, then runs and emits final text.
func (c *Client) StreamAssistantMessageWithFile(ctx context.Context, threadID, prompt, filePath string) (<-chan string, error) {
	if c.key == "" || c.AssistantID == "" {
		return nil, errors.New("assistants not configured")
	}
	// Ensure vector store for this thread
	vsID, err := c.ensureVectorStore(ctx, threadID)
	if err != nil {
		return nil, err
	}
	// Upload file (with cache)
	fileID, err := c.UploadAssistantFile(ctx, threadID, filePath)
	if err != nil {
		return nil, err
	}
	// Poll until processed
	if err := c.pollFileProcessed(ctx, fileID, 60*time.Second); err != nil {
		return nil, err
	}
	// Add file to vector store
	if err := c.addFileToVectorStore(ctx, vsID, fileID); err != nil {
		return nil, err
	}
	// Create message (text only) - usar contexto independiente para evitar timeout
	// Aumentado a 90s + retry automático para manejar latencias extremas de OpenAI API
	addMsgCtx, cancelAddMsg := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelAddMsg()

	if err := c.addMessage(addMsgCtx, threadID, prompt); err != nil {
		return nil, err
	}
	log.Printf("[assist][StreamAssistantMessageWithFile][start] thread=%s vs=%s file=%s prompt_len=%d", threadID, vsID, filepath.Base(filePath), len(prompt))
	out := make(chan string, 1)
	go func() {
		defer close(out)
		// Constrain the run to only use the vector store
		// Ajuste: permitir saludos y conversación general; solo emitir 'No encontré...' cuando la pregunta exige datos del documento y realmente no existen.
		strict := "Si la entrada del usuario es un saludo o una frase genérica (por ejemplo: hola, buenas, gracias, cómo estás) RESPONDE cordialmente y ofrece ayuda sobre el/los documento(s). Para preguntas que requieren datos concretos del/los documento(s) usa solo la información recuperada. Si tras revisar no hay evidencia suficiente responde exactamente: 'No encontré información en el archivo adjunto.' Cita fragmentos textuales relevantes siempre que haya evidencia y no inventes datos. Al final agrega: 'Verificación: Alineado con el documento: Sí/No' y 'Bibliografía' con el/los archivo(s) y, si es posible, páginas o fragmentos."
		// Bias to this uploaded file
		if base := filepath.Base(filePath); base != "" {
			strict = strict + " Responde sobre el archivo recientemente subido '" + base + "' sin pedir confirmación, salvo que el usuario indique otro documento."
		}
		text, err := c.runAndWait(ctx, threadID, strict, vsID)
		if err == nil && text != "" {
			outHash := ""
			if len(text) > 120 {
				outHash = fmt.Sprintf("%x", sha256.Sum256([]byte(text)))[:12]
			} else {
				outHash = fmt.Sprintf("len%d", len(text))
			}
			log.Printf("[assist][StreamAssistantMessageWithFile][done] thread=%s file=%s out_hash=%s chars=%d", threadID, filepath.Base(filePath), outHash, len(text))
			// En modo test, guardar respuesta completa en archivo temporal
			if os.Getenv("TEST_CAPTURE_FULL") == "1" {
				tmpFile := "/tmp/assistant_full_" + threadID + ".txt"
				os.WriteFile(tmpFile, []byte(text), 0644)
			}
			out <- text
		}
		if err != nil {
			log.Printf("[assist][StreamAssistantMessageWithFile][error] thread=%s file=%s err=%v", threadID, filepath.Base(filePath), err)
		}
	}()
	return out, nil
}

// StreamAssistantMessageWithImage uploads an image and creates a message with vision,
// then runs the assistant (GPT-4o/GPT-4o-mini with vision) and streams the response.
// The prompt should ask about the image content.
func (c *Client) StreamAssistantMessageWithImage(ctx context.Context, threadID, prompt, imagePath string) (<-chan string, error) {
	if c.key == "" || c.AssistantID == "" {
		return nil, errors.New("assistants not configured")
	}

	log.Printf("[image][stream][start] thread=%s file=%s", threadID, filepath.Base(imagePath))

	// 1. Upload image to OpenAI
	imageFileID, err := c.UploadImageFile(ctx, imagePath)
	if err != nil {
		log.Printf("[image][stream][upload_error] thread=%s file=%s err=%v", threadID, filepath.Base(imagePath), err)
		return nil, fmt.Errorf("failed to upload image: %v", err)
	}

	// 2. Add message with image to thread
	addMsgCtx, cancelAddMsg := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelAddMsg()

	if err := c.addMessageWithImage(addMsgCtx, threadID, prompt, imageFileID); err != nil {
		log.Printf("[image][stream][add_message_error] thread=%s err=%v", threadID, err)
		return nil, fmt.Errorf("failed to add message with image: %v", err)
	}

	// 3. Create run and stream response
	out := make(chan string, 1)
	go func() {
		defer close(out)

		// Instrucciones específicas para análisis de imágenes médicas
		instructions := `Eres un asistente médico experto analizando una imagen clínica.

REGLAS CRÍTICAS:
1. Analiza ÚNICAMENTE lo que ves en la imagen proporcionada
2. NO inventes información que no sea visible
3. Sé específico sobre hallazgos visuales
4. Si la imagen no es clara o no es médica, indícalo

FORMATO DE RESPUESTA:
- Descripción de lo observado en la imagen
- Hallazgos relevantes (si aplica)
- Consideraciones clínicas basadas en la imagen
- Limitaciones de tu análisis (calidad de imagen, ángulo, etc.)

Si la pregunta del usuario es específica (ej: "¿hay fractura?"), responde esa pregunta directamente basándote en la imagen.

Mantén un tono profesional pero accesible.`

		// Run con Chat Completions API (no Assistants) para evitar conflictos con reasoning_effort
		text, err := c.runVisionAndWait(ctx, threadID, instructions, imagePath)
		if err == nil && text != "" {
			log.Printf("[image][stream][done] thread=%s file=%s chars=%d",
				threadID, filepath.Base(imagePath), len(text))
			out <- text
		}
		if err != nil {
			log.Printf("[image][stream][error] thread=%s file=%s err=%v",
				threadID, filepath.Base(imagePath), err)
		}
	}()
	return out, nil
}

// StreamAssistantJSON runs the assistant using file_search but WITHOUT overriding tool_resources (vector_store_ids),
// so it uses the assistant's pre-configured RAG. It enforces custom JSON-style instructions via the run.
func (c *Client) StreamAssistantJSON(ctx context.Context, threadID, userPrompt, jsonInstructions string) (<-chan string, error) {
	// Fallback path: if we have no AssistantID but do have an API key + model, emulate via Chat Completions.
	if c.key == "" { // sin API key no podemos llamar a OpenAI
		return nil, errors.New("openai api key not configured")
	}
	if c.AssistantID == "" { // usar chat completions como fallback
		model := c.Model
		if strings.TrimSpace(model) == "" {
			model = "gpt-4-turbo" // default con mejor contexto conversacional
		}
		out := make(chan string, 1)
		go func() {
			defer close(out)
			log.Printf("[openai][StreamAssistantJSON] usando fallback chat completions model=%s", model)
			// Construimos un prompt estructurado: system con instrucciones JSON y user con el prompt del usuario.
			// Reutilizamos cliente subyacente (c.api) directamente.
			if c.api == nil {
				return
			}
			req := openai.ChatCompletionRequest{
				Model: model,
				Messages: []openai.ChatCompletionMessage{
					{Role: openai.ChatMessageRoleSystem, Content: jsonInstructions + "\nResponde UNICAMENTE JSON válido."},
					{Role: openai.ChatMessageRoleUser, Content: userPrompt},
				},
			}
			resp, err := c.api.CreateChatCompletion(ctx, req)
			if err != nil || len(resp.Choices) == 0 {
				return
			}
			txt := resp.Choices[0].Message.Content
			if strings.TrimSpace(txt) != "" {
				out <- txt
			}
		}()
		return out, nil
	}
	// Ruta normal Assistants v2
	// Usar contexto independiente para addMessage para evitar timeout si el request HTTP ya consumió tiempo
	// Aumentado a 90s + retry automático para manejar latencias extremas de OpenAI API
	addMsgCtx, cancelAddMsg := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelAddMsg()

	if err := c.addMessage(addMsgCtx, threadID, userPrompt); err != nil {
		return nil, err
	}
	out := make(chan string, 1)
	go func() {
		defer close(out)
		// IMPORTANT: pass empty vectorStoreID to avoid overriding assistant-level vector store
		text, err := c.runAndWait(ctx, threadID, jsonInstructions, "")
		if err == nil && text != "" {
			out <- text
		}
	}()
	return out, nil
}

// --- Cleanup helpers (Delete) --- //

// DeleteThread deletes an Assistants thread (best effort, 404 ignored).
func (c *Client) DeleteThread(ctx context.Context, threadID string) error {
	if c.key == "" || strings.TrimSpace(threadID) == "" {
		return nil
	}
	resp, err := c.doJSON(ctx, http.MethodDelete, "/threads/"+threadID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return okOrNotFound(resp)
}

// deleteVectorStore deletes a vector store by id (best effort).
func (c *Client) deleteVectorStore(ctx context.Context, vsID string) error {
	if c.key == "" || strings.TrimSpace(vsID) == "" {
		return nil
	}
	resp, err := c.doJSON(ctx, http.MethodDelete, "/vector_stores/"+vsID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return okOrNotFound(resp)
}

// deleteFile deletes an uploaded file by id (best effort).
func (c *Client) deleteFile(ctx context.Context, fileID string) error {
	if c.key == "" || strings.TrimSpace(fileID) == "" {
		return nil
	}
	resp, err := c.doJSON(ctx, http.MethodDelete, "/files/"+fileID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return okOrNotFound(resp)
}

// DeleteThreadArtifacts removes vector store, uploaded files, and thread; and clears local mappings.
func (c *Client) DeleteThreadArtifacts(ctx context.Context, threadID string) error {
	// Gather vector store id
	var vsID string
	c.vsMu.RLock()
	if id, ok := c.vectorStore[threadID]; ok {
		vsID = id
	}
	c.vsMu.RUnlock()

	// Collect file IDs from cache for this thread
	var toDelete []string
	c.fileMu.RLock()
	for k, fid := range c.fileCache {
		if strings.HasPrefix(k, threadID+"|") {
			toDelete = append(toDelete, fid)
		}
	}
	c.fileMu.RUnlock()

	// Delete files first (best-effort)
	for _, fid := range toDelete {
		_ = c.deleteFile(ctx, fid)
	}
	// Delete vector store SOLO si NO es el vector store compartido de libros
	// El vector store de libros es permanente y usado por todos los threads del chat general
	const sharedBooksVectorID = "vs_680fc484cef081918b2b9588b701e2f4"
	if vsID != "" && vsID != sharedBooksVectorID {
		log.Printf("[delete_artifacts] deleting vector store thread=%s vs=%s", threadID, vsID)
		_ = c.deleteVectorStore(ctx, vsID)
	} else if vsID == sharedBooksVectorID {
		log.Printf("[delete_artifacts] preserving shared books vector store thread=%s vs=%s", threadID, vsID)
	}
	// Delete thread
	_ = c.DeleteThread(ctx, threadID)

	// Cleanup local maps and persist
	c.vsMu.Lock()
	if vsID != "" {
		delete(c.vectorStore, threadID)
	}
	vsSnap := make(map[string]string, len(c.vectorStore))
	for k, v := range c.vectorStore {
		vsSnap[k] = v
	}
	c.vsMu.Unlock()
	_ = saveVectorStoreFile(vsSnap)

	c.fileMu.Lock()
	if len(toDelete) > 0 {
		// Remove only keys for this thread
		for k := range c.fileCache {
			if strings.HasPrefix(k, threadID+"|") {
				delete(c.fileCache, k)
			}
		}
	}
	fcSnap := make(map[string]string, len(c.fileCache))
	for k, v := range c.fileCache {
		fcSnap[k] = v
	}
	c.fileMu.Unlock()
	_ = saveFileCache(fcSnap)

	c.sessMu.Lock()
	delete(c.sessBytes, threadID)
	delete(c.sessFiles, threadID)
	c.sessMu.Unlock()

	return nil
}

// Satisfy chat.AIClient interface signature using generic any for ctx when invoked indirectly.
func (c *Client) DeleteThreadArtifactsAny(ctx any, threadID string) error {
	if realCtx, ok := ctx.(context.Context); ok {
		return c.DeleteThreadArtifacts(realCtx, threadID)
	}
	return c.DeleteThreadArtifacts(context.Background(), threadID)
}

// --- Vector Store maintenance & inspection helpers --- //

// ForceNewVectorStore drops current vector store (best-effort) and creates a new empty one.
func (c *Client) ForceNewVectorStore(ctx context.Context, threadID string) (string, error) {
	if threadID == "" {
		return "", errors.New("threadID vacío")
	}
	c.vsMu.Lock()
	old := c.vectorStore[threadID]
	delete(c.vectorStore, threadID)
	delete(c.vsLastAccess, threadID)
	c.vsMu.Unlock()

	// Resetear contador de archivos ya que estamos eliminando el vector store
	c.sessMu.Lock()
	c.sessFiles[threadID] = 0
	c.sessMu.Unlock()

	if old != "" {
		log.Printf("[vector_store][force_new] thread=%s deleting_old_vs=%s", threadID, old)
		_ = c.deleteVectorStore(ctx, old)
	}
	return c.ensureVectorStore(ctx, threadID)
}

// ListVectorStoreFiles lists file ids currently attached to the vector store for a thread.
func (c *Client) ListVectorStoreFiles(ctx context.Context, threadID string) ([]string, error) {
	if threadID == "" {
		return nil, errors.New("threadID vacío")
	}
	vsID := c.GetVectorStoreID(threadID)
	if vsID == "" {
		return []string{}, nil
	}
	resp, err := c.doJSON(ctx, http.MethodGet, "/vector_stores/"+vsID+"/files?limit=100", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list files failed: %s", string(b))
	}
	var data struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(data.Data))
	for _, d := range data.Data {
		if d.ID != "" {
			ids = append(ids, d.ID)
		}
	}
	return ids, nil
}

// GetVectorStoreID returns existing vector store id without creating a new one.
func (c *Client) GetVectorStoreID(threadID string) string {
	c.vsMu.RLock()
	defer c.vsMu.RUnlock()
	return c.vectorStore[threadID]
}

// GetLastFileMetadata returns PDF metadata for the last file uploaded to a thread
func (c *Client) GetLastFileMetadata(threadID string) *PDFMetadata {
	c.lastMu.RLock()
	defer c.lastMu.RUnlock()
	if lf, ok := c.lastFile[threadID]; ok {
		return lf.Metadata
	}
	return nil
}

// ExtractPDFMetadataFromPath analyzes a PDF file and extracts metadata including text detection
// Returns nil if file is not a PDF or cannot be read
func (c *Client) ExtractPDFMetadataFromPath(filePath string) *PDFMetadata {
	return extractPDFMetadata(filePath)
}

// touchVectorStore updates last access for TTL logic.
func (c *Client) touchVectorStore(threadID string) {
	if threadID == "" || c.vsTTL == 0 {
		return
	}
	c.vsMu.Lock()
	c.vsLastAccess[threadID] = time.Now()
	c.vsMu.Unlock()
}

// maybeCleanupVectorStores removes expired vector stores based on TTL.
func (c *Client) maybeCleanupVectorStores() {
	if c.vsTTL == 0 {
		return
	}
	if time.Since(c.lastCleanup) < 5*time.Minute {
		return
	}
	c.vsMu.Lock()
	now := time.Now()
	expired := make([]string, 0)
	// CRÍTICO: Proteger el vector store compartido de libros médicos (permanente)
	const sharedBooksVectorID = "vs_680fc484cef081918b2b9588b701e2f4"
	for t, id := range c.vectorStore {
		// NUNCA limpiar el vector store de libros - es compartido y permanente
		if id == sharedBooksVectorID {
			continue
		}
		last := c.vsLastAccess[t]
		if last.IsZero() {
			c.vsLastAccess[t] = now
			continue
		}
		if now.Sub(last) > c.vsTTL {
			expired = append(expired, t)
			log.Printf("[cleanup][expire] thread=%s vs=%s unused_for=%v", t, id, now.Sub(last))
			_ = c.deleteVectorStore(context.Background(), id)
		}
	}
	if len(expired) > 0 {
		for _, t := range expired {
			delete(c.vectorStore, t)
			delete(c.vsLastAccess, t)
		}
		snap := make(map[string]string, len(c.vectorStore))
		for k, v := range c.vectorStore {
			snap[k] = v
		}
		go saveVectorStoreFile(snap)
	}
	c.lastCleanup = now
	c.vsMu.Unlock()
}

// VectorSearchResult contiene tanto el contenido encontrado como metadatos de la fuente
type VectorSearchResult struct {
	Content   string       `json:"content"`
	Source    string       `json:"source"`             // Título del documento o nombre del archivo
	VectorID  string       `json:"vector_id"`          // ID del vector store
	HasResult bool         `json:"has_result"`         // Indica si se encontró información relevante
	Section   string       `json:"section,omitempty"`  // Sección/capítulo si es posible
	Metadata  *PDFMetadata `json:"metadata,omitempty"` // Metadatos del PDF si está disponible
}

// QuickVectorSearch intenta recuperar fragmentos usando el endpoint directo de vector stores (más liviano que crear runs).
// Retorna el contenido Y el nombre REAL del archivo (no adivinado), evitando desalineación de fuentes.
func (c *Client) QuickVectorSearch(ctx context.Context, vectorStoreID, query string) (*VectorSearchResult, error) {
	if c.key == "" || strings.TrimSpace(vectorStoreID) == "" {
		return nil, errors.New("vector store search not configured")
	}

	payload := map[string]any{"query": query}
	resp, err := c.doJSON(ctx, http.MethodPost, "/vector_stores/"+vectorStoreID+"/search", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vector search failed: %s", string(b))
	}
	var data struct {
		Data []struct {
			FileID   string          `json:"file_id"`
			Metadata map[string]any  `json:"metadata"`
			Content  json.RawMessage `json:"content"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if len(data.Data) == 0 {
		return &VectorSearchResult{VectorID: vectorStoreID, HasResult: false}, nil
	}
	// CRÍTICO: Solo tomar el primer resultado para mantener compatibilidad con código existente
	// Para múltiples resultados, usar QuickVectorSearchMultiple
	entry := data.Data[0]
	snippet := extractSnippetFromContent(entry.Content)
	if isLikelyNoDataResponse(snippet) {
		snippet = ""
	}
	result := &VectorSearchResult{VectorID: vectorStoreID, Content: snippet}
	if entry.FileID != "" {
		if name, err := c.getFileName(ctx, entry.FileID); err == nil {
			result.Source = friendlyDocName(name)
		}
	}
	if result.Source == "" && len(entry.Metadata) > 0 {
		if raw, ok := entry.Metadata["source"].(string); ok {
			result.Source = friendlyDocName(raw)
		}
		if section, ok := entry.Metadata["section"].(string); ok {
			result.Section = strings.TrimSpace(section)
		}
		if result.Section == "" {
			if page, ok := entry.Metadata["page_label"].(string); ok {
				result.Section = strings.TrimSpace(page)
			}
		}
		if result.Section == "" {
			if pageNum, ok := entry.Metadata["page"].(float64); ok {
				result.Section = fmt.Sprintf("Página %d", int(pageNum))
			}
		}
	}
	if result.Source != "" || result.Content != "" || strings.TrimSpace(result.Section) != "" {
		result.HasResult = true
	}
	return result, nil
}

// QuickVectorSearchMultiple devuelve MÚLTIPLES resultados del vector store (hasta maxResults)
// Esto permite citar varios libros cuando la información aparece en más de uno
func (c *Client) QuickVectorSearchMultiple(ctx context.Context, vectorStoreID, query string, maxResults int) ([]*VectorSearchResult, error) {
	if c.key == "" || strings.TrimSpace(vectorStoreID) == "" {
		return nil, errors.New("vector store search not configured")
	}

	if maxResults <= 0 {
		maxResults = 3 // Default: hasta 3 libros diferentes
	}

	payload := map[string]any{"query": query}
	resp, err := c.doJSON(ctx, http.MethodPost, "/vector_stores/"+vectorStoreID+"/search", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vector search failed: %s", string(b))
	}
	var data struct {
		Data []struct {
			FileID   string          `json:"file_id"`
			Metadata map[string]any  `json:"metadata"`
			Content  json.RawMessage `json:"content"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if len(data.Data) == 0 {
		return []*VectorSearchResult{}, nil
	}

	// Procesar múltiples resultados, deduplicando por nombre de archivo
	results := make([]*VectorSearchResult, 0, maxResults)
	seenSources := make(map[string]bool)

	log.Printf("[openai][QuickVectorSearchMultiple] total_results=%d max_results=%d query=%q",
		len(data.Data), maxResults, sanitizePreview(query))

	for i, entry := range data.Data {
		if len(results) >= maxResults {
			break
		}

		snippet := extractSnippetFromContent(entry.Content)
		if isLikelyNoDataResponse(snippet) || strings.TrimSpace(snippet) == "" {
			log.Printf("[openai][QuickVectorSearchMultiple][skip.%d] reason=no_content", i)
			continue
		}

		result := &VectorSearchResult{VectorID: vectorStoreID, Content: snippet}

		// Obtener nombre del archivo
		if entry.FileID != "" {
			if name, err := c.getFileName(ctx, entry.FileID); err == nil {
				result.Source = friendlyDocName(name)
			}
		}

		// Fallback a metadata si no obtuvimos el nombre del FileID
		if result.Source == "" && len(entry.Metadata) > 0 {
			if raw, ok := entry.Metadata["source"].(string); ok {
				result.Source = friendlyDocName(raw)
			}
		}

		// Metadatos adicionales
		if len(entry.Metadata) > 0 {
			if section, ok := entry.Metadata["section"].(string); ok {
				result.Section = strings.TrimSpace(section)
			}
			if result.Section == "" {
				if page, ok := entry.Metadata["page_label"].(string); ok {
					result.Section = strings.TrimSpace(page)
				}
			}
			if result.Section == "" {
				if pageNum, ok := entry.Metadata["page"].(float64); ok {
					result.Section = fmt.Sprintf("Página %d", int(pageNum))
				}
			}
		}

		// CRÍTICO: Deduplicar por nombre de archivo para evitar repetir el mismo libro
		sourceKey := strings.ToLower(strings.TrimSpace(result.Source))
		if sourceKey == "" {
			sourceKey = fmt.Sprintf("unknown_%d", i)
		}

		if seenSources[sourceKey] {
			log.Printf("[openai][QuickVectorSearchMultiple][skip.%d] reason=duplicate_source source=%q", i, result.Source)
			continue
		}

		seenSources[sourceKey] = true
		result.HasResult = true
		results = append(results, result)

		log.Printf("[openai][QuickVectorSearchMultiple][added.%d] source=%q content_len=%d section=%q",
			i, result.Source, len(result.Content), result.Section)
	}

	log.Printf("[openai][QuickVectorSearchMultiple][complete] query=%q total_sources=%d unique_sources=%d",
		sanitizePreview(query), len(data.Data), len(results))

	return results, nil
}

func extractSnippetFromContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var items []any
	if err := json.Unmarshal(raw, &items); err == nil {
		for _, item := range items {
			switch v := item.(type) {
			case map[string]any:
				if txt := grabStringField(v, "text"); txt != "" {
					return txt
				}
				if txt := grabStringField(v, "value"); txt != "" {
					return txt
				}
			case string:
				if s := strings.TrimSpace(v); s != "" {
					return s
				}
			}
		}
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return strings.TrimSpace(single)
	}
	return ""
}

func grabStringField(m map[string]any, key string) string {
	if raw, ok := m[key]; ok {
		switch val := raw.(type) {
		case string:
			return strings.TrimSpace(val)
		case map[string]any:
			if inner, ok := val["value"].(string); ok {
				return strings.TrimSpace(inner)
			}
			if inner, ok := val["text"].(string); ok {
				return strings.TrimSpace(inner)
			}
		}
	}
	return ""
}

var docTitleSmallWords = map[string]struct{}{
	"de": {}, "del": {}, "la": {}, "el": {}, "los": {}, "las": {}, "y": {}, "en": {}, "of": {}, "the": {}, "a": {}, "an": {}, "para": {}, "con": {}, "sobre": {}, "por": {}, "un": {}, "una": {}, "al": {}, "da": {}, "dos": {}, "vs": {}, "vs.": {},
}

var docNameOverrides = map[string]string{
	"medical_management_of_the_pregnant_patie":   "Medical Management of the Pregnant Patient",
	"medical_management_of_the_pregnant_patient": "Medical Management of the Pregnant Patient",
}

func canonicalDocKey(raw string) string {
	base := strings.TrimSpace(strings.ToLower(raw))
	if base == "" {
		return ""
	}
	for _, ext := range []string{".pdf", ".docx", ".doc", ".txt"} {
		if strings.HasSuffix(base, ext) {
			base = strings.TrimSuffix(base, ext)
		}
	}
	base = strings.ReplaceAll(base, "-", "_")
	base = strings.ReplaceAll(base, " ", "_")
	for strings.Contains(base, "__") {
		base = strings.ReplaceAll(base, "__", "_")
	}
	return base
}

func friendlyDocName(raw string) string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return "Documento médico"
	}
	key := canonicalDocKey(clean)
	if name, ok := docNameOverrides[key]; ok {
		return name
	}
	name := clean
	if idx := strings.LastIndex(name, "."); idx > -1 {
		name = name[:idx]
	}
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.TrimSpace(name)
	if name == "" {
		return "Documento médico"
	}
	words := strings.Fields(name)
	if len(words) == 0 {
		return "Documento médico"
	}
	for i, w := range words {
		lw := strings.ToLower(w)
		if i > 0 {
			if _, ok := docTitleSmallWords[lw]; ok {
				words[i] = lw
				continue
			}
		}
		runes := []rune(lw)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

// SearchInVectorStore busca información específica en un vector store dado y devuelve metadatos
func (c *Client) SearchInVectorStore(ctx context.Context, vectorStoreID, query string) (string, error) {
	result, err := c.SearchInVectorStoreWithMetadata(ctx, vectorStoreID, query)
	if err != nil {
		return "", err
	}
	if !result.HasResult {
		return "", nil // No se encontró información
	}
	return result.Content, nil
}

// SearchInVectorStoreWithMetadata busca información y devuelve metadatos completos
func (c *Client) SearchInVectorStoreWithMetadata(ctx context.Context, vectorStoreID, query string) (*VectorSearchResult, error) {
	if c.key == "" || c.AssistantID == "" {
		return nil, errors.New("assistants not configured")
	}

	// Buscar threadID para este vectorStoreID (para obtener metadata)
	var threadID string
	c.vsMu.RLock()
	for tid, vid := range c.vectorStore {
		if vid == vectorStoreID {
			threadID = tid
			break
		}
	}
	c.vsMu.RUnlock()

	// Obtener metadatos del PDF si hay lastFile para este thread
	var pdfMetadata *PDFMetadata
	if threadID != "" {
		c.lastMu.RLock()
		if lf, ok := c.lastFile[threadID]; ok {
			pdfMetadata = lf.Metadata
		}
		c.lastMu.RUnlock()
	}

	// Intento rápido: usar el endpoint directo para evitar runs lentos.
	if quick, err := c.QuickVectorSearch(ctx, vectorStoreID, query); err == nil {
		// Agregar metadatos al resultado rápido
		quick.Metadata = pdfMetadata
		if quick.HasResult {
			return quick, nil
		}
		// Si el endpoint respondió pero sin resultados, devolvemos eso sin escalar a instrucciones caras.
		if !quick.HasResult {
			return quick, nil
		}
	} else {
		log.Printf("[vector][quick_search][error] %v", err)
	}

	// Crear un thread temporal para la búsqueda
	tempThreadID, err := c.CreateThread(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary thread: %v", err)
	}
	defer func() {
		_ = c.DeleteThreadArtifacts(ctx, tempThreadID)
	}()

	// Obtener información de los archivos en el vector store para generar metadatos
	files, err := c.listVectorStoreFilesWithNames(ctx, vectorStoreID)
	if err != nil {
		log.Printf("[vector][search][warning] could not get file metadata: %v", err)
		files = []string{"Documento médico"} // Fallback genérico
	}

	// Construir prompt para salida estructurada (JSON) con metadatos
	searchPrompt := fmt.Sprintf(`Eres un asistente de recuperación de información clínica.
TAREA: Busca en el vector store información relevante sobre: "%s".

REGLAS:
- Usa EXCLUSIVAMENTE el contenido disponible en el vector store.
- Si no hay información, responde exactamente: NO_FOUND
- Si hay información, responde ESTRICTAMENTE en JSON válido con esta forma:
  {"source_book":"<título o nombre del archivo>","section":"<capítulo o sección si se detecta, o vacío>","snippet":"<fragmento breve y literal>"}
- No añadas texto fuera del JSON.

Documentos disponibles: %s
`, query, strings.Join(files, ", "))

	// Ejecutar búsqueda anclada al vector store
	text, err := c.runAndWaitWithVectorStore(ctx, tempThreadID, searchPrompt, vectorStoreID)
	if err != nil {
		return nil, fmt.Errorf("vector store search failed: %v", err)
	}

	result := &VectorSearchResult{VectorID: vectorStoreID, HasResult: false, Metadata: pdfMetadata}

	trimmed := strings.TrimSpace(text)
	if trimmed == "" || strings.Contains(strings.ToUpper(trimmed), "NO_FOUND") || isLikelyNoDataResponse(trimmed) {
		return result, nil
	}

	// Intentar parsear JSON estructurado
	var tmp struct {
		SourceBook string `json:"source_book"`
		Section    string `json:"section"`
		Snippet    string `json:"snippet"`
	}
	js := trimmed
	if strings.HasPrefix(js, "{") && strings.HasSuffix(js, "}") && json.Unmarshal([]byte(js), &tmp) == nil {
		// Rellenar con lo recibido
		rawSource := strings.TrimSpace(tmp.SourceBook)
		if rawSource != "" {
			result.Source = friendlyDocName(rawSource)
		}
		result.Section = strings.TrimSpace(tmp.Section)
		result.Content = strings.TrimSpace(tmp.Snippet)
		result.HasResult = result.Source != "" || result.Content != ""
		// Fallback de título si viene vacío: usar primer filename
		if result.Source == "" && len(files) > 0 {
			result.Source = friendlyDocName(files[0])
		}
		return result, nil
	}

	// Fallback: usar texto libre como snippet y primer filename como fuente
	fallbackSource := "Documento médico"
	if len(files) > 0 {
		fallbackSource = friendlyDocName(files[0])
	}
	result.Source = fallbackSource
	if isLikelyNoDataResponse(trimmed) {
		return result, nil
	}
	result.Content = trimmed
	result.HasResult = result.Content != ""
	return result, nil
}

// listVectorStoreFilesWithNames obtiene nombres de archivos en el vector store
func (c *Client) listVectorStoreFilesWithNames(ctx context.Context, vectorStoreID string) ([]string, error) {
	resp, err := c.doJSON(ctx, http.MethodGet, "/vector_stores/"+vectorStoreID+"/files?limit=20", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list files failed: %s", string(b))
	}

	var data struct {
		Data []struct {
			ID     string `json:"id"`
			Object string `json:"object"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	names := make([]string, 0)
	seen := map[string]struct{}{}
	for _, file := range data.Data {
		// Obtener detalles del archivo para el nombre
		if fileName, err := c.getFileName(ctx, file.ID); err == nil && fileName != "" {
			friendly := friendlyDocName(fileName)
			if _, ok := seen[friendly]; ok {
				continue
			}
			seen[friendly] = struct{}{}
			names = append(names, friendly)
		}
	}

	if len(names) == 0 {
		names = []string{"Documento médico"}
	}

	return names, nil
}

// getFileName obtiene el nombre de un archivo por su ID
func (c *Client) getFileName(ctx context.Context, fileID string) (string, error) {
	resp, err := c.doJSON(ctx, http.MethodGet, "/files/"+fileID, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("get file failed")
	}

	var file struct {
		Filename string `json:"filename"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
		return "", err
	}

	return file.Filename, nil
}

// SearchPubMed busca información en PubMed usando E-utilities API de NCBI
func (c *Client) SearchPubMed(ctx context.Context, query string) (string, error) {
	log.Printf("[openai][SearchPubMed][start] query_len=%d query_preview=%s", len(query), sanitizePreview(query))
	start := time.Now()

	// Traducir query a términos médicos en inglés si es necesario
	translatedQuery := c.translateMedicalQuery(ctx, query)
	if translatedQuery != query {
		log.Printf("[openai][SearchPubMed][translated] original_preview=%s translated_preview=%s", sanitizePreview(query), sanitizePreview(translatedQuery))
		query = translatedQuery
	}

	// Limpiar puntuación problemática para PubMed
	query = strings.ReplaceAll(query, "?", "")
	query = strings.ReplaceAll(query, "¿", "")
	query = strings.ReplaceAll(query, "!", "")
	query = strings.ReplaceAll(query, "¡", "")
	query = strings.TrimSpace(query)

	// Contexto con timeout para PubMed API (35s total: 20s búsqueda + 15s detalles)
	pubmedCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()

	// PASO 1: Búsqueda en PubMed con E-Search
	// MEJORA: Aumentar retmax a 8 y cambiar mindate a 2020 para mayor calidad de fuentes
	// Priorizar revisiones sistemáticas, meta-análisis y guías clínicas
	searchURL := fmt.Sprintf(
		"https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?db=pubmed&term=%s&retmode=json&retmax=8&sort=relevance&datetype=pdat&mindate=2020",
		url.QueryEscape(query),
	)

	log.Printf("[openai][SearchPubMed][esearch] url=%s", searchURL)
	searchReq, err := http.NewRequestWithContext(pubmedCtx, http.MethodGet, searchURL, nil)
	if err != nil {
		log.Printf("[openai][SearchPubMed][error] create_search_request_failed err=%v", err)
		return "", fmt.Errorf("failed to create search request: %v", err)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	searchResp, err := client.Do(searchReq)
	if err != nil {
		log.Printf("[openai][SearchPubMed][error] esearch_failed err=%v", err)
		return "", nil // Tratar error de red como sin resultados
	}
	defer searchResp.Body.Close()

	if searchResp.StatusCode == http.StatusTooManyRequests {
		log.Printf("[openai][SearchPubMed][rate_limit] waiting 1s before retry")
		time.Sleep(1 * time.Second)
		// Reintentar una vez después del rate limit
		searchResp2, err2 := client.Do(searchReq)
		if err2 != nil {
			log.Printf("[openai][SearchPubMed][error] esearch_retry_failed err=%v", err2)
			return "", nil
		}
		searchResp.Body.Close()
		searchResp = searchResp2
	}

	if searchResp.StatusCode != http.StatusOK {
		log.Printf("[openai][SearchPubMed][error] esearch_status=%d", searchResp.StatusCode)
		return "", nil
	}

	// Parsear respuesta JSON de E-Search
	var searchResult struct {
		ESearchResult struct {
			Count    string   `json:"count"`
			IDList   []string `json:"idlist"`
			RetMax   string   `json:"retmax"`
			RetStart string   `json:"retstart"`
		} `json:"esearchresult"`
	}

	if err := json.NewDecoder(searchResp.Body).Decode(&searchResult); err != nil {
		log.Printf("[openai][SearchPubMed][error] parse_esearch_failed err=%v", err)
		return "", nil
	}

	pmids := searchResult.ESearchResult.IDList
	if len(pmids) == 0 {
		log.Printf("[openai][SearchPubMed][no_results] count=%s total_elapsed_ms=%d", searchResult.ESearchResult.Count, time.Since(start).Milliseconds())
		return "", nil
	}

	log.Printf("[openai][SearchPubMed][esearch.success] pmids_found=%d count=%s elapsed_ms=%d", len(pmids), searchResult.ESearchResult.Count, time.Since(start).Milliseconds())

	// PASO 2: Obtener detalles de los artículos con E-Fetch (XML)
	// MEJORA: Aumentar a 6 artículos para mayor solidez de fuentes
	// Luego filtraremos por relevancia y diversidad
	if len(pmids) > 6 {
		pmids = pmids[:6]
	}

	pmidList := strings.Join(pmids, ",")
	fetchURL := fmt.Sprintf(
		"https://eutils.ncbi.nlm.nih.gov/entrez/eutils/efetch.fcgi?db=pubmed&id=%s&retmode=xml",
		pmidList,
	)

	log.Printf("[openai][SearchPubMed][efetch] pmids=%s", pmidList)

	// Pequeño delay para respetar rate limits de NCBI (recomiendan 3 requests/segundo sin API key)
	time.Sleep(350 * time.Millisecond)

	fetchReq, err := http.NewRequestWithContext(pubmedCtx, http.MethodGet, fetchURL, nil)
	if err != nil {
		log.Printf("[openai][SearchPubMed][error] create_fetch_request_failed err=%v", err)
		return "", fmt.Errorf("failed to create fetch request: %v", err)
	}

	fetchResp, err := client.Do(fetchReq)
	if err != nil {
		log.Printf("[openai][SearchPubMed][error] efetch_failed err=%v", err)
		return "", nil
	}
	defer fetchResp.Body.Close()

	if fetchResp.StatusCode == http.StatusTooManyRequests {
		log.Printf("[openai][SearchPubMed][rate_limit] efetch waiting 1s before retry")
		time.Sleep(1 * time.Second)
		fetchResp2, err2 := client.Do(fetchReq)
		if err2 != nil {
			log.Printf("[openai][SearchPubMed][error] efetch_retry_failed err=%v", err2)
			return "", nil
		}
		fetchResp.Body.Close()
		fetchResp = fetchResp2
	}

	if fetchResp.StatusCode != http.StatusOK {
		log.Printf("[openai][SearchPubMed][error] efetch_status=%d", fetchResp.StatusCode)
		return "", nil
	}

	// Parsear XML de PubMed
	xmlData, err := io.ReadAll(fetchResp.Body)
	if err != nil {
		log.Printf("[openai][SearchPubMed][error] read_efetch_failed err=%v", err)
		return "", nil
	}

	articles, err := parsePubMedXML(xmlData)
	if err != nil {
		log.Printf("[openai][SearchPubMed][error] parse_xml_failed err=%v", err)
		return "", nil
	}

	if len(articles) == 0 {
		log.Printf("[openai][SearchPubMed][no_results] no_articles_parsed total_elapsed_ms=%d", time.Since(start).Milliseconds())
		return "", nil
	}

	log.Printf("[openai][SearchPubMed][efetch.success] articles_parsed=%d elapsed_ms=%d", len(articles), time.Since(start).Milliseconds())

	// PASO 2.5: Filtrar y priorizar artículos por relevancia y diversidad
	filteredArticles := filterAndPrioritizeArticles(articles, query)
	log.Printf("[openai][SearchPubMed][filtered] original=%d filtered=%d", len(articles), len(filteredArticles))

	// PASO 3: Formatear resultados en JSON estructurado
	result := map[string]interface{}{
		"summary": generateSummary(filteredArticles, query),
		"studies": filteredArticles,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		log.Printf("[openai][SearchPubMed][error] json_marshal_failed err=%v", err)
		return "", nil
	}

	log.Printf("[openai][SearchPubMed][complete] articles=%d json_len=%d total_elapsed_ms=%d", len(articles), len(jsonData), time.Since(start).Milliseconds())
	return string(jsonData), nil
}

// parsePubMedXML parsea el XML de PubMed y extrae información de artículos
func parsePubMedXML(xmlData []byte) ([]map[string]interface{}, error) {
	type Author struct {
		LastName string `xml:"LastName"`
		ForeName string `xml:"ForeName"`
		Initials string `xml:"Initials"`
	}

	type ArticleId struct {
		IdType string `xml:"IdType,attr"`
		Value  string `xml:",chardata"`
	}

	type PubmedData struct {
		ArticleIdList struct {
			ArticleIds []ArticleId `xml:"ArticleId"`
		} `xml:"ArticleIdList"`
	}

	type Article struct {
		ArticleTitle string `xml:"ArticleTitle"`
		Abstract     struct {
			AbstractText []string `xml:"AbstractText"`
		} `xml:"Abstract"`
		AuthorList struct {
			Authors []Author `xml:"Author"`
		} `xml:"AuthorList"`
		Journal struct {
			Title        string `xml:"Title"`
			JournalIssue struct {
				PubDate struct {
					Year  string `xml:"Year"`
					Month string `xml:"Month"`
				} `xml:"PubDate"`
			} `xml:"JournalIssue"`
		} `xml:"Journal"`
	}

	type PubmedArticle struct {
		MedlineCitation struct {
			PMID    string  `xml:"PMID"`
			Article Article `xml:"Article"`
		} `xml:"MedlineCitation"`
		PubmedData PubmedData `xml:"PubmedData"`
	}

	type PubmedArticleSet struct {
		Articles []PubmedArticle `xml:"PubmedArticle"`
	}

	var articleSet PubmedArticleSet
	if err := xml.Unmarshal(xmlData, &articleSet); err != nil {
		return nil, fmt.Errorf("xml unmarshal failed: %v", err)
	}

	var results []map[string]interface{}
	for _, pubmedArticle := range articleSet.Articles {
		art := pubmedArticle.MedlineCitation.Article
		pmid := pubmedArticle.MedlineCitation.PMID

		// Extraer autores (máximo 6)
		authors := []string{}
		for i, author := range art.AuthorList.Authors {
			if i >= 6 {
				break
			}
			authorName := ""
			if author.LastName != "" {
				authorName = author.LastName
				if author.Initials != "" {
					authorName += " " + author.Initials
				} else if author.ForeName != "" {
					// Tomar primera inicial del nombre
					authorName += " " + string([]rune(author.ForeName)[0])
				}
			}
			if authorName != "" {
				authors = append(authors, authorName)
			}
		}

		// Extraer año
		year := 0
		if art.Journal.JournalIssue.PubDate.Year != "" {
			if y, err := strconv.Atoi(art.Journal.JournalIssue.PubDate.Year); err == nil {
				year = y
			}
		}

		// Extraer DOI
		doi := ""
		for _, artID := range pubmedArticle.PubmedData.ArticleIdList.ArticleIds {
			if artID.IdType == "doi" {
				doi = "doi:" + artID.Value
				break
			}
		}

		// Extraer puntos clave del abstract (primeras 2-3 oraciones)
		keyPoints := extractKeyPoints(art.Abstract.AbstractText)

		articleData := map[string]interface{}{
			"title":      cleanText(art.ArticleTitle),
			"pmid":       pmid,
			"authors":    authors,
			"year":       year,
			"journal":    cleanText(art.Journal.Title),
			"key_points": keyPoints,
		}

		if doi != "" {
			articleData["doi"] = doi
		}

		results = append(results, articleData)
	}

	return results, nil
}

// extractKeyPoints extrae puntos clave del abstract con enfoque en hallazgos principales
func extractKeyPoints(abstractTexts []string) []string {
	if len(abstractTexts) == 0 {
		return []string{}
	}

	// Combinar todos los textos del abstract
	fullAbstract := strings.Join(abstractTexts, " ")
	fullAbstract = cleanText(fullAbstract)

	// Dividir en oraciones
	sentences := splitSentences(fullAbstract)

	if len(sentences) == 0 {
		return []string{}
	}

	keyPoints := []string{}

	// MEJORA: Priorizar oraciones que contengan hallazgos principales
	// Detectar secciones RESULTS, CONCLUSIONS, FINDINGS
	conclusionKeywords := []string{
		"conclusion", "found that", "showed that", "demonstrated that",
		"results indicate", "findings suggest", "significantly", "associated with",
		"increased risk", "reduced risk", "efficacy", "effective",
	}

	// Buscar oraciones con hallazgos relevantes
	relevantSentences := []string{}
	for _, sent := range sentences {
		lowerSent := strings.ToLower(sent)
		isRelevant := false

		for _, kw := range conclusionKeywords {
			if strings.Contains(lowerSent, kw) {
				isRelevant = true
				break
			}
		}

		if isRelevant {
			relevantSentences = append(relevantSentences, sent)
		}
	}

	// Si encontramos oraciones relevantes, priorizarlas
	if len(relevantSentences) > 0 {
		// Tomar hasta 3 oraciones relevantes
		for i := 0; i < 3 && i < len(relevantSentences); i++ {
			// MEJORA: Aumentar truncado a 200 chars para mayor contexto
			keyPoints = append(keyPoints, truncateText(relevantSentences[i], 200))
		}
		return keyPoints
	}

	// Fallback: si no hay oraciones con keywords, tomar inicio y final
	// Primera oración (contexto)
	if len(sentences) > 0 {
		keyPoints = append(keyPoints, truncateText(sentences[0], 200))
	}

	// Oración del medio (métodos/resultados)
	if len(sentences) > 2 {
		midIndex := len(sentences) / 2
		keyPoints = append(keyPoints, truncateText(sentences[midIndex], 200))
	}

	// Última oración (conclusiones)
	if len(sentences) > 1 {
		keyPoints = append(keyPoints, truncateText(sentences[len(sentences)-1], 200))
	}

	return keyPoints
}

// splitSentences divide texto en oraciones
func splitSentences(text string) []string {
	// Regex simple para dividir en oraciones
	re := regexp.MustCompile(`[.!?]+\s+`)
	sentences := re.Split(text, -1)

	var result []string
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if len(s) > 20 { // Filtrar fragmentos muy cortos
			result = append(result, s)
		}
	}
	return result
}

// cleanText limpia texto de PubMed (elimina etiquetas, espacios extra)
func cleanText(text string) string {
	// Eliminar saltos de línea y espacios múltiples
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// truncateText trunca texto a un máximo de caracteres
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	// Truncar en espacio más cercano
	truncated := text[:maxLen]
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}
	return truncated + "..."
}

// filterAndPrioritizeArticles filtra y prioriza artículos por relevancia temática y diversidad
func filterAndPrioritizeArticles(articles []map[string]interface{}, query string) []map[string]interface{} {
	if len(articles) == 0 {
		return articles
	}

	// Extraer keywords principales de la query (términos médicos significativos)
	queryKeywords := extractMedicalKeywords(query)
	log.Printf("[openai][filterArticles] query_keywords=%v", queryKeywords)

	// Calcular score de relevancia para cada artículo
	type scoredArticle struct {
		article map[string]interface{}
		score   int
		journal string
		year    int
	}

	scored := make([]scoredArticle, 0, len(articles))
	journalCount := make(map[string]int)

	for _, art := range articles {
		title := ""
		if t, ok := art["title"].(string); ok {
			title = strings.ToLower(t)
		}

		journal := ""
		if j, ok := art["journal"].(string); ok {
			journal = j
		}

		year := 0
		if y, ok := art["year"].(int); ok {
			year = y
		}

		// Calcular score de relevancia
		score := 0

		// +3 puntos por cada keyword que aparezca en el título
		for _, kw := range queryKeywords {
			if strings.Contains(title, strings.ToLower(kw)) {
				score += 3
			}
		}

		// +2 puntos por año reciente (2023-2025)
		if year >= 2023 {
			score += 2
		} else if year >= 2021 {
			score += 1
		}

		// +1 punto si tiene DOI (indica artículo formal)
		if _, hasDOI := art["doi"]; hasDOI {
			score += 1
		}

		// Detectar tipo de publicación por keywords en título
		titleLower := strings.ToLower(title)
		if strings.Contains(titleLower, "systematic review") || strings.Contains(titleLower, "meta-analysis") {
			score += 4 // PRIORIZAR revisiones sistemáticas y meta-análisis
		} else if strings.Contains(titleLower, "guideline") || strings.Contains(titleLower, "consensus") {
			score += 3 // PRIORIZAR guías clínicas
		} else if strings.Contains(titleLower, "randomized") || strings.Contains(titleLower, "clinical trial") {
			score += 2 // PRIORIZAR ensayos clínicos
		}

		scored = append(scored, scoredArticle{
			article: art,
			score:   score,
			journal: journal,
			year:    year,
		})

		journalCount[journal]++
	}

	// Ordenar por score descendente
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		// Desempate por año (más reciente primero)
		return scored[i].year > scored[j].year
	})

	// Seleccionar artículos con diversidad de journals
	selected := make([]map[string]interface{}, 0, 4)
	journalsUsed := make(map[string]int)
	minScore := 0

	// Calcular score mínimo aceptable (promedio - 20%)
	if len(scored) > 0 {
		totalScore := 0
		for _, s := range scored {
			totalScore += s.score
		}
		avgScore := totalScore / len(scored)
		minScore = int(float64(avgScore) * 0.8)
		log.Printf("[openai][filterArticles] avg_score=%d min_score=%d", avgScore, minScore)
	}

	for _, s := range scored {
		// Criterio de parada: ya tenemos suficientes artículos
		if len(selected) >= 4 {
			break
		}

		// Filtrar artículos con score muy bajo (irrelevantes)
		if s.score < minScore && len(selected) > 0 {
			log.Printf("[openai][filterArticles][skip] title=%q score=%d min_score=%d", s.article["title"], s.score, minScore)
			continue
		}

		// Diversificar journals: no más de 2 del mismo journal
		if journalsUsed[s.journal] >= 2 {
			log.Printf("[openai][filterArticles][skip_journal] title=%q journal=%q count=%d", s.article["title"], s.journal, journalsUsed[s.journal])
			continue
		}

		selected = append(selected, s.article)
		journalsUsed[s.journal]++
		log.Printf("[openai][filterArticles][selected] title=%q score=%d year=%d journal=%q", s.article["title"], s.score, s.year, s.journal)
	}

	// Si no hay suficientes artículos de alta calidad, relajar filtros
	if len(selected) < 2 && len(scored) > 0 {
		log.Printf("[openai][filterArticles][fallback] selected_count=%d taking_top_2", len(selected))
		for i := 0; i < 2 && i < len(scored); i++ {
			alreadySelected := false
			for _, sel := range selected {
				if sel["pmid"] == scored[i].article["pmid"] {
					alreadySelected = true
					break
				}
			}
			if !alreadySelected {
				selected = append(selected, scored[i].article)
			}
		}
	}

	return selected
}

// extractMedicalKeywords extrae términos médicos significativos de una query
func extractMedicalKeywords(query string) []string {
	// Limpiar y normalizar
	lower := strings.ToLower(query)

	// Remover stopwords comunes en español e inglés
	stopwords := map[string]bool{
		"el": true, "la": true, "los": true, "las": true, "un": true, "una": true,
		"de": true, "del": true, "al": true, "en": true, "con": true, "por": true,
		"para": true, "que": true, "qué": true, "cual": true, "cuál": true,
		"como": true, "cómo": true, "es": true, "son": true, "sobre": true,
		"the": true, "and": true, "for": true, "from": true, "with": true,
		"what": true, "which": true, "how": true, "about": true,
	}

	// Extraer palabras individuales
	words := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	keywords := []string{}
	seen := make(map[string]bool)

	for _, word := range words {
		word = strings.TrimSpace(word)

		// Filtrar palabras muy cortas o stopwords
		if len(word) < 4 || stopwords[word] {
			continue
		}

		// Evitar duplicados
		if seen[word] {
			continue
		}
		seen[word] = true

		keywords = append(keywords, word)

		// Limitar a 5 keywords principales
		if len(keywords) >= 5 {
			break
		}
	}

	return keywords
}

// generateSummary genera un resumen breve basado en los artículos encontrados
func generateSummary(articles []map[string]interface{}, query string) string {
	if len(articles) == 0 {
		return ""
	}

	// Extraer años para rango
	years := []int{}
	for _, art := range articles {
		if y, ok := art["year"].(int); ok && y > 0 {
			years = append(years, y)
		}
	}

	yearsInfo := ""
	if len(years) > 0 {
		minYear, maxYear := years[0], years[0]
		for _, y := range years {
			if y < minYear {
				minYear = y
			}
			if y > maxYear {
				maxYear = y
			}
		}
		if minYear == maxYear {
			yearsInfo = fmt.Sprintf(" (%d)", maxYear)
		} else {
			yearsInfo = fmt.Sprintf(" (%d-%d)", minYear, maxYear)
		}
	}

	return fmt.Sprintf("Se encontraron %d estudios recientes en PubMed%s relacionados con: %s",
		len(articles), yearsInfo, query)
}

// StreamAssistantWithSpecificVectorStore ejecuta el assistant con un vector store específico
func (c *Client) StreamAssistantWithSpecificVectorStore(ctx context.Context, threadID, prompt, vectorStoreID string) (<-chan string, error) {
	if c.key == "" || c.AssistantID == "" {
		return nil, errors.New("assistants not configured")
	}

	// Crear contexto nuevo con timeout específico para addMessage (desacoplado del request HTTP)
	// Esto evita que falle si el contexto padre ya consumió tiempo en búsquedas (PubMed, vector)
	// Aumentado a 90s + retry automático para manejar latencias extremas de OpenAI API
	addMsgCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if err := c.addMessage(addMsgCtx, threadID, prompt); err != nil {
		return nil, err
	}

	log.Printf("[assist][StreamWithSpecificVector][start] thread=%s vs=%s", threadID, vectorStoreID)
	out := make(chan string, 1)
	go func() {
		defer close(out)
		text, err := c.runAndWaitWithVectorStore(ctx, threadID, prompt, vectorStoreID)
		if err == nil && text != "" {
			log.Printf("[assist][StreamWithSpecificVector][done] thread=%s vs=%s chars=%d", threadID, vectorStoreID, len(text))
			// DEBUG: Enviar texto completo sin chunking
			out <- text
		}
		if err != nil {
			log.Printf("[assist][StreamWithSpecificVector][error] thread=%s vs=%s err=%v", threadID, vectorStoreID, err)
		}
	}()
	return out, nil
}

// StreamAssistantWithInstructions separa el mensaje del usuario (que se guarda en el thread)
// de las instrucciones del sistema (que solo se usan en el run).
// Esto evita contaminar el historial del thread con prompts gigantes del sistema.
func (c *Client) StreamAssistantWithInstructions(ctx context.Context, threadID, userMessage, instructions, vectorStoreID string) (<-chan string, error) {
	if c.key == "" || c.AssistantID == "" {
		return nil, errors.New("assistants not configured")
	}

	// Crear contexto nuevo con timeout específico para addMessage (desacoplado del request HTTP)
	addMsgCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// CRÍTICO: Guardar solo el mensaje del usuario (pregunta limpia), NO las instrucciones completas
	if err := c.addMessage(addMsgCtx, threadID, userMessage); err != nil {
		return nil, err
	}

	log.Printf("[assist][StreamWithInstructions][start] thread=%s vs=%s user_msg_len=%d instructions_len=%d",
		threadID, vectorStoreID, len(userMessage), len(instructions))

	out := make(chan string, 100) // Buffer más grande para chunks
	go func() {
		defer close(out)
		// CRÍTICO: Usar contexto independiente para el run (desacoplado del HTTP request timeout)
		// OpenAI puede tardar 180+ segundos en runs complejos con file_search en vectores grandes
		// Aumentado a 240s (4 min) porque OpenAI API está experimentando latencias extremas
		runCtx, runCancel := context.WithTimeout(context.Background(), 240*time.Second)
		defer runCancel()
		log.Printf("[assist][StreamWithInstructions][run_context] thread=%s timeout=240s", threadID)

		// Usar las instrucciones completas solo para el run, no se guardan en el thread
		text, err := c.runAndWaitWithVectorStore(runCtx, threadID, instructions, vectorStoreID)
		if err == nil && text != "" {
			log.Printf("[assist][StreamWithInstructions][done] thread=%s vs=%s chars=%d", threadID, vectorStoreID, len(text))
			// DEBUG: Enviar texto completo sin chunking
			out <- text
		}
		if err != nil {
			log.Printf("[assist][StreamWithInstructions][error] thread=%s vs=%s err=%v", threadID, vectorStoreID, err)
		}
	}()
	return out, nil
}

// runAndWaitWithVectorStore ejecuta un run del assistant con un vector store específico
func (c *Client) runAndWaitWithVectorStore(ctx context.Context, threadID, instructions, vectorStoreID string) (string, error) {
	// Usar el método existente que ya maneja todo el polling y la lógica de run
	return c.runAndWait(ctx, threadID, instructions, vectorStoreID)
}

// sanitizeBody trims whitespace and limits size for log output
func sanitizeBody(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 400 {
		return s[:400] + "..."
	}
	return s
}

// sanitizePreview helper para logs (más corto que sanitizeBody)
func sanitizePreview(s string) string {
	if len(s) > 80 {
		s = s[:80] + "..."
	}
	return strings.TrimSpace(s)
}

// cleanAssistantAnnotations elimina anotaciones internas de OpenAI Assistants API
// que aparecen como 【0†source】 o fileciteturn0fileN en las respuestas.
// Estas son markers de citación que OpenAI inserta pero no queremos mostrar al usuario.
func cleanAssistantAnnotations(text string) string {
	// Patrón 1: 【número†source】 (brackets Unicode + número + dagger + source)
	re1 := regexp.MustCompile(`【\d+†[^】]*】`)
	text = re1.ReplaceAllString(text, "")

	// Patrón 2: fileciteturnNfileM (donde N y M son números)
	re2 := regexp.MustCompile(`fileciteturn\d+file\d+`)
	text = re2.ReplaceAllString(text, "")

	// Patrón 3: Otros markers comunes de OpenAI
	re3 := regexp.MustCompile(`\[citation:\d+\]`)
	text = re3.ReplaceAllString(text, "")

	// CRÍTICO: Limpiar espacios extras PERO PRESERVAR saltos de línea para Markdown
	// NO usar \s (incluye \n) sino solo espacios/tabs horizontales
	// `[ \t]{2,}` = dos o más espacios/tabs → un espacio
	re4 := regexp.MustCompile(`[ \t]{2,}`)
	text = re4.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// translateMedicalQuery traduce queries médicas del español al inglés usando OpenAI
func (c *Client) translateMedicalQuery(ctx context.Context, query string) string {
	lowerQuery := strings.ToLower(query)

	// Detectar si la query parece estar en español
	// Incluye: caracteres especiales, palabras interrogativas, palabras comunes médicas en español
	spanishIndicators := []string{
		"¿", "á", "é", "í", "ó", "ú", "ñ", // Caracteres especiales
		"cómo", "cuál", "qué", "cuales", "cual", "como", "que", // Interrogativos
		"tratamiento", "paciente", "enfermedad", "síntoma", "diagnóstico", "terapia", // Términos médicos
		"mejor", "para", "con", "del", "por", "sin", "sobre", // Preposiciones comunes
		"tumor", "cáncer", "cardíaca", "renal", "hepático", "pulmonar", // Términos anatómicos
	}
	isSpanish := false
	for _, indicator := range spanishIndicators {
		if strings.Contains(lowerQuery, indicator) {
			isSpanish = true
			break
		}
	}

	if !isSpanish {
		log.Printf("[openai][translateQuery][skip] query appears to be in English, no translation needed")
		return query // Ya está en inglés, no traducir
	}

	log.Printf("[openai][translateQuery][detected] Spanish detected, will translate query_preview=%s", truncateText(query, 80))

	// Usar OpenAI para traducción médica precisa
	if c.api == nil {
		log.Printf("[openai][translateQuery][fallback] No API key available, using simple cleanup")
		// Fallback: limpieza simple si no hay API disponible
		result := strings.ReplaceAll(query, "¿", "")
		result = strings.ReplaceAll(result, "¡", "")
		return result
	}

	translationCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	messages := []openai.ChatCompletionMessage{
		{
			Role: "system",
			Content: "You are a medical search query translator. Convert the Spanish medical question into English search terms for PubMed. " +
				"Extract only the key medical concepts (disease, treatment, anatomy, etc.). " +
				"Remove question words (qué, cuál, cómo). Return ONLY the English search terms, no punctuation.",
		},
		{
			Role:    "user",
			Content: query,
		},
	}

	resp, err := c.api.CreateChatCompletion(translationCtx, openai.ChatCompletionRequest{
		Model:       "gpt-4o-mini", // Modelo rápido y económico para traducciones
		Messages:    messages,
		Temperature: 0.1, // Baja temperatura para traducción consistente
		MaxTokens:   200,
	})

	if err != nil {
		log.Printf("[openai][translateQuery][error] err=%v, using original query", err)
		// Fallback: limpiar caracteres especiales
		result := strings.ReplaceAll(query, "¿", "")
		result = strings.ReplaceAll(result, "¡", "")
		return result
	}

	if len(resp.Choices) == 0 {
		log.Printf("[openai][translateQuery][error] No choices in response, using original query")
		return query
	}

	translated := strings.TrimSpace(resp.Choices[0].Message.Content)
	log.Printf("[openai][translateQuery][success] original=%s translated=%s", truncateText(query, 50), truncateText(translated, 50))
	return translated
}

// simplifyMedicalQuery extrae conceptos clave médicos de una query compleja
func simplifyMedicalQuery(query string) string {
	// Eliminar palabras de pregunta comunes
	q := strings.ToLower(query)
	stopWords := []string{"qué es", "que es", "cuál es", "cual es", "cómo se", "como se", "explica", "define", "dime sobre", "información sobre", "¿", "?"}
	for _, sw := range stopWords {
		q = strings.ReplaceAll(q, sw, "")
	}
	// Limpiar y limitar a primeros 60 caracteres de términos clave
	q = strings.TrimSpace(q)
	if len(q) > 60 {
		q = q[:60]
	}
	return q
}

func isLikelyNoDataResponse(s string) bool {
	t := strings.ToLower(strings.TrimSpace(s))
	if t == "" {
		return true
	}
	keywords := []string{
		"no encontré información",
		"no encontre informacion",
		"no se encontró información",
		"no se encontro informacion",
		"no hay información relevante",
		"no hay informacion relevante",
		"no se encontraron coincidencias",
		"no se encontraron resultados",
		"sin información disponible",
		"sin informacion disponible",
		"no hay contenido disponible",
	}
	for _, k := range keywords {
		if strings.Contains(t, k) {
			return true
		}
	}
	return false
}
