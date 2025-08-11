# conn package: MySQL connection

Overview
- Provides NewMySQL() to open a MySQL connection and ensure the target database exists.
- Creates the database if missing, then returns a live *sql.DB.

Environment variables
- DB_HOST: MySQL host (e.g., 127.0.0.1)
- DB_PORT: MySQL port (e.g., 3306)
- DB_USER: MySQL username
- DB_PASSWORD: MySQL password
- DB_NAME: Database name to create/use

How it works
- Builds a DSN without DB to connect as admin and CREATE DATABASE IF NOT EXISTS <DB_NAME>.
- Opens a second DSN including the DB_NAME with parseTime=true.
- Pings both connections to validate availability.

Good practices
- Use least-privileged DB_USER limited to the target schema.
- Avoid embedding credentials in code; keep them in .env or secret store.
- Set reasonable MySQL timeouts via DSN if needed (readTimeout, writeTimeout).
- Reuse the returned *sql.DB as a long-lived pool; close only on shutdown.

Architecture notes
- conn is a low-level infra package; it should not import app logic.
- It’s used by main.go to construct the DB and pass it to other packages.
