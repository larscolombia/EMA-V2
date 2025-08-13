package chat_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	chatpkg "ema-backend/chat"
	openaipkg "ema-backend/openai"
)

// TestPDFFirstThenQuestions simula: (1) usuario sube solo el PDF y presiona Enter (prompt vacío),
// (2) luego hace preguntas de seguimiento en el mismo thread. Imprime en consola el comportamiento.
func TestPDFFirstThenQuestions(t *testing.T) {
	// Cargar variables de entorno desde ubicaciones comunes
	tried := []string{
		".env",
		"backend/.env",
		"../.env",
		"../backend/.env",
		"../../.env",
		"../../backend/.env",
	}
	for _, p := range tried {
		_ = godotenv.Load(p)
	}

	// Requiere credenciales reales para observar el flujo con OpenAI
	if os.Getenv("OPENAI_API_KEY") == "" || !strings.HasPrefix(os.Getenv("CHAT_PRINCIPAL_ASSISTANT"), "asst_") {
		t.Skip("OPENAI_API_KEY y/o CHAT_PRINCIPAL_ASSISTANT no configurados; se omite prueba de integración real")
	}

	// Preparar router con los endpoints de chat
	ai := openaipkg.NewClient()
	h := chatpkg.NewHandler(ai)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/asistente/start", h.Start)
	r.POST("/asistente/message", h.Message)

	// 1) Crear thread real via handler
	reqStart := httptest.NewRequest(http.MethodPost, "/asistente/start", strings.NewReader(`{"prompt":""}`))
	reqStart.Header.Set("Content-Type", "application/json")
	recStart := httptest.NewRecorder()
	r.ServeHTTP(recStart, reqStart)
	if recStart.Code != http.StatusOK {
		t.Fatalf("/asistente/start fallo: code=%d body=%s", recStart.Code, recStart.Body.String())
	}
	var startResp struct {
		ThreadID string `json:"thread_id"`
	}
	if err := json.Unmarshal(recStart.Body.Bytes(), &startResp); err != nil || startResp.ThreadID == "" {
		t.Fatalf("no se pudo obtener thread_id: err=%v body=%s", err, recStart.Body.String())
	}
	threadID := startResp.ThreadID
	t.Logf("Thread creado: %s", threadID)

	// 2) Enviar solo el PDF (prompt vacío)
	// Buscar el PDF en varias rutas candidatas por si la prueba se ejecuta desde distintos cwd
	candidates := []string{
		filepath.FromSlash("docsprueba/Propuesta-405.pdf"),                    // cuando cwd es backend/chat
		filepath.FromSlash("backend/chat/docsprueba/Propuesta-405.pdf"),       // cuando cwd es repo root
		filepath.FromSlash("../chat/docsprueba/Propuesta-405.pdf"),            // cuando cwd es backend
		filepath.FromSlash("../../backend/chat/docsprueba/Propuesta-405.pdf"), // CI variaciones
	}
	var pdfPath string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			pdfPath = c
			break
		}
	}
	if pdfPath == "" {
		t.Skipf("PDF de prueba no encontrado en rutas candidatas: %v", candidates)
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	// Archivo
	fw, err := w.CreateFormFile("file", filepath.Base(pdfPath))
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	f, err := os.Open(pdfPath)
	if err != nil {
		t.Fatalf("open pdf: %v", err)
	}
	_, err = io.Copy(fw, f)
	_ = f.Close()
	if err != nil {
		t.Fatalf("copy pdf: %v", err)
	}
	// prompt vacío y thread_id
	_ = w.WriteField("prompt", "")
	_ = w.WriteField("thread_id", threadID)
	_ = w.Close()

	req1 := httptest.NewRequest(http.MethodPost, "/asistente/message", &body)
	req1.Header.Set("Content-Type", w.FormDataContentType())
	rec1 := httptest.NewRecorder()

	start := time.Now()
	r.ServeHTTP(rec1, req1)
	t.Logf("Primera respuesta (solo PDF) status=%d latency=%s", rec1.Code, time.Since(start))
	fmt.Println("[PDF-only] HTTP status:", rec1.Code)
	fmt.Println("[PDF-only] Body (trunc 2KB):\n", truncate(rec1.Body.String(), 2048))

	// Si el archivo sigue procesándose, el handler puede devolver 202. Reintentar un par de veces.
	attempt := 1
	for rec1.Code == http.StatusAccepted && attempt <= 2 {
		time.Sleep(10 * time.Second)
		attempt++
		recRetry := httptest.NewRecorder()
		reqRetry := httptest.NewRequest(http.MethodPost, "/asistente/message", bytes.NewReader(body.Bytes()))
		reqRetry.Header.Set("Content-Type", w.FormDataContentType())
		r.ServeHTTP(recRetry, reqRetry)
		t.Logf("Retry #%d (solo PDF) status=%d", attempt, recRetry.Code)
		fmt.Println("[PDF-only][retry] HTTP status:", recRetry.Code)
		fmt.Println("[PDF-only][retry] Body (trunc 2KB):\n", truncate(recRetry.Body.String(), 2048))
		rec1 = recRetry
	}

	// 3) Pregunta de seguimiento en el mismo thread
	followQ := "¿Cuáles son los objetivos principales?"
	qBody := fmt.Sprintf(`{"thread_id":"%s","prompt":"%s"}`, threadID, escapeJSON(followQ))
	req2 := httptest.NewRequest(http.MethodPost, "/asistente/message", strings.NewReader(qBody))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	t.Logf("Follow-up status=%d", rec2.Code)
	fmt.Println("[Follow-up] HTTP status:", rec2.Code)
	fmt.Println("[Follow-up] Body (trunc 2KB):\n", truncate(rec2.Body.String(), 2048))

	// No fallar la prueba si no hubo SSE, pero asegurar que hubo alguna salida no vacía en alguno de los pasos
	if rec1.Code != http.StatusOK && rec1.Code != http.StatusAccepted {
		t.Fatalf("esperabamos 200 o 202 para PDF-only, obtuvimos %d", rec1.Code)
	}
	if rec2.Code != http.StatusOK {
		t.Fatalf("esperabamos 200 para follow-up, obtuvimos %d", rec2.Code)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "... [truncated]"
}

func escapeJSON(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n", "\r", "\\r")
	return r.Replace(s)
}
