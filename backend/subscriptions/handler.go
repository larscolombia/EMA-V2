package subscriptions

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"ema-backend/login"
	"ema-backend/migrations"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo   *Repository
	stripe *StripeService
}

func NewHandler(repo *Repository) *Handler {
	s := NewStripeFromEnv(repo)
	return &Handler{repo: repo, stripe: s}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/plans", h.getPlans)
	r.POST("/plans", h.createPlan)
	r.PUT("/plans/:id", h.updatePlan)
	r.DELETE("/plans/:id", h.deletePlan)

	r.GET("/admin/plans", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		data, err := os.ReadFile("subscriptions/admin.html")
		if err != nil { c.String(500, "admin ui not found"); return }
		c.Writer.Write(data)
	})

	r.GET("/subscriptions", h.getSubscriptions)
	r.POST("/subscriptions", h.createSubscription)
	r.PUT("/subscriptions/:id", h.updateSubscription)
	r.DELETE("/subscriptions/:id", h.deleteSubscription)

	r.POST("/cancel-subscription", h.cancelSubscription)
	r.POST("/checkout", h.checkout)
	r.POST("/stripe/webhook", h.handleStripeWebhook)
	r.POST("/stripe/confirm", h.confirmSession) // confirmación manual (idempotente) basada en session_id
	r.GET("/suscription-plans", h.getPlans)
}

func (h *Handler) getPlans(c *gin.Context) {
	plans, err := h.repo.GetPlans()
	if err != nil {
		log.Printf("/plans error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auth := c.GetHeader("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	var activePlanID int
	if token != "" {
		if email, ok := login.GetEmailFromToken(token); ok {
			if u := migrations.GetUserByEmail(email); u != nil {
				if sub, err2 := h.repo.GetActiveSubscription(u.ID); err2 == nil && sub != nil {
					activePlanID = sub.PlanID
				}
			}
		}
	}
	out := []gin.H{}
	for _, p := range plans {
		out = append(out, gin.H{
			"id": p.ID, "name": p.Name, "currency": p.Currency, "price": p.Price, "billing": p.Billing,
			"consultations": p.Consultations, "questionnaires": p.Questionnaires, "clinical_cases": p.ClinicalCases, "files": p.Files,
			"stripe_product_id": p.StripeProductID, "stripe_price_id": p.StripePriceID, "statistics": p.Statistics,
			"active": p.ID == activePlanID,
		})
	}
	resp := gin.H{"data": out}
	if activePlanID != 0 { resp["active_plan_id"] = activePlanID }
	c.JSON(http.StatusOK, resp)
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
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
		return
	}
	mode := "set"
	if m, ok := payload["mode"].(string); ok && m == "decrement" { mode = "decrement" }
	allowedKeys := []string{"consultations", "questionnaires", "clinical_cases", "files"}
	if mode == "decrement" {
		deltas := map[string]int{}
		for _, key := range allowedKeys {
			if v, ok := payload[key]; ok && v != nil {
				switch vv := v.(type) {
				case float64:
					if vv > 0 { deltas[key] = int(vv) }
				case int:
					if vv > 0 { deltas[key] = vv }
				}
			}
		}
		if err := h.repo.DecrementSubscriptionFields(id, deltas); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "mode": mode, "deltas": deltas})
		return
	}
	updated := map[string]int{}
	for _, key := range allowedKeys {
		if v, ok := payload[key]; ok && v != nil {
			switch vv := v.(type) {
			case float64:
				if vv >= 0 { updated[key] = int(vv) }
			case int:
				if vv >= 0 { updated[key] = vv }
			}
		}
	}
	if len(updated) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sin campos para actualizar"})
		return
	}
	for k, v := range updated {
		if err := h.repo.SetQuotaValue(id, k, v); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "field": k})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "mode": mode, "updated": updated})
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
	plan, _ := h.repo.GetPlanByID(body.PlanID)
	if h.stripe != nil && plan != nil && plan.Price > 0 {
		url, sessionID, err := h.stripe.CreateCheckoutSessionWithID(c.Request.Context(), body.UserID, body.PlanID, body.Frequency)
		if err != nil {
			if errors.Is(err, ErrStripeInvalidAPIKey) {
				c.JSON(http.StatusBadGateway, gin.H{"error": "stripe_api_key_invalida"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if os.Getenv("STRIPE_AUTO_SUBSCRIBE") == "1" { // dev shortcut
			sub := &Subscription{UserID: body.UserID, PlanID: plan.ID, StartDate: time.Now(), Frequency: body.Frequency}
			if err := h.repo.CreateSubscription(sub); err != nil {
				log.Printf("[checkout][auto_subscribe] create failed: %v", err)
			} else {
				log.Printf("[checkout][auto_subscribe] user=%d plan=%d subscription_id=%d", body.UserID, plan.ID, sub.ID)
				c.JSON(http.StatusOK, gin.H{"checkout_url": url, "session_id": sessionID, "auto_subscribed": true, "subscription_id": sub.ID})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"checkout_url": url, "session_id": sessionID, "auto_subscribed": false})
		return
	}
	// Direct (free plan or stripe disabled)
	s := &Subscription{UserID: body.UserID, PlanID: body.PlanID, StartDate: time.Now(), Frequency: body.Frequency}
	if err := h.repo.CreateSubscription(s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"checkout_url": "https://example.com/checkout/success", "auto_subscribed": true, "subscription_id": s.ID})
}

// confirmSession: fallback idempotente por si el webhook se demora o se perdió.
// Body: {"session_id":"cs_test_..."}
func (h *Handler) confirmSession(c *gin.Context) {
	if h.stripe == nil { c.JSON(400, gin.H{"error":"stripe no configurado"}); return }
	var body struct { SessionID string `json:"session_id"` }
	if err := c.ShouldBindJSON(&body); err != nil || body.SessionID == "" { c.JSON(400, gin.H{"error":"session_id requerido"}); return }
	created, subID, err := h.stripe.ConfirmSession(body.SessionID)
	if err != nil { c.JSON(500, gin.H{"error":err.Error()}); return }
	c.JSON(200, gin.H{"status":"ok","created":created,"subscription_id":subID})
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
