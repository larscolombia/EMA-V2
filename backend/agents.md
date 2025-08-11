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

Boot sequence (main.go)
1) Load .env (optional).
2) Connect MySQL via conn.NewMySQL().
3) migrations.Init(db); Migrate(); SeedDefaultUser(); SeedDefaultPlans().
4) Register routes for auth, profile, media, subscriptions, countries, chat.
5) Start Gin server on PORT (default 8080).

Env variables (.env)
- DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
- PORT (optional, default 8080)
- OPENAI_API_KEY, CHAT_PRINCIPAL_ASSISTANT (optional for chat)

Good practices
- Replace plaintext passwords with bcrypt and sessions with JWTs/expirable storage.
- Add request auth middleware for protected routes.
- Introduce versioned migrations; add logging/metrics.
- Parameterize media root and CDN strategy.
