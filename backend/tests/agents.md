# tests package: Quiz generation endpoints

Overview
- Provides HTTP endpoints to generate medical quizzes using OpenAI.
- Persists generated quizzes and questions in MySQL.

Environment variables
- CUESTIONARIOS_MEDICOS_GENERALES: Assistant identifier used for quiz generation.

How it works
- POST /tests/generate/:userId: accepts {num_questions, nivel, id_categoria} and returns {success, data{test_id, thread_id, questions}}.
- Uses openai client for LLM interaction and stores results via repository.

Good practices
- Validate input parameters.
- Handle and log OpenAI/API errors gracefully.
- Sanitize and parameterize SQL statements to avoid injection.

Architecture notes
- Repository wraps database operations; handler deals with HTTP and AI orchestration.
