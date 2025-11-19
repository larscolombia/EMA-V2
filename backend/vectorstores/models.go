package vectorstores

import "time"

// VectorStore representa un vector store de OpenAI gestionado en la base de datos
type VectorStore struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	VectorStoreID string    `json:"vector_store_id"`
	Description   string    `json:"description"`
	Category      string    `json:"category"`
	IsDefault     bool      `json:"is_default"`
	FileCount     int       `json:"file_count"`
	TotalBytes    int64     `json:"total_bytes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// VectorStoreFile representa un archivo dentro de un vector store
type VectorStoreFile struct {
	ID            int       `json:"id"`
	VectorStoreID string    `json:"vector_store_id"`
	FileID        string    `json:"file_id"`
	Filename      string    `json:"filename"`
	FileSize      int64     `json:"file_size"`
	Status        string    `json:"status"`                // processing, completed, failed
	UploadedBy    *int      `json:"uploaded_by,omitempty"` // Cambiado a *int para soportar NULL
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateVectorStoreRequest para crear un nuevo vector store
type CreateVectorStoreRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// UploadFileRequest para subir archivo a un vector store
type UploadFileRequest struct {
	VectorStoreID string `json:"vector_store_id" binding:"required"`
	// El archivo viene en multipart/form-data
}
