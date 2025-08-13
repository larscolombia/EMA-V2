package openai

import "database/sql"

// persistDB, when set, enables DB-backed persistence for vector stores and file cache.
var persistDB *sql.DB

// SetPersistDB allows main() to enable DB-backed persistence.
func SetPersistDB(db *sql.DB) {
	persistDB = db
}
