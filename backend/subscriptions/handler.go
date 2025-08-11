package subscriptions

import (
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
)

type Handler struct {
    repo *Repository
}

func NewHandler(repo *Repository) *Handler {
    return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
    r.GET("/plans", h.getPlans)
    r.POST("/plans", h.createPlan)
    r.PUT("/plans/:id", h.updatePlan)
    r.DELETE("/plans/:id", h.deletePlan)

    r.GET("/subscriptions", h.getSubscriptions)
    r.POST("/subscriptions", h.createSubscription)
    r.PUT("/subscriptions/:id", h.updateSubscription)
    r.DELETE("/subscriptions/:id", h.deleteSubscription)
}

func (h *Handler) getPlans(c *gin.Context) {
    plans, err := h.repo.GetPlans()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": plans})
}

func (h *Handler) createPlan(c *gin.Context) {
    var p Plan
    if err := c.ShouldBindJSON(&p); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
        return
    }
    if err := h.repo.CreatePlan(&p); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, p)
}

func (h *Handler) updatePlan(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
        return
    }
    var p Plan
    if err := c.ShouldBindJSON(&p); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
        return
    }
    if err := h.repo.UpdatePlan(id, &p); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) deletePlan(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
        return
    }
    if err := h.repo.DeletePlan(id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) getSubscriptions(c *gin.Context) {
    userParam := c.Query("user_id")
    userID := 0
    if userParam != "" {
        if id, err := strconv.Atoi(userParam); err == nil {
            userID = id
        }
    }
    subs, err := h.repo.GetSubscriptions(userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": subs})
}

func (h *Handler) createSubscription(c *gin.Context) {
    var s Subscription
    if err := c.ShouldBindJSON(&s); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
        return
    }
    if s.StartDate.IsZero() {
        s.StartDate = time.Now()
    }
    if err := h.repo.CreateSubscription(&s); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, s)
}

func (h *Handler) updateSubscription(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
        return
    }
    var s Subscription
    if err := c.ShouldBindJSON(&s); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
        return
    }
    if err := h.repo.UpdateSubscription(id, &s); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) deleteSubscription(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
        return
    }
    if err := h.repo.DeleteSubscription(id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

