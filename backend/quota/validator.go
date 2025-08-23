package quota

import (
    "context"
    "errors"
    "os"
    "strings"
    "log"

    "ema-backend/login"
    "ema-backend/subscriptions"
    "github.com/gin-gonic/gin"
)

// Flow to subscription field mapping
var flowField = map[string]string{
    // Clinical case generation counts as 1 usage
    "analytical_generate":      "clinical_cases",
    // Interactive case initial generation/start counts
    "interactive_generate":     "clinical_cases",
    "interactive_strict_start": "clinical_cases",
    // Chats inside an existing case (analytical_chat, interactive_chat, interactive_strict_message)
    // now DO NOT consume additional quota; they are intentionally omitted.
    // General assistant chat mapped to consultations bucket
    "chat_message":              "consultations",
}

// Validator provides quota validation wired into handlers.
type Validator struct {
    subs *subscriptions.Repository
}

func NewValidator(repo *subscriptions.Repository) *Validator { return &Validator{subs: repo} }

// ValidateAndConsume identifies user from Authorization token, fetches active subscription and decrements the mapped field by 1.
func (v *Validator) ValidateAndConsume(ctx context.Context, c *gin.Context, flow string) error {
    field, ok := flowField[flow]
    if !ok { // Unknown flow -> allow
    log.Printf("[quota][skip] flow=%s reason=unknown_flow", flow)
        return nil
    }
    if os.Getenv("QUOTA_DISABLE") == "1" {
        // Bypass entirely for debugging; annotate headers for client awareness
        c.Set("quota_field", field)
        c.Set("quota_remaining", "debug-infinite")
    log.Printf("[quota][bypass] flow=%s field=%s QUOTA_DISABLE=1", flow, field)
        return nil
    }
    auth := c.GetHeader("Authorization")
    token := strings.TrimPrefix(auth, "Bearer ")
    if token == "" {
    log.Printf("[quota][deny] flow=%s field=%s reason=missing_token", flow, field)
        return errors.New("missing token")
    }
    email, ok := login.GetEmailFromToken(token)
    if !ok {
        log.Printf("[quota][deny] flow=%s field=%s reason=invalid_session token_prefix=%s" , flow, field, tokenSummary(token))
        return errors.New("invalid session")
    }
    // Resolve user
    u := userResolver(email)
    if u == nil {
    log.Printf("[quota][deny] flow=%s field=%s email=%s reason=user_not_found", flow, field, email)
        return errors.New("user not found")
    }
    sub, err := v.subs.GetActiveSubscription(u.ID)
    if err != nil {
    log.Printf("[quota][error] flow=%s field=%s user_id=%d email=%s err=%v", flow, field, u.ID, email, err)
        return err
    }
    if sub == nil {
    log.Printf("[quota][deny] flow=%s field=%s user_id=%d email=%s reason=no_subscription", flow, field, u.ID, email)
        return errors.New("no subscription")
    }
    // Fast path check
    var remaining int
    switch field {
    case "clinical_cases":
        remaining = sub.ClinicalCases
    case "consultations":
        remaining = sub.Consultations
    case "questionnaires":
        remaining = sub.Questionnaires
    case "files":
        remaining = sub.Files
    }
    if remaining <= 0 {
        // attach structured info so handler can format JSON
        c.Set("quota_error_field", field)
        c.Set("quota_error_reason", "exhausted")
        log.Printf("[quota][exhausted] flow=%s field=%s user_id=%d sub_id=%d email=%s remaining=%d", flow, field, u.ID, sub.ID, email, remaining)
        return errors.New("quota exhausted")
    }
    log.Printf("[quota][consume] flow=%s field=%s user_id=%d sub_id=%d email=%s remaining_before=%d amount=1", flow, field, u.ID, sub.ID, email, remaining)
    consumed, err := v.subs.ConsumeQuota(sub.ID, field, 1)
    if err != nil {
        log.Printf("[quota][error] flow=%s field=%s user_id=%d sub_id=%d email=%s err=%v", flow, field, u.ID, sub.ID, email, err)
        return err
    }
    if !consumed {
        c.Set("quota_error_field", field)
        c.Set("quota_error_reason", "exhausted")
        log.Printf("[quota][race_exhausted] flow=%s field=%s user_id=%d sub_id=%d email=%s remaining_precheck=%d", flow, field, u.ID, sub.ID, email, remaining)
        return errors.New("quota exhausted")
    }
    // Store remaining (after decrement) in context for handlers to propagate via headers
    c.Set("quota_field", field)
    c.Set("quota_remaining", remaining-1)
    log.Printf("[quota][ok] flow=%s field=%s user_id=%d sub_id=%d email=%s remaining_after=%d", flow, field, u.ID, sub.ID, email, remaining-1)
    return nil
}

// tokenSummary returns a short (safe) representation of a token for logs
func tokenSummary(t string) string {
    if len(t) <= 8 { return t }
    return t[:4] + "..." + t[len(t)-4:]
}

// --- User resolver adapter ---
// We keep this indirection to avoid tight coupling with migrations/internal user structures.

var userResolver = func(email string) *UserLite { return nil }

// RegisterUserResolver allows main to provide a resolver.
func RegisterUserResolver(fn func(email string) *UserLite) { userResolver = fn }

// UserLite minimal projection
type UserLite struct { ID int; Email string }

// Middleware helper (not used yet)
func (v *Validator) Middleware(flow string) gin.HandlerFunc {
    return func(c *gin.Context) {
        if err := v.ValidateAndConsume(c.Request.Context(), c, flow); err != nil {
            c.JSON(403, gin.H{"error": err.Error()})
            c.Abort()
            return
        }
        c.Next()
    }
}
