# sse package: Server-Sent Events

Overview
- Utility to stream tokens/messages over HTTP using SSE.

Environment variables
- None.

How it works
- Sets appropriate headers and uses gin.Context.Stream to emit "message" events while the channel provides data.

Good practices
- Keep events small and frequent; handle client disconnects.
- Use heartbeat/ping messages if needed in long streams.

Architecture notes
- Pure helper; used by chat handler for streaming responses.
