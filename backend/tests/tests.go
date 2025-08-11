package tests

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ema-backend/openai"
)

// Question represents a quiz question payload.
type Question struct {
	ID       int      `json:"id"`
	Question string   `json:"question"`
	Answer   string   `json:"answer"`
	Type     string   `json:"type"`
	Options  []string `json:"options"`
	Category any      `json:"category"`
}

// Repository handles DB operations for tests.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new repository with the given DB.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateTest inserts a new test for a user and returns its ID.
func (r *Repository) CreateTest(userID int) (int64, error) {
	res, err := r.db.Exec("INSERT INTO tests (user_id) VALUES (?)", userID)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// InsertQuestion stores a question for the given test.
func (r *Repository) InsertQuestion(testID int64, q Question) error {
	opts, err := json.Marshal(q.Options)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`INSERT INTO test_questions (test_id, question, answer, type, options, category) VALUES (?,?,?,?,?,?)`, testID, q.Question, q.Answer, q.Type, string(opts), q.Category)
	return err
}

// Handler exposes quiz endpoints.
type Handler struct {
	repo *Repository
	ai   *openai.Client
}

// NewHandler constructs a Handler.
func NewHandler(repo *Repository, ai *openai.Client) *Handler {
	return &Handler{repo: repo, ai: ai}
}

// RegisterRoutes sets up HTTP routes for quizzes.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/tests/generate/:userId", h.Generate)
}

type generateReq struct {
	NumQuestions int    `json:"num_questions"`
	Nivel        string `json:"nivel"`
	IdCategoria  []int  `json:"id_categoria"`
}

// Generate creates a quiz for a user and returns it in the expected JSON format.
func (h *Handler) Generate(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "usuario inválido"})
		return
	}
	var req generateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parámetros inválidos"})
		return
	}
	prompt := fmt.Sprintf(`Crea un cuestionario de medicina interna.#las siguiente reglas deben ser tomadas en cuenta a la hora de generar el cuestionarioReglas:1. Formato de las preguntas:- Incluye 4 o 5 opciones, con una Única respuesta correcta.- Incluye preguntas de "excepto" y "sin excepto" según corresponda.

Por favor, genera un JSON con un objeto que contenga las siguientes claves:
"questions": Un array de objetos de preguntas de tamaño %d. Cada objeto debe contener:
- "id": Un número entero acendente.
- "question": El texto de la pregunta.
- "answer": La respuesta correcta para la pregunta.
- "type": El tipo de pregunta. Puede ser uno de los siguientes: "true_false", "open_ended", "single_choice".
- "options": Solo para preguntas de tipo "single_choice", un array de opciones posibles.
- "category": Puede ser null si no hay una categoría asignada.

El numero de preguntas debe ser igual a la cantidad solicitada. Los tipos de pregunta deben ser seleccionados aleatoriamente entre los tres tipos disponibles: "true_false", "open_ended" y "single_choice".
Responde estrictamente en formato JSON, sin texto adicional. Limita el contenido a medicina interna. El nivel de dificultad debe ser %s.`, req.NumQuestions, req.Nivel)

	stream, err := h.ai.StreamMessage(c, prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var sb strings.Builder
	for token := range stream {
		sb.WriteString(token)
	}
	var payload struct {
		Questions []Question `json:"questions"`
	}
	if err := json.Unmarshal([]byte(sb.String()), &payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "respuesta inválida"})
		return
	}
	testID, err := h.repo.CreateTest(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo crear el test"})
		return
	}
	for _, q := range payload.Questions {
		if err := h.repo.InsertQuestion(testID, q); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la pregunta"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"test_id": testID, "thread_id": "", "questions": payload.Questions}})
}

// NewQuizAI returns an OpenAI client configured for quiz generation.
func NewQuizAI() *openai.Client {
	c := openai.NewClient()
	if id := os.Getenv("CUESTIONARIOS_MEDICOS_GENERALES"); id != "" {
		c.AssistantID = id
	}
	return c
}
