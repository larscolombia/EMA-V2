# subscriptions package: Plans and user subscriptions

Overview
- Endpoints:
  - Plans: GET/POST/PUT/DELETE /plans (plus GET /suscription-plans alias)
  - Subscriptions: GET/POST/PUT/DELETE /subscriptions
  - Cancel: POST /cancel-subscription { subscription_id }
  - Checkout (stub): POST /checkout -> { checkout_url }
- Models map to subscription_plans and subscriptions tables, with joined Plan in responses.

Environment variables
- None directly. Uses the shared DB connection.

How it works
- Plans CRUD operates on subscription_plans.
- Subscriptions CRUD operates on subscriptions, returning joined plan as subscription_plan.
- cancel-subscription deletes a subscription by id.
- checkout returns a stub checkout_url for WebView flow; replace with real provider.

Good practices
- Add auth and RBAC checks (only owners/admins can CRUD subscriptions/plans).
- Validate frequency and plan existence; enforce constraints in DB and code.
- Implement soft delete or end_date semantics for cancellations.
- For real payments, integrate Stripe/MercadoPago and sign webhook callbacks.

Architecture notes
- subscriptions exposes HTTP handlers and uses a Repository for DB access.
- Keep business rules in repository/service layer to simplify handlers.
