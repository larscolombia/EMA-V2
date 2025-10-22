package openai

import (
	"bytes"
	"context"
	"crypto/sha256"
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
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	openai "github.com/sashabaranov/go-openai"
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
	Title    string `json:"title,omitempty"`
	Author   string `json:"author,omitempty"`
	Subject  string `json:"subject,omitempty"`
	Keywords string `json:"keywords,omitempty"`
	Creator  string `json:"creator,omitempty"`
	Producer string `json:"producer,omitempty"`
	Created  string `json:"created,omitempty"`  // Fecha de creación
	Modified string `json:"modified,omitempty"` // Fecha de modificación
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
		httpClient:   &http.Client{Timeout: 120 * time.Second}, // Aumentado para runs largos
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

// extractPDFMetadata extrae metadatos de un archivo PDF
// Retorna nil si no es PDF o si falla la extracción (no crítico)
func extractPDFMetadata(filePath string) *PDFMetadata {
	// Solo procesar archivos .pdf
	if !strings.HasSuffix(strings.ToLower(filePath), ".pdf") {
		return nil
	}

	// Validar que el archivo existe y es accesible
	if _, err := os.Stat(filePath); err != nil {
		log.Printf("[pdf][metadata][warning] file not accessible %s: %v", filePath, err)
		return nil
	}

	meta := &PDFMetadata{}

	// Extraer título del nombre de archivo
	// Nota: rsc.io/pdf es una librería muy simple que no expone metadatos fácilmente
	// Para extracción real de metadatos (Author, CreationDate, etc.) se necesitaría
	// una librería más completa como github.com/pdfcpu/pdfcpu
	baseName := filepath.Base(filePath)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// Limpiar y capitalizar el nombre del archivo
	cleaned := strings.ReplaceAll(nameWithoutExt, "_", " ")
	cleaned = strings.ReplaceAll(cleaned, "-", " ")
	meta.Title = cleanPDFString(cleaned)

	log.Printf("[pdf][metadata][extracted] file=%s title=%q (from filename)",
		filepath.Base(filePath), meta.Title)

	return meta
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

	// Intentar hasta 2 veces si falla por timeout
	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		resp, err := c.doJSON(ctx, http.MethodPost, "/threads/"+threadID+"/messages", payload)
		if err != nil {
			lastErr = err
			if attempt < 2 && (strings.Contains(err.Error(), "deadline exceeded") || strings.Contains(err.Error(), "timeout")) {
				log.Printf("[addMessage][retry] thread=%s attempt=%d err=%v", threadID, attempt, err)
				time.Sleep(2 * time.Second) // Pequeña pausa antes de reintentar
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
	tools := []map[string]any{{"type": "file_search"}}
	payload["tools"] = tools
	if vectorStoreID != "" {
		payload["tool_resources"] = map[string]any{
			"file_search": map[string]any{
				"vector_store_ids": []string{vectorStoreID},
			},
		}
		log.Printf("[runAndWait][DEBUG] thread=%s OVERRIDE_VECTOR_STORE=%s (forcing file_search to use ONLY this vector)", threadID, vectorStoreID)
	}

	// DEBUG: Log payload completo para verificar qué estamos enviando
	payloadJSON, _ := json.Marshal(payload)
	log.Printf("[runAndWait][DEBUG] thread=%s run_payload=%s", threadID, string(payloadJSON))

	resp, err := c.doJSON(ctx, http.MethodPost, "/threads/"+threadID+"/runs", payload)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
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

	// poll
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(400 * time.Millisecond):
		}

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
						text := buf.String()
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
		rresp, rerr := c.doJSON(ctx, http.MethodGet, "/threads/"+threadID+"/runs/"+run.ID, nil)
		if rerr != nil {
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
		if r.Status == "completed" {
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
	mresp, merr := c.doJSON(ctx, http.MethodGet, "/threads/"+threadID+"/messages?limit=10&order=desc", nil)
	if merr != nil {
		return "", merr
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
			finalText := buf.String()
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
		if text := content.String(); text != "" {
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
	// Delete vector store
	if vsID != "" {
		_ = c.deleteVectorStore(ctx, vsID)
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
	if old != "" {
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
	for t, id := range c.vectorStore {
		last := c.vsLastAccess[t]
		if last.IsZero() {
			c.vsLastAccess[t] = now
			continue
		}
		if now.Sub(last) > c.vsTTL {
			expired = append(expired, t)
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
	searchURL := fmt.Sprintf(
		"https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?db=pubmed&term=%s&retmode=json&retmax=5&sort=relevance&datetype=pdat&mindate=2018",
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
	// Limitar a máximo 4 artículos para no sobrecargar
	if len(pmids) > 4 {
		pmids = pmids[:4]
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

	// PASO 3: Formatear resultados en JSON estructurado
	result := map[string]interface{}{
		"summary": generateSummary(articles, query),
		"studies": articles,
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

// extractKeyPoints extrae puntos clave del abstract
func extractKeyPoints(abstractTexts []string) []string {
	if len(abstractTexts) == 0 {
		return []string{}
	}

	// Combinar todos los textos del abstract
	fullAbstract := strings.Join(abstractTexts, " ")
	fullAbstract = cleanText(fullAbstract)

	// Dividir en oraciones
	sentences := splitSentences(fullAbstract)

	// Tomar máximo 3 oraciones más relevantes (primeras 2 y última)
	keyPoints := []string{}
	if len(sentences) > 0 {
		keyPoints = append(keyPoints, truncateText(sentences[0], 150))
	}
	if len(sentences) > 1 {
		keyPoints = append(keyPoints, truncateText(sentences[1], 150))
	}
	if len(sentences) > 3 {
		keyPoints = append(keyPoints, truncateText(sentences[len(sentences)-1], 150))
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
			out <- text
		}
		if err != nil {
			log.Printf("[assist][StreamWithSpecificVector][error] thread=%s vs=%s err=%v", threadID, vectorStoreID, err)
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
