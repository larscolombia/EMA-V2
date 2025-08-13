package subscriptions

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo   *Repository
	stripe *StripeService
}

func NewHandler(repo *Repository) *Handler {
	// Initialize Stripe service from environment (optional)
	s := NewStripeFromEnv(repo)
	return &Handler{repo: repo, stripe: s}
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

	// Aliases used by Flutter
	r.POST("/cancel-subscription", h.cancelSubscription)
	// Minimal checkout stub to satisfy clients expecting /checkout
	r.POST("/checkout", h.checkout)
	// Stripe webhook endpoint (only active when STRIPE_WEBHOOK_SECRET is set)
	r.POST("/stripe/webhook", h.handleStripeWebhook)
	// Handle common misspelling if present in some older clients
	r.GET("/suscription-plans", h.getPlans)
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

// cancelSubscription handles POST /cancel-subscription with body { subscription_id }
func (h *Handler) cancelSubscription(c *gin.Context) {
	var body struct {
		SubscriptionID int `json:"subscription_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.SubscriptionID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscription_id requerido"})
		return
	}
	if err := h.repo.DeleteSubscription(body.SubscriptionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) updateSubscription(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	// Treat provided fields as decrement deltas, not absolute overwrite
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
		return
	}
	deltas := map[string]int{}
	for _, key := range []string{"consultations", "questionnaires", "clinical_cases", "files"} {
		if v, ok := payload[key]; ok && v != nil {
			switch vv := v.(type) {
			case float64:
				if vv > 0 {
					deltas[key] = int(vv)
				}
			case int:
				if vv > 0 {
					deltas[key] = vv
				}
			}
		}
	}
	if err := h.repo.DecrementSubscriptionFields(id, deltas); err != nil {
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

// checkout provides a stub checkout URL for clients integrating a webview flow.
// Expected body: { "user_id": number, "plan_id": number, "frequency": number }
// Response: { "checkout_url": string }
func (h *Handler) checkout(c *gin.Context) {
	var body struct {
		UserID    int `json:"user_id"`
		PlanID    int `json:"plan_id"`
		Frequency int `json:"frequency"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.UserID == 0 || body.PlanID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
		return
	}
	// Decide flow based on plan price and Stripe availability
	plan, _ := h.repo.GetPlanByID(body.PlanID)
	// If Stripe configured and plan has price > 0, create a real checkout session
	if h.stripe != nil && plan != nil && plan.Price > 0 {
		url, err := h.stripe.CreateCheckoutSession(c.Request.Context(), body.UserID, body.PlanID, body.Frequency)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"checkout_url": url})
		return
	}
	// Otherwise (free plan or Stripe disabled), create subscription immediately and return success URL.
	s := &Subscription{UserID: body.UserID, PlanID: body.PlanID, StartDate: time.Now(), Frequency: body.Frequency}
	if err := h.repo.CreateSubscription(s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"checkout_url": "https://example.com/checkout/success"})
}

// handleStripeWebhook processes Stripe webhook events to finalize subscriptions on successful payments
func (h *Handler) handleStripeWebhook(c *gin.Context) {
	if h.stripe == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "stripe no configurado"})
		return
	}
	if err := h.stripe.HandleWebhook(c.Writer, c.Request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
}
