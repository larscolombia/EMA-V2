# Casos Clínicos (Analítico e Interactivo)

Endpoints y contratos que satisface el frontend Flutter.

## Variables de entorno
- `CASOS_CLINICOS_ANALITICO`: ID del Assistant para flujo analítico (static). Opcional.
- `CASOS_CLINICOS_INTERACTIVO`: ID del Assistant para flujo interactivo. Si no se define, reutiliza el analítico.
- `CLINICAL_APPEND_REFS`: Habilita RAG (Retrieval-Augmented Generation) con vector store + PubMed para evaluación crítica fundamentada. Por defecto: `true` (habilitado). Establecer a `false` para deshabilitar.
- `INTERACTIVE_VECTOR_ID`: ID del vector store de libros médicos. Por defecto: `vs_680fc484cef081918b2b9588b701e2f4`.

## Evaluación Crítica con RAG

Los casos clínicos analíticos implementan evaluación crítica fundamentada en evidencia:

1. **Búsqueda automática de evidencia**: Antes de responder, el sistema busca en el vector store de libros médicos y PubMed para fundamentar la evaluación.

2. **Evaluación explícita**: Las respuestas se marcan como:
   - ✅ **CORRECTAS**: Con refuerzo de conceptos clave
   - ❌ **INCORRECTAS**: Con justificación clara basada en evidencia científica

3. **Sin condescendencia**: El sistema evita validar errores con lenguaje diplomático ("podría ser una opción"). Identifica errores médicos directamente.

4. **Referencias incluidas**: Cuando se usa evidencia para evaluar, las referencias se incluyen al final de la retroalimentación.

**Ejemplo de evaluación INCORRECTA:**
> ❌ El TAC de tórax NO está indicado en este contexto. La mononucleosis infecciosa se diagnostica clínicamente y con pruebas serológicas (Monospot, anticuerpos heterófilos). El manejo es conservador (reposo, hidratación, analgésicos). La imagenología avanzada solo se considera ante complicaciones atípicas como rotura esplénica o compromiso respiratorio grave, que NO se presentan en este caso.

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
