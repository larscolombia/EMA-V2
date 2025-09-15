package openai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

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
	ID   string
	Name string
	At   time.Time
	Hash string
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
		httpClient:   &http.Client{Timeout: 60 * time.Second},
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
			// Assistant IDs are not models; default to a general model unless CHAT_MODEL is provided
			model = "gpt-4o-mini"
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
			fbModel = "gpt-4o-mini"
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
				fbModel = "gpt-4o-mini"
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
		c.lastMu.Lock()
		c.lastFile[threadID] = LastFileInfo{ID: id, Name: filepath.Base(filePath), At: time.Now(), Hash: hash}
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

	// remember last uploaded file for this thread
	c.lastMu.Lock()
	c.lastFile[threadID] = LastFileInfo{ID: data.ID, Name: filepath.Base(filePath), At: time.Now(), Hash: hash}
	c.lastMu.Unlock()
	return data.ID, nil
}

func encoding_hex(b []byte) string { return hex.EncodeToString(b) }

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
func (c *Client) addMessage(ctx context.Context, threadID, prompt string) error {
	payload := map[string]any{
		"role": "user",
		// Assistants v2 messages expect content items like {type:"text", text:"..."}
		"content": []map[string]any{{"type": "text", "text": prompt}},
	}
	resp, err := c.doJSON(ctx, http.MethodPost, "/threads/"+threadID+"/messages", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add message failed: %s", string(b))
	}
	return nil
}

// addMessageWithAttachment posts a user message with a file attachment (for file_search tool).

// runAndWait creates a run (optionally with instructions) and polls until completion, then returns the assistant text.
func (c *Client) runAndWait(ctx context.Context, threadID string, instructions string, vectorStoreID string) (string, error) {
	// Máxima duración interna antes de intentar devolver contenido parcial
	const maxRunDuration = 55 * time.Second
	start := time.Now()
	// create run
	payload := map[string]any{"assistant_id": c.AssistantID}
	// Endurecer instrucciones cuando hay vector store (archivos adjuntos) para exigir verificación y bibliografía
	hardened := strings.TrimSpace(instructions)
	if strings.TrimSpace(vectorStoreID) != "" {
		extra := "\n\nREGLAS DE RIGOR (obligatorias cuando existan archivos adjuntos):\n- Usa EXCLUSIVAMENTE la información recuperada de los archivos del hilo.\n- Si la pregunta no puede responderse con el contenido disponible, responde exactamente: 'No encontré información en el archivo adjunto.'\n- Incluye fragmentos textuales breves cuando cites.\n- Al final agrega: 'Verificación: Alineado con el documento: Sí/No' y una sección 'Bibliografía' listando el/los archivo(s) y, si es posible, páginas o fragmentos.\n- No inventes fuentes ni datos."
		if hardened == "" {
			hardened = extra
		} else {
			hardened = hardened + extra
		}
	}
	if hardened != "" {
		payload["instructions"] = hardened
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
	}
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
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return "", err
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
							return text + "\n\n[Nota: respuesta parcial generada antes de completar el proceso para evitar timeout]", nil
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
			Status string `json:"status"`
		}
		_ = json.NewDecoder(rresp.Body).Decode(&r)
		rresp.Body.Close()
		if r.Status == "completed" {
			break
		}
		if r.Status == "failed" || r.Status == "cancelled" || r.Status == "expired" {
			return "", fmt.Errorf("run status: %s", r.Status)
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

// StreamAssistantMessage uses Assistants API but emits the final text as a single chunk for simplicity.
func (c *Client) StreamAssistantMessage(ctx context.Context, threadID, prompt string) (<-chan string, error) {
	if c.key == "" || c.AssistantID == "" {
		return nil, errors.New("assistants not configured")
	}
	vsID, _ := c.ensureVectorStore(ctx, threadID) // ignore error; run still works without
	if err := c.addMessage(ctx, threadID, prompt); err != nil {
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
		// Ajuste: permitir saludos/conversación natural sin forzar mensaje de 'No encontré...' si el usuario solo saluda.
		strict := "Si la entrada del usuario es un saludo o una frase genérica (por ejemplo: hola, buenas, gracias, cómo estás) RESPONDE cordialmente y ofrece ayuda sobre el/los documento(s) sin pedir que reformule. Para preguntas que requieren datos concretos del/los documento(s) debes usar EXCLUSIVAMENTE la información recuperada de los documentos de este hilo. Si después de revisar no hay evidencia suficiente para responder esa pregunta específica, responde exactamente: 'No encontré información en el archivo adjunto.' Siempre que cites información encontrada incluye fragmentos textuales concisos. No inventes contenido que no esté en los documentos. Al final agrega: 'Verificación: Alineado con el documento: Sí/No' y 'Bibliografía' con el/los archivo(s) y, si es posible, páginas o fragmentos."
		// Bias to the most recently uploaded file if any
		c.lastMu.RLock()
		if lf, ok := c.lastFile[threadID]; ok && strings.TrimSpace(lf.Name) != "" {
			strict = strict + " Prioriza el archivo más reciente de este hilo ('" + lf.Name + "') y no pidas confirmación a menos que el usuario lo contradiga."
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
	// Create message (text only)
	if err := c.addMessage(ctx, threadID, prompt); err != nil {
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
			model = "gpt-4o-mini" // default razonable
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
	if err := c.addMessage(ctx, threadID, userPrompt); err != nil {
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
	Content   string `json:"content"`
	Source    string `json:"source"`     // Título del documento o nombre del archivo
	VectorID  string `json:"vector_id"`  // ID del vector store
	HasResult bool   `json:"has_result"` // Indica si se encontró información relevante
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

	// Crear un thread temporal para la búsqueda
	threadID, err := c.CreateThread(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary thread: %v", err)
	}

	// Obtener información de los archivos en el vector store para generar metadatos
	files, err := c.listVectorStoreFilesWithNames(ctx, vectorStoreID)
	if err != nil {
		log.Printf("[vector][search][warning] could not get file metadata: %v", err)
		files = []string{"Documento médico"} // Fallback genérico
	}

	// Construir prompt enriquecido que incluye solicitud de metadatos
	searchPrompt := fmt.Sprintf(`Busca información relevante sobre: "%s"

IMPORTANTE: 
- Devuelve SOLO la información encontrada en los documentos del vector store
- No elabores ni agregues información externa
- Si no encuentras información relevante, responde únicamente: "NO_FOUND"
- Incluye fragmentos textuales específicos cuando sea posible
- Al inicio de tu respuesta, indica de qué documento(s) proviene la información

Documentos disponibles en este vector store: %s
`, query, strings.Join(files, ", "))

	// Usamos el assistant con instrucciones específicas para búsqueda
	text, err := c.runAndWaitWithVectorStore(ctx, threadID, searchPrompt, vectorStoreID)
	if err != nil {
		return nil, fmt.Errorf("vector store search failed: %v", err)
	}

	// Limpiar el thread temporal
	_ = c.DeleteThreadArtifacts(ctx, threadID)

	result := &VectorSearchResult{
		VectorID:  vectorStoreID,
		HasResult: false,
	}

	if strings.Contains(strings.ToUpper(text), "NO_FOUND") {
		result.Content = ""
		result.Source = ""
		return result, nil
	}

	// Extraer fuente del contenido si está presente
	source := "Base de conocimiento médico"
	if len(files) > 0 {
		source = files[0] // Usar el primer archivo como fuente principal
	}

	result.Content = text
	result.Source = source
	result.HasResult = true

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
	for _, file := range data.Data {
		// Obtener detalles del archivo para el nombre
		if fileName, err := c.getFileName(ctx, file.ID); err == nil && fileName != "" {
			names = append(names, fileName)
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

// SearchPubMed busca información en PubMed usando el assistant con acceso web
func (c *Client) SearchPubMed(ctx context.Context, query string) (string, error) {
	if c.key == "" || c.AssistantID == "" {
		return "", errors.New("assistants not configured")
	}

	// Crear un thread temporal para la búsqueda en PubMed
	threadID, err := c.CreateThread(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary thread: %v", err)
	}

	pubmedPrompt := fmt.Sprintf(`Busca información médica sobre: "%s" en PubMed (https://pubmed.ncbi.nlm.nih.gov/)

IMPORTANTE:
- Busca SOLO en PubMed oficial
- Incluye PMIDs cuando estén disponibles
- Prioriza estudios recientes y de alta calidad
- Si no encuentras información relevante, responde: "NO_PUBMED_FOUND"
- Incluye citas bibliográficas específicas
- Mantén rigor científico
`, query)

	text, err := c.runAndWait(ctx, threadID, pubmedPrompt, "")
	if err != nil {
		return "", fmt.Errorf("pubmed search failed: %v", err)
	}

	// Limpiar el thread temporal
	_ = c.DeleteThreadArtifacts(ctx, threadID)

	if strings.Contains(strings.ToUpper(text), "NO_PUBMED_FOUND") {
		return "", nil // No se encontró información en PubMed
	}

	return text, nil
}

// StreamAssistantWithSpecificVectorStore ejecuta el assistant con un vector store específico
func (c *Client) StreamAssistantWithSpecificVectorStore(ctx context.Context, threadID, prompt, vectorStoreID string) (<-chan string, error) {
	if c.key == "" || c.AssistantID == "" {
		return nil, errors.New("assistants not configured")
	}

	if err := c.addMessage(ctx, threadID, prompt); err != nil {
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
