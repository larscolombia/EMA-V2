# login package: Auth and sessions

Overview
- Provides endpoints: POST /login, GET /session, POST /logout, POST /register, POST /password/forgot, POST /password/reset, POST /password/change.
- Maintains a simple in-memory token store (not persistent) mapping token -> email.
- Implements password reset flow with token-based email verification.

Environment variables
- None directly. Uses DB via migrations package to fetch users.
- Uses email package which requires SMTP variables for sending recovery emails.
- FRONTEND_URL: URL base del frontend para generar enlaces de recuperación (default: http://localhost:5173)

How it works
- /login: validates credentials against users table; on success generates a random token and returns { token, user }.
- /session: reads Bearer token; returns the user if token is valid.
- /logout: removes token from in-memory store.
- /register: creates un nuevo usuario si el correo no está registrado y envía correo de bienvenida.
- /password/forgot: generates a unique reset token, stores it in password_resets table with 1-hour expiry, and sends recovery email with link.
- /password/reset: validates token from password_resets table, updates user password, deletes used token, and sends confirmation email.
- /password/change: actualiza la contraseña del usuario autenticado y envía notificación.

Good practices
- Replace in-memory sessions with signed JWTs or DB-backed sessions with expiry.
- Hash passwords (e.g., bcrypt) instead of storing plaintext (current code is demo-only).
- Password reset tokens expire after 1 hour for security.
- Used tokens are automatically deleted after successful password reset.
- Email enumeration is prevented by always returning success message regardless of email existence.
- Rate-limit login and register endpoints, add audit logging.
- Validate and normalize emails; avoid leaking whether an email exists.

Architecture notes
- login depends on migrations for user queries (DB). It should remain thin.
- Share auth helpers (e.g., token validation) with other packages that need auth.
