# EMA - Educación Médica Avanzada

## Correcciones Aplicadas para Chat (2025-10-22 - ACTUALIZACIÓN CRÍTICA)

### Problema 1: Chat General - Pérdida de Contexto y Calidad de Respuestas
**Síntomas**: 
- El chat pierde el hilo entre preguntas consecutivas
- No integra datos demográficos dados en mensajes anteriores
- Genera hipótesis incoherentes (ej: MPNST pediátrico en adulta joven)
- Citas vagas sin capítulos/páginas específicas
- Incluye texto literal "[Respuesta 250-350 palabras]" en la salida

**Causas Raíz**:
- Modelo anterior (`o3-mini`) tenía limitaciones severas en tracking conversacional
- Faltaba estructura de STATE para mantener demografía/síntomas entre turnos
- No verificaba coherencia demográfica (edad) con hipótesis diagnósticas
- Instrucciones genéricas sin reglas de citación específicas

**Corrección Aplicada (2025-10-22)**:
1. ✅ **Sistema de STATE Obligatorio** (backend/conversations_ia/handler.go + openai/client.go)
   - CADA respuesta debe iniciar con bloque [STATE] actualizado
   - Campos: Demografía, Síntomas clave, Duración, Signos alarma, Hipótesis activas, Decisiones previas
   - **Regla crítica**: Integrar datos NUEVOS sin resetear anteriores (acumulación)
   - **Prohibido**: Decir "no hay información suficiente" si ya hay datos previos en el thread
   
2. ✅ **Verificación de Coherencia Demográfica**
   - Hipótesis deben explicar síntomas actuales Y demografía
   - NO permitir hipótesis pediátricas en adultos (y viceversa)
   - Probabilidad etiquetada (alta/media/baja) con criterios específicos que la sostienen

3. ✅ **Formato de Citas APA Estricto**
   - Libros: `Apellido, A. (año). *Título* (ed.). Editorial. Capítulo X, pp. Y-Z.`
   - PubMed: `Apellido, A. et al. (año). Título. *Revista*, vol(núm), pp. PMID: ######`
   - **Obligatorio**: Capítulos y páginas exactas de donde se extrajo la información
   - **Prohibido**: Citar PMIDs no pertinentes al caso actual

4. ✅ **Eliminación de Texto de Relleno**
   - Eliminado literal "[Respuesta 250-350 palabras]" de instrucciones
   - Respuestas directas y concisas sin marcadores artificiales
   - Máximo 3 hipótesis diferenciales con justificación clínica

5. ✅ **Modelo Optimizado: gpt-4-turbo**
   - Ventana de contexto: 128,000 tokens (conversaciones muy largas)
   - Superior tracking de hilos conversacionales vs gpt-4o
   - Mejor comprensión de pronombres y referencias temporales

6. ✅ **CRÍTICO - Fuentes Verificadas ÚNICAMENTE**
   - El chat principal SOLO usa:
     - **Prioridad 1**: Biblioteca médica (libros de texto especializados)
     - **Prioridad 2**: PubMed (literatura científica ≥2020)
   - **PROHIBIDO**: Usar conocimiento general del modelo sin fuente verificada
   - Si no hay información en las fuentes: el bot lo indica claramente (pero mantiene STATE)

**Ejemplo de STATE en acción (CONSULTA CLÍNICA)**:
```
Usuario MSG 1: "Tengo dolor de cabeza desde ayer, pulsátil, con náuseas"
Usuario MSG 2: "Tengo 28 años, mujer, no estoy embarazada. Sin fiebre."

Bot MSG 2 debe mostrar:
[STATE]
Demografía: 28 años, mujer, no embarazada
Síntomas clave: [cefalea pulsátil desde hace 1 día, náuseas]
Duración/curso: 24 horas
Signos de alarma: fiebre ausente
Hipótesis activas: [
  {Migraña, probabilidad: alta, criterios: cefalea pulsátil unilateral + náuseas + edad/sexo compatible},
  {Cefalea tensional, probabilidad: media, criterios: cefalea pero falta bilateralidad opresiva},
  {HTA urgencia, probabilidad: baja, criterios: requiere PA ≥180 mmHg no reportada}
]
[/STATE]
```
**NO debe decir**: "No tengo suficiente información" porque YA tiene demografía y síntomas de mensajes previos.

**Ejemplo de CONSULTA TEÓRICA (sin STATE)**:
```
Usuario: "Qué es el glioblastoma y cuál es su tratamiento?"

Bot responde DIRECTAMENTE sin [STATE]:
El glioblastoma es un tumor cerebral maligno grado IV según la OMS...
Tratamiento estándar: resección quirúrgica máxima segura seguida de...

## Fuentes:
Principles of Neurosurgery (3ra ed.). Cap 15, pp. 234-245.
```

**IMPORTANTE - [STATE] es INTERNO, NO VISIBLE**:
- El sistema usa [STATE] internamente para razonar y mantener coherencia
- El usuario NO ve el bloque [STATE] en el chat
- La respuesta es natural, fluida y profesional
- Ejemplo: En lugar de mostrar "Demografía: 28 años, mujer...", el bot dice: "Considerando que se trata de una mujer de 28 años con cefalea pulsátil..."

7. ✅ **Detección Automática de Tipo de Consulta**
   - El sistema identifica si es **consulta clínica** (caso de paciente) o **consulta teórica** (definiciones, tratamientos generales)
   - **Consulta clínica**: Usa [STATE], mantiene contexto demográfico, genera hipótesis diferenciales
   - **Consulta teórica**: Responde directamente sin [STATE], enfoque en definición/fisiopatología/tratamiento
   - Ejemplos clínicos: "Tengo 28 años con cefalea...", "Paciente de 45 años con dolor torácico..."
   - Ejemplos teóricos: "Qué es el síndrome de Cushing?", "Tratamiento de hipertensión arterial", "Capítulo 3 del Harrison"

### Problema 2: Chat con PDF - No Responde sobre el PDF
**Síntoma**: Subes un PDF, haces una pregunta sobre él y el bot responde sobre otra cosa o dice que no contiene esa información cuando sí la contiene.

**Causas Raíz Identificadas**:
1. Delay de propagación de OpenAI después de indexar (5s era insuficiente)
2. El `file_search` tool no se estaba invocando correctamente
3. Las instrucciones no forzaban el uso prioritario del PDF

**Correcciones Aplicadas**:
1. ✅ Aumentado tiempo de espera post-indexación de 5s a 15s (backend/conversations_ia/handler.go - línea ~1215)
   - OpenAI necesita tiempo para que el índice se propague en su infraestructura
   - 15s asegura que el file_search pueda encontrar el contenido

2. ✅ Mejoradas instrucciones para forzar uso del file_search tool
   - Instrucciones más explícitas: "USA el tool 'file_search' INMEDIATAMENTE"
   - Clarifica que el PDF YA ESTÁ indexado y disponible
   - Prioriza búsqueda en PDF antes que conocimiento general

3. ✅ Detección automática de documentos en el thread (backend/conversations_ia/handler.go - línea ~285)
   - Si el thread tiene documentos, SIEMPRE activa modo doc_only
   - Esto cubre casos donde el usuario pregunta sin mencionar explícitamente "el PDF"
   - Ejemplo: "Capítulo 1 qué dice?" ahora buscará en el PDF automáticamente

---

## 🔥 Correcciones Críticas Post-Testing (2025-10-22 FINAL)

### Problemas Detectados en Testing Real

Se detectaron **4 fallos graves** en las instrucciones anteriores:

#### **FALLO 1: No capturó primera persona como consulta clínica**
```
Usuario: "Tengo dolor de cabeza desde ayer..."
Bot: [STATE] Demografía: Adulto (edad y sexo no especificados) ❌
```
**Problema**: No reconoció "Tengo" (primera persona) como indicador de consulta clínica.

#### **FALLO 2: Solo 1 hipótesis en lugar de 3**
```
Bot: Hipótesis activas: 1. Migraña (alta probabilidad) ❌
```
**Problema**: Debió dar 3 hipótesis diferenciales por defecto.

#### **FALLO 3: No incluyó PMIDs cuando se solicitaron explícitamente**
```
Usuario: "Cita 2 fuentes con PMID reales"
Bot: Manual Protocolos... (sin PMID) ❌
```
**Problema**: Usuario pidió PMIDs explícitamente y no los dio.

#### **FALLO 4: PÉRDIDA TOTAL DEL HILO - Cambió de caso completamente** (CRÍTICO)
```
Usuario MSG 1-3: Caso de cefalea en mujer de 28 años
Usuario MSG 4: "Añade recomendaciones iniciales no farmacológicas..."
Bot MSG 4: "Para el manejo inicial de un paciente con agitación (psicosis)..." ❌❌❌
```
**Problema**: Saltó de cefalea a agitación/psicosis sin relación. Esto es exactamente lo que debíamos evitar.

### Correcciones Aplicadas (FINAL)

1. ✅ **Detección de primera persona obligatoria**
   - "Tengo X", "Me duele Y" → automáticamente consulta clínica
   - Demografía marcada como "pendiente" si no se especifica edad/sexo

2. ✅ **Número exacto de hipótesis/signos**
   - Regla: "SIEMPRE 3 hipótesis (o el número que pida el usuario)"
   - Si pide "dame N hipótesis" → da EXACTAMENTE N

3. ✅ **PMIDs obligatorios cuando se solicitan**
   - Regla: "Si usuario pidió 'con PMID reales' → OBLIGATORIO incluir PMID: ######"
   - Aplica a CADA cita de PubMed

4. ✅ **Anti-deriva: Mantener coherencia del caso**
   - Regla: "Si preguntan sobre el MISMO CASO clínico → MANTÉN coherencia, NO cambies de tema"
   - Regla: "Si hablaban de cefalea, NO saltes a agitación/psicosis sin razón"
   - Campo STATE: "Decisiones previas: [recomendaciones ya dadas]" para recordar el contexto

5. ✅ **Acumulación estricta de datos**
   - Regla: "ACUMULA datos de TODOS los mensajes (NO resetees)"
   - Regla: "'Ahora supón que X empeora' → MANTÉN datos previos + AÑADE el empeoramiento"

6. ✅ **[STATE] INTERNO - NO VISIBLE AL USUARIO** (2025-10-22 CRÍTICO)
   - El modelo usa [STATE] mentalmente para razonar y mantener coherencia
   - La respuesta al usuario es natural, fluida y profesional
   - NO aparece texto "[STATE]", "Demografía:", "Síntomas clave:", etc. en el chat
   - Ejemplo: En lugar de "[STATE] Demografía: 28 años...", el bot integra naturalmente: "Considerando que se trata de una mujer de 28 años con cefalea pulsátil desde ayer..."
   - Extensión adecuada: 250-400 palabras para casos clínicos, 200-350 para consultas teóricas

---

## 🧠 Mecanismos de Coherencia Conversacional

El sistema implementa múltiples estrategias para mantener coherencia en conversaciones largas:

### 1. Modelo Optimizado: gpt-4-turbo
- **Ventana de contexto**: 128,000 tokens (permite conversaciones muy largas)
- **Capacidad de seguimiento**: Superior tracking de hilos conversacionales vs gpt-4o
- **Manejo de referencias**: Mejor comprensión de pronombres y referencias temporales

### 2. Instrucciones Explícitas de Coherencia

El sistema instruye al modelo para:

**Seguimiento de Historial:**
```
- Lee el historial COMPLETO de este thread antes de responder
- Identifica el tema/contexto de mensajes anteriores
- Mantén continuidad: si hablaban de diabetes, asume ese contexto
```

**Resolución de Pronombres:**
```
- "Y el tratamiento?" → identifica DE QUÉ enfermedad hablan en mensajes previos
- "Qué complicaciones?" → usa el tema de la pregunta anterior
- "Y eso?" → identifica el referente en la respuesta anterior
```

**Referencias Temporales:**
```
- "Luego de eso" → identifica QUÉ pasó antes en la conversación
- "Después de X" → busca X en el historial y continúa desde ahí
- "Lo anterior" → referencia a mensaje/tema previo
```

**Continuidad Temática:**
```
- Si preguntan "Capítulo 1 qué dice?" seguido de "Y el capítulo 2?"
  → Entiende que siguen explorando el mismo documento
- Si hablan de síntomas de diabetes, luego "Y el tratamiento?"
  → Entiende que preguntan tratamiento DE DIABETES
```

### 3. Arquitectura Assistants API v2
- Cada conversación = 1 thread persistente
- El historial se mantiene automáticamente en el backend de OpenAI
- El modelo tiene acceso a TODOS los mensajes del thread
- No es necesario re-enviar historial manualmente

### 4. Ejemplos de Coherencia en Acción

**Escenario 1: Pronombres**
```
Usuario: "Cuáles son los síntomas de diabetes tipo 2?"
Bot: [Responde sobre síntomas]
Usuario: "Y el tratamiento?"
Bot: [Entiende que preguntan sobre tratamiento de diabetes tipo 2]
```

**Escenario 2: Continuación Temática**
```
Usuario: "Háblame del Capítulo 3 de Harrison"
Bot: [Responde sobre Cap 3]
Usuario: "Qué dice el siguiente capítulo?"
Bot: [Entiende que preguntan por Capítulo 4 de Harrison]
```

**Escenario 3: Referencias Complejas**
```
Usuario: "Cuál es la fisiopatología de la hipertensión?"
Bot: [Responde con mecanismos]
Usuario: "Y eso cómo afecta el riñón?"
Bot: [Identifica que "eso" = hipertensión, responde sobre efectos renales]
```

---

### Configuración Recomendada (.env)

Para obtener el mejor rendimiento, configura estas variables en tu archivo `.env`:

```bash
# API Key de OpenAI (obligatoria)
OPENAI_API_KEY=tu_api_key_aqui

# Assistant ID (obligatorio) - debe empezar con asst_
CHAT_PRINCIPAL_ASSISTANT=asst_xxxxxxxxxxxxxxxxxxxxx

# Modelo a usar (opcional - por defecto usa gpt-4-turbo si no se especifica)
# Opciones disponibles:
# - gpt-4-turbo: Mejor coherencia conversacional, ventana de contexto 128k (RECOMENDADO)
# - gpt-4o: Balance contexto/velocidad, ventana 128k
# - gpt-4: Máxima calidad pero más costoso
CHAT_MODEL=gpt-4-turbo

# Configuración de Vector Store para PDFs
VS_MAX_FILES=5              # Máximo archivos por sesión
VS_MAX_MB=100               # Máximo MB por sesión
VS_POLL_SEC=8               # Segundos para esperar procesamiento de PDF
VS_INDEX_TIMEOUT_SEC=120    # Timeout para indexación de PDF (aumentar para PDFs grandes)
VS_TTL_MINUTES=60           # Tiempo de vida del vector store en memoria

# ID del Vector Store de la biblioteca médica (si tienes uno pre-configurado)
BOOKS_VECTOR_ID=vs_680fc484cef081918b2b9588b701e2f4
```

### 🔒 Política Estricta de Fuentes Verificadas

El sistema ahora implementa una política ESTRICTA de fuentes verificadas para garantizar información médica confiable:

#### Chat Principal (sin PDF del usuario)

**FUENTES PERMITIDAS:**
1. ✅ Biblioteca Médica - Libros de texto especializados precargados en el vector store
2. ✅ PubMed - Literatura científica revisada por pares (solo estudios ≥2020)

**FUENTES PROHIBIDAS:**
- ❌ Conocimiento general del modelo GPT
- ❌ Información no verificada o sin cita
- ❌ Inferencias o extrapolaciones sin respaldo en fuentes

**COMPORTAMIENTO:**
- Si encuentra información en libros → la usa y cita el libro
- Si no encuentra en libros pero sí en PubMed → usa PubMed y cita PMID
- Si encuentra en ambos → integra priorizando libros para fundamentos
- Si NO encuentra en ninguna fuente → responde: *"No encontré información verificada en la biblioteca médica ni en PubMed para responder esta pregunta. Por favor, reformula tu pregunta o proporciona más detalles."*

#### Chat con PDF del Usuario

**FUENTES PERMITIDAS:**
1. ✅ SOLO el documento PDF subido por el usuario

**FUENTES PROHIBIDAS:**
- ❌ Conocimiento médico general
- ❌ Otros documentos o fuentes externas
- ❌ Biblioteca médica o PubMed

**COMPORTAMIENTO:**
- Usa tool `file_search` para buscar en el PDF
- Si encuentra información → la cita textualmente
- Si NO encuentra → responde: *"No encontré información sobre esto en el documento"*
- NUNCA complementa con información externa

#### Ejemplos de Respuestas Correctas

**Pregunta con fuentes disponibles:**
```
Usuario: "Cuáles son los síntomas de la diabetes tipo 2?"

Bot: "Los síntomas principales de la diabetes tipo 2 incluyen:
- Poliuria (micción frecuente)
- Polidipsia (sed excesiva)
- Polifagia (hambre aumentada)
- Pérdida de peso inexplicable
- Fatiga
- Visión borrosa

Los síntomas suelen desarrollarse gradualmente a lo largo de varios años.

## Fuentes
- Harrison. Principios de Medicina Interna. 20ª edición. [PDF]
- American Diabetes Association. Standards of Medical Care in Diabetes—2023 (PMID: 36507649, 2023)"
```

**Pregunta sin fuentes disponibles:**
```
Usuario: "Qué opinas sobre la homeopatía para tratar el cáncer?"

Bot: "No encontré información verificada en la biblioteca médica ni en PubMed para responder esta pregunta. Por favor, reformula tu pregunta o proporciona más detalles para que pueda buscar en las fuentes médicas especializadas."
```

**Chat con PDF:**
```
Usuario: [Sube PDF] "Qué dice sobre el tratamiento?"

Bot: "El documento indica que el tratamiento de primera línea consiste en:

1. Metformina 500mg vía oral, aumentando gradualmente hasta 2000mg/día
2. Modificaciones del estilo de vida: dieta mediterránea y ejercicio aeróbico 150 min/semana
3. Monitoreo de HbA1c cada 3 meses

Fuente: Guía_Diabetes_2023.pdf"
```

### Cómo Probar las Correcciones

#### Test 1: Contexto Conversacional
```
1. Inicia un nuevo chat
2. Pregunta: "Cuáles son los síntomas de la diabetes?"
3. Espera respuesta
4. Pregunta: "Y cuál sería el tratamiento?" (nota: usa "Y" para referirse a diabetes)
5. Espera respuesta
6. Pregunta: "Qué complicaciones puede tener?" (sin mencionar diabetes explícitamente)

✅ ESPERADO: El bot debe entender que sigues hablando de diabetes en todas las preguntas
❌ ANTES: El bot perdía el contexto después de la 2da o 3ra pregunta
```

#### Test 2: Chat con PDF
```
1. Inicia un nuevo chat
2. Sube un PDF médico (por ejemplo, un artículo o capítulo de libro)
3. Espera confirmación de carga (debe decir "Documento cargado y procesado")
4. Pregunta específica del contenido, por ejemplo: "Qué dice el capítulo 1?"
5. Espera respuesta con contenido del PDF y "Fuente: [nombre del PDF]"
6. Haz una segunda pregunta sin mencionar el PDF: "Y el capítulo 2?"

✅ ESPERADO: 
   - Primera pregunta debe responder con contenido real del PDF
   - Segunda pregunta debe buscar automáticamente en el mismo PDF
   - Ambas respuestas deben incluir "Fuente: [nombre.pdf]"

❌ ANTES: 
   - Respondía con conocimiento general, no del PDF
   - Decía "no encontré información" cuando sí estaba en el PDF
```

### Notas Técnicas

**Arquitectura del Sistema:**
- Frontend (Flutter/Dart) → API Gateway → Backend (Go/Gin)
- Backend usa OpenAI Assistants API v2 (no Chat Completions)
- Cada conversación = 1 thread en OpenAI
- PDFs se indexan en vector stores específicos por thread
- file_search tool es invocado automáticamente por el Assistant

**Endpoints Activos:**
- `/conversations/start` - Iniciar nuevo chat (crea thread de OpenAI)
- `/conversations/message` - Enviar mensaje (con o sin PDF)

**Flujos:**
1. **Chat General** → SOLO usa fuentes verificadas:
   - Biblioteca médica (libros de texto especializados) como fuente primaria
   - PubMed (literatura científica ≥2020) como complemento
   - NUNCA usa conocimiento general del modelo sin citar fuente
   - Si no encuentra información: indica claramente que las fuentes no contienen esa información

2. **Chat con PDF** → modo doc_only (SOLO usa el PDF subido por el usuario)
   - Búsqueda exclusiva en el documento cargado
   - Prohibido usar conocimiento externo o fuentes generales
   - Si no encuentra información: indica que el documento no contiene esa información

3. **Small Talk** (saludos) → respuesta directa sin RAG

### Troubleshooting

**Si el chat sigue perdiendo contexto:**
1. Verifica que `CHAT_MODEL=gpt-4-turbo` esté configurado (gpt-4-turbo tiene mejor coherencia conversacional)
2. Revisa los logs del backend: `grep "\[assist\]\[StreamAssistantMessage\]" logs.txt`
3. Asegúrate de que el thread_id se mantiene entre mensajes

**Si el PDF sigue sin responder correctamente:**
1. Verifica que el PDF se indexó: busca en logs `[conv][PDF][indexing.ready]`
2. Aumenta `VS_INDEX_TIMEOUT_SEC` si el PDF es muy grande (>10MB)
3. Revisa que aparezca `[conv][PDF][post_index_wait] wait_complete` antes de la primera pregunta

**Si ves errores de quota:**
- Verifica tu saldo en OpenAI
- Revisa límites de rate limiting en tu cuenta
- Usa `gpt-4-turbo` (RECOMENDADO) para máxima coherencia conversacional
- Considera `gpt-4o-mini` solo si el costo es crítico (pero perderás calidad en conversaciones largas)

### Próximos Pasos Recomendados

1. **Monitoreo**: Implementar métricas para tracking de:
   - Tiempo de respuesta por modelo
   - Tasa de uso correcto del file_search tool
   - Satisfacción del usuario (thumbs up/down)

2. **Optimizaciones**:
   - Cache de búsquedas frecuentes en vector stores
   - Warm-up de threads al iniciar sesión
   - Prefetch de contexto conversacional

3. **Testing**:
   - Suite de tests automatizados para contexto conversacional
   - Tests de integración con PDFs reales
   - Load testing para validar escalabilidad

---

# EMA - Educación Médica Avanzada (Documentación Original)

A new Flutter project.
