package openai

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

var persistMu sync.Mutex

const vectorStoreFile = "data/vectorstores.json"

// loadVectorStoreFile loads persistent mapping thread_id -> vector_store_id.
func loadVectorStoreFile() (map[string]string, error) {
	if persistDB != nil {
		return loadVectorStoreDB(persistDB)
	}
	b, err := os.ReadFile(vectorStoreFile)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]string{}
	}
	return m, nil
}

// saveVectorStoreFile persists the mapping.
func saveVectorStoreFile(m map[string]string) error {
	if persistDB != nil {
		return saveVectorStoreDB(persistDB, m)
	}
	if err := os.MkdirAll(filepath.Dir(vectorStoreFile), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(vectorStoreFile, b, 0o644)
}

// DB adapters (simple key-value table)
func loadVectorStoreDB(db *sql.DB) (map[string]string, error) {
	m := map[string]string{}
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS ai_vectorstores (thread_id VARCHAR(191) PRIMARY KEY, vector_store_id VARCHAR(255))`)
	rows, err := db.Query(`SELECT thread_id, vector_store_id FROM ai_vectorstores`)
	if err != nil {
		return m, nil
	}
	defer rows.Close()
	for rows.Next() {
		var t, v string
		if err := rows.Scan(&t, &v); err == nil {
			m[t] = v
		}
	}
	return m, nil
}

func saveVectorStoreDB(db *sql.DB, m map[string]string) error {
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS ai_vectorstores (thread_id VARCHAR(191) PRIMARY KEY, vector_store_id VARCHAR(255))`)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM ai_vectorstores`); err != nil {
		return err
	}
	for t, v := range m {
		if _, err := tx.Exec(`INSERT INTO ai_vectorstores (thread_id, vector_store_id) VALUES (?, ?)`, t, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}
