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
	// COALESCE to avoid scanning NULL into string fields; statistics: heuristic (price>0 => 1)
	rows, err := r.db.Query(`SELECT id, name, currency, price, billing, consultations, questionnaires, clinical_cases, files, COALESCE(stripe_product_id,''), COALESCE(stripe_price_id,''), CASE WHEN price>0 THEN 1 ELSE 0 END AS statistics FROM subscription_plans`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plans := []Plan{}
	for rows.Next() {
		var p Plan
	if err := rows.Scan(&p.ID, &p.Name, &p.Currency, &p.Price, &p.Billing, &p.Consultations, &p.Questionnaires, &p.ClinicalCases, &p.Files, &p.StripeProductID, &p.StripePriceID, &p.Statistics); err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, nil
}

// GetPlanByID returns a plan by its ID
func (r *Repository) GetPlanByID(id int) (*Plan, error) {
	row := r.db.QueryRow(`SELECT id, name, currency, price, billing, consultations, questionnaires, clinical_cases, files, COALESCE(stripe_product_id,''), COALESCE(stripe_price_id,''), CASE WHEN price>0 THEN 1 ELSE 0 END AS statistics FROM subscription_plans WHERE id=? LIMIT 1`, id)
	var p Plan
	if err := row.Scan(&p.ID, &p.Name, &p.Currency, &p.Price, &p.Billing, &p.Consultations, &p.Questionnaires, &p.ClinicalCases, &p.Files, &p.StripeProductID, &p.StripePriceID, &p.Statistics); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repository) CreatePlan(p *Plan) error {
	res, err := r.db.Exec(`INSERT INTO subscription_plans (name, currency, price, billing, consultations, questionnaires, clinical_cases, files, stripe_product_id, stripe_price_id) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		p.Name, p.Currency, p.Price, p.Billing, p.Consultations, p.Questionnaires, p.ClinicalCases, p.Files, p.StripeProductID, p.StripePriceID)
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
	_, err := r.db.Exec(`UPDATE subscription_plans SET name=?, currency=?, price=?, billing=?, consultations=?, questionnaires=?, clinical_cases=?, files=?, stripe_product_id=?, stripe_price_id=? WHERE id=?`,
		p.Name, p.Currency, p.Price, p.Billing, p.Consultations, p.Questionnaires, p.ClinicalCases, p.Files, p.StripeProductID, p.StripePriceID, id)
	return err
}

func (r *Repository) DeletePlan(id int) error {
	_, err := r.db.Exec(`DELETE FROM subscription_plans WHERE id=?`, id)
	return err
}

func (r *Repository) GetSubscriptions(userID int) ([]Subscription, error) {
	rows, err := r.db.Query(`SELECT s.id, s.user_id, s.plan_id, s.start_date, s.end_date, s.frequency, s.consultations, s.questionnaires, s.clinical_cases, s.files, p.id, p.name, p.currency, p.price, p.billing, p.consultations, p.questionnaires, p.clinical_cases, p.files, CASE WHEN p.price>0 THEN 1 ELSE 0 END AS statistics FROM subscriptions s JOIN subscription_plans p ON s.plan_id = p.id WHERE (?=0 OR s.user_id=?)`, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	subs := []Subscription{}
	for rows.Next() {
		var s Subscription
		var plan Plan
		err := rows.Scan(&s.ID, &s.UserID, &s.PlanID, &s.StartDate, &s.EndDate, &s.Frequency, &s.Consultations, &s.Questionnaires, &s.ClinicalCases, &s.Files,
			&plan.ID, &plan.Name, &plan.Currency, &plan.Price, &plan.Billing, &plan.Consultations, &plan.Questionnaires, &plan.ClinicalCases, &plan.Files, &s.Statistics)
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
		"files":          true,
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

// GetActiveSubscription returns the most recent (or first) subscription for a user.
// Heuristic: no specific status field exists, so we pick the latest by id (DESC).
func (r *Repository) GetActiveSubscription(userID int) (*Subscription, error) {
	row := r.db.QueryRow(`SELECT s.id, s.user_id, s.plan_id, s.start_date, s.end_date, s.frequency, s.consultations, s.questionnaires, s.clinical_cases, s.files,
			p.id, p.name, p.currency, p.price, p.billing, p.consultations, p.questionnaires, p.clinical_cases, p.files, CASE WHEN p.price>0 THEN 1 ELSE 0 END AS statistics
			FROM subscriptions s JOIN subscription_plans p ON s.plan_id = p.id WHERE s.user_id=? ORDER BY s.id DESC LIMIT 1`, userID)
	var s Subscription
	var plan Plan
	err := row.Scan(&s.ID, &s.UserID, &s.PlanID, &s.StartDate, &s.EndDate, &s.Frequency, &s.Consultations, &s.Questionnaires, &s.ClinicalCases, &s.Files,
		&plan.ID, &plan.Name, &plan.Currency, &plan.Price, &plan.Billing, &plan.Consultations, &plan.Questionnaires, &plan.ClinicalCases, &plan.Files, &s.Statistics)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.Plan = &plan
	return &s, nil
}

// ConsumeQuota atomically decrements a single quota field by amount if it is > 0.
// Returns (true,nil) if consumed, (false,nil) if not enough quota, (false,err) on error.
func (r *Repository) ConsumeQuota(subscriptionID int, field string, amount int) (bool, error) {
	if amount <= 0 {
		return true, nil
	}
	allowed := map[string]bool{"consultations": true, "questionnaires": true, "clinical_cases": true, "files": true}
	if !allowed[field] {
		return false, fmt.Errorf("invalid quota field: %s", field)
	}
	// Conditional update ensures atomic check & decrement
	res, err := r.db.Exec("UPDATE subscriptions SET " + field + "=" + field + "-? WHERE id=? AND " + field + ">= ?", amount, subscriptionID, amount)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows == 1, nil
}

// SetQuotaValue sets a specific quota field to an exact value for the given subscription id (debug / admin usage only)
func (r *Repository) SetQuotaValue(subscriptionID int, field string, value int) error {
	allowed := map[string]bool{"consultations": true, "questionnaires": true, "clinical_cases": true, "files": true}
	if !allowed[field] { return fmt.Errorf("invalid quota field: %s", field) }
	if value < 0 { value = 0 }
	_, err := r.db.Exec("UPDATE subscriptions SET "+field+"=? WHERE id=?", value, subscriptionID)
	return err
}

// ResetSubscriptionQuotasToPlan sets the subscription quotas back to the plan defaults.
func (r *Repository) ResetSubscriptionQuotasToPlan(subID int) error {
	// Get plan id
	row := r.db.QueryRow(`SELECT plan_id FROM subscriptions WHERE id=? LIMIT 1`, subID)
	var planID int
	if err := row.Scan(&planID); err != nil { return err }
	plan, err := r.GetPlanByID(planID)
	if err != nil { return err }
	if plan == nil { return fmt.Errorf("plan not found for subscription %d", subID) }
	_, err = r.db.Exec(`UPDATE subscriptions SET consultations=?, questionnaires=?, clinical_cases=?, files=? WHERE id=?`, plan.Consultations, plan.Questionnaires, plan.ClinicalCases, plan.Files, subID)
	return err
}

