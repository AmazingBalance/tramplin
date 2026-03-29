package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

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

func (s *Store) GetCurrentCuratorProfile(userID string) (*model.CurrentCuratorProfile, *commonstore.AppError) {
	profile, err := scanCuratorProfile(s.db.QueryRow(context.Background(), `
		SELECT cp.user_id, cp.full_name, cp.is_admin, cp.created_at
		FROM curator_profiles cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.user_id = $1
			AND u.role = 'curator'
			AND u.is_active = TRUE
	`, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(500, "internal_error", "failed to load curator profile", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load curator profile", nil)
	}
	return &model.CurrentCuratorProfile{IsAdmin: profile.IsAdmin}, nil
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
