package subscriptions

import (
    "database/sql"
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

func (r *Repository) UpdateSubscription(id int, s *Subscription) error {
    _, err := r.db.Exec(`UPDATE subscriptions SET consultations=?, questionnaires=?, clinical_cases=?, files=? WHERE id=?`,
        s.Consultations, s.Questionnaires, s.ClinicalCases, s.Files, id)
    return err
}

func (r *Repository) DeleteSubscription(id int) error {
    _, err := r.db.Exec(`DELETE FROM subscriptions WHERE id=?`, id)
    return err
}

