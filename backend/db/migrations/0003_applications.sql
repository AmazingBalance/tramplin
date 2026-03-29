BEGIN;

DO $$
BEGIN
    CREATE TYPE application_status AS ENUM ('submitted', 'reviewing', 'reserve', 'accepted', 'rejected', 'withdrawn');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS applications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id uuid NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,
    applicant_user_id uuid NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    cover_letter text,
    status application_status NOT NULL DEFAULT 'submitted',
    applied_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT applications_opportunity_id_applicant_user_id_key UNIQUE (opportunity_id, applicant_user_id)
);

CREATE INDEX IF NOT EXISTS applications_applicant_status_idx
    ON applications (applicant_user_id, status, applied_at DESC);

CREATE INDEX IF NOT EXISTS applications_opportunity_status_idx
    ON applications (opportunity_id, status, applied_at DESC);

CREATE TABLE IF NOT EXISTS application_status_history (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id uuid NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    old_status application_status,
    new_status application_status NOT NULL,
    changed_by_user_id uuid NOT NULL REFERENCES users(id),
    comment text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS application_status_history_application_created_idx
    ON application_status_history (application_id, created_at);

COMMIT;
