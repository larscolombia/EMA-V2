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
	Source    string `json:"source"`            // Título del documento o nombre del archivo
	VectorID  string `json:"vector_id"`         // ID del vector store
	HasResult bool   `json:"has_result"`        // Indica si se encontró información relevante
	Section   string `json:"section,omitempty"` // Sección/capítulo si es posible
}

// quickVectorSearch intenta recuperar fragmentos usando el endpoint directo de vector stores (más liviano que crear runs).
func (c *Client) quickVectorSearch(ctx context.Context, vectorStoreID, query string) (*VectorSearchResult, error) {
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

	// Intento rápido: usar el endpoint directo para evitar runs lentos.
	if quick, err := c.quickVectorSearch(ctx, vectorStoreID, query); err == nil {
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
	threadID, err := c.CreateThread(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary thread: %v", err)
	}
	defer func() {
		_ = c.DeleteThreadArtifacts(ctx, threadID)
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
	text, err := c.runAndWaitWithVectorStore(ctx, threadID, searchPrompt, vectorStoreID)
	if err != nil {
		return nil, fmt.Errorf("vector store search failed: %v", err)
	}

	result := &VectorSearchResult{VectorID: vectorStoreID, HasResult: false}

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

// SearchPubMed busca información en PubMed usando el assistant con acceso web
func (c *Client) SearchPubMed(ctx context.Context, query string) (string, error) {
	if c.key == "" || c.AssistantID == "" {
		log.Printf("[openai][SearchPubMed][error] assistants_not_configured")
		return "", errors.New("assistants not configured")
	}

	log.Printf("[openai][SearchPubMed][start] query_len=%d query_preview=%s", len(query), sanitizePreview(query))
	start := time.Now()

	// Contexto con timeout específico para PubMed (más generoso que el vector store)
	pubmedCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()

	// Crear un thread temporal para la búsqueda en PubMed
	threadID, err := c.CreateThread(pubmedCtx)
	if err != nil {
		log.Printf("[openai][SearchPubMed][error] create_thread_failed err=%v", err)
		return "", fmt.Errorf("failed to create temporary thread: %v", err)
	}
	log.Printf("[openai][SearchPubMed][thread_created] thread=%s elapsed_ms=%d", threadID, time.Since(start).Milliseconds())

	// Intento 1: Query completa con formato estructurado
	pubmedPrompt := fmt.Sprintf(`INSTRUCCIÓN CRÍTICA: Debes buscar en PubMed (https://pubmed.ncbi.nlm.nih.gov/) usando tus herramientas de búsqueda web.

Tema de búsqueda: "%s"

SI TIENES ACCESO A BÚSQUEDA WEB:
- Busca artículos reales en PubMed sobre el tema
- Devuelve el resultado en el formato JSON especificado abajo
- Prioriza estudios recientes (≥2020)

SI NO TIENES ACCESO A BÚSQUEDA WEB:
- Responde EXACTAMENTE: NO_PUBMED_FOUND
- NO inventes estudios ni datos
- NO digas "voy a intentar" o "hubo un error"
- SOLO responde: NO_PUBMED_FOUND

FORMATO DE RESPUESTA OBLIGATORIO (solo si encontraste resultados reales):
{
	"summary": "Síntesis clínica en 2-3 frases (máximo 80 palabras)",
  "studies": [
    {
      "title": "Título exacto del estudio de PubMed",
      "pmid": "PMID numérico real",
			"authors": ["Apellido Inicial", "Apellido Inicial"],
      "year": 2024,
      "journal": "Nombre de la revista",
			"doi": "doi:10.xxxx/xxxxx",
			"key_points": ["Hallazgo clave 1", "Hallazgo clave 2"]
    }
  ]
}

REGLAS CRÍTICAS:
- ❌ PROHIBIDO inventar PMIDs, títulos o datos que no sean de PubMed real
- ❌ PROHIBIDO responder con mensajes de error como "parece que hubo un error"
- ❌ PROHIBIDO decir "voy a intentar" o "ajustaré la consulta"
- ✅ OBLIGATORIO: Si no puedes acceder a PubMed web, responde: NO_PUBMED_FOUND
- ✅ OBLIGATORIO: Usa SOLO resultados reales de PubMed que puedas verificar
- Incluye máximo 4 estudios, priorizando publicaciones desde 2018 (ideal ≥2020)
- key_points debe contener frases breves (≤25 palabras) con hallazgos clínicos concretos y verificables
- year debe ser numérico exacto del artículo; si no está disponible, omítelo
- journal es el nombre exacto de la revista donde se publicó
- authors debe listar de 1 a 6 autores en orden, usando "Apellido Inicial" del artículo real
- doi es opcional, inclúyelo solo si está disponible en PubMed
- title debe ser el título exacto del artículo en PubMed
- PMID debe ser el identificador numérico real y verificable
- No agregues texto fuera del JSON válido
`, query)

	text, err := c.runAndWait(pubmedCtx, threadID, pubmedPrompt, "")

	// Manejo de timeout: intentar query simplificada
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "context deadline exceeded") {
			log.Printf("[openai][SearchPubMed][timeout] first_attempt thread=%s elapsed_ms=%d, trying_simplified_query", threadID, time.Since(start).Milliseconds())

			// Crear nuevo contexto para reintento
			retryCtx, retryCancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer retryCancel()

			// Query simplificada: solo conceptos clave
			simplifiedQuery := simplifyMedicalQuery(query)
			log.Printf("[openai][SearchPubMed][retry] simplified_query=%s", sanitizePreview(simplifiedQuery))

			simplifiedPrompt := fmt.Sprintf(`Busca en PubMed: "%s". Devuelve JSON con máximo 3 estudios recientes (≥2020) con: title, pmid, year, key_points. Si no encuentras nada, responde: NO_PUBMED_FOUND`, simplifiedQuery)

			text, err = c.runAndWait(retryCtx, threadID, simplifiedPrompt, "")
			if err != nil {
				log.Printf("[openai][SearchPubMed][error] retry_also_failed err=%v total_elapsed_ms=%d", err, time.Since(start).Milliseconds())
				_ = c.DeleteThreadArtifacts(context.Background(), threadID)
				return "", fmt.Errorf("pubmed search timed out after retry: %v", err)
			}
			log.Printf("[openai][SearchPubMed][retry.success] text_len=%d total_elapsed_ms=%d", len(text), time.Since(start).Milliseconds())
		} else {
			log.Printf("[openai][SearchPubMed][error] runAndWait_failed err=%v elapsed_ms=%d", err, time.Since(start).Milliseconds())
			_ = c.DeleteThreadArtifacts(context.Background(), threadID)
			return "", fmt.Errorf("pubmed search failed: %v", err)
		}
	} else {
		log.Printf("[openai][SearchPubMed][success] first_attempt text_len=%d elapsed_ms=%d", len(text), time.Since(start).Milliseconds())
	}

	// Limpiar el thread temporal
	_ = c.DeleteThreadArtifacts(context.Background(), threadID)

	// Verificar si no se encontró información
	if strings.Contains(strings.ToUpper(text), "NO_PUBMED_FOUND") {
		log.Printf("[openai][SearchPubMed][no_results] total_elapsed_ms=%d", time.Since(start).Milliseconds())
		return "", nil // No se encontró información en PubMed
	}

	// Detectar mensajes de error del asistente (cuando no tiene acceso a web search)
	lowerText := strings.ToLower(text)
	errorPhrases := []string{
		"parece que se produjo un error",
		"parece que hubo un error",
		"error al procesar",
		"error al realizar",
		"formato de la consulta no fue aceptado",
		"ajustaré la consulta",
		"intentaré de nuevo",
		"intentaré nuevamente",
		"voy a intentar",
		"por favor, espera",
		"actualizando consulta",
	}

	for _, phrase := range errorPhrases {
		if strings.Contains(lowerText, phrase) {
			log.Printf("[openai][SearchPubMed][assistant_error_detected] phrase=\"%s\" text_preview=%s total_elapsed_ms=%d", phrase, sanitizePreview(text), time.Since(start).Milliseconds())
			return "", nil // El asistente no puede acceder a PubMed, tratar como sin resultados
		}
	}

	// Verificar que la respuesta tenga contenido mínimo
	if len(strings.TrimSpace(text)) < 50 {
		log.Printf("[openai][SearchPubMed][insufficient_content] text_len=%d total_elapsed_ms=%d", len(text), time.Since(start).Milliseconds())
		return "", nil
	}

	// Validar que realmente sea JSON con estructura de estudios
	if !strings.Contains(text, "{") || !strings.Contains(text, "\"studies\"") {
		log.Printf("[openai][SearchPubMed][invalid_json_structure] text_len=%d total_elapsed_ms=%d", len(text), time.Since(start).Milliseconds())
		return "", nil
	}

	log.Printf("[openai][SearchPubMed][complete] text_len=%d total_elapsed_ms=%d", len(text), time.Since(start).Milliseconds())
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

// sanitizePreview helper para logs (más corto que sanitizeBody)
func sanitizePreview(s string) string {
	if len(s) > 80 {
		s = s[:80] + "..."
	}
	return strings.TrimSpace(s)
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
