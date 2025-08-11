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
}

