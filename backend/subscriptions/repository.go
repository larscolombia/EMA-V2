package subscriptions

import (
    "database/sql"
    "fmt"
)

type Repository struct {
    db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
    return &Repository{db: db}
}

func (r *Repository) GetPlans() ([]Plan, error) {
    rows, err := r.db.Query(`SELECT id, name, currency, price, billing, consultations, questionnaires, clinical_cases, files FROM subscription_plans`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    plans := []Plan{}
    for rows.Next() {
        var p Plan
        if err := rows.Scan(&p.ID, &p.Name, &p.Currency, &p.Price, &p.Billing, &p.Consultations, &p.Questionnaires, &p.ClinicalCases, &p.Files); err != nil {
            return nil, err
        }
        plans = append(plans, p)
    }
    return plans, nil
}

// GetPlanByID returns a plan by its ID
func (r *Repository) GetPlanByID(id int) (*Plan, error) {
    row := r.db.QueryRow(`SELECT id, name, currency, price, billing, consultations, questionnaires, clinical_cases, files FROM subscription_plans WHERE id=? LIMIT 1`, id)
    var p Plan
    if err := row.Scan(&p.ID, &p.Name, &p.Currency, &p.Price, &p.Billing, &p.Consultations, &p.Questionnaires, &p.ClinicalCases, &p.Files); err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }
    return &p, nil
}

func (r *Repository) CreatePlan(p *Plan) error {
    res, err := r.db.Exec(`INSERT INTO subscription_plans (name, currency, price, billing, consultations, questionnaires, clinical_cases, files) VALUES (?,?,?,?,?,?,?,?)`,
        p.Name, p.Currency, p.Price, p.Billing, p.Consultations, p.Questionnaires, p.ClinicalCases, p.Files)
    if err != nil {
        return err
    }
    id, err := res.LastInsertId()
    if err != nil {
        return err
    }
    p.ID = int(id)
    return nil
}

func (r *Repository) UpdatePlan(id int, p *Plan) error {
    _, err := r.db.Exec(`UPDATE subscription_plans SET name=?, currency=?, price=?, billing=?, consultations=?, questionnaires=?, clinical_cases=?, files=? WHERE id=?`,
        p.Name, p.Currency, p.Price, p.Billing, p.Consultations, p.Questionnaires, p.ClinicalCases, p.Files, id)
    return err
}

func (r *Repository) DeletePlan(id int) error {
    _, err := r.db.Exec(`DELETE FROM subscription_plans WHERE id=?`, id)
    return err
}

func (r *Repository) GetSubscriptions(userID int) ([]Subscription, error) {
    rows, err := r.db.Query(`SELECT s.id, s.user_id, s.plan_id, s.start_date, s.end_date, s.frequency, s.consultations, s.questionnaires, s.clinical_cases, s.files, p.id, p.name, p.currency, p.price, p.billing, p.consultations, p.questionnaires, p.clinical_cases, p.files FROM subscriptions s JOIN subscription_plans p ON s.plan_id = p.id WHERE (?=0 OR s.user_id=?)`, userID, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    subs := []Subscription{}
    for rows.Next() {
        var s Subscription
        var plan Plan
        err := rows.Scan(&s.ID, &s.UserID, &s.PlanID, &s.StartDate, &s.EndDate, &s.Frequency, &s.Consultations, &s.Questionnaires, &s.ClinicalCases, &s.Files,
            &plan.ID, &plan.Name, &plan.Currency, &plan.Price, &plan.Billing, &plan.Consultations, &plan.Questionnaires, &plan.ClinicalCases, &plan.Files)
        if err != nil {
            return nil, err
        }
        s.Plan = &plan
        subs = append(subs, s)
    }
    return subs, nil
}

func (r *Repository) CreateSubscription(s *Subscription) error {
    // If quotas are zero/unset, initialize them from the selected plan
    if s.Consultations == 0 && s.Questionnaires == 0 && s.ClinicalCases == 0 && s.Files == 0 {
        plan, err := r.GetPlanByID(s.PlanID)
        if err != nil {
            return err
        }
        if plan != nil {
            s.Consultations = plan.Consultations
            s.Questionnaires = plan.Questionnaires
            s.ClinicalCases = plan.ClinicalCases
            s.Files = plan.Files
        }
    }
    res, err := r.db.Exec(`INSERT INTO subscriptions (user_id, plan_id, start_date, end_date, frequency, consultations, questionnaires, clinical_cases, files) VALUES (?,?,?,?,?,?,?,?,?)`,
        s.UserID, s.PlanID, s.StartDate, s.EndDate, s.Frequency, s.Consultations, s.Questionnaires, s.ClinicalCases, s.Files)
    if err != nil {
        return err
    }
    id, err := res.LastInsertId()
    if err != nil {
        return err
    }
    s.ID = int(id)
    return nil
}

// DecrementSubscriptionFields decreases the provided fields by the given amounts (non-negative), clamped at 0
func (r *Repository) DecrementSubscriptionFields(id int, deltas map[string]int) error {
    if len(deltas) == 0 {
        return nil
    }
    allowed := map[string]bool{
        "consultations":  true,
        "questionnaires": true,
        "clinical_cases": true,
        "files":         true,
    }
    sets := []string{}
    args := []interface{}{}
    for k, v := range deltas {
        if !allowed[k] {
            continue
        }
        if v < 0 {
            return fmt.Errorf("delta for %s must be >= 0", k)
        }
        // Only update when there's a positive decrement
        if v > 0 {
            sets = append(sets, k+" = GREATEST("+k+" - ?, 0)")
            args = append(args, v)
        }
    }
    if len(sets) == 0 {
        return nil
    }
    args = append(args, id)
    query := "UPDATE subscriptions SET " + joinWithComma(sets) + " WHERE id=?"
    _, err := r.db.Exec(query, args...)
    return err
}

func joinWithComma(parts []string) string {
    if len(parts) == 0 {
        return ""
    }
    out := parts[0]
    for i := 1; i < len(parts); i++ {
        out += ", " + parts[i]
    }
    return out
}

func (r *Repository) DeleteSubscription(id int) error {
    _, err := r.db.Exec(`DELETE FROM subscriptions WHERE id=?`, id)
    return err
}

