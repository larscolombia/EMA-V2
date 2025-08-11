# openai package: LLM client

Overview
- Wraps go-openai client and exposes a simple streaming message API.

Environment variables
- OPENAI_API_KEY: API key for OpenAI.
- CHAT_PRINCIPAL_ASSISTANT: Assistant ID (asst_...) used by your workflow.
- CHAT_MODEL: Chat model name (e.g., gpt-4o-mini). If omitted and CHAT_PRINCIPAL_ASSISTANT is an assistant ID, backend defaults to gpt-4o-mini.

Example .env
```
# Database
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=root
DB_NAME=ema

# Server
PORT=8080

# OpenAI
OPENAI_API_KEY=sk-xxx_your_api_key
# If you use an Assistant ID, set it here and also set CHAT_MODEL
CHAT_PRINCIPAL_ASSISTANT=asst_xxx
CHAT_MODEL=gpt-4o-mini
```

How it works
- NewClient() reads env vars, creates go-openai client.
- StreamMessage(ctx, prompt) opens a streaming chat completion and returns a channel of tokens.

Good practices
- Validate env vars at startup; fail fast if missing.
- Consider rate limiting and retries.
- Avoid leaking secrets in logs; rotate keys periodically.

Troubleshooting
- 500 on POST /asistente/start: ensure both OPENAI_API_KEY and CHAT_PRINCIPAL_ASSISTANT are present and valid.
- If you need to test without OpenAI, the backend will still return a thread_id; then sending messages will stream an empty response.

Architecture notes
- Consumed by chat handler; isolated from HTTP concerns.
