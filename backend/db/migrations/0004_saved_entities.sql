BEGIN;

CREATE TABLE IF NOT EXISTS applicant_saved_opportunities (
    applicant_user_id uuid NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    opportunity_id uuid NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (applicant_user_id, opportunity_id)
);

CREATE INDEX IF NOT EXISTS applicant_saved_opportunities_created_idx
    ON applicant_saved_opportunities (applicant_user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS applicant_saved_companies (
    applicant_user_id uuid NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    company_id uuid NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (applicant_user_id, company_id)
);

CREATE INDEX IF NOT EXISTS applicant_saved_companies_created_idx
    ON applicant_saved_companies (applicant_user_id, created_at DESC);

COMMIT;
