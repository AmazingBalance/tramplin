BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    CREATE TYPE user_role AS ENUM ('applicant', 'employer', 'curator');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE location_precision AS ENUM ('exact_address', 'city_only');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE tag_type AS ENUM ('technology', 'role', 'domain', 'level', 'employment_type', 'format', 'custom');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE visibility_scope AS ENUM ('private', 'contacts_only', 'employers_only', 'authenticated_public');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE company_verification_status AS ENUM ('pending', 'under_review', 'verified', 'rejected', 'expired');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE company_membership_status AS ENUM ('pending', 'approved', 'rejected', 'revoked');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE company_member_role AS ENUM ('owner', 'recruiter', 'hr', 'manager');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email varchar(255) NOT NULL UNIQUE,
    password_hash varchar(255) NOT NULL,
    display_name varchar(120) NOT NULL,
    role user_role NOT NULL,
    email_verified_at timestamptz,
    is_active boolean NOT NULL DEFAULT true,
    avatar_media_id uuid,
    token_version int NOT NULL DEFAULT 0,
    password_changed_at timestamptz,
    last_login_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_ui_settings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    settings_json jsonb NOT NULL,
    schema_version int NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS locations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    precision location_precision NOT NULL,
    country varchar(120) NOT NULL,
    region varchar(120),
    city varchar(120) NOT NULL,
    address_line varchar(255),
    postal_code varchar(20),
    place_name varchar(255),
    latitude numeric(9, 6),
    longitude numeric(9, 6),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tags (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name varchar(120) NOT NULL,
    tag_type tag_type NOT NULL,
    is_system boolean NOT NULL DEFAULT false,
    is_active boolean NOT NULL DEFAULT true,
    created_by_user_id uuid REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tags_name_tag_type_key UNIQUE (name, tag_type)
);

CREATE TABLE IF NOT EXISTS applicant_profiles (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    last_name varchar(120) NOT NULL,
    first_name varchar(120) NOT NULL,
    middle_name varchar(120),
    university_name varchar(255),
    faculty varchar(255),
    program_name varchar(255),
    study_year smallint,
    graduation_year smallint,
    city varchar(120),
    about text,
    resume_media_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS applicant_privacy_settings (
    applicant_user_id uuid PRIMARY KEY REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    profile_visibility visibility_scope NOT NULL DEFAULT 'authenticated_public',
    resume_visibility visibility_scope NOT NULL DEFAULT 'contacts_only',
    applications_visibility visibility_scope NOT NULL DEFAULT 'private',
    contacts_visibility visibility_scope NOT NULL DEFAULT 'contacts_only',
    show_career_interests boolean NOT NULL DEFAULT true,
    allow_recommendations boolean NOT NULL DEFAULT true,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS applicant_tags (
    applicant_user_id uuid NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    tag_id uuid NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (applicant_user_id, tag_id)
);

CREATE TABLE IF NOT EXISTS employer_profiles (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    full_name varchar(255) NOT NULL,
    job_title varchar(120),
    phone varchar(32),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS companies (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    legal_name varchar(255) NOT NULL,
    brand_name varchar(255),
    slug varchar(150) NOT NULL UNIQUE,
    inn varchar(20) UNIQUE,
    description text,
    industry varchar(120),
    website_url varchar(500),
    corporate_email_domain varchar(120),
    headquarters_location_id uuid REFERENCES locations(id),
    logo_media_id uuid,
    banner_media_id uuid,
    verification_status company_verification_status NOT NULL DEFAULT 'pending',
    verified_at timestamptz,
    verified_by_curator_user_id uuid,
    created_by_user_id uuid REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS company_memberships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id uuid NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employer_user_id uuid NOT NULL REFERENCES employer_profiles(user_id) ON DELETE CASCADE,
    invited_by_user_id uuid REFERENCES employer_profiles(user_id),
    status company_membership_status NOT NULL DEFAULT 'pending',
    member_role company_member_role NOT NULL,
    is_primary_contact boolean NOT NULL DEFAULT false,
    status_changed_by_user_id uuid REFERENCES users(id),
    status_comment text,
    status_updated_at timestamptz NOT NULL DEFAULT now(),
    approved_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT company_memberships_company_id_employer_user_id_key UNIQUE (company_id, employer_user_id)
);

CREATE INDEX IF NOT EXISTS company_memberships_company_status_idx
    ON company_memberships (company_id, status);

CREATE INDEX IF NOT EXISTS company_memberships_employer_status_idx
    ON company_memberships (employer_user_id, status);

CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    in_app_enabled boolean NOT NULL DEFAULT true,
    email_enabled boolean NOT NULL DEFAULT true,
    recommendation_enabled boolean NOT NULL DEFAULT true,
    application_status_enabled boolean NOT NULL DEFAULT true,
    employer_messages_enabled boolean NOT NULL DEFAULT true,
    system_enabled boolean NOT NULL DEFAULT true,
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO tags (name, tag_type, is_system, is_active)
VALUES
    ('Go', 'technology', true, true),
    ('Python', 'technology', true, true),
    ('Backend', 'role', true, true),
    ('Junior', 'level', true, true),
    ('Remote', 'format', true, true)
ON CONFLICT (name, tag_type) DO UPDATE
SET is_system = EXCLUDED.is_system,
    is_active = EXCLUDED.is_active;

COMMIT;
