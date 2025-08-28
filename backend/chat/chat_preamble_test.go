package chat

import (
  "bytes"
  "mime/multipart"
  "net/http/httptest"
  "os"
  "path/filepath"
  "testing"
  "github.com/gin-gonic/gin"
)

// Reutiliza mockAIPrompt del otro test.
func TestPDFPreambleApplied(t *testing.T){
  gin.SetMode(gin.TestMode)
  mk := &mockAIPrompt{AssistantID: "asst_dummy"}
  h := NewHandler(mk)
  os.Setenv("DOC_SUMMARY_PREAMBLE", "INSTRUCCION_GLOBAL_TEST")
  os.Setenv("STRUCTURED_PDF_SUMMARY", "1")
  pdfPath := filepath.Join("files", "Propuesta Comerccial LARS - Inforcid.pdf")
  if _, err := os.Stat(pdfPath); err != nil { t.Skip("PDF no disponible") }

  body := &bytes.Buffer{}
  w := multipart.NewWriter(body)
  fw, _ := w.CreateFormFile("file", filepath.Base(pdfPath))
  data, _ := os.ReadFile(pdfPath)
  fw.Write(data[:min(1024,len(data))])
  w.WriteField("thread_id", "thread_local")
  w.Close()
  req := httptest.NewRequest("POST", "/asistente/message", body)
  req.Header.Set("Content-Type", w.FormDataContentType())
  rr := httptest.NewRecorder()
  c,_ := gin.CreateTestContext(rr)
  c.Request = req
  h.Message(c)
  if mk.lastPrompt == "" { t.Fatal("prompt vacío") }
  if mk.lastPrompt[:len("INSTRUCCION_GLOBAL_TEST")] != "INSTRUCCION_GLOBAL_TEST" { t.Errorf("preamble no aplicado: %s", mk.lastPrompt) }
  // limpiar
  os.Unsetenv("DOC_SUMMARY_PREAMBLE")
}
