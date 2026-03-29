BEGIN;

DO $$
BEGIN
    CREATE TYPE moderation_target_type AS ENUM (
        'company',
        'profile',
        'opportunity',
        'tag',
        'media',
        'verification'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

CREATE TABLE IF NOT EXISTS moderation_cases (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type moderation_target_type NOT NULL,
    target_id uuid NOT NULL,
    submitted_by_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    assigned_curator_user_id uuid REFERENCES curator_profiles(user_id) ON DELETE SET NULL,
    resolved_by_curator_user_id uuid REFERENCES curator_profiles(user_id) ON DELETE SET NULL,
    status moderation_status NOT NULL DEFAULT 'pending',
    reason text,
    created_at timestamptz NOT NULL DEFAULT now(),
    resolved_at timestamptz
);

CREATE INDEX IF NOT EXISTS moderation_cases_status_created_idx
    ON moderation_cases (status, created_at DESC);

CREATE INDEX IF NOT EXISTS moderation_cases_target_created_idx
    ON moderation_cases (target_type, target_id, created_at DESC);

COMMIT;
