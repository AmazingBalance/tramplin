BEGIN;

DO $$
BEGIN
    CREATE TYPE media_file_status AS ENUM ('pending', 'uploaded', 'failed', 'deleted');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE media_file_purpose AS ENUM (
        'avatar',
        'resume',
        'company_logo',
        'company_banner',
        'company_media',
        'opportunity_cover',
        'opportunity_media',
        'verification_evidence',
        'other'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS media_files (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    file_key varchar(500) NOT NULL UNIQUE,
    original_name varchar(255),
    mime_type varchar(255),
    file_size bigint,
    uploaded_by_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    purpose media_file_purpose,
    status media_file_status NOT NULL DEFAULT 'pending',
    etag varchar(255),
    completed_at timestamptz,
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS media_files_uploaded_by_user_idx
    ON media_files (uploaded_by_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS media_files_status_idx
    ON media_files (status, created_at DESC);

COMMIT;
