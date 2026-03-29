BEGIN;

DO $$
BEGIN
    CREATE TYPE opportunity_type AS ENUM ('internship', 'vacancy', 'mentor_program', 'event');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE opportunity_status AS ENUM ('draft', 'planned', 'active', 'closed', 'rejected', 'archived');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE moderation_status AS ENUM ('pending', 'approved', 'rejected', 'needs_changes');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE participation_format AS ENUM ('offline', 'hybrid', 'remote', 'online');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE employment_type AS ENUM ('full_time', 'part_time', 'project', 'contract');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE experience_level AS ENUM ('trainee', 'junior', 'middle', 'senior');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE link_type AS ENUM ('apply', 'info', 'registration', 'social', 'other');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS opportunities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id uuid NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    created_by_user_id uuid NOT NULL REFERENCES users(id),
    title varchar(255) NOT NULL,
    summary varchar(500) NOT NULL,
    slug varchar(200) NOT NULL UNIQUE,
    description text NOT NULL,
    type opportunity_type NOT NULL,
    status opportunity_status NOT NULL DEFAULT 'draft',
    moderation_status moderation_status NOT NULL DEFAULT 'pending',
    participation_format participation_format NOT NULL,
    location_id uuid REFERENCES locations(id),
    contact_email varchar(255),
    contact_phone varchar(32),
    cover_media_id uuid,
    published_at timestamptz,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS opportunity_vacancy_details (
    opportunity_id uuid PRIMARY KEY REFERENCES opportunities(id) ON DELETE CASCADE,
    employment_type employment_type NOT NULL,
    experience_level experience_level NOT NULL,
    salary_from int,
    salary_to int,
    currency varchar(3)
);

CREATE TABLE IF NOT EXISTS opportunity_mentor_program_details (
    opportunity_id uuid PRIMARY KEY REFERENCES opportunities(id) ON DELETE CASCADE,
    start_at timestamptz,
    end_at timestamptz,
    seats_count int,
    mentor_requirements text
);

CREATE TABLE IF NOT EXISTS opportunity_event_details (
    opportunity_id uuid PRIMARY KEY REFERENCES opportunities(id) ON DELETE CASCADE,
    start_at timestamptz NOT NULL,
    end_at timestamptz NOT NULL,
    registration_deadline timestamptz,
    capacity int,
    venue_note varchar(255)
);

CREATE TABLE IF NOT EXISTS opportunity_links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id uuid NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,
    link_type link_type NOT NULL,
    title varchar(255) NOT NULL,
    url varchar(500) NOT NULL,
    sort_order int NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS opportunity_links_opportunity_sort_idx
    ON opportunity_links (opportunity_id, sort_order);

CREATE TABLE IF NOT EXISTS opportunity_media (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id uuid NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,
    media_file_id uuid NOT NULL,
    title varchar(255),
    sort_order int NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS opportunity_stats (
    opportunity_id uuid PRIMARY KEY REFERENCES opportunities(id) ON DELETE CASCADE,
    views_count int NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS opportunity_tags (
    opportunity_id uuid NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,
    tag_id uuid NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (opportunity_id, tag_id)
);

COMMIT;
