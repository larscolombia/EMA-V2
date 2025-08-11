# Backend architecture overview

Packages
- conn: DB connection factory reading env vars and ensuring DB exists.
- migrations: schema creation, seeds, and user/profile queries.
- login: auth endpoints with in-memory sessions.
- profile: user profile CRUD and media upload; serves relative paths under /media via main.
- subscriptions: plans and subscriptions CRUD, cancel endpoint, and a stub /checkout.
- countries: static country list for the app.
- openai: LLM client (go-openai) reading OPENAI_API_KEY and CHAT_PRINCIPAL_ASSISTANT.
- chat: assistant HTTP endpoints; uses SSE for streaming.
- sse: helper for Server-Sent Events.
- marketing: envía correos y pushes periódicos a usuarios con plan gratuito.

Boot sequence (main.go)
1) Load .env (optional).
2) Connect MySQL via conn.NewMySQL().
3) migrations.Init(db); Migrate(); SeedDefaultUser(); SeedDefaultPlans().
4) Register routes for auth, profile, media, subscriptions, countries, chat.
5) Start Gin server on PORT (default 8080).

Env variables (.env)
- DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
- PORT (optional, default 8080)
- OPENAI_API_KEY, CHAT_PRINCIPAL_ASSISTANT (assistant ID), CHAT_MODEL (model name) — required for chat
- SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, SMTP_FROM (opcional para correos)

OpenAI setup (.env)
- Add your OpenAI credentials to backend/.env:
	- OPENAI_API_KEY=sk-...your-key...
	- CHAT_PRINCIPAL_ASSISTANT=asst_... (assistant id)
	- CHAT_MODEL=gpt-4o-mini (or the model you want to use)
- Restart the backend after changes so env vars are loaded.
- If you don’t provide these, the chat endpoints will still create a thread_id but won’t call the OpenAI API.

Quick check
- Health: GET http://localhost:8080/health → {"status":"ok"}
- Profile: Bearer <token> → GET /user-detail/1
- Start chat: POST /asistente/start with {"prompt":"hola"} → returns { thread_id, text }
	- If text is empty, verify OPENAI_API_KEY and CHAT_PRINCIPAL_ASSISTANT are set.

Good practices
- Replace plaintext passwords with bcrypt and sessions with JWTs/expirable storage.
- Add request auth middleware for protected routes.
- Introduce versioned migrations; add logging/metrics.
- Parameterize media root and CDN strategy.
