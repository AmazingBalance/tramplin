BEGIN;

DO $$
BEGIN
    CREATE TYPE notification_type AS ENUM (
        'recommendation_received',
        'application_status_changed',
        'employer_message',
        'employer_broadcast',
        'system'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE notification_source_type AS ENUM (
        'recommendation',
        'application',
        'campaign',
        'system'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS notifications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    type notification_type NOT NULL,
    source_type notification_source_type NOT NULL,
    source_id uuid,
    company_id uuid REFERENCES companies(id) ON DELETE SET NULL,
    opportunity_id uuid REFERENCES opportunities(id) ON DELETE SET NULL,
    application_id uuid REFERENCES applications(id) ON DELETE SET NULL,
    title varchar(255) NOT NULL,
    body text,
    is_read boolean NOT NULL DEFAULT false,
    read_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS notifications_recipient_created_idx
    ON notifications (recipient_user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS notifications_recipient_unread_idx
    ON notifications (recipient_user_id, is_read, created_at DESC, id DESC);

COMMIT;
