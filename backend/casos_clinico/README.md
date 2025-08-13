# Casos Clínicos (Analítico e Interactivo)

Endpoints y contratos que satisface el frontend Flutter.

## Variables de entorno
- `CASOS_CLINICOS_ANALITICO`: ID del Assistant para flujo analítico (static). Opcional.
- `CASOS_CLINICOS_INTERACTIVO`: ID del Assistant para flujo interactivo. Si no se define, reutiliza el analítico.

## Rutas
- POST `/caso-clinico`
  - Body: `{ "age": string, "sex": string, "type": string, "pregnant": boolean }`
  - Respuesta: `{ "case": { id, title, type:"static", age, sex, gestante|pregnant, is_real, anamnesis, physical_examination, diagnostic_tests, final_diagnosis, management }, "thread_id": string }`

- POST `/casos-clinicos/conversar`
  - Body: `{ "thread_id": string, "mensaje": string }`
  - Respuesta: `{ "respuesta": { "text": string } }`

- POST `/casos-clinicos/interactivo`
  - Body: `{ "age": string, "sex": string, "type": string, "pregnant": boolean }`
  - Respuesta: `{ "case": {..., type:"interactive"}, "data": { "questions": { "texto": string, "tipo": "open_ended"|"single_choice", "opciones": string[] } }, "thread_id": string }`

- POST `/casos-clinicos/interactivo/conversar`
  - Body: `{ "thread_id": string, "mensaje": string }`
  - Respuesta: `{ "data": { "feedback": string, "question": { "texto": string, "tipo": "open_ended"|"single_choice", "opciones": string[] }, "thread_id": string } }`

## Robustez y normalización
- Respuestas del assistant se fuerzan a JSON estricto; si no es válido, se intenta una reparación automática una vez.
- Se garantizan mínimos por defecto para evitar fallos del UI (claves y tipos presentes).
- Hook de validación de cuota disponible vía `SetQuotaValidator(ctx, c, flow)` con `flow` ∈ {`analytical_generate`,`analytical_chat`,`interactive_generate`,`interactive_chat`}.

## Pruebas
- Tests unitarios en este paquete validan 200 OK y formas mínimas.
- Tests de integración de `openai/` y `testsapi/` se omiten si falta `OPENAI_API_KEY`.
