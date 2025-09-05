# Implementación del Sistema de Fuentes Inteligente para Chat Principal

## Resumen

Se ha implementado un sistema de búsqueda inteligente en el chat principal (`conversations_ia`) que utiliza un flujo específico de fuentes para responder las consultas de los usuarios:

1. **Búsqueda en RAG específico**: Vector store `vs_680fc484cef081918b2b9588b701e2f4`
2. **Fallback a PubMed**: Si no se encuentra información en el RAG
3. **Citado preciso de fuentes**: Indicación clara de qué fuente se utilizó

## Cambios Implementados

### 1. Nuevos Métodos en AIClient Interface (`conversations_ia/handler.go`)

Se agregaron tres nuevos métodos al interface `AIClient`:

```go
// Nuevos métodos para búsqueda específica en RAG y PubMed
SearchInVectorStore(ctx context.Context, vectorStoreID, query string) (string, error)
SearchPubMed(ctx context.Context, query string) (string, error)
StreamAssistantWithSpecificVectorStore(ctx context.Context, threadID, prompt, vectorStoreID string) (<-chan string, error)
```

### 2. Función SmartMessage (`conversations_ia/handler.go`)

Nueva función que implementa el flujo inteligente:

```go
func (h *Handler) SmartMessage(ctx context.Context, threadID, prompt string) (<-chan string, string, error)
```

**Flujo de la función:**

1. **Buscar en RAG específico** (`vs_680fc484cef081918b2b9588b701e2f4`)
   - Si encuentra información → devuelve respuesta con fuente "rag"
   - Si no encuentra → continúa al paso 2

2. **Buscar en PubMed**
   - Si encuentra información → devuelve respuesta con fuente "pubmed"
   - Si no encuentra → continúa al paso 3

3. **Respuesta de no encontrado**
   - Devuelve mensaje indicando que no se encontró información en ninguna fuente
   - Fuente "none"

### 3. Implementación en Cliente OpenAI (`openai/client.go`)

Se implementaron los tres nuevos métodos:

#### `SearchInVectorStore`
- Crea un thread temporal
- Busca específicamente en el vector store indicado
- Devuelve información encontrada o vacío si no hay resultados

#### `SearchPubMed`
- Crea un thread temporal
- Usa el assistant con acceso web para buscar en PubMed
- Devuelve información con PMIDs cuando están disponibles

#### `StreamAssistantWithSpecificVectorStore`
- Ejecuta el assistant usando un vector store específico
- Mantiene el formato de streaming consistente

### 4. Actualización del Flujo Principal

Se modificaron todas las rutas del handler para usar `SmartMessage`:

- **JSON requests** (`/conversations/message`)
- **Multipart sin archivos** (solo texto)
- **Audio transcrito** (después de transcripción)
- **Otros archivos** (ignorando el archivo, solo prompt)

**IMPORTANTE**: Los PDFs subidos por el usuario mantienen su funcionamiento original (RAG personalizado del usuario).

### 5. Headers de Respuesta

Se agregó un nuevo header para indicar la fuente utilizada:

```
X-Source-Used: rag|pubmed|none
```

### 6. Tests (`conversations_ia/handler_test.go`)

Se crearon tests completos que verifican:

- ✅ Búsqueda exitosa en RAG específico
- ✅ Fallback a PubMed cuando RAG está vacío
- ✅ Respuesta apropiada cuando ninguna fuente tiene información
- ✅ Uso correcto del vector store específico

## Comportamiento del Sistema

### Para Consultas sin Archivo PDF

1. **Usuario pregunta sobre medicina**
   ```
   Pregunta: "¿Qué es la diabetes mellitus tipo 2?"
   ```

2. **Sistema busca en RAG específico**
   ```
   Vector Store: vs_680fc484cef081918b2b9588b701e2f4
   ```

3. **Si encuentra información en RAG:**
   ```
   Respuesta: [Información del RAG]
   Fuente: Base de conocimiento médico interno
   Header: X-Source-Used: rag
   ```

4. **Si NO encuentra en RAG, busca en PubMed:**
   ```
   Respuesta: [Información de PubMed con PMIDs]
   Fuente: PubMed (https://pubmed.ncbi.nlm.nih.gov/)
   Header: X-Source-Used: pubmed
   ```

5. **Si NO encuentra en ninguna fuente:**
   ```
   Respuesta: No se encontró información relevante...
   Fuente: Búsqueda sin resultados
   Header: X-Source-Used: none
   ```

### Para PDFs del Usuario

**NO CAMBIA**: Los archivos PDF subidos por el usuario mantienen su comportamiento original, usando su propio vector store personalizado.

## Configuración

El sistema utiliza el vector store hardcodeado:
```go
const targetVectorID = "vs_680fc484cef081918b2b9588b701e2f4"
```

## Ventajas

1. **Fuentes confiables**: Solo RAG específico y PubMed oficial
2. **Transparencia**: El usuario sabe exactamente qué fuente se usó
3. **Eficiencia**: Búsqueda jerárquica (RAG primero, PubMed segundo)
4. **Compatibilidad**: No afecta funcionalidad existente de PDFs del usuario
5. **Trazabilidad**: Headers y logs detallados para debugging

## Logs

El sistema genera logs detallados para seguimiento:

```
[conv][SmartMessage][start] thread=thread_123 target_vector=vs_680fc484cef081918b2b9588b701e2f4
[conv][SmartMessage][rag_found] thread=thread_123 chars=245
[conv][SmartMessage][rag_empty] thread=thread_123, trying_pubmed
[conv][SmartMessage][pubmed_found] thread=thread_123 chars=189
[conv][SmartMessage][no_sources] thread=thread_123
```

## Verificación

Todos los tests pasan correctamente:
```bash
cd backend/conversations_ia && go test -v
=== RUN   TestSmartMessage_RAGFound
--- PASS: TestSmartMessage_RAGFound (0.00s)
=== RUN   TestSmartMessage_PubMedFallback  
--- PASS: TestSmartMessage_PubMedFallback (0.00s)
=== RUN   TestSmartMessage_NoSourcesFound
--- PASS: TestSmartMessage_NoSourcesFound (0.00s)
=== RUN   TestVectorStoreIDUsage
--- PASS: TestVectorStoreIDUsage (0.00s)
PASS
```
