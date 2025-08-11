package conn

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// NewMySQL abre una nueva conexión a MySQL utilizando
// las variables de entorno.
func NewMySQL() (*sql.DB, error) {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")

	// Ensure database exists by connecting without DB and creating it if needed
	adminDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true", user, pass, host, port)
	adminDB, err := sql.Open("mysql", adminDSN)
	if err != nil {
		return nil, err
	}
	if err := adminDB.Ping(); err != nil {
		adminDB.Close()
		return nil, err
	}
	if _, err := adminDB.Exec("CREATE DATABASE IF NOT EXISTS `" + name + "` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		adminDB.Close()
		return nil, err
	}
	adminDB.Close()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, name)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
