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
	return &StripeService{
		repo:          repo,
		secretKey:     key,
		webhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		successURL:    success,
		cancelURL:     cancel,
	}
}

// CreateCheckoutSession simulates creating a checkout session and returns a URL.
// Replace the body with real Stripe SDK/API calls if desired.
func (s *StripeService) CreateCheckoutSession(ctx context.Context, userID, planID, frequency int) (string, error) {
	if s == nil {
		return "", errors.New("stripe no configurado")
	}
	// In a real integration you would:
	// 1) Look up plan by ID, map to Stripe Price ID
	// 2) Create a Checkout Session via Stripe API with success/cancel URLs
	// 3) Return session.URL
	// For now we return success URL immediately for development.
	return s.successURL, nil
}

// HandleWebhook consumes webhook payloads. For a successful checkout event,
// it creates a subscription record for the user/plan encoded in metadata.
func (s *StripeService) HandleWebhook(w http.ResponseWriter, r *http.Request) error {
	if s == nil {
		return errors.New("stripe no configurado")
	}
	// Read body
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	// Optionally verify signature with s.webhookSecret
	_ = s.webhookSecret // Not used in this lightweight implementation

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
