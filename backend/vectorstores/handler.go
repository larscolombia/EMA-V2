package vectorstores

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ema-backend/openai"

	"github.com/gin-gonic/gin"
)

// OpenAIClient interface para interactuar con OpenAI
type OpenAIClient interface {
	UploadFile(ctx context.Context, filePath, purpose string) (string, error)
	AddFileToVectorStore(ctx context.Context, vsID, fileID string) error
	PollVectorStoreFileIndexed(ctx context.Context, vsID, fileID string, timeout time.Duration) error
	DeleteFile(ctx context.Context, fileID string) error
	ListVectorStoreFilesDetailed(ctx context.Context, vsID string) ([]openai.VectorStoreFileDetail, error)
	GetFileInfo(ctx context.Context, fileID string) (*openai.FileInfoResponse, error)
}

type Handler struct {
	db *sql.DB
	ai OpenAIClient
}

func NewHandler(db *sql.DB, ai OpenAIClient) *Handler {
	return &Handler{db: db, ai: ai}
}

// ListVectorStores - GET /admin/vectorstores
func (h *Handler) ListVectorStores(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, name, vector_store_id, description, category, is_default, 
		       file_count, total_bytes, created_at, updated_at
		FROM vector_stores
		ORDER BY is_default DESC, created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar vector stores"})
		return
	}
	defer rows.Close()

	stores := []VectorStore{} // Inicializar con array vacío
	for rows.Next() {
		var vs VectorStore
		if err := rows.Scan(&vs.ID, &vs.Name, &vs.VectorStoreID, &vs.Description, &vs.Category,
			&vs.IsDefault, &vs.FileCount, &vs.TotalBytes, &vs.CreatedAt, &vs.UpdatedAt); err != nil {
			continue
		}
		stores = append(stores, vs)
	}

	c.JSON(http.StatusOK, gin.H{"data": stores})
}

// GetVectorStore - GET /admin/vectorstores/:id
func (h *Handler) GetVectorStore(c *gin.Context) {
	id := c.Param("id")

	var vs VectorStore
	err := h.db.QueryRow(`
		SELECT id, name, vector_store_id, description, category, is_default,
		       file_count, total_bytes, created_at, updated_at
		FROM vector_stores WHERE id = ?
	`, id).Scan(&vs.ID, &vs.Name, &vs.VectorStoreID, &vs.Description, &vs.Category,
		&vs.IsDefault, &vs.FileCount, &vs.TotalBytes, &vs.CreatedAt, &vs.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vector store no encontrado"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener vector store"})
		return
	}

	c.JSON(http.StatusOK, vs)
}

// CreateVectorStore - POST /admin/vectorstores
func (h *Handler) CreateVectorStore(c *gin.Context) {
	var req CreateVectorStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Crear vector store en OpenAI
	vsID, err := h.createOpenAIVectorStore(c.Request.Context(), req.Name)
	if err != nil {
		log.Printf("[vectorstores] Error creating OpenAI vector store: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear vector store en OpenAI"})
		return
	}

	// Guardar en BD
	result, err := h.db.Exec(`
		INSERT INTO vector_stores (name, vector_store_id, description, category, is_default, file_count, total_bytes)
		VALUES (?, ?, ?, ?, FALSE, 0, 0)
	`, req.Name, vsID, req.Description, req.Category)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar vector store"})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{
		"id":              id,
		"vector_store_id": vsID,
		"name":            req.Name,
	})
}

// UpdateVectorStore - PUT /admin/vectorstores/:id
func (h *Handler) UpdateVectorStore(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		IsDefault   *bool  `json:"is_default,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Si se marca como default, desmarcar los demás
	if req.IsDefault != nil && *req.IsDefault {
		_, _ = h.db.Exec("UPDATE vector_stores SET is_default = FALSE WHERE is_default = TRUE")
	}

	_, err := h.db.Exec(`
		UPDATE vector_stores 
		SET name = ?, description = ?, category = ?, is_default = COALESCE(?, is_default)
		WHERE id = ?
	`, req.Name, req.Description, req.Category, req.IsDefault, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar vector store"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// DeleteVectorStore - DELETE /admin/vectorstores/:id
func (h *Handler) DeleteVectorStore(c *gin.Context) {
	id := c.Param("id")

	// Verificar que no sea el default
	var isDefault bool
	err := h.db.QueryRow("SELECT is_default FROM vector_stores WHERE id = ?", id).Scan(&isDefault)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vector store no encontrado"})
		return
	}
	if isDefault {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se puede eliminar el vector store por defecto"})
		return
	}

	// Eliminar archivos asociados
	_, _ = h.db.Exec("DELETE FROM vector_store_files WHERE vector_store_id = (SELECT vector_store_id FROM vector_stores WHERE id = ?)", id)

	// Eliminar vector store
	_, err = h.db.Exec("DELETE FROM vector_stores WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar vector store"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// ListFiles - GET /admin/vectorstores/:id/files
func (h *Handler) ListFiles(c *gin.Context) {
	id := c.Param("id")

	// Obtener vector_store_id
	var vsID string
	err := h.db.QueryRow("SELECT vector_store_id FROM vector_stores WHERE id = ?", id).Scan(&vsID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vector store no encontrado"})
		return
	}

	// Obtener archivos desde OpenAI
	files, err := h.ai.ListVectorStoreFilesDetailed(c.Request.Context(), vsID)
	if err != nil {
		log.Printf("[vectorstores] Error listing files from OpenAI: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar archivos"})
		return
	}

	// Sincronizar con BD
	h.syncFilesWithDB(c.Request.Context(), vsID, files)

	// Actualizar contadores después del sync
	h.updateVectorStoreCounters(vsID)

	// Obtener archivos de BD con metadata adicional
	rows, err := h.db.Query(`
		SELECT id, vector_store_id, file_id, filename, file_size, status, uploaded_by, created_at, updated_at
		FROM vector_store_files
		WHERE vector_store_id = ?
		ORDER BY created_at DESC
	`, vsID)
	if err != nil {
		log.Printf("[vectorstores] Error querying DB files: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener archivos"})
		return
	}
	defer rows.Close()

	dbFiles := []VectorStoreFile{} // Inicializar con array vacío en lugar de nil
	for rows.Next() {
		var f VectorStoreFile
		if err := rows.Scan(&f.ID, &f.VectorStoreID, &f.FileID, &f.Filename, &f.FileSize,
			&f.Status, &f.UploadedBy, &f.CreatedAt, &f.UpdatedAt); err != nil {
			log.Printf("[vectorstores] Error scanning file row: %v", err)
			continue
		}
		dbFiles = append(dbFiles, f)
	}

	log.Printf("[vectorstores] ListFiles vs=%s openai_count=%d db_count=%d", vsID, len(files), len(dbFiles))
	c.JSON(http.StatusOK, gin.H{"data": dbFiles})
}

// UploadFile - POST /admin/vectorstores/:id/upload
func (h *Handler) UploadFile(c *gin.Context) {
	id := c.Param("id")

	// Obtener vector_store_id
	var vsID string
	err := h.db.QueryRow("SELECT vector_store_id FROM vector_stores WHERE id = ?", id).Scan(&vsID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vector store no encontrado"})
		return
	}

	// Obtener archivo del form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Archivo requerido"})
		return
	}

	// Validar extensión
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" && ext != ".txt" && ext != ".md" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Solo se permiten archivos PDF, TXT o MD"})
		return
	}

	// Guardar temporalmente
	tempDir := filepath.Join(os.TempDir(), "ema_uploads")
	os.MkdirAll(tempDir, 0755)
	tempPath := filepath.Join(tempDir, fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename))

	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar archivo"})
		return
	}
	defer os.Remove(tempPath)

	// Subir a OpenAI
	fileID, err := h.ai.UploadFile(c.Request.Context(), tempPath, "assistants")
	if err != nil {
		log.Printf("[vectorstores] Error uploading to OpenAI: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al subir archivo a OpenAI"})
		return
	}

	// Agregar al vector store
	if err := h.ai.AddFileToVectorStore(c.Request.Context(), vsID, fileID); err != nil {
		log.Printf("[vectorstores] Error adding file to vector store: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al agregar archivo al vector store"})
		return
	}

	// Guardar en BD
	_, err = h.db.Exec(`
		INSERT INTO vector_store_files (vector_store_id, file_id, filename, file_size, status)
		VALUES (?, ?, ?, ?, 'processing')
	`, vsID, fileID, file.Filename, file.Size)

	if err != nil {
		log.Printf("[vectorstores] Error saving file to DB: %v", err)
	}

	// Actualizar contadores
	h.updateVectorStoreCounters(vsID)

	// Esperar indexación en background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := h.ai.PollVectorStoreFileIndexed(ctx, vsID, fileID, 2*time.Minute); err == nil {
			h.db.Exec("UPDATE vector_store_files SET status = 'completed' WHERE file_id = ?", fileID)
		} else {
			h.db.Exec("UPDATE vector_store_files SET status = 'failed' WHERE file_id = ?", fileID)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"file_id":  fileID,
		"filename": file.Filename,
		"status":   "processing",
	})
}

// DeleteFile - DELETE /admin/vectorstores/:id/files/:fileId
func (h *Handler) DeleteFile(c *gin.Context) {
	id := c.Param("id")
	fileID := c.Param("fileId")

	// Obtener vector_store_id
	var vsID string
	err := h.db.QueryRow("SELECT vector_store_id FROM vector_stores WHERE id = ?", id).Scan(&vsID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vector store no encontrado"})
		return
	}

	// Eliminar de OpenAI
	if err := h.ai.DeleteFile(c.Request.Context(), fileID); err != nil {
		log.Printf("[vectorstores] Error deleting file from OpenAI: %v", err)
		// Continuar para eliminarlo de BD de todas formas
	}

	// Eliminar de BD
	_, err = h.db.Exec("DELETE FROM vector_store_files WHERE file_id = ? AND vector_store_id = ?", fileID, vsID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar archivo"})
		return
	}

	// Actualizar contadores
	h.updateVectorStoreCounters(vsID)

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// Helper: crear vector store en OpenAI
func (h *Handler) createOpenAIVectorStore(ctx context.Context, name string) (string, error) {
	// Usar API directa de OpenAI
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not configured")
	}

	payload := map[string]interface{}{
		"name": name,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/vector_stores", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenAI-Beta", "assistants=v2")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error: %s", string(bodyBytes))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ID, nil
}

// Helper: sincronizar archivos con BD
func (h *Handler) syncFilesWithDB(ctx context.Context, vsID string, files []openai.VectorStoreFileDetail) {
	log.Printf("[vectorstores] syncFilesWithDB vs=%s incoming_files=%d", vsID, len(files))

	// Obtener IDs de archivos existentes en BD
	rows, _ := h.db.Query("SELECT file_id FROM vector_store_files WHERE vector_store_id = ?", vsID)
	existingIDs := make(map[string]bool)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				existingIDs[id] = true
			}
		}
	}
	log.Printf("[vectorstores] syncFilesWithDB existing_count=%d", len(existingIDs))

	// Solo insertar archivos nuevos que no existen en BD
	newCount := 0
	updateCount := 0
	for _, f := range files {
		if existingIDs[f.ID] {
			// Ya existe, solo actualizar status si cambió
			h.db.Exec("UPDATE vector_store_files SET status = ? WHERE file_id = ?", f.Status, f.ID)
			updateCount++
			continue
		}

		// Archivo nuevo: obtener metadata desde OpenAI Files API
		var filename string
		var fileSize int64

		if fileInfo, err := h.ai.GetFileInfo(ctx, f.ID); err == nil {
			filename = fileInfo.Filename
			fileSize = fileInfo.Bytes
			log.Printf("[vectorstores] Fetched metadata for new file %s: %s (%d bytes)", f.ID, filename, fileSize)
		} else {
			log.Printf("[vectorstores] Warning: Could not fetch metadata for %s: %v", f.ID, err)
			filename = f.ID // Fallback al ID
			fileSize = 0
		}

		result, err := h.db.Exec(`
			INSERT INTO vector_store_files (vector_store_id, file_id, filename, file_size, status)
			VALUES (?, ?, ?, ?, ?)
		`, vsID, f.ID, filename, fileSize, f.Status)

		if err != nil {
			log.Printf("[vectorstores] Error inserting file %s: %v", f.ID, err)
		} else {
			newCount++
			lastID, _ := result.LastInsertId()
			log.Printf("[vectorstores] Inserted new file id=%d file_id=%s filename=%s", lastID, f.ID, filename)
		}
	}

	log.Printf("[vectorstores] syncFilesWithDB complete: new=%d updated=%d", newCount, updateCount)
}

// Helper: actualizar contadores
func (h *Handler) updateVectorStoreCounters(vsID string) {
	var count int
	var totalBytes int64

	h.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(file_size), 0)
		FROM vector_store_files
		WHERE vector_store_id = ?
	`, vsID).Scan(&count, &totalBytes)

	result, err := h.db.Exec(`
		UPDATE vector_stores
		SET file_count = ?, total_bytes = ?
		WHERE vector_store_id = ?
	`, count, totalBytes, vsID)

	if err != nil {
		log.Printf("[vectorstores] Error updating counters for %s: %v", vsID, err)
	} else {
		rows, _ := result.RowsAffected()
		log.Printf("[vectorstores] Updated counters for %s: count=%d bytes=%d rows_affected=%d", vsID, count, totalBytes, rows)
	}
}

// RegisterRoutes registra las rutas del módulo
func RegisterRoutes(r *gin.RouterGroup, db *sql.DB, ai OpenAIClient) {
	h := NewHandler(db, ai)

	r.GET("/vectorstores", h.ListVectorStores)
	r.GET("/vectorstores/:id", h.GetVectorStore)
	r.POST("/vectorstores", h.CreateVectorStore)
	r.PUT("/vectorstores/:id", h.UpdateVectorStore)
	r.DELETE("/vectorstores/:id", h.DeleteVectorStore)

	r.GET("/vectorstores/:id/files", h.ListFiles)
	r.POST("/vectorstores/:id/upload", h.UploadFile)
	r.DELETE("/vectorstores/:id/files/:fileId", h.DeleteFile)
}
