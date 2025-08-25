package subscriptions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
	"log"
	"strings"

	stripe "github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/client"
	"github.com/stripe/stripe-go/v78/webhook"
)

// StripeService is a thin abstraction to create checkout sessions and handle webhooks.
// Real Stripe calls are optional; if STRIPE_SECRET_KEY is not set, the service is disabled (nil).
type StripeService struct {
	repo          *Repository
	secretKey     string
	webhookSecret string
	// Base URL where the Flutter app listens for success/cancel via WebView
	successURL string
	cancelURL  string
	sc *client.API
	invalidKey bool // once detected, short-circuit further remote calls
}

var ErrStripeInvalidAPIKey = errors.New("stripe_invalid_api_key")

func maskKey(k string) string {
	if len(k) < 12 { return "****" }
	return k[:7] + "..." + k[len(k)-4:]
}

// NewStripeFromEnv returns a configured service or nil when missing env vars.
func NewStripeFromEnv(repo *Repository) *StripeService {
	key := os.Getenv("STRIPE_SECRET_KEY")
	if key == "" {
		return nil
	}
	success := os.Getenv("STRIPE_SUCCESS_URL")
	if success == "" {
		// Default to example success marker consumed by Flutter webview
		success = "https://example.com/checkout/success"
	}
	cancel := os.Getenv("STRIPE_CANCEL_URL")
	if cancel == "" {
		cancel = "https://example.com/checkout/cancel"
	}
	sc := &client.API{}
	sc.Init(key, nil)
	return &StripeService{
		repo:          repo,
		secretKey:     key,
		webhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		successURL:    success,
		cancelURL:     cancel,
		sc:            sc,
	}
}

// CreateCheckoutSession simulates creating a checkout session and returns a URL.
// Replace the body with real Stripe SDK/API calls if desired.
func (s *StripeService) ensureStripeProductAndPrice(ctx context.Context, p *Plan) error {
	if p.Price == 0 { // Free plan: no Stripe objects needed
		return nil
	}
	// Create product if missing
	if p.StripeProductID == "" {
		prod, err := s.sc.Products.New(&stripe.ProductParams{Name: stripe.String(p.Name)})
		if err != nil { return err }
		p.StripeProductID = prod.ID
	}
	// Ensure price: fetch existing to compare amount (if stored)
	if p.StripePriceID != "" {
		if pr, err := s.sc.Prices.Get(p.StripePriceID, nil); err == nil {
			current := pr.UnitAmount
			desired := int64(p.Price * 100)
			if current != desired { // create new price; keep old for historic invoices
				priceParams := &stripe.PriceParams{
					Product:    stripe.String(p.StripeProductID),
					Currency:   stripe.String(p.Currency),
					UnitAmount: stripe.Int64(desired),
					Recurring: &stripe.PriceRecurringParams{Interval: stripe.String("month")},
				}
				price, err := s.sc.Prices.New(priceParams)
				if err != nil { return err }
				p.StripePriceID = price.ID
			}
		} else { // price id invalid -> recreate
			p.StripePriceID = ""
		}
	}
	if p.StripePriceID == "" { // create if missing
		unitAmount := int64(p.Price * 100)
		priceParams := &stripe.PriceParams{
			Product:    stripe.String(p.StripeProductID),
			Currency:   stripe.String(p.Currency),
			UnitAmount: stripe.Int64(unitAmount),
			Recurring: &stripe.PriceRecurringParams{Interval: stripe.String("month")},
		}
		price, err := s.sc.Prices.New(priceParams)
		if err != nil { return err }
		p.StripePriceID = price.ID
	}
	return nil
}

// CreateCheckoutSession creates a real Stripe Checkout Session (one-off) for a plan.
// Backward compatible (deprecated) wrapper
func (s *StripeService) CreateCheckoutSession(ctx context.Context, userID, planID, frequency int) (string, error) {
	url, _, err := s.CreateCheckoutSessionWithID(ctx, userID, planID, frequency)
	return url, err
}

// New: returns URL + sessionID
func (s *StripeService) CreateCheckoutSessionWithID(ctx context.Context, userID, planID, frequency int) (string, string, error) {
	if s == nil { return "", "", errors.New("stripe no configurado") }
	plan, err := s.repo.GetPlanByID(planID)
	if err != nil || plan == nil { return "", "", fmt.Errorf("plan inválido") }
	if plan.Price == 0 {
		sub := &Subscription{UserID: userID, PlanID: plan.ID, StartDate: time.Now(), Frequency: frequency}
		if err := s.repo.CreateSubscription(sub); err != nil { return "", "", err }
		return s.successURL, "", nil
	}
	if err := s.ensureStripeProductAndPrice(ctx, plan); err != nil {
		var se *stripe.Error
		if errors.As(err, &se) && (se.HTTPStatusCode == 401 || strings.Contains(strings.ToLower(se.Msg), "invalid api key")) {
			log.Printf("[STRIPE][ensure] invalid api key (%s): %v", maskKey(s.secretKey), se)
			s.invalidKey = true
			return "", "", ErrStripeInvalidAPIKey
		}
		return "", "", err
	}
	_ = s.repo.UpdatePlan(plan.ID, plan)
	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String(s.successURL),
		CancelURL:  stripe.String(s.cancelURL),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			Price:    stripe.String(plan.StripePriceID),
			Quantity: stripe.Int64(1),
		}},
		Metadata: map[string]string{
			"user_id": strconv.Itoa(userID),
			"plan_id": strconv.Itoa(planID),
			"frequency": strconv.Itoa(frequency),
		},
	}
	if s.invalidKey { return "", "", ErrStripeInvalidAPIKey }
	sess, err := s.sc.CheckoutSessions.New(params)
	if err != nil {
		var se *stripe.Error
		if errors.As(err, &se) && (se.HTTPStatusCode == 401 || strings.Contains(strings.ToLower(se.Msg), "invalid api key")) {
			log.Printf("[STRIPE][checkout] invalid api key (%s): %v", maskKey(s.secretKey), se)
			s.invalidKey = true
			return "", "", ErrStripeInvalidAPIKey
		}
		log.Printf("[STRIPE][checkout] error: %v", err)
		return "", "", err
	}
	return sess.URL, sess.ID, nil
}

// HandleWebhook consumes webhook payloads. For a successful checkout event,
// it creates a subscription record for the user/plan encoded in metadata.
func (s *StripeService) HandleWebhook(w http.ResponseWriter, r *http.Request) error {
	if s == nil {
		return errors.New("stripe no configurado")
	}
	// Read body (preserve for verification)
	payload, err := io.ReadAll(r.Body)
	if err != nil { return err }
	sig := r.Header.Get("Stripe-Signature")
	if s.webhookSecret != "" {
		if _, err := webhook.ConstructEvent(payload, sig, s.webhookSecret); err != nil {
			return fmt.Errorf("firma inválida: %w", err)
		}
	}

	// Expected minimal payload we support during development:
	// {
	//   "type": "checkout.session.completed",
	//   "data": {"object": {"metadata": {"user_id":"1","plan_id":"2","frequency":"1"}}}
	// }
	var event struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				Metadata map[string]string `json:"metadata"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}
	if event.Type != "checkout.session.completed" {
		// Ignore other events
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ignored"))
		return nil
	}
	// Parse metadata
	uid, _ := strconv.Atoi(event.Data.Object.Metadata["user_id"])
	pid, _ := strconv.Atoi(event.Data.Object.Metadata["plan_id"])
	freq, _ := strconv.Atoi(event.Data.Object.Metadata["frequency"])
	if uid == 0 || pid == 0 {
		return fmt.Errorf("metadata incompleta")
	}
	// Create subscription record initialized with plan quotas
	now := time.Now()
	sub := &Subscription{UserID: uid, PlanID: pid, StartDate: now, Frequency: freq}
	if err := s.repo.CreateSubscription(sub); err != nil {
		return err
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
	return nil
}

// ConfirmSession: query Stripe; if completed and subscription not yet created, create it (idempotent)
func (s *StripeService) ConfirmSession(sessionID string) (bool, int, error) {
	if s == nil { return false, 0, errors.New("stripe no configurado") }
	if sessionID == "" { return false, 0, errors.New("session_id vacío") }
	sess, err := s.sc.CheckoutSessions.Get(sessionID, nil)
	if err != nil { return false, 0, err }
	if sess.Status != stripe.CheckoutSessionStatusComplete { return false, 0, nil }
	uid, _ := strconv.Atoi(sess.Metadata["user_id"])
	pid, _ := strconv.Atoi(sess.Metadata["plan_id"])
	freq, _ := strconv.Atoi(sess.Metadata["frequency"])
	if uid == 0 || pid == 0 { return false, 0, errors.New("metadata incompleta") }
	// If already active with same plan, no new creation
	sub, _ := s.repo.GetActiveSubscription(uid)
	if sub != nil && sub.PlanID == pid { return false, sub.ID, nil }
	now := time.Now()
	newSub := &Subscription{UserID: uid, PlanID: pid, StartDate: now, Frequency: freq}
	if err := s.repo.CreateSubscription(newSub); err != nil { return false, 0, err }
	return true, newSub.ID, nil
}
