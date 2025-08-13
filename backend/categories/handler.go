package categories

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func NewHandler(r *Repository) *Handler { return &Handler{repo: r} }

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/categoria-medicas", h.list)
}

func (h *Handler) list(c *gin.Context) {
	items, err := h.repo.All()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}
