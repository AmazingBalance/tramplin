BEGIN;

CREATE TABLE IF NOT EXISTS applicant_social_links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    applicant_user_id uuid NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    platform varchar(120) NOT NULL,
    url varchar(500) NOT NULL,
    is_public boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS applicant_social_links_applicant_created_idx
    ON applicant_social_links (applicant_user_id, created_at, id);

CREATE TABLE IF NOT EXISTS company_social_links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id uuid NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    platform varchar(120) NOT NULL,
    url varchar(500) NOT NULL
);

CREATE INDEX IF NOT EXISTS company_social_links_company_idx
    ON company_social_links (company_id, id);

CREATE TABLE IF NOT EXISTS company_media (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id uuid NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    media_file_id uuid NOT NULL,
    title varchar(255),
    sort_order int NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS company_media_company_sort_idx
    ON company_media (company_id, sort_order, created_at, id);

COMMIT;
