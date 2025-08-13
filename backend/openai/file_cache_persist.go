package openai

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const fileCacheFile = "data/filecache.json"

func loadFileCache() (map[string]string, error) {
	if persistDB != nil {
		return loadFileCacheDB(persistDB)
	}
	b, err := os.ReadFile(fileCacheFile)
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

func saveFileCache(m map[string]string) error {
	if persistDB != nil {
		return saveFileCacheDB(persistDB, m)
	}
	if err := os.MkdirAll(filepath.Dir(fileCacheFile), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fileCacheFile, b, 0o644)
}

func loadFileCacheDB(db *sql.DB) (map[string]string, error) {
	m := map[string]string{}
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS ai_filecache (ckey VARCHAR(191) PRIMARY KEY, file_id VARCHAR(255))`)
	rows, err := db.Query(`SELECT ckey, file_id FROM ai_filecache`)
	if err != nil {
		return m, nil
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err == nil {
			m[k] = v
		}
	}
	return m, nil
}

func saveFileCacheDB(db *sql.DB, m map[string]string) error {
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS ai_filecache (ckey VARCHAR(191) PRIMARY KEY, file_id VARCHAR(255))`)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM ai_filecache`); err != nil {
		return err
	}
	for k, v := range m {
		if _, err := tx.Exec(`INSERT INTO ai_filecache (ckey, file_id) VALUES (?, ?)`, k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}
