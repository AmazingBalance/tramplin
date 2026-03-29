BEGIN;

DO $$
BEGIN
    CREATE TYPE connection_status AS ENUM ('pending', 'accepted', 'rejected', 'blocked');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS applicant_connections (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    applicant_low_user_id uuid NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    applicant_high_user_id uuid NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    initiator_user_id uuid NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    status connection_status NOT NULL DEFAULT 'pending',
    initiator_note text,
    responded_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT applicant_connections_unique_pair UNIQUE (applicant_low_user_id, applicant_high_user_id),
    CONSTRAINT applicant_connections_distinct_pair CHECK (applicant_low_user_id <> applicant_high_user_id),
    CONSTRAINT applicant_connections_initiator_in_pair CHECK (
        initiator_user_id = applicant_low_user_id OR initiator_user_id = applicant_high_user_id
    )
);

CREATE INDEX IF NOT EXISTS applicant_connections_low_status_idx
    ON applicant_connections (applicant_low_user_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS applicant_connections_high_status_idx
    ON applicant_connections (applicant_high_user_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS opportunity_recommendations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    recommender_user_id uuid NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    recipient_user_id uuid NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    opportunity_id uuid NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,
    message text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT opportunity_recommendations_distinct_users CHECK (recommender_user_id <> recipient_user_id)
);

CREATE INDEX IF NOT EXISTS opportunity_recommendations_recipient_created_idx
    ON opportunity_recommendations (recipient_user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS opportunity_recommendations_recommender_created_idx
    ON opportunity_recommendations (recommender_user_id, created_at DESC, id DESC);

COMMIT;
