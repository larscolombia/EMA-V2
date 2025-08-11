# login package: Auth and sessions

Overview
- Provides endpoints: POST /login, GET /session, POST /logout, POST /register, POST /password/forgot.
- Maintains a simple in-memory token store (not persistent) mapping token -> email.

Environment variables
- None directly. Uses DB via migrations package to fetch users.

How it works
- /login: validates credentials against users table; on success generates a random token and returns { token, user }.
- /session: reads Bearer token; returns the user if token is valid.
- /logout: removes token from in-memory store.
- /register: creates a new user if email not taken.
- /password/forgot: dummy acknowledgment without sending emails.

Good practices
- Replace in-memory sessions with signed JWTs or DB-backed sessions with expiry.
- Hash passwords (e.g., bcrypt) instead of storing plaintext (current code is demo-only).
- Rate-limit login and register endpoints, add audit logging.
- Validate and normalize emails; avoid leaking whether an email exists.

Architecture notes
- login depends on migrations for user queries (DB). It should remain thin.
- Share auth helpers (e.g., token validation) with other packages that need auth.
