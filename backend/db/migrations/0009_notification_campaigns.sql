BEGIN;

DO $$
BEGIN
    CREATE TYPE notification_campaign_audience_type AS ENUM (
        'single_applicant',
        'selected_applicants',
        'all_applicants_of_opportunity'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

DO $$
BEGIN
    CREATE TYPE notification_campaign_status AS ENUM (
        'draft',
        'scheduled',
        'processing',
        'sent',
        'cancelled'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

CREATE TABLE IF NOT EXISTS notification_campaigns (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id uuid NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    created_by_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    opportunity_id uuid REFERENCES opportunities(id) ON DELETE SET NULL,
    audience_type notification_campaign_audience_type NOT NULL,
    status notification_campaign_status NOT NULL DEFAULT 'draft',
    title varchar(255) NOT NULL,
    body text NOT NULL,
    send_via_in_app boolean NOT NULL DEFAULT TRUE,
    send_via_email boolean NOT NULL DEFAULT FALSE,
    scheduled_at timestamptz,
    sent_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS notification_campaigns_company_created_idx
    ON notification_campaigns (company_id, created_at DESC);

CREATE INDEX IF NOT EXISTS notification_campaigns_company_status_created_idx
    ON notification_campaigns (company_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS notification_campaign_recipients (
    campaign_id uuid NOT NULL REFERENCES notification_campaigns(id) ON DELETE CASCADE,
    applicant_user_id uuid NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    application_id uuid REFERENCES applications(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (campaign_id, applicant_user_id)
);

CREATE INDEX IF NOT EXISTS notification_campaign_recipients_campaign_idx
    ON notification_campaign_recipients (campaign_id);

COMMIT;
