BEGIN;

DO $$
BEGIN
    CREATE TYPE verification_method AS ENUM (
        'corporate_email',
        'inn',
        'official_website',
        'manual_review'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

DO $$
BEGIN
    CREATE TYPE verification_request_status AS ENUM (
        'pending',
        'under_review',
        'approved',
        'rejected',
        'needs_changes'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

CREATE TABLE IF NOT EXISTS curator_profiles (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    full_name varchar(255) NOT NULL,
    is_admin boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now()
);

DO $$
BEGIN
    ALTER TABLE companies
        ADD CONSTRAINT companies_verified_by_curator_user_id_fkey
        FOREIGN KEY (verified_by_curator_user_id) REFERENCES curator_profiles(user_id);
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

CREATE TABLE IF NOT EXISTS verification_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id uuid NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    submitted_by_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    method verification_method NOT NULL,
    status verification_request_status NOT NULL DEFAULT 'pending',
    submitted_comment text,
    review_comment text,
    reviewed_by_curator_user_id uuid REFERENCES curator_profiles(user_id),
    reviewed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS verification_requests_company_created_idx
    ON verification_requests (company_id, created_at DESC);

CREATE INDEX IF NOT EXISTS verification_requests_status_created_idx
    ON verification_requests (status, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS verification_requests_company_active_key
    ON verification_requests (company_id)
    WHERE status IN ('pending', 'under_review');

CREATE TABLE IF NOT EXISTS verification_evidence (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    verification_request_id uuid NOT NULL REFERENCES verification_requests(id) ON DELETE CASCADE,
    evidence_type varchar(80) NOT NULL,
    value text,
    evidence_file_id uuid REFERENCES media_files(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS verification_evidence_request_idx
    ON verification_evidence (verification_request_id, created_at ASC);

COMMIT;
