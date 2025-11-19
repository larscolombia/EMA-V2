-- Tabla para gestionar múltiples vector stores
CREATE TABLE IF NOT EXISTS vector_stores (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL COMMENT 'Nombre descriptivo del vector store',
    vector_store_id VARCHAR(255) NOT NULL UNIQUE COMMENT 'ID del vector store en OpenAI (vs_xxx)',
    description TEXT COMMENT 'Descripción del propósito del vector store',
    category VARCHAR(100) COMMENT 'Categoría (medicina_general, cardiologia, neurologia, etc)',
    is_default BOOLEAN DEFAULT FALSE COMMENT 'Si es el vector store por defecto',
    file_count INT DEFAULT 0 COMMENT 'Número de archivos en el vector store',
    total_bytes BIGINT DEFAULT 0 COMMENT 'Tamaño total en bytes',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_category (category),
    INDEX idx_default (is_default)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Tabla para tracking de archivos en cada vector store
CREATE TABLE IF NOT EXISTS vector_store_files (
    id INT AUTO_INCREMENT PRIMARY KEY,
    vector_store_id VARCHAR(255) NOT NULL COMMENT 'ID del vector store en OpenAI',
    file_id VARCHAR(255) NOT NULL COMMENT 'ID del archivo en OpenAI (file-xxx)',
    filename VARCHAR(500) NOT NULL COMMENT 'Nombre original del archivo',
    file_size BIGINT COMMENT 'Tamaño del archivo en bytes',
    status VARCHAR(50) DEFAULT 'processing' COMMENT 'Estado: processing, completed, failed',
    uploaded_by INT COMMENT 'ID del admin que subió el archivo',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_file_vector (vector_store_id, file_id),
    INDEX idx_vector_store (vector_store_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insertar el vector store global existente
INSERT INTO vector_stores (name, vector_store_id, description, category, is_default, file_count)
VALUES ('Biblioteca Médica General', 'vs_680fc484cef081918b2b9588b701e2f4', 'Vector store principal con libros de medicina general y especialidades', 'medicina_general', TRUE, 0)
ON DUPLICATE KEY UPDATE name = VALUES(name), description = VALUES(description);
