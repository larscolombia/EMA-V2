# openai package: LLM client

Overview
- Wraps go-openai client and exposes a simple streaming message API.

Environment variables
- OPENAI_API_KEY: API key for OpenAI.
- CHAT_PRINCIPAL_ASSISTANT: Model/assistant identifier used in requests.

How it works
- NewClient() reads env vars, creates go-openai client.
- StreamMessage(ctx, prompt) opens a streaming chat completion and returns a channel of tokens.

Good practices
- Validate env vars at startup; fail fast if missing.
- Consider rate limiting and retries.
- Avoid leaking secrets in logs; rotate keys periodically.

Architecture notes
- Consumed by chat handler; isolated from HTTP concerns.
