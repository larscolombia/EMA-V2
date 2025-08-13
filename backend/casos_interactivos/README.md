# Casos Interactivos (nuevo flujo)

Rutas nuevas para un flujo de caso clínico interactivo con contrato JSON estricto por turno.

## Variable de entorno
- `CASOS_INTERACTIVOS_ASSISTANT`: ID del Assistant específico para este flujo (por ejemplo: `asst_Zd0Mv8mRKjceEalmgGEICxaw`).
- `CASOS_INTERACTIVOS_MAX_PREGUNTAS`: número máximo de preguntas del caso (default: 4). Al llegar al límite, se responde con `finish: 1` y `{ hallazgos: {}, pregunta: {} }`.

## Rutas
- POST `/casos-interactivos/iniciar`
  - Body: `{ "age": string, "sex": string, "type": string, "pregnant": boolean }`
  - Respuesta:
    - `{ "case": {..., type:"interactive"}, "data": { "feedback": string, "next": { "hallazgos": object, "pregunta": { "tipo": "single-choice", "texto": string, "opciones": string[] } }, "finish": 0 }, "thread_id": string }`

- POST `/casos-interactivos/mensaje`
  - Body: `{ "thread_id": string, "mensaje": string }`
  - Respuesta:
    - `{ "data": { "feedback": string, "next": { "hallazgos": object, "pregunta": { "tipo": "single-choice"|"open_ended", "texto": string, "opciones": string[] } }, "finish": 0|1, "thread_id": string } }`

## Robustez
- Forzamos JSON estricto; si falla, reparamos una vez y, de no ser posible, devolvemos valores mínimos válidos.
- Siempre se incluye `thread_id` en las respuestas.

## Pruebas
- `handler_test.go`: valida respuestas 200 y forma mínima.
