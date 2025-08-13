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
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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
	vsMu        sync.RWMutex
	vectorStore map[string]string
	fileMu      sync.RWMutex
	fileCache   map[string]string // key: threadID+"|"+sha256 -> fileID
	// session usage tracking (in-memory)
	sessMu    sync.RWMutex
	sessBytes map[string]int64
	sessFiles map[string]int
	// last uploaded file per thread to bias instructions
	lastMu   sync.RWMutex
	lastFile map[string]LastFileInfo
}

type LastFileInfo struct {
	ID   string
	Name string
	At   time.Time
}

func NewClient() *Client {
	key := os.Getenv("OPENAI_API_KEY")
	assistant := os.Getenv("CHAT_PRINCIPAL_ASSISTANT")
	model := os.Getenv("CHAT_MODEL")
	var api *openai.Client
	if key != "" {
		api = openai.NewClient(key)
	}
	vsMap, _ := loadVectorStoreFile()
	fcMap, _ := loadFileCache()
	return &Client{
		api:         api,
		AssistantID: assistant,
		Model:       model,
		key:         key,
		httpClient:  &http.Client{Timeout: 60 * time.Second},
		vectorStore: vsMap,
		fileCache:   fcMap,
		sessBytes:   make(map[string]int64),
		sessFiles:   make(map[string]int),
		lastFile:    make(map[string]LastFileInfo),
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
		println("[OPENAI STREAM INIT ERROR] ", err.Error())
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
			println("[OPENAI FALLBACK ERROR] ", err2.Error())
			return nil, err
		}
		out := make(chan string, 1)
		go func() {
			defer close(out)
			if len(resp.Choices) > 0 {
				msg := resp.Choices[0].Message.Content
				if msg != "" {
					println("[OPENAI FALLBACK MSG] ", msg)
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
					println("[OPENAI STREAM END] ", err.Error())
				}
				break
			}
			if len(resp.Choices) == 0 {
				println("[OPENAI WARNING] empty choices in delta")
				continue
			}
			token := resp.Choices[0].Delta.Content
			if token != "" {
				anyToken = true
				println("[OPENAI TOKEN] ", token)
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
				println("[OPENAI FALLBACK ERROR] ", err.Error())
				return
			}
			if len(resp.Choices) > 0 {
				msg := resp.Choices[0].Message.Content
				if msg != "" {
					println("[OPENAI FALLBACK MSG] ", msg)
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

	c.vsMu.RLock()
	if id, ok := c.vectorStore[threadID]; ok {
		c.vsMu.RUnlock()
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
	c.lastFile[threadID] = LastFileInfo{ID: data.ID, Name: filepath.Base(filePath), At: time.Now()}
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
	resp, err := c.doJSON(ctx, http.MethodPost, "/threads", map[string]any{})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
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
	// create run
	payload := map[string]any{"assistant_id": c.AssistantID}
	if strings.TrimSpace(instructions) != "" {
		payload["instructions"] = instructions
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
			for _, c := range m.Content {
				if c.Type == "text" && c.Text.Value != "" {
					buf.WriteString(c.Text.Value)
				}
			}
			if s := buf.String(); s != "" {
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
	out := make(chan string, 1)
	go func() {
		defer close(out)
		strict := "Responde únicamente usando información recuperada de los documentos de este chat. Si no hay evidencia suficiente responde exactamente: 'No encontré información en el archivo adjunto.' Cita fragmentos cuando sea posible."
		// Bias to the most recently uploaded file if any
		c.lastMu.RLock()
		if lf, ok := c.lastFile[threadID]; ok && strings.TrimSpace(lf.Name) != "" {
			strict = strict + " Prioriza el archivo más reciente de este hilo ('" + lf.Name + "') y no pidas confirmación a menos que el usuario lo contradiga."
		}
		c.lastMu.RUnlock()
		text, err := c.runAndWait(ctx, threadID, strict, vsID)
		if err == nil && text != "" {
			out <- text
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
	out := make(chan string, 1)
	go func() {
		defer close(out)
		// Constrain the run to only use the vector store
		strict := "Responde únicamente usando información recuperada de los documentos de este chat. Si no hay evidencia suficiente responde exactamente: 'No encontré información en el archivo adjunto.' Cita fragmentos cuando sea posible."
		// Bias to this uploaded file
		if base := filepath.Base(filePath); base != "" {
			strict = strict + " Responde sobre el archivo recientemente subido '" + base + "' sin pedir confirmación, salvo que el usuario indique otro documento."
		}
		text, err := c.runAndWait(ctx, threadID, strict, vsID)
		if err == nil && text != "" {
			out <- text
		}
	}()
	return out, nil
}

// StreamAssistantJSON runs the assistant using file_search but WITHOUT overriding tool_resources (vector_store_ids),
// so it uses the assistant's pre-configured RAG. It enforces custom JSON-style instructions via the run.
func (c *Client) StreamAssistantJSON(ctx context.Context, threadID, userPrompt, jsonInstructions string) (<-chan string, error) {
	if c.key == "" || c.AssistantID == "" {
		return nil, errors.New("assistants not configured")
	}
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
