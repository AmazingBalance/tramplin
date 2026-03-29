package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

type Store struct {
	db *pgxpool.Pool
}

type scanner interface {
	Scan(dest ...any) error
}

type queryable interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

var _ commonstore.Repository = (*Store)(nil)

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) CreateApplicant(input model.RegisterApplicantInput, passwordHash string) (*model.User, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create applicant", nil)
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	user, err := scanUser(tx.QueryRow(ctx, `
		INSERT INTO users (
			email, password_hash, display_name, role, is_active, token_version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, TRUE, 0, $5, $5)
		RETURNING id, email, password_hash, display_name, role, is_active, avatar_media_id,
			email_verified_at, last_login_at, token_version, created_at, updated_at
	`, normalizeEmail(input.Email), passwordHash, input.DisplayName, model.UserRoleApplicant, now))
	if err != nil {
		if isUniqueViolation(err, "users_email_key") {
			return nil, appErr(409, "conflict", "user with this email already exists", nil)
		}
		return nil, appErr(500, "internal_error", "failed to create applicant", nil)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO applicant_profiles (
			user_id, last_name, first_name, middle_name, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $5)
	`, user.ID, input.LastName, input.FirstName, input.MiddleName, now)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create applicant profile", nil)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO applicant_privacy_settings (
			applicant_user_id, profile_visibility, resume_visibility, applications_visibility,
			contacts_visibility, show_career_interests, allow_recommendations, updated_at
		) VALUES ($1, 'authenticated_public', 'contacts_only', 'private', 'contacts_only', TRUE, TRUE, $2)
	`, user.ID, now)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create applicant privacy settings", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to create applicant", nil)
	}
	return user, nil
}

func (s *Store) CreateEmployer(input model.RegisterEmployerInput, passwordHash string) (*model.User, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create employer", nil)
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	user, err := scanUser(tx.QueryRow(ctx, `
		INSERT INTO users (
			email, password_hash, display_name, role, is_active, token_version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, TRUE, 0, $5, $5)
		RETURNING id, email, password_hash, display_name, role, is_active, avatar_media_id,
			email_verified_at, last_login_at, token_version, created_at, updated_at
	`, normalizeEmail(input.Email), passwordHash, input.DisplayName, model.UserRoleEmployer, now))
	if err != nil {
		if isUniqueViolation(err, "users_email_key") {
			return nil, appErr(409, "conflict", "user with this email already exists", nil)
		}
		return nil, appErr(500, "internal_error", "failed to create employer", nil)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO employer_profiles (
			user_id, full_name, job_title, phone, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $5)
	`, user.ID, input.FullName, input.JobTitle, input.Phone, now)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create employer profile", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to create employer", nil)
	}
	return user, nil
}

func (s *Store) GetUserByEmail(email string) (*model.User, *commonstore.AppError) {
	user, err := scanUser(s.db.QueryRow(context.Background(), `
		SELECT id, email, password_hash, display_name, role, is_active, avatar_media_id,
			email_verified_at, last_login_at, token_version, created_at, updated_at
		FROM users
		WHERE email = $1
	`, normalizeEmail(email)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(401, "unauthorized", "invalid credentials", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load user", nil)
	}
	return user, nil
}

func (s *Store) GetUserByID(userID string) (*model.User, *commonstore.AppError) {
	user, err := scanUser(s.db.QueryRow(context.Background(), `
		SELECT id, email, password_hash, display_name, role, is_active, avatar_media_id,
			email_verified_at, last_login_at, token_version, created_at, updated_at
		FROM users
		WHERE id = $1
	`, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(401, "unauthorized", "authentication failed", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load user", nil)
	}
	return user, nil
}

func (s *Store) TouchLastLogin(userID string) {
	now := time.Now().UTC()
	_, _ = s.db.Exec(context.Background(), `
		UPDATE users
		SET last_login_at = $1, updated_at = $1
		WHERE id = $2
	`, now, userID)
}

func (s *Store) BumpTokenVersion(userID string) {
	now := time.Now().UTC()
	_, _ = s.db.Exec(context.Background(), `
		UPDATE users
		SET token_version = token_version + 1, updated_at = $1
		WHERE id = $2
	`, now, userID)
}

func (s *Store) GetUISettings(userID string) (*model.UISettings, *commonstore.AppError) {
	ctx := context.Background()
	settings, err := scanUISettings(s.db.QueryRow(ctx, `
		SELECT user_id, settings_json, schema_version, updated_at
		FROM user_ui_settings
		WHERE user_id = $1
	`, userID))
	if err == nil {
		return settings, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, appErr(500, "internal_error", "failed to load ui settings", nil)
	}

	now := time.Now().UTC()
	_, err = s.db.Exec(ctx, `
		INSERT INTO user_ui_settings (user_id, settings_json, schema_version, updated_at)
		VALUES ($1, '{}'::jsonb, 1, $2)
		ON CONFLICT (user_id) DO NOTHING
	`, userID, now)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to initialize ui settings", nil)
	}

	settings, err = scanUISettings(s.db.QueryRow(ctx, `
		SELECT user_id, settings_json, schema_version, updated_at
		FROM user_ui_settings
		WHERE user_id = $1
	`, userID))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load ui settings", nil)
	}
	return settings, nil
}

func (s *Store) PutUISettings(userID string, input model.PutUISettingsInput) (*model.UISettings, *commonstore.AppError) {
	ctx := context.Background()
	now := time.Now().UTC()
	schemaVersion := 1
	if input.SchemaVersion != nil {
		schemaVersion = *input.SchemaVersion
	}
	payload, err := json.Marshal(cloneMap(input.SettingsJSON))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to encode ui settings", nil)
	}

	settings, err := scanUISettings(s.db.QueryRow(ctx, `
		INSERT INTO user_ui_settings (user_id, settings_json, schema_version, updated_at)
		VALUES ($1, $2::jsonb, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		SET settings_json = EXCLUDED.settings_json,
			schema_version = EXCLUDED.schema_version,
			updated_at = EXCLUDED.updated_at
		RETURNING user_id, settings_json, schema_version, updated_at
	`, userID, string(payload), schemaVersion, now))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to save ui settings", nil)
	}
	return settings, nil
}

func (s *Store) GetApplicantProfile(userID string) (*model.ApplicantProfile, *commonstore.AppError) {
	profile, err := scanApplicantProfile(s.db.QueryRow(context.Background(), `
		SELECT user_id, first_name, last_name, middle_name, university_name, faculty,
			program_name, study_year, graduation_year, city, about, resume_media_id,
			created_at, updated_at
		FROM applicant_profiles
		WHERE user_id = $1
	`, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load applicant profile", nil)
	}
	profile.TagIDs = []string{}
	profile.SocialLinks = []model.ApplicantSocialLink{}
	return profile, nil
}

func (s *Store) UpdateApplicantProfile(userID string, input model.UpdateApplicantProfileInput) (*model.ApplicantProfile, *commonstore.AppError) {
	ctx := context.Background()
	profile, repoErr := s.GetApplicantProfile(userID)
	if repoErr != nil {
		return nil, repoErr
	}

	if input.FirstName != nil {
		profile.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		profile.LastName = *input.LastName
	}
	if input.MiddleName != nil {
		profile.MiddleName = input.MiddleName
	}
	if input.UniversityName != nil {
		profile.UniversityName = input.UniversityName
	}
	if input.Faculty != nil {
		profile.Faculty = input.Faculty
	}
	if input.ProgramName != nil {
		profile.ProgramName = input.ProgramName
	}
	if input.StudyYear != nil {
		profile.StudyYear = input.StudyYear
	}
	if input.GraduationYear != nil {
		profile.GraduationYear = input.GraduationYear
	}
	if input.City != nil {
		profile.City = input.City
	}
	if input.About != nil {
		profile.About = input.About
	}
	if input.ResumeMediaID != nil {
		profile.ResumeMediaID = input.ResumeMediaID
	}
	profile.UpdatedAt = time.Now().UTC()

	_, err := s.db.Exec(ctx, `
		UPDATE applicant_profiles
		SET first_name = $1, last_name = $2, middle_name = $3, university_name = $4,
			faculty = $5, program_name = $6, study_year = $7, graduation_year = $8,
			city = $9, about = $10, resume_media_id = $11, updated_at = $12
		WHERE user_id = $13
	`, profile.FirstName, profile.LastName, profile.MiddleName, profile.UniversityName,
		profile.Faculty, profile.ProgramName, int16Ptr(profile.StudyYear), int16Ptr(profile.GraduationYear),
		profile.City, profile.About, profile.ResumeMediaID, profile.UpdatedAt, userID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update applicant profile", nil)
	}
	return profile, nil
}

func (s *Store) GetApplicantPrivacy(userID string) (*model.ApplicantPrivacySettings, *commonstore.AppError) {
	privacy, err := scanApplicantPrivacy(s.db.QueryRow(context.Background(), `
		SELECT applicant_user_id, profile_visibility, resume_visibility, applications_visibility,
			contacts_visibility, show_career_interests, allow_recommendations, updated_at
		FROM applicant_privacy_settings
		WHERE applicant_user_id = $1
	`, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load applicant privacy settings", nil)
	}
	return privacy, nil
}

func (s *Store) UpdateApplicantPrivacy(userID string, input model.UpdateApplicantPrivacyInput) (*model.ApplicantPrivacySettings, *commonstore.AppError) {
	ctx := context.Background()
	privacy, repoErr := s.GetApplicantPrivacy(userID)
	if repoErr != nil {
		return nil, repoErr
	}

	if input.ProfileVisibility != nil {
		privacy.ProfileVisibility = *input.ProfileVisibility
	}
	if input.ResumeVisibility != nil {
		privacy.ResumeVisibility = *input.ResumeVisibility
	}
	if input.ApplicationsVisibility != nil {
		privacy.ApplicationsVisibility = *input.ApplicationsVisibility
	}
	if input.ContactsVisibility != nil {
		privacy.ContactsVisibility = *input.ContactsVisibility
	}
	if input.ShowCareerInterests != nil {
		privacy.ShowCareerInterests = *input.ShowCareerInterests
	}
	if input.AllowRecommendations != nil {
		privacy.AllowRecommendations = *input.AllowRecommendations
	}
	privacy.UpdatedAt = time.Now().UTC()

	_, err := s.db.Exec(ctx, `
		UPDATE applicant_privacy_settings
		SET profile_visibility = $1, resume_visibility = $2, applications_visibility = $3,
			contacts_visibility = $4, show_career_interests = $5, allow_recommendations = $6,
			updated_at = $7
		WHERE applicant_user_id = $8
	`, privacy.ProfileVisibility, privacy.ResumeVisibility, privacy.ApplicationsVisibility,
		privacy.ContactsVisibility, privacy.ShowCareerInterests, privacy.AllowRecommendations,
		privacy.UpdatedAt, userID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update applicant privacy settings", nil)
	}
	return privacy, nil
}

func (s *Store) ReplaceApplicantTags(userID string, tagIDs []string) ([]*model.Tag, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to replace applicant tags", nil)
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM applicant_profiles WHERE user_id = $1)`, userID).Scan(&exists); err != nil {
		return nil, appErr(500, "internal_error", "failed to load applicant profile", nil)
	}
	if !exists {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}

	uniqueIDs := uniqueSortedStrings(tagIDs)
	tags := make([]*model.Tag, 0, len(uniqueIDs))
	for _, tagID := range uniqueIDs {
		tag, err := scanTag(tx.QueryRow(ctx, `
			SELECT id, name, tag_type, is_system, is_active, created_by_user_id, created_at
			FROM tags
			WHERE id = $1
		`, tagID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, appErr(422, "validation_error", "unknown tag id", map[string]any{"tagId": tagID})
			}
			return nil, appErr(500, "internal_error", "failed to load tags", nil)
		}
		tags = append(tags, tag)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM applicant_tags WHERE applicant_user_id = $1`, userID); err != nil {
		return nil, appErr(500, "internal_error", "failed to replace applicant tags", nil)
	}
	for _, tagID := range uniqueIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO applicant_tags (applicant_user_id, tag_id)
			VALUES ($1, $2)
		`, userID, tagID); err != nil {
			return nil, appErr(500, "internal_error", "failed to replace applicant tags", nil)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE applicant_profiles
		SET updated_at = $1
		WHERE user_id = $2
	`, time.Now().UTC(), userID); err != nil {
		return nil, appErr(500, "internal_error", "failed to update applicant tags", nil)
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to replace applicant tags", nil)
	}
	return tags, nil
}

func (s *Store) GetApplicantTags(userID string) ([]*model.Tag, *commonstore.AppError) {
	ctx := context.Background()
	rows, err := s.db.Query(ctx, `
		SELECT t.id, t.name, t.tag_type, t.is_system, t.is_active, t.created_by_user_id, t.created_at
		FROM applicant_tags at
		JOIN tags t ON t.id = at.tag_id
		WHERE at.applicant_user_id = $1
		ORDER BY t.name ASC
	`, userID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load applicant tags", nil)
	}
	defer rows.Close()

	items := make([]*model.Tag, 0)
	for rows.Next() {
		tag, err := scanTag(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load applicant tags", nil)
		}
		items = append(items, tag)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load applicant tags", nil)
	}
	return items, nil
}

func (s *Store) GetEmployerProfile(userID string) (*model.EmployerProfile, *commonstore.AppError) {
	profile, err := scanEmployerProfile(s.db.QueryRow(context.Background(), `
		SELECT user_id, full_name, job_title, phone, created_at, updated_at
		FROM employer_profiles
		WHERE user_id = $1
	`, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(403, "forbidden", "current user is not an employer", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load employer profile", nil)
	}
	return profile, nil
}

func (s *Store) UpdateEmployerProfile(userID string, input model.UpdateEmployerProfileInput) (*model.EmployerProfile, *commonstore.AppError) {
	ctx := context.Background()
	profile, repoErr := s.GetEmployerProfile(userID)
	if repoErr != nil {
		return nil, repoErr
	}

	if input.FullName != nil {
		profile.FullName = *input.FullName
	}
	if input.JobTitle != nil {
		profile.JobTitle = input.JobTitle
	}
	if input.Phone != nil {
		profile.Phone = input.Phone
	}
	profile.UpdatedAt = time.Now().UTC()

	_, err := s.db.Exec(ctx, `
		UPDATE employer_profiles
		SET full_name = $1, job_title = $2, phone = $3, updated_at = $4
		WHERE user_id = $5
	`, profile.FullName, profile.JobTitle, profile.Phone, profile.UpdatedAt, userID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update employer profile", nil)
	}
	return profile, nil
}

func (s *Store) ListPublicTags(tagType, q string) ([]*model.Tag, *commonstore.AppError) {
	ctx := context.Background()
	rows, err := s.db.Query(ctx, `
		SELECT id, name, tag_type, is_system, is_active, created_by_user_id, created_at
		FROM tags
		WHERE is_active = TRUE
			AND ($1 = '' OR tag_type::text = $1)
			AND ($2 = '' OR name ILIKE '%' || $2 || '%')
		ORDER BY name ASC
	`, tagType, q)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load tags", nil)
	}
	defer rows.Close()

	items := make([]*model.Tag, 0)
	for rows.Next() {
		tag, err := scanTag(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load tags", nil)
		}
		items = append(items, tag)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load tags", nil)
	}
	return items, nil
}

func (s *Store) CreateOrReuseLocation(input model.LocationInput) (*model.Location, *commonstore.AppError) {
	ctx := context.Background()
	location, err := s.findMatchingLocation(ctx, s.db, input)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load location", nil)
	}
	if location != nil {
		return location, nil
	}

	now := time.Now().UTC()
	location, err = scanLocation(s.db.QueryRow(ctx, `
		INSERT INTO locations (
			precision, country, region, city, address_line, postal_code, place_name,
			latitude, longitude, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, precision, country, region, city, address_line, postal_code,
			place_name, latitude::float8, longitude::float8, created_at
	`, input.Precision, input.Country, input.Region, input.City, input.AddressLine,
		input.PostalCode, input.PlaceName, input.Latitude, input.Longitude, now))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create location", nil)
	}
	return location, nil
}

func (s *Store) SearchLocations(q string, page, pageSize int) ([]*model.Location, int, *commonstore.AppError) {
	ctx := context.Background()
	rows, err := s.db.Query(ctx, `
		SELECT id, precision, country, region, city, address_line, postal_code,
			place_name, latitude::float8, longitude::float8, created_at
		FROM locations
		WHERE $1 = ''
			OR country ILIKE '%' || $1 || '%'
			OR city ILIKE '%' || $1 || '%'
			OR COALESCE(region, '') ILIKE '%' || $1 || '%'
			OR COALESCE(address_line, '') ILIKE '%' || $1 || '%'
			OR COALESCE(place_name, '') ILIKE '%' || $1 || '%'
		ORDER BY created_at DESC
	`, q)
	if err != nil {
		return nil, 0, appErr(500, "internal_error", "failed to search locations", nil)
	}
	defer rows.Close()

	items := make([]*model.Location, 0)
	for rows.Next() {
		location, err := scanLocation(rows)
		if err != nil {
			return nil, 0, appErr(500, "internal_error", "failed to search locations", nil)
		}
		items = append(items, location)
	}
	if rows.Err() != nil {
		return nil, 0, appErr(500, "internal_error", "failed to search locations", nil)
	}
	paged, total := paginate(items, page, pageSize)
	return paged, total, nil
}

func (s *Store) CreateCompany(actorUserID string, input model.CreateCompanyInput) (*model.Company, *model.CompanyMembership, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, appErr(500, "internal_error", "failed to create company", nil)
	}
	defer tx.Rollback(ctx)

	if !s.employerExists(ctx, tx, actorUserID) {
		return nil, nil, appErr(403, "forbidden", "current user is not an employer", nil)
	}
	if input.HeadquartersLocationID != nil && input.HeadquartersLocation != nil {
		return nil, nil, appErr(422, "validation_error", "provide either headquartersLocationId or headquartersLocation", nil)
	}
	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		return nil, nil, appErr(422, "validation_error", "slug is required", nil)
	}

	locationID, repoErr := s.resolveLocation(ctx, tx, input.HeadquartersLocationID, input.HeadquartersLocation)
	if repoErr != nil {
		return nil, nil, repoErr
	}

	now := time.Now().UTC()
	company, err := scanCompany(tx.QueryRow(ctx, `
		INSERT INTO companies (
			legal_name, brand_name, slug, inn, description, industry, website_url,
			corporate_email_domain, headquarters_location_id, logo_media_id, banner_media_id,
			verification_status, created_by_user_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'pending', $12, $13, $13)
		RETURNING id, legal_name, brand_name, slug, inn, description, industry, website_url,
			corporate_email_domain, headquarters_location_id, logo_media_id, banner_media_id,
			verification_status, verified_at, verified_by_curator_user_id, created_by_user_id,
			created_at, updated_at
	`, input.LegalName, input.BrandName, slug, input.INN, input.Description, input.Industry,
		input.WebsiteURL, input.CorporateEmailDomain, locationID, input.LogoMediaID,
		input.BannerMediaID, actorUserID, now))
	if err != nil {
		if isUniqueViolation(err, "companies_slug_key") || isUniqueViolation(err, "companies_inn_key") {
			return nil, nil, appErr(409, "conflict", "company slug already exists", nil)
		}
		return nil, nil, appErr(500, "internal_error", "failed to create company", nil)
	}

	membership, err := scanCompanyMembership(tx.QueryRow(ctx, `
		INSERT INTO company_memberships (
			company_id, employer_user_id, status, member_role, is_primary_contact,
			status_updated_at, approved_at, updated_at, created_at
		) VALUES ($1, $2, 'approved', 'owner', TRUE, $3, $3, $3, $3)
		RETURNING id, company_id, employer_user_id, invited_by_user_id, status, member_role,
			is_primary_contact, status_changed_by_user_id, status_comment, status_updated_at,
			approved_at, updated_at, created_at
	`, company.ID, actorUserID, now))
	if err != nil {
		return nil, nil, appErr(500, "internal_error", "failed to create company membership", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, appErr(500, "internal_error", "failed to create company", nil)
	}
	return company, membership, nil
}

func (s *Store) UpdateCompany(actorUserID, companyID string, input model.UpdateCompanyInput) (*model.Company, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update company", nil)
	}
	defer tx.Rollback(ctx)

	if _, repoErr := s.requireApprovedMembership(ctx, tx, actorUserID, companyID); repoErr != nil {
		return nil, repoErr
	}
	if input.HeadquartersLocationID != nil && input.HeadquartersLocation != nil {
		return nil, appErr(422, "validation_error", "provide either headquartersLocationId or headquartersLocation", nil)
	}

	company, err := scanCompany(tx.QueryRow(ctx, `
		SELECT id, legal_name, brand_name, slug, inn, description, industry, website_url,
			corporate_email_domain, headquarters_location_id, logo_media_id, banner_media_id,
			verification_status, verified_at, verified_by_curator_user_id, created_by_user_id,
			created_at, updated_at
		FROM companies
		WHERE id = $1
	`, companyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "company not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load company", nil)
	}

	if input.Slug != nil {
		company.Slug = *input.Slug
	}
	if input.LegalName != nil {
		company.LegalName = *input.LegalName
	}
	if input.BrandName != nil {
		company.BrandName = input.BrandName
	}
	if input.INN != nil {
		company.INN = input.INN
	}
	if input.Description != nil {
		company.Description = input.Description
	}
	if input.Industry != nil {
		company.Industry = input.Industry
	}
	if input.WebsiteURL != nil {
		company.WebsiteURL = input.WebsiteURL
	}
	if input.CorporateEmailDomain != nil {
		company.CorporateEmailDomain = input.CorporateEmailDomain
	}
	if input.LogoMediaID != nil {
		company.LogoMediaID = input.LogoMediaID
	}
	if input.BannerMediaID != nil {
		company.BannerMediaID = input.BannerMediaID
	}
	if input.HeadquartersLocationID != nil || input.HeadquartersLocation != nil {
		locationID, repoErr := s.resolveLocation(ctx, tx, input.HeadquartersLocationID, input.HeadquartersLocation)
		if repoErr != nil {
			return nil, repoErr
		}
		company.HeadquartersLocationID = locationID
	}
	company.UpdatedAt = time.Now().UTC()

	_, err = tx.Exec(ctx, `
		UPDATE companies
		SET legal_name = $1, brand_name = $2, slug = $3, inn = $4, description = $5,
			industry = $6, website_url = $7, corporate_email_domain = $8,
			headquarters_location_id = $9, logo_media_id = $10, banner_media_id = $11,
			updated_at = $12
		WHERE id = $13
	`, company.LegalName, company.BrandName, company.Slug, company.INN, company.Description,
		company.Industry, company.WebsiteURL, company.CorporateEmailDomain, company.HeadquartersLocationID,
		company.LogoMediaID, company.BannerMediaID, company.UpdatedAt, company.ID)
	if err != nil {
		if isUniqueViolation(err, "companies_slug_key") || isUniqueViolation(err, "companies_inn_key") {
			return nil, appErr(409, "conflict", "company slug already exists", nil)
		}
		return nil, appErr(500, "internal_error", "failed to update company", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update company", nil)
	}
	return company, nil
}

func (s *Store) ListEmployerMemberships(userID string) ([]*model.CompanyMembership, *commonstore.AppError) {
	ctx := context.Background()
	if !s.employerExists(ctx, s.db, userID) {
		return nil, appErr(403, "forbidden", "current user is not an employer", nil)
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, company_id, employer_user_id, invited_by_user_id, status, member_role,
			is_primary_contact, status_changed_by_user_id, status_comment, status_updated_at,
			approved_at, updated_at, created_at
		FROM company_memberships
		WHERE employer_user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load memberships", nil)
	}
	defer rows.Close()
	return scanMemberships(rows)
}

func (s *Store) GetCompanyForEmployer(userID, companyID string) (*model.Company, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireApprovedMembership(ctx, s.db, userID, companyID); repoErr != nil {
		return nil, repoErr
	}
	return s.GetPublicCompanyByID(companyID)
}

func (s *Store) ListCompanyMemberships(actorUserID, companyID, status string) ([]*model.CompanyMembership, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireApprovedMembership(ctx, s.db, actorUserID, companyID); repoErr != nil {
		return nil, repoErr
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, company_id, employer_user_id, invited_by_user_id, status, member_role,
			is_primary_contact, status_changed_by_user_id, status_comment, status_updated_at,
			approved_at, updated_at, created_at
		FROM company_memberships
		WHERE company_id = $1
			AND ($2 = '' OR status::text = $2)
		ORDER BY created_at ASC
	`, companyID, status)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load memberships", nil)
	}
	defer rows.Close()
	return scanMemberships(rows)
}

func (s *Store) CreateCompanyMembership(actorUserID, companyID string, input model.CreateCompanyMembershipInput) (*model.CompanyMembership, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create membership", nil)
	}
	defer tx.Rollback(ctx)

	actorMembership, repoErr := s.requireApprovedMembership(ctx, tx, actorUserID, companyID)
	if repoErr != nil {
		return nil, repoErr
	}

	targetUser, err := scanUser(tx.QueryRow(ctx, `
		SELECT u.id, u.email, u.password_hash, u.display_name, u.role, u.is_active, u.avatar_media_id,
			u.email_verified_at, u.last_login_at, u.token_version, u.created_at, u.updated_at
		FROM users u
		JOIN employer_profiles ep ON ep.user_id = u.id
		WHERE u.email = $1 AND u.role = 'employer'
	`, normalizeEmail(input.EmployerEmail)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "target employer account not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load target employer", nil)
	}

	if !canInviteIntoRole(actorMembership.MemberRole, input.MemberRole, input.IsPrimaryContact) {
		return nil, appErr(403, "forbidden", "current employer cannot invite members to this company or requested role is not allowed", nil)
	}

	existing, err := scanCompanyMembership(tx.QueryRow(ctx, `
		SELECT id, company_id, employer_user_id, invited_by_user_id, status, member_role,
			is_primary_contact, status_changed_by_user_id, status_comment, status_updated_at,
			approved_at, updated_at, created_at
		FROM company_memberships
		WHERE company_id = $1 AND employer_user_id = $2
		FOR UPDATE
	`, companyID, targetUser.ID))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, appErr(500, "internal_error", "failed to load membership", nil)
	}

	now := time.Now().UTC()
	if err == nil {
		if existing.Status == model.CompanyMembershipPending || existing.Status == model.CompanyMembershipApproved {
			return nil, appErr(409, "conflict", "membership already exists in pending or approved state", nil)
		}

		existing.Status = model.CompanyMembershipPending
		existing.MemberRole = input.MemberRole
		existing.IsPrimaryContact = input.IsPrimaryContact
		existing.InvitedByUserID = stringPtr(actorUserID)
		existing.StatusChangedByUserID = stringPtr(actorUserID)
		existing.StatusComment = input.Comment
		existing.StatusUpdatedAt = now
		existing.UpdatedAt = now

		_, err = tx.Exec(ctx, `
			UPDATE company_memberships
			SET invited_by_user_id = $1, status = 'pending', member_role = $2,
				is_primary_contact = $3, status_changed_by_user_id = $4, status_comment = $5,
				status_updated_at = $6, updated_at = $6
			WHERE id = $7
		`, existing.InvitedByUserID, existing.MemberRole, existing.IsPrimaryContact,
			existing.StatusChangedByUserID, existing.StatusComment, now, existing.ID)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to reopen membership", nil)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, appErr(500, "internal_error", "failed to reopen membership", nil)
		}
		return existing, nil
	}

	membership, err := scanCompanyMembership(tx.QueryRow(ctx, `
		INSERT INTO company_memberships (
			company_id, employer_user_id, invited_by_user_id, status, member_role,
			is_primary_contact, status_changed_by_user_id, status_comment, status_updated_at,
			updated_at, created_at
		) VALUES ($1, $2, $3, 'pending', $4, $5, $6, $7, $8, $8, $8)
		RETURNING id, company_id, employer_user_id, invited_by_user_id, status, member_role,
			is_primary_contact, status_changed_by_user_id, status_comment, status_updated_at,
			approved_at, updated_at, created_at
	`, companyID, targetUser.ID, actorUserID, input.MemberRole, input.IsPrimaryContact, actorUserID, input.Comment, now))
	if err != nil {
		if isUniqueViolation(err, "company_memberships_company_id_employer_user_id_key") {
			return nil, appErr(409, "conflict", "membership already exists in pending or approved state", nil)
		}
		return nil, appErr(500, "internal_error", "failed to create membership", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to create membership", nil)
	}
	return membership, nil
}

func (s *Store) UpdateCompanyMembership(actorUserID, companyID, membershipID string, input model.UpdateCompanyMembershipInput) (*model.CompanyMembership, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update membership", nil)
	}
	defer tx.Rollback(ctx)

	actorMembership, repoErr := s.requireApprovedMembership(ctx, tx, actorUserID, companyID)
	if repoErr != nil {
		return nil, repoErr
	}

	target, err := scanCompanyMembership(tx.QueryRow(ctx, `
		SELECT id, company_id, employer_user_id, invited_by_user_id, status, member_role,
			is_primary_contact, status_changed_by_user_id, status_comment, status_updated_at,
			approved_at, updated_at, created_at
		FROM company_memberships
		WHERE id = $1 AND company_id = $2
		FOR UPDATE
	`, membershipID, companyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "membership not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load membership", nil)
	}
	if target.Status != model.CompanyMembershipApproved {
		return nil, appErr(409, "conflict", "requested change would violate membership lifecycle or owner invariants", nil)
	}
	if !canEditMembership(actorMembership.MemberRole, target.MemberRole) {
		return nil, appErr(403, "forbidden", "current employer cannot change this membership", nil)
	}

	if input.MemberRole != nil {
		if !canAssignRole(actorMembership.MemberRole, target.MemberRole, *input.MemberRole) {
			return nil, appErr(403, "forbidden", "current employer cannot change this membership", nil)
		}
		if target.MemberRole == model.CompanyRoleOwner && *input.MemberRole != model.CompanyRoleOwner {
			owners, err := s.countApprovedOwners(ctx, tx, companyID)
			if err != nil {
				return nil, appErr(500, "internal_error", "failed to validate owner invariants", nil)
			}
			if owners <= 1 {
				return nil, appErr(409, "conflict", "requested change would violate membership lifecycle or owner invariants", nil)
			}
		}
		target.MemberRole = *input.MemberRole
	}

	if input.IsPrimaryContact != nil {
		if *input.IsPrimaryContact {
			if actorMembership.MemberRole != model.CompanyRoleOwner {
				return nil, appErr(403, "forbidden", "current employer cannot change this membership", nil)
			}
			if _, err := tx.Exec(ctx, `
				UPDATE company_memberships
				SET is_primary_contact = FALSE, updated_at = $1
				WHERE company_id = $2 AND is_primary_contact = TRUE
			`, time.Now().UTC(), companyID); err != nil {
				return nil, appErr(500, "internal_error", "failed to update primary contact", nil)
			}
			target.IsPrimaryContact = true
		} else if target.IsPrimaryContact {
			return nil, appErr(409, "conflict", "requested change would violate membership lifecycle or owner invariants", nil)
		}
	}

	target.UpdatedAt = time.Now().UTC()
	_, err = tx.Exec(ctx, `
		UPDATE company_memberships
		SET member_role = $1, is_primary_contact = $2, updated_at = $3
		WHERE id = $4
	`, target.MemberRole, target.IsPrimaryContact, target.UpdatedAt, target.ID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update membership", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update membership", nil)
	}
	return target, nil
}

func (s *Store) ApproveCompanyMembership(actorUserID, companyID, membershipID string, comment *string) (*model.CompanyMembership, *commonstore.AppError) {
	return s.transitionMembership(actorUserID, companyID, membershipID, model.CompanyMembershipApproved, comment)
}

func (s *Store) RejectCompanyMembership(actorUserID, companyID, membershipID string, comment string) (*model.CompanyMembership, *commonstore.AppError) {
	return s.transitionMembership(actorUserID, companyID, membershipID, model.CompanyMembershipRejected, &comment)
}

func (s *Store) RevokeCompanyMembership(actorUserID, companyID, membershipID string, comment string) (*model.CompanyMembership, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to revoke membership", nil)
	}
	defer tx.Rollback(ctx)

	actorMembership, repoErr := s.requireApprovedMembership(ctx, tx, actorUserID, companyID)
	if repoErr != nil {
		return nil, repoErr
	}
	target, err := scanCompanyMembership(tx.QueryRow(ctx, `
		SELECT id, company_id, employer_user_id, invited_by_user_id, status, member_role,
			is_primary_contact, status_changed_by_user_id, status_comment, status_updated_at,
			approved_at, updated_at, created_at
		FROM company_memberships
		WHERE id = $1 AND company_id = $2
		FOR UPDATE
	`, membershipID, companyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "membership not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load membership", nil)
	}
	if target.Status != model.CompanyMembershipApproved {
		return nil, appErr(409, "conflict", "membership is not approved or revocation would remove the last approved owner", nil)
	}
	if !canFinalizeMembership(actorMembership.MemberRole, target.MemberRole) {
		return nil, appErr(403, "forbidden", "current employer cannot revoke this membership", nil)
	}
	if target.MemberRole == model.CompanyRoleOwner {
		owners, err := s.countApprovedOwners(ctx, tx, companyID)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to validate owner invariants", nil)
		}
		if owners <= 1 {
			return nil, appErr(409, "conflict", "membership is not approved or revocation would remove the last approved owner", nil)
		}
	}

	now := time.Now().UTC()
	target.Status = model.CompanyMembershipRevoked
	target.StatusChangedByUserID = stringPtr(actorUserID)
	target.StatusComment = &comment
	target.StatusUpdatedAt = now
	target.UpdatedAt = now
	target.IsPrimaryContact = false

	_, err = tx.Exec(ctx, `
		UPDATE company_memberships
		SET status = 'revoked', status_changed_by_user_id = $1, status_comment = $2,
			status_updated_at = $3, updated_at = $3, is_primary_contact = FALSE
		WHERE id = $4
	`, actorUserID, comment, now, target.ID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to revoke membership", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to revoke membership", nil)
	}
	return target, nil
}

func (s *Store) ListPublicCompanies(q, industry, city string, page, pageSize int) ([]*model.Company, int) {
	ctx := context.Background()
	rows, err := s.db.Query(ctx, `
		SELECT c.id, c.legal_name, c.brand_name, c.slug, c.inn, c.description, c.industry,
			c.website_url, c.corporate_email_domain, c.headquarters_location_id, c.logo_media_id,
			c.banner_media_id, c.verification_status, c.verified_at, c.verified_by_curator_user_id,
			c.created_by_user_id, c.created_at, c.updated_at
		FROM companies c
		LEFT JOIN locations l ON l.id = c.headquarters_location_id
		WHERE ($1 = '' OR c.legal_name ILIKE '%' || $1 || '%' OR COALESCE(c.brand_name, '') ILIKE '%' || $1 || '%')
			AND ($2 = '' OR COALESCE(c.industry, '') ILIKE '%' || $2 || '%')
			AND ($3 = '' OR COALESCE(l.city, '') ILIKE '%' || $3 || '%')
		ORDER BY c.created_at DESC
	`, q, industry, city)
	if err != nil {
		return []*model.Company{}, 0
	}
	defer rows.Close()

	items := make([]*model.Company, 0)
	for rows.Next() {
		company, scanErr := scanCompany(rows)
		if scanErr != nil {
			return []*model.Company{}, 0
		}
		items = append(items, company)
	}
	paged, total := paginate(items, page, pageSize)
	return paged, total
}

func (s *Store) GetPublicCompanyByID(companyID string) (*model.Company, *commonstore.AppError) {
	company, err := scanCompany(s.db.QueryRow(context.Background(), `
		SELECT id, legal_name, brand_name, slug, inn, description, industry, website_url,
			corporate_email_domain, headquarters_location_id, logo_media_id, banner_media_id,
			verification_status, verified_at, verified_by_curator_user_id, created_by_user_id,
			created_at, updated_at
		FROM companies
		WHERE id = $1
	`, companyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "company not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load company", nil)
	}
	return company, nil
}

func (s *Store) GetPublicCompanyBySlug(slug string) (*model.Company, *commonstore.AppError) {
	company, err := scanCompany(s.db.QueryRow(context.Background(), `
		SELECT id, legal_name, brand_name, slug, inn, description, industry, website_url,
			corporate_email_domain, headquarters_location_id, logo_media_id, banner_media_id,
			verification_status, verified_at, verified_by_curator_user_id, created_by_user_id,
			created_at, updated_at
		FROM companies
		WHERE slug = $1
	`, slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "company not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load company", nil)
	}
	return company, nil
}

func (s *Store) GetLocationByID(locationID *string) *model.Location {
	if locationID == nil {
		return nil
	}
	location, err := scanLocation(s.db.QueryRow(context.Background(), `
		SELECT id, precision, country, region, city, address_line, postal_code,
			place_name, latitude::float8, longitude::float8, created_at
		FROM locations
		WHERE id = $1
	`, *locationID))
	if err != nil {
		return nil
	}
	return location
}

func (s *Store) GetEmployerSnapshot(userID string) (*model.User, *model.EmployerProfile) {
	ctx := context.Background()
	user, err := scanUser(s.db.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, role, is_active, avatar_media_id,
			email_verified_at, last_login_at, token_version, created_at, updated_at
		FROM users
		WHERE id = $1
	`, userID))
	if err != nil {
		return nil, nil
	}
	profile, err := scanEmployerProfile(s.db.QueryRow(ctx, `
		SELECT user_id, full_name, job_title, phone, created_at, updated_at
		FROM employer_profiles
		WHERE user_id = $1
	`, userID))
	if err != nil {
		return user, nil
	}
	return user, profile
}

func (s *Store) GetNotificationPreferences(userID string) (*model.NotificationPreferences, *commonstore.AppError) {
	return s.ensureNotificationPreferences(userID)
}

func (s *Store) PatchNotificationPreferences(
	userID string,
	inAppEnabled, emailEnabled, recommendationEnabled, applicationStatusEnabled, employerMessagesEnabled, systemEnabled *bool,
) (*model.NotificationPreferences, *commonstore.AppError) {
	preferences, repoErr := s.ensureNotificationPreferences(userID)
	if repoErr != nil {
		return nil, repoErr
	}

	if inAppEnabled != nil {
		preferences.InAppEnabled = *inAppEnabled
	}
	if emailEnabled != nil {
		preferences.EmailEnabled = *emailEnabled
	}
	if recommendationEnabled != nil {
		preferences.RecommendationEnabled = *recommendationEnabled
	}
	if applicationStatusEnabled != nil {
		preferences.ApplicationStatusEnabled = *applicationStatusEnabled
	}
	if employerMessagesEnabled != nil {
		preferences.EmployerMessagesEnabled = *employerMessagesEnabled
	}
	if systemEnabled != nil {
		preferences.SystemEnabled = *systemEnabled
	}
	preferences.UpdatedAt = time.Now().UTC()

	updated, err := scanNotificationPreferences(s.db.QueryRow(context.Background(), `
		UPDATE notification_preferences
		SET in_app_enabled = $1, email_enabled = $2, recommendation_enabled = $3,
			application_status_enabled = $4, employer_messages_enabled = $5, system_enabled = $6,
			updated_at = $7
		WHERE user_id = $8
		RETURNING user_id, in_app_enabled, email_enabled, recommendation_enabled,
			application_status_enabled, employer_messages_enabled, system_enabled, updated_at
	`, preferences.InAppEnabled, preferences.EmailEnabled, preferences.RecommendationEnabled,
		preferences.ApplicationStatusEnabled, preferences.EmployerMessagesEnabled, preferences.SystemEnabled,
		preferences.UpdatedAt, userID))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update notification preferences", nil)
	}
	return updated, nil
}

func (s *Store) ensureNotificationPreferences(userID string) (*model.NotificationPreferences, *commonstore.AppError) {
	ctx := context.Background()
	preferences, err := scanNotificationPreferences(s.db.QueryRow(ctx, `
		SELECT user_id, in_app_enabled, email_enabled, recommendation_enabled,
			application_status_enabled, employer_messages_enabled, system_enabled, updated_at
		FROM notification_preferences
		WHERE user_id = $1
	`, userID))
	if err == nil {
		return preferences, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, appErr(500, "internal_error", "failed to load notification preferences", nil)
	}

	now := time.Now().UTC()
	_, err = s.db.Exec(ctx, `
		INSERT INTO notification_preferences (
			user_id, in_app_enabled, email_enabled, recommendation_enabled,
			application_status_enabled, employer_messages_enabled, system_enabled, updated_at
		) VALUES ($1, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, $2)
		ON CONFLICT (user_id) DO NOTHING
	`, userID, now)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to initialize notification preferences", nil)
	}

	preferences, err = scanNotificationPreferences(s.db.QueryRow(ctx, `
		SELECT user_id, in_app_enabled, email_enabled, recommendation_enabled,
			application_status_enabled, employer_messages_enabled, system_enabled, updated_at
		FROM notification_preferences
		WHERE user_id = $1
	`, userID))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load notification preferences", nil)
	}
	return preferences, nil
}

func (s *Store) transitionMembership(actorUserID, companyID, membershipID, targetStatus string, comment *string) (*model.CompanyMembership, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update membership", nil)
	}
	defer tx.Rollback(ctx)

	actorMembership, repoErr := s.requireApprovedMembership(ctx, tx, actorUserID, companyID)
	if repoErr != nil {
		return nil, repoErr
	}
	target, err := scanCompanyMembership(tx.QueryRow(ctx, `
		SELECT id, company_id, employer_user_id, invited_by_user_id, status, member_role,
			is_primary_contact, status_changed_by_user_id, status_comment, status_updated_at,
			approved_at, updated_at, created_at
		FROM company_memberships
		WHERE id = $1 AND company_id = $2
		FOR UPDATE
	`, membershipID, companyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "membership not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load membership", nil)
	}
	if target.Status != model.CompanyMembershipPending {
		if targetStatus == model.CompanyMembershipApproved {
			return nil, appErr(409, "conflict", "membership is not pending or approval would violate owner-role rules", nil)
		}
		return nil, appErr(409, "conflict", "membership is not pending or rejection would violate owner-role rules", nil)
	}
	if !canFinalizeMembership(actorMembership.MemberRole, target.MemberRole) {
		if targetStatus == model.CompanyMembershipApproved {
			return nil, appErr(403, "forbidden", "current employer cannot approve this membership", nil)
		}
		return nil, appErr(403, "forbidden", "current employer cannot reject this membership", nil)
	}

	now := time.Now().UTC()
	target.Status = targetStatus
	target.StatusChangedByUserID = stringPtr(actorUserID)
	target.StatusComment = comment
	target.StatusUpdatedAt = now
	target.UpdatedAt = now
	if targetStatus == model.CompanyMembershipApproved {
		target.ApprovedAt = &now
	}

	_, err = tx.Exec(ctx, `
		UPDATE company_memberships
		SET status = $1, status_changed_by_user_id = $2, status_comment = $3,
			status_updated_at = $4, updated_at = $4, approved_at = CASE WHEN $1 = 'approved' THEN $4 ELSE approved_at END
		WHERE id = $5
	`, targetStatus, actorUserID, comment, now, target.ID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update membership", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update membership", nil)
	}
	return target, nil
}

func (s *Store) employerExists(ctx context.Context, q queryable, userID string) bool {
	var exists bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM employer_profiles WHERE user_id = $1)`, userID).Scan(&exists); err != nil {
		return false
	}
	return exists
}

func (s *Store) requireApprovedMembership(ctx context.Context, q queryable, userID, companyID string) (*model.CompanyMembership, *commonstore.AppError) {
	membership, err := scanCompanyMembership(q.QueryRow(ctx, `
		SELECT id, company_id, employer_user_id, invited_by_user_id, status, member_role,
			is_primary_contact, status_changed_by_user_id, status_comment, status_updated_at,
			approved_at, updated_at, created_at
		FROM company_memberships
		WHERE company_id = $1 AND employer_user_id = $2
	`, companyID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(403, "forbidden", "current employer has no access to this company", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load membership", nil)
	}
	if membership.Status != model.CompanyMembershipApproved {
		return nil, appErr(403, "forbidden", "current employer has no access to this company", nil)
	}
	return membership, nil
}

func (s *Store) countApprovedOwners(ctx context.Context, q queryable, companyID string) (int, error) {
	var count int
	err := q.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM company_memberships
		WHERE company_id = $1 AND status = 'approved' AND member_role = 'owner'
	`, companyID).Scan(&count)
	return count, err
}

func (s *Store) resolveLocation(ctx context.Context, q queryable, locationID *string, input *model.LocationInput) (*string, *commonstore.AppError) {
	if locationID != nil {
		var exists bool
		if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM locations WHERE id = $1)`, *locationID).Scan(&exists); err != nil {
			return nil, appErr(500, "internal_error", "failed to validate location", nil)
		}
		if !exists {
			return nil, appErr(422, "validation_error", "unknown location id", nil)
		}
		return locationID, nil
	}
	if input == nil {
		return nil, nil
	}

	location, err := s.findMatchingLocation(ctx, q, *input)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load location", nil)
	}
	if location != nil {
		return stringPtr(location.ID), nil
	}

	now := time.Now().UTC()
	location, err = scanLocation(q.QueryRow(ctx, `
		INSERT INTO locations (
			precision, country, region, city, address_line, postal_code, place_name,
			latitude, longitude, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, precision, country, region, city, address_line, postal_code,
			place_name, latitude::float8, longitude::float8, created_at
	`, input.Precision, input.Country, input.Region, input.City, input.AddressLine,
		input.PostalCode, input.PlaceName, input.Latitude, input.Longitude, now))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create location", nil)
	}
	return stringPtr(location.ID), nil
}

func (s *Store) findMatchingLocation(ctx context.Context, q queryable, input model.LocationInput) (*model.Location, error) {
	location, err := scanLocation(q.QueryRow(ctx, `
		SELECT id, precision, country, region, city, address_line, postal_code,
			place_name, latitude::float8, longitude::float8, created_at
		FROM locations
		WHERE precision::text = $1
			AND country = $2
			AND city = $3
			AND region IS NOT DISTINCT FROM $4
			AND address_line IS NOT DISTINCT FROM $5
			AND postal_code IS NOT DISTINCT FROM $6
			AND place_name IS NOT DISTINCT FROM $7
			AND latitude IS NOT DISTINCT FROM $8
			AND longitude IS NOT DISTINCT FROM $9
		LIMIT 1
	`, input.Precision, input.Country, input.City, input.Region, input.AddressLine,
		input.PostalCode, input.PlaceName, input.Latitude, input.Longitude))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return location, err
}

func scanMemberships(rows pgx.Rows) ([]*model.CompanyMembership, *commonstore.AppError) {
	items := make([]*model.CompanyMembership, 0)
	for rows.Next() {
		membership, err := scanCompanyMembership(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load memberships", nil)
		}
		items = append(items, membership)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load memberships", nil)
	}
	return items, nil
}

func scanUser(src scanner) (*model.User, error) {
	var user model.User
	var avatarMediaID sql.NullString
	var emailVerifiedAt sql.NullTime
	var lastLoginAt sql.NullTime
	if err := src.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.Role,
		&user.IsActive,
		&avatarMediaID,
		&emailVerifiedAt,
		&lastLoginAt,
		&user.TokenVersion,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}
	user.AvatarMediaID = nullStringPtr(avatarMediaID)
	user.EmailVerifiedAt = nullTimePtr(emailVerifiedAt)
	user.LastLoginAt = nullTimePtr(lastLoginAt)
	return &user, nil
}

func scanUISettings(src scanner) (*model.UISettings, error) {
	var settings model.UISettings
	var payload []byte
	if err := src.Scan(&settings.UserID, &payload, &settings.SchemaVersion, &settings.UpdatedAt); err != nil {
		return nil, err
	}
	settings.SettingsJSON = map[string]any{}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &settings.SettingsJSON); err != nil {
			return nil, err
		}
	}
	return &settings, nil
}

func scanApplicantProfile(src scanner) (*model.ApplicantProfile, error) {
	var profile model.ApplicantProfile
	var middleName, universityName, faculty, programName, city, about, resumeMediaID sql.NullString
	var studyYear, graduationYear sql.NullInt16
	if err := src.Scan(
		&profile.UserID,
		&profile.FirstName,
		&profile.LastName,
		&middleName,
		&universityName,
		&faculty,
		&programName,
		&studyYear,
		&graduationYear,
		&city,
		&about,
		&resumeMediaID,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	); err != nil {
		return nil, err
	}
	profile.MiddleName = nullStringPtr(middleName)
	profile.UniversityName = nullStringPtr(universityName)
	profile.Faculty = nullStringPtr(faculty)
	profile.ProgramName = nullStringPtr(programName)
	profile.StudyYear = nullInt16Ptr(studyYear)
	profile.GraduationYear = nullInt16Ptr(graduationYear)
	profile.City = nullStringPtr(city)
	profile.About = nullStringPtr(about)
	profile.ResumeMediaID = nullStringPtr(resumeMediaID)
	return &profile, nil
}

func scanApplicantPrivacy(src scanner) (*model.ApplicantPrivacySettings, error) {
	var privacy model.ApplicantPrivacySettings
	if err := src.Scan(
		&privacy.ApplicantUserID,
		&privacy.ProfileVisibility,
		&privacy.ResumeVisibility,
		&privacy.ApplicationsVisibility,
		&privacy.ContactsVisibility,
		&privacy.ShowCareerInterests,
		&privacy.AllowRecommendations,
		&privacy.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &privacy, nil
}

func scanEmployerProfile(src scanner) (*model.EmployerProfile, error) {
	var profile model.EmployerProfile
	var jobTitle, phone sql.NullString
	if err := src.Scan(&profile.UserID, &profile.FullName, &jobTitle, &phone, &profile.CreatedAt, &profile.UpdatedAt); err != nil {
		return nil, err
	}
	profile.JobTitle = nullStringPtr(jobTitle)
	profile.Phone = nullStringPtr(phone)
	return &profile, nil
}

func scanLocation(src scanner) (*model.Location, error) {
	var location model.Location
	var region, addressLine, postalCode, placeName sql.NullString
	var latitude, longitude sql.NullFloat64
	if err := src.Scan(
		&location.ID,
		&location.Precision,
		&location.Country,
		&region,
		&location.City,
		&addressLine,
		&postalCode,
		&placeName,
		&latitude,
		&longitude,
		&location.CreatedAt,
	); err != nil {
		return nil, err
	}
	location.Region = nullStringPtr(region)
	location.AddressLine = nullStringPtr(addressLine)
	location.PostalCode = nullStringPtr(postalCode)
	location.PlaceName = nullStringPtr(placeName)
	location.Latitude = nullFloat64Ptr(latitude)
	location.Longitude = nullFloat64Ptr(longitude)
	return &location, nil
}

func scanTag(src scanner) (*model.Tag, error) {
	var tag model.Tag
	var createdBy sql.NullString
	if err := src.Scan(&tag.ID, &tag.Name, &tag.TagType, &tag.IsSystem, &tag.IsActive, &createdBy, &tag.CreatedAt); err != nil {
		return nil, err
	}
	tag.CreatedByUserID = nullStringPtr(createdBy)
	return &tag, nil
}

func scanCompany(src scanner) (*model.Company, error) {
	var company model.Company
	var brandName, inn, description, industry, websiteURL, corporateEmailDomain sql.NullString
	var headquartersLocationID, logoMediaID, bannerMediaID sql.NullString
	var verifiedAt sql.NullTime
	var verifiedBy, createdBy sql.NullString
	if err := src.Scan(
		&company.ID,
		&company.LegalName,
		&brandName,
		&company.Slug,
		&inn,
		&description,
		&industry,
		&websiteURL,
		&corporateEmailDomain,
		&headquartersLocationID,
		&logoMediaID,
		&bannerMediaID,
		&company.VerificationStatus,
		&verifiedAt,
		&verifiedBy,
		&createdBy,
		&company.CreatedAt,
		&company.UpdatedAt,
	); err != nil {
		return nil, err
	}
	company.BrandName = nullStringPtr(brandName)
	company.INN = nullStringPtr(inn)
	company.Description = nullStringPtr(description)
	company.Industry = nullStringPtr(industry)
	company.WebsiteURL = nullStringPtr(websiteURL)
	company.CorporateEmailDomain = nullStringPtr(corporateEmailDomain)
	company.HeadquartersLocationID = nullStringPtr(headquartersLocationID)
	company.LogoMediaID = nullStringPtr(logoMediaID)
	company.BannerMediaID = nullStringPtr(bannerMediaID)
	company.VerifiedAt = nullTimePtr(verifiedAt)
	company.VerifiedByCuratorUserID = nullStringPtr(verifiedBy)
	company.CreatedByUserID = nullStringPtr(createdBy)
	return &company, nil
}

func scanCompanyMembership(src scanner) (*model.CompanyMembership, error) {
	var membership model.CompanyMembership
	var invitedBy, statusChangedBy, statusComment sql.NullString
	var approvedAt sql.NullTime
	if err := src.Scan(
		&membership.ID,
		&membership.CompanyID,
		&membership.EmployerUserID,
		&invitedBy,
		&membership.Status,
		&membership.MemberRole,
		&membership.IsPrimaryContact,
		&statusChangedBy,
		&statusComment,
		&membership.StatusUpdatedAt,
		&approvedAt,
		&membership.UpdatedAt,
		&membership.CreatedAt,
	); err != nil {
		return nil, err
	}
	membership.InvitedByUserID = nullStringPtr(invitedBy)
	membership.StatusChangedByUserID = nullStringPtr(statusChangedBy)
	membership.StatusComment = nullStringPtr(statusComment)
	membership.ApprovedAt = nullTimePtr(approvedAt)
	return &membership, nil
}

func scanNotificationPreferences(src scanner) (*model.NotificationPreferences, error) {
	var preferences model.NotificationPreferences
	if err := src.Scan(
		&preferences.UserID,
		&preferences.InAppEnabled,
		&preferences.EmailEnabled,
		&preferences.RecommendationEnabled,
		&preferences.ApplicationStatusEnabled,
		&preferences.EmployerMessagesEnabled,
		&preferences.SystemEnabled,
		&preferences.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &preferences, nil
}

func appErr(status int, code, message string, details map[string]any) *commonstore.AppError {
	return &commonstore.AppError{Status: status, Code: code, Message: message, Details: details}
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	return constraint == "" || pgErr.ConstraintName == constraint
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	items := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	sort.Strings(items)
	return items
}

func paginate[T any](items []T, page, pageSize int) ([]T, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	total := len(items)
	start := (page - 1) * pageSize
	if start >= total {
		return []T{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return items[start:end], total
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	copy := make(map[string]any, len(input))
	for key, value := range input {
		copy[key] = value
	}
	return copy
}

func canInviteIntoRole(actorRole, requestedRole string, requestedPrimary bool) bool {
	if actorRole == model.CompanyRoleOwner {
		return true
	}
	if requestedPrimary || requestedRole == model.CompanyRoleOwner {
		return false
	}
	return requestedRole == model.CompanyRoleRecruiter || requestedRole == model.CompanyRoleHR || requestedRole == model.CompanyRoleManager
}

func canEditMembership(actorRole, targetRole string) bool {
	switch actorRole {
	case model.CompanyRoleOwner:
		return true
	case model.CompanyRoleManager:
		return targetRole != model.CompanyRoleOwner
	default:
		return false
	}
}

func canAssignRole(actorRole, targetRole, newRole string) bool {
	switch actorRole {
	case model.CompanyRoleOwner:
		return true
	case model.CompanyRoleManager:
		if targetRole == model.CompanyRoleOwner {
			return false
		}
		return newRole != model.CompanyRoleOwner
	default:
		return false
	}
}

func canFinalizeMembership(actorRole, targetRole string) bool {
	switch actorRole {
	case model.CompanyRoleOwner:
		return true
	case model.CompanyRoleManager:
		return targetRole != model.CompanyRoleOwner
	default:
		return false
	}
}

func stringPtr(value string) *string {
	return &value
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func nullInt16Ptr(value sql.NullInt16) *int {
	if !value.Valid {
		return nil
	}
	number := int(value.Int16)
	return &number
}

func nullFloat64Ptr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}

func int16Ptr(value *int) any {
	if value == nil {
		return nil
	}
	return int16(*value)
}
