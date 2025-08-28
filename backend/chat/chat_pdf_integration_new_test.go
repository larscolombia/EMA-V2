//go:build integration
// +build integration

package chat

import (
  "bytes"
  "mime/multipart"
  "net/http/httptest"
  "os"
  "path/filepath"
  "strings"
  "testing"
  "time"
  "github.com/joho/godotenv"
  "github.com/gin-gonic/gin"
  openaipkg "ema-backend/openai"
)

// Integration test que lee archivo temporal para obtener respuesta completa
func TestIntegrationPDFUploadStructured(t *testing.T) {
  _ = godotenv.Load("../.env") // intentar cargar desde raíz backend
  if os.Getenv("OPENAI_API_KEY") == "" || os.Getenv("CHAT_PRINCIPAL_ASSISTANT") == "" {
    t.Skip("OPENAI_API_KEY/CHAT_PRINCIPAL_ASSISTANT no definidos (carga .env falló o variables vacías)")
  }
  os.Setenv("STRUCTURED_PDF_SUMMARY", "1")
  os.Setenv("TEST_CAPTURE_FULL", "1")
  gin.SetMode(gin.TestMode)
  ai := openaipkg.NewClient()
  h := NewHandler(ai)

  // Cuando se ejecuta dentro del paquete chat, los archivos viven en ../files
  pdfPath := filepath.Join("..", "files", "Propuesta Comerccial LARS - Inforcid.pdf")
  if _, err := os.Stat(pdfPath); err != nil { t.Fatalf("PDF no encontrado: %v", err) }

  body := &bytes.Buffer{}
  w := multipart.NewWriter(body)
  fw, _ := w.CreateFormFile("file", filepath.Base(pdfPath))
  data, _ := os.ReadFile(pdfPath)
  fw.Write(data) // enviar completo
  threadId := "test-thread-int-full"
  w.WriteField("thread_id", threadId)
  w.Close()

  req := httptest.NewRequest("POST", "/asistente/message", body)
  req.Header.Set("Content-Type", w.FormDataContentType())
  rr := httptest.NewRecorder()
  c,_ := gin.CreateTestContext(rr)
  c.Request = req

  done := make(chan struct{})
  go func(){ h.Message(c); close(done) }()
  select {
  case <-done:
  case <-time.After(90 * time.Second):
    t.Fatal("timeout esperando respuesta del assistant")
  }
  if rr.Code != 200 { t.Fatalf("status inesperado: %d body=%s", rr.Code, rr.Body.String()) }
  
  // Leer archivo temporal generado por cliente OpenAI
  // El threadId real será mapeado, buscar archivos que empiecen con assistant_full_thread_
  matches, _ := filepath.Glob("/tmp/assistant_full_thread_*.txt")
  if len(matches) == 0 {
    t.Fatal("archivo temporal con respuesta completa no encontrado - verifica que TEST_CAPTURE_FULL=1")
  }
  
  // Tomar el más reciente
  var newestFile string
  var newestTime time.Time
  for _, f := range matches {
    if info, err := os.Stat(f); err == nil && info.ModTime().After(newestTime) {
      newestTime = info.ModTime()
      newestFile = f
    }
  }
  
  if newestFile == "" { t.Fatal("no se pudo determinar archivo temporal más reciente") }
  
  fullResp, err := os.ReadFile(newestFile)
  if err != nil { t.Fatalf("error leyendo archivo temporal: %v", err) }
  
  aiText := string(fullResp)
  t.Logf("AI respuesta completa (%d chars):\n%s", len(aiText), aiText)
  
  // Limpiar archivo temporal
  os.Remove(newestFile)
  
  // Validaciones básicas
  if !strings.Contains(aiText, "1. Resumen Ejecutivo") {
    t.Errorf("respuesta no contiene formato estructurado esperado")
  }
  if len(aiText) < 1000 {
    t.Errorf("respuesta muy corta (%d chars), posiblemente truncada", len(aiText))
  }
}
