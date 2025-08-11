# chat package: Assistant endpoints

Overview
- Endpoints: POST /asistente/start, POST /asistente/message.
- Generates responses using the openai client; supports streaming via SSE for /message.

Environment variables
- Inherited from openai package: OPENAI_API_KEY, CHAT_PRINCIPAL_ASSISTANT.

How it works
- /asistente/start: accepts {prompt}, gets stream, buffers to full text, returns thread_id + text.
- /asistente/message: accepts {thread_id, prompt}, streams tokens to client via Server-Sent Events.

Good practices
- Validate input sizes; set timeouts and error handling on streams.
- Authenticate requests if sensitive; throttle to prevent abuse.

Architecture notes
- Thin HTTP layer delegating to openai client; SSE helper in sse package.
