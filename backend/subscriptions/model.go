package subscriptions

import "time"

type Plan struct {
    ID            int     `json:"id"`
    Name          string  `json:"name"`
    Currency      string  `json:"currency"`
    Price         float64 `json:"price"`
    Billing       string  `json:"billing"`
    Consultations int     `json:"consultations"`
    Questionnaires int    `json:"questionnaires"`
    ClinicalCases int     `json:"clinical_cases"`
    Files         int     `json:"files"`
    StripeProductID string `json:"stripe_product_id,omitempty"`
    StripePriceID   string `json:"stripe_price_id,omitempty"`
    Statistics    int     `json:"statistics"` // 1 = incluye estadísticas premium
}

type Subscription struct {
    ID            int        `json:"id"`
    UserID        int        `json:"user_id"`
    PlanID        int        `json:"plan_id"`
    StartDate     time.Time  `json:"start_date"`
    EndDate       *time.Time `json:"end_date"`
    Frequency     int        `json:"frequency"`
    Consultations int        `json:"consultations"`
    Questionnaires int       `json:"questionnaires"`
    ClinicalCases int        `json:"clinical_cases"`
    Files         int        `json:"files"`
    Plan          *Plan      `json:"subscription_plan,omitempty"`
    Statistics    int        `json:"statistics"` // copia denormalizada para acceso rápido
}

