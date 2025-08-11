# migrations package: Schema and data seeding

Overview
- Manages DB schema creation for users, subscription_plans, subscriptions.
- Provides query helpers for users and updates for profile fields.
- Seeds a default admin user and default subscription plans.

Environment variables
- None directly (uses connection provided by conn).

How it works
- Init(db) stores a shared *sql.DB for operations.
- Migrate() executes CREATE TABLE IF NOT EXISTS and ALTERs for new columns.
- SeedDefaultUser() and SeedDefaultPlans() insert initial records when missing.
- Helpers: GetUserByEmail/ID, CreateUser, EmailExists, UpdateUserProfile/Image.

Good practices
- Use real migration tooling (e.g., goose, migrate) for versioned migrations.
- Wrap DDL/DML in transactions where supported and safe.
- Add indexes on frequently filtered columns.
- Avoid plaintext passwords; hash before insert.

Architecture notes
- Central DB access point for core user data; reused by login/profile.
