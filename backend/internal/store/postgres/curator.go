package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

func (s *Store) ListCuratorUsers(actorUserID string, input model.ListCuratorUsersInput) ([]*model.User, int, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireCuratorActor(ctx, s.db, actorUserID); repoErr != nil {
		return nil, 0, repoErr
	}

	filterByActive := false
	filterActiveValue := false
	if input.IsActive != nil {
		filterByActive = true
		filterActiveValue = *input.IsActive
	}

	rows, err := s.db.Query(ctx, `
		SELECT u.id, u.email, u.password_hash, u.display_name, u.role, u.is_active,
			u.avatar_media_id, u.email_verified_at, u.last_login_at, u.token_version,
			u.created_at, u.updated_at
		FROM users u
		LEFT JOIN applicant_profiles ap ON ap.user_id = u.id
		LEFT JOIN employer_profiles ep ON ep.user_id = u.id
		WHERE u.role IN ('applicant', 'employer')
			AND ($1 = '' OR u.role::text = $1)
			AND ($2 = FALSE OR u.is_active = $3)
			AND (
				$4 = ''
				OR u.email ILIKE '%' || $4 || '%'
				OR u.display_name ILIKE '%' || $4 || '%'
				OR COALESCE(ap.first_name, '') ILIKE '%' || $4 || '%'
				OR COALESCE(ap.last_name, '') ILIKE '%' || $4 || '%'
				OR COALESCE(ep.full_name, '') ILIKE '%' || $4 || '%'
			)
		ORDER BY u.created_at DESC, u.id DESC
	`, input.Role, filterByActive, filterActiveValue, strings.TrimSpace(input.Q))
	if err != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load users", nil)
	}
	defer rows.Close()

	items := make([]*model.User, 0)
	for rows.Next() {
		user, scanErr := scanUser(rows)
		if scanErr != nil {
			return nil, 0, appErr(500, "internal_error", "failed to load users", nil)
		}
		items = append(items, user)
	}
	if rows.Err() != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load users", nil)
	}

	paged, total := paginate(items, input.Page, input.PageSize)
	return paged, total, nil
}

func (s *Store) GetCuratorUser(actorUserID, targetUserID string) (*model.User, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireCuratorActor(ctx, s.db, actorUserID); repoErr != nil {
		return nil, repoErr
	}
	return s.loadModeratedUser(ctx, s.db, targetUserID)
}

func (s *Store) UpdateCuratorUser(actorUserID, targetUserID string, input model.UpdateCuratorUserInput) (*model.User, *commonstore.AppError) {
	if strings.TrimSpace(input.Reason) == "" {
		return nil, appErr(422, "validation_error", "reason is required", nil)
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update user", nil)
	}
	defer tx.Rollback(ctx)

	if _, repoErr := s.requireCuratorActor(ctx, tx, actorUserID); repoErr != nil {
		return nil, repoErr
	}

	user, repoErr := s.loadModeratedUser(ctx, tx, targetUserID)
	if repoErr != nil {
		return nil, repoErr
	}

	changed := false
	if input.DisplayName != nil {
		user.DisplayName = strings.TrimSpace(*input.DisplayName)
		changed = true
	}
	if input.IsActive != nil {
		user.IsActive = *input.IsActive
		changed = true
	}
	if changed {
		user.UpdatedAt = time.Now().UTC()
		if _, err := tx.Exec(ctx, `
			UPDATE users
			SET display_name = $1, is_active = $2, updated_at = $3
			WHERE id = $4
		`, user.DisplayName, user.IsActive, user.UpdatedAt, user.ID); err != nil {
			return nil, appErr(500, "internal_error", "failed to update user", nil)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update user", nil)
	}
	return s.GetCuratorUser(actorUserID, targetUserID)
}

func (s *Store) GetCuratorApplicant(actorUserID, targetUserID string) (*model.CuratorApplicantProfileView, *commonstore.AppError) {
	ctx := context.Background()
	return s.loadCuratorApplicantView(ctx, s.db, actorUserID, targetUserID)
}

func (s *Store) UpdateCuratorApplicant(actorUserID, targetUserID string, input model.UpdateCuratorApplicantInput) (*model.CuratorApplicantProfileView, *commonstore.AppError) {
	if strings.TrimSpace(input.Reason) == "" {
		return nil, appErr(422, "validation_error", "reason is required", nil)
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update applicant", nil)
	}
	defer tx.Rollback(ctx)

	if _, repoErr := s.requireCuratorActor(ctx, tx, actorUserID); repoErr != nil {
		return nil, repoErr
	}

	profile, err := scanApplicantProfile(tx.QueryRow(ctx, `
		SELECT user_id, first_name, last_name, middle_name, university_name, faculty,
			program_name, study_year, graduation_year, city, about, resume_media_id,
			created_at, updated_at
		FROM applicant_profiles
		WHERE user_id = $1
		FOR UPDATE
	`, targetUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "applicant not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load applicant profile", nil)
	}

	profileChanged := false
	if input.Profile.FirstName != nil {
		profile.FirstName = *input.Profile.FirstName
		profileChanged = true
	}
	if input.Profile.LastName != nil {
		profile.LastName = *input.Profile.LastName
		profileChanged = true
	}
	if input.Profile.MiddleName != nil {
		profile.MiddleName = input.Profile.MiddleName
		profileChanged = true
	}
	if input.Profile.UniversityName != nil {
		profile.UniversityName = input.Profile.UniversityName
		profileChanged = true
	}
	if input.Profile.Faculty != nil {
		profile.Faculty = input.Profile.Faculty
		profileChanged = true
	}
	if input.Profile.ProgramName != nil {
		profile.ProgramName = input.Profile.ProgramName
		profileChanged = true
	}
	if input.Profile.StudyYear != nil {
		profile.StudyYear = input.Profile.StudyYear
		profileChanged = true
	}
	if input.Profile.GraduationYear != nil {
		profile.GraduationYear = input.Profile.GraduationYear
		profileChanged = true
	}
	if input.Profile.City != nil {
		profile.City = input.Profile.City
		profileChanged = true
	}
	if input.Profile.About != nil {
		profile.About = input.Profile.About
		profileChanged = true
	}
	if input.Profile.ResumeMediaID != nil {
		profile.ResumeMediaID = input.Profile.ResumeMediaID
		profileChanged = true
	}
	if profileChanged {
		profile.UpdatedAt = time.Now().UTC()
		if _, err := tx.Exec(ctx, `
			UPDATE applicant_profiles
			SET first_name = $1, last_name = $2, middle_name = $3, university_name = $4,
				faculty = $5, program_name = $6, study_year = $7, graduation_year = $8,
				city = $9, about = $10, resume_media_id = $11, updated_at = $12
			WHERE user_id = $13
		`, profile.FirstName, profile.LastName, profile.MiddleName, profile.UniversityName,
			profile.Faculty, profile.ProgramName, int16Ptr(profile.StudyYear), int16Ptr(profile.GraduationYear),
			profile.City, profile.About, profile.ResumeMediaID, profile.UpdatedAt, targetUserID); err != nil {
			return nil, appErr(500, "internal_error", "failed to update applicant profile", nil)
		}
	}

	if input.Privacy != nil {
		privacy, err := scanApplicantPrivacy(tx.QueryRow(ctx, `
			SELECT applicant_user_id, profile_visibility, resume_visibility, applications_visibility,
				contacts_visibility, show_career_interests, allow_recommendations, updated_at
			FROM applicant_privacy_settings
			WHERE applicant_user_id = $1
			FOR UPDATE
		`, targetUserID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, appErr(404, "not_found", "applicant not found", nil)
			}
			return nil, appErr(500, "internal_error", "failed to load applicant privacy settings", nil)
		}

		privacyChanged := false
		if input.Privacy.ProfileVisibility != nil {
			privacy.ProfileVisibility = *input.Privacy.ProfileVisibility
			privacyChanged = true
		}
		if input.Privacy.ResumeVisibility != nil {
			privacy.ResumeVisibility = *input.Privacy.ResumeVisibility
			privacyChanged = true
		}
		if input.Privacy.ApplicationsVisibility != nil {
			privacy.ApplicationsVisibility = *input.Privacy.ApplicationsVisibility
			privacyChanged = true
		}
		if input.Privacy.ContactsVisibility != nil {
			privacy.ContactsVisibility = *input.Privacy.ContactsVisibility
			privacyChanged = true
		}
		if input.Privacy.ShowCareerInterests != nil {
			privacy.ShowCareerInterests = *input.Privacy.ShowCareerInterests
			privacyChanged = true
		}
		if input.Privacy.AllowRecommendations != nil {
			privacy.AllowRecommendations = *input.Privacy.AllowRecommendations
			privacyChanged = true
		}
		if privacyChanged {
			privacy.UpdatedAt = time.Now().UTC()
			if _, err := tx.Exec(ctx, `
				UPDATE applicant_privacy_settings
				SET profile_visibility = $1, resume_visibility = $2, applications_visibility = $3,
					contacts_visibility = $4, show_career_interests = $5, allow_recommendations = $6,
					updated_at = $7
				WHERE applicant_user_id = $8
			`, privacy.ProfileVisibility, privacy.ResumeVisibility, privacy.ApplicationsVisibility,
				privacy.ContactsVisibility, privacy.ShowCareerInterests, privacy.AllowRecommendations,
				privacy.UpdatedAt, targetUserID); err != nil {
				return nil, appErr(500, "internal_error", "failed to update applicant privacy settings", nil)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update applicant", nil)
	}
	return s.GetCuratorApplicant(actorUserID, targetUserID)
}

func (s *Store) GetCuratorEmployer(actorUserID, targetUserID string) (*model.EmployerProfileView, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireCuratorActor(ctx, s.db, actorUserID); repoErr != nil {
		return nil, repoErr
	}
	return s.loadEmployerProfileView(ctx, s.db, targetUserID)
}

func (s *Store) UpdateCuratorEmployer(actorUserID, targetUserID string, input model.UpdateCuratorEmployerInput) (*model.EmployerProfileView, *commonstore.AppError) {
	if strings.TrimSpace(input.Reason) == "" {
		return nil, appErr(422, "validation_error", "reason is required", nil)
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update employer", nil)
	}
	defer tx.Rollback(ctx)

	if _, repoErr := s.requireCuratorActor(ctx, tx, actorUserID); repoErr != nil {
		return nil, repoErr
	}

	user, repoErr := s.loadModeratedUser(ctx, tx, targetUserID)
	if repoErr != nil {
		return nil, repoErr
	}
	if user.Role != model.UserRoleEmployer {
		return nil, appErr(404, "not_found", "employer not found", nil)
	}

	profile, err := scanEmployerProfile(tx.QueryRow(ctx, `
		SELECT user_id, full_name, job_title, phone, created_at, updated_at
		FROM employer_profiles
		WHERE user_id = $1
		FOR UPDATE
	`, targetUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "employer not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load employer profile", nil)
	}

	changed := false
	if input.Profile.FullName != nil {
		profile.FullName = *input.Profile.FullName
		changed = true
	}
	if input.Profile.JobTitle != nil {
		profile.JobTitle = input.Profile.JobTitle
		changed = true
	}
	if input.Profile.Phone != nil {
		profile.Phone = input.Profile.Phone
		changed = true
	}
	if changed {
		profile.UpdatedAt = time.Now().UTC()
		if _, err := tx.Exec(ctx, `
			UPDATE employer_profiles
			SET full_name = $1, job_title = $2, phone = $3, updated_at = $4
			WHERE user_id = $5
		`, profile.FullName, profile.JobTitle, profile.Phone, profile.UpdatedAt, targetUserID); err != nil {
			return nil, appErr(500, "internal_error", "failed to update employer profile", nil)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update employer", nil)
	}
	return s.GetCuratorEmployer(actorUserID, targetUserID)
}

func (s *Store) GetCuratorCompany(actorUserID, companyID string) (*model.Company, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireCuratorActor(ctx, s.db, actorUserID); repoErr != nil {
		return nil, repoErr
	}
	return s.loadCompanyByID(ctx, s.db, companyID)
}

func (s *Store) UpdateCuratorCompany(actorUserID, companyID string, input model.UpdateCuratorCompanyInput) (*model.Company, *commonstore.AppError) {
	if strings.TrimSpace(input.Reason) == "" {
		return nil, appErr(422, "validation_error", "reason is required", nil)
	}
	if input.Company.HeadquartersLocationID != nil && input.Company.HeadquartersLocation != nil {
		return nil, appErr(422, "validation_error", "provide either headquartersLocationId or headquartersLocation", nil)
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update company", nil)
	}
	defer tx.Rollback(ctx)

	if _, repoErr := s.requireCuratorActor(ctx, tx, actorUserID); repoErr != nil {
		return nil, repoErr
	}

	company, err := scanCompany(tx.QueryRow(ctx, `
		SELECT id, legal_name, brand_name, slug, inn, description, industry, website_url,
			corporate_email_domain, headquarters_location_id, logo_media_id, banner_media_id,
			verification_status, verified_at, verified_by_curator_user_id, created_by_user_id,
			created_at, updated_at
		FROM companies
		WHERE id = $1
		FOR UPDATE
	`, companyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "company not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load company", nil)
	}

	changed := false
	if input.Company.Slug != nil {
		company.Slug = *input.Company.Slug
		changed = true
	}
	if input.Company.LegalName != nil {
		company.LegalName = *input.Company.LegalName
		changed = true
	}
	if input.Company.BrandName != nil {
		company.BrandName = input.Company.BrandName
		changed = true
	}
	if input.Company.INN != nil {
		company.INN = input.Company.INN
		changed = true
	}
	if input.Company.Description != nil {
		company.Description = input.Company.Description
		changed = true
	}
	if input.Company.Industry != nil {
		company.Industry = input.Company.Industry
		changed = true
	}
	if input.Company.WebsiteURL != nil {
		company.WebsiteURL = input.Company.WebsiteURL
		changed = true
	}
	if input.Company.CorporateEmailDomain != nil {
		company.CorporateEmailDomain = input.Company.CorporateEmailDomain
		changed = true
	}
	if input.Company.LogoMediaID != nil {
		company.LogoMediaID = input.Company.LogoMediaID
		changed = true
	}
	if input.Company.BannerMediaID != nil {
		company.BannerMediaID = input.Company.BannerMediaID
		changed = true
	}
	if input.Company.HeadquartersLocationID != nil || input.Company.HeadquartersLocation != nil {
		locationID, repoErr := s.resolveLocation(ctx, tx, input.Company.HeadquartersLocationID, input.Company.HeadquartersLocation)
		if repoErr != nil {
			return nil, repoErr
		}
		company.HeadquartersLocationID = locationID
		changed = true
	}

	if changed {
		company.UpdatedAt = time.Now().UTC()
		if _, err := tx.Exec(ctx, `
			UPDATE companies
			SET legal_name = $1, brand_name = $2, slug = $3, inn = $4, description = $5,
				industry = $6, website_url = $7, corporate_email_domain = $8,
				headquarters_location_id = $9, logo_media_id = $10, banner_media_id = $11,
				updated_at = $12
			WHERE id = $13
		`, company.LegalName, company.BrandName, company.Slug, company.INN, company.Description,
			company.Industry, company.WebsiteURL, company.CorporateEmailDomain, company.HeadquartersLocationID,
			company.LogoMediaID, company.BannerMediaID, company.UpdatedAt, company.ID); err != nil {
			if isUniqueViolation(err, "companies_slug_key") || isUniqueViolation(err, "companies_inn_key") {
				return nil, appErr(409, "conflict", "company slug already exists", nil)
			}
			return nil, appErr(500, "internal_error", "failed to update company", nil)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update company", nil)
	}
	return s.GetCuratorCompany(actorUserID, companyID)
}

func (s *Store) GetCuratorOpportunity(actorUserID, opportunityID string) (*model.Opportunity, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireCuratorActor(ctx, s.db, actorUserID); repoErr != nil {
		return nil, repoErr
	}
	return s.loadOpportunityByID(ctx, s.db, opportunityID)
}

func (s *Store) UpdateCuratorOpportunity(actorUserID, opportunityID string, input model.UpdateCuratorOpportunityInput) (*model.Opportunity, *commonstore.AppError) {
	if strings.TrimSpace(input.Reason) == "" {
		return nil, appErr(422, "validation_error", "reason is required", nil)
	}
	if input.LocationID != nil && input.Location != nil {
		return nil, appErr(422, "validation_error", "provide either locationId or location", nil)
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update opportunity", nil)
	}
	defer tx.Rollback(ctx)

	if _, repoErr := s.requireCuratorActor(ctx, tx, actorUserID); repoErr != nil {
		return nil, repoErr
	}

	opportunity, err := scanOpportunity(tx.QueryRow(ctx, `
		SELECT id, company_id, created_by_user_id, title, summary, slug, description,
			type, status, moderation_status, participation_format, location_id,
			contact_email, contact_phone, cover_media_id, published_at, expires_at,
			created_at, updated_at
		FROM opportunities
		WHERE id = $1
		FOR UPDATE
	`, opportunityID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "opportunity not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load opportunity", nil)
	}

	if repoErr := s.validateCuratorOpportunityInput(opportunity, input, tx); repoErr != nil {
		return nil, repoErr
	}

	changed := false
	if input.Title != nil {
		opportunity.Title = *input.Title
		changed = true
	}
	if input.Summary != nil {
		opportunity.Summary = *input.Summary
		changed = true
	}
	if input.Slug != nil {
		opportunity.Slug = *input.Slug
		changed = true
	}
	if input.Description != nil {
		opportunity.Description = *input.Description
		changed = true
	}
	if input.ParticipationFormat != nil {
		opportunity.ParticipationFormat = *input.ParticipationFormat
		changed = true
	}
	if input.ContactEmail != nil {
		opportunity.ContactEmail = input.ContactEmail
		changed = true
	}
	if input.ContactPhone != nil {
		opportunity.ContactPhone = input.ContactPhone
		changed = true
	}
	if input.CoverMediaID != nil {
		opportunity.CoverMediaID = input.CoverMediaID
		changed = true
	}
	if input.PublishedAt != nil {
		opportunity.PublishedAt = input.PublishedAt
		changed = true
	}
	if input.ExpiresAt != nil {
		opportunity.ExpiresAt = input.ExpiresAt
		changed = true
	}
	if input.LocationID != nil || input.Location != nil {
		locationID, repoErr := s.resolveLocation(ctx, tx, input.LocationID, input.Location)
		if repoErr != nil {
			return nil, repoErr
		}
		opportunity.LocationID = locationID
		changed = true
	}

	if changed {
		opportunity.UpdatedAt = time.Now().UTC()
		if _, err := tx.Exec(ctx, `
			UPDATE opportunities
			SET title = $1, summary = $2, slug = $3, description = $4,
				participation_format = $5, location_id = $6, contact_email = $7,
				contact_phone = $8, cover_media_id = $9, published_at = $10,
				expires_at = $11, updated_at = $12
			WHERE id = $13
		`, opportunity.Title, opportunity.Summary, opportunity.Slug, opportunity.Description,
			opportunity.ParticipationFormat, opportunity.LocationID, opportunity.ContactEmail,
			opportunity.ContactPhone, opportunity.CoverMediaID, opportunity.PublishedAt,
			opportunity.ExpiresAt, opportunity.UpdatedAt, opportunity.ID); err != nil {
			if isUniqueViolation(err, "opportunities_slug_key") {
				return nil, appErr(409, "conflict", "opportunity slug already exists", nil)
			}
			return nil, appErr(500, "internal_error", "failed to update opportunity", nil)
		}
	}

	if input.VacancyDetails != nil || input.MentorProgramDetails != nil || input.EventDetails != nil {
		if repoErr := s.replaceOpportunityDetails(ctx, tx, opportunity.ID, opportunity.Type, input.VacancyDetails, input.MentorProgramDetails, input.EventDetails); repoErr != nil {
			return nil, repoErr
		}
	}
	if input.ReplaceTagIDs {
		if repoErr := s.replaceOpportunityTags(ctx, tx, opportunity.ID, input.TagIDs); repoErr != nil {
			return nil, repoErr
		}
	}
	if input.ReplaceLinks {
		if repoErr := s.replaceOpportunityLinks(ctx, tx, opportunity.ID, input.Links); repoErr != nil {
			return nil, repoErr
		}
	}
	if input.ReplaceMedia {
		if repoErr := s.replaceOpportunityMedia(ctx, tx, opportunity.ID, input.Media); repoErr != nil {
			return nil, repoErr
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update opportunity", nil)
	}
	return s.GetCuratorOpportunity(actorUserID, opportunityID)
}

func (s *Store) requireCuratorActor(ctx context.Context, q queryable, actorUserID string) (*model.User, *commonstore.AppError) {
	if _, repoErr := s.requireCuratorProfile(ctx, q, actorUserID); repoErr != nil {
		return nil, repoErr
	}

	user, err := scanUser(q.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, role, is_active, avatar_media_id,
			email_verified_at, last_login_at, token_version, created_at, updated_at
		FROM users
		WHERE id = $1
	`, actorUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(401, "unauthorized", "authentication failed", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load user", nil)
	}
	return user, nil
}

func (s *Store) loadModeratedUser(ctx context.Context, q queryable, targetUserID string) (*model.User, *commonstore.AppError) {
	user, err := scanUser(q.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, role, is_active, avatar_media_id,
			email_verified_at, last_login_at, token_version, created_at, updated_at
		FROM users
		WHERE id = $1
	`, targetUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "user not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load user", nil)
	}
	if user.Role == model.UserRoleCurator {
		return nil, appErr(403, "forbidden", "target user is a curator account", nil)
	}
	if user.Role != model.UserRoleApplicant && user.Role != model.UserRoleEmployer {
		return nil, appErr(404, "not_found", "user not found", nil)
	}
	return user, nil
}

func (s *Store) loadCuratorApplicantView(ctx context.Context, q queryable, actorUserID, targetUserID string) (*model.CuratorApplicantProfileView, *commonstore.AppError) {
	viewer, repoErr := s.requireCuratorActor(ctx, q, actorUserID)
	if repoErr != nil {
		return nil, repoErr
	}

	view, repoErr := s.loadApplicantProfileView(ctx, q, viewer, targetUserID, false)
	if repoErr != nil {
		return nil, repoErr
	}
	privacy, err := scanApplicantPrivacy(q.QueryRow(ctx, `
		SELECT applicant_user_id, profile_visibility, resume_visibility, applications_visibility,
			contacts_visibility, show_career_interests, allow_recommendations, updated_at
		FROM applicant_privacy_settings
		WHERE applicant_user_id = $1
	`, targetUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "applicant not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load applicant privacy settings", nil)
	}
	return &model.CuratorApplicantProfileView{
		View:    view,
		Privacy: privacy,
	}, nil
}

func (s *Store) loadEmployerProfileView(ctx context.Context, q queryable, targetUserID string) (*model.EmployerProfileView, *commonstore.AppError) {
	user, err := scanUser(q.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, role, is_active, avatar_media_id,
			email_verified_at, last_login_at, token_version, created_at, updated_at
		FROM users
		WHERE id = $1
	`, targetUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "employer not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load employer", nil)
	}
	if user.Role != model.UserRoleEmployer {
		return nil, appErr(404, "not_found", "employer not found", nil)
	}

	profile, err := scanEmployerProfile(q.QueryRow(ctx, `
		SELECT user_id, full_name, job_title, phone, created_at, updated_at
		FROM employer_profiles
		WHERE user_id = $1
	`, targetUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "employer not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load employer profile", nil)
	}

	rows, err := q.Query(ctx, `
		SELECT id, company_id, employer_user_id, invited_by_user_id, status, member_role,
			is_primary_contact, status_changed_by_user_id, status_comment, status_updated_at,
			approved_at, updated_at, created_at
		FROM company_memberships
		WHERE employer_user_id = $1
		ORDER BY created_at DESC, id DESC
	`, targetUserID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load memberships", nil)
	}
	defer rows.Close()

	memberships := make([]*model.CompanyMembership, 0)
	for rows.Next() {
		membership, scanErr := scanCompanyMembership(rows)
		if scanErr != nil {
			return nil, appErr(500, "internal_error", "failed to load memberships", nil)
		}
		memberships = append(memberships, membership)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load memberships", nil)
	}

	return &model.EmployerProfileView{
		Profile:       profile,
		DisplayName:   user.DisplayName,
		AvatarMediaID: user.AvatarMediaID,
		Companies:     memberships,
	}, nil
}

func (s *Store) loadCompanyByID(ctx context.Context, q queryable, companyID string) (*model.Company, *commonstore.AppError) {
	company, err := scanCompany(q.QueryRow(ctx, `
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
	if repoErr := s.hydrateCompanyPublicFields(ctx, q, company); repoErr != nil {
		return nil, repoErr
	}
	return company, nil
}

func (s *Store) loadOpportunityByID(ctx context.Context, q queryable, opportunityID string) (*model.Opportunity, *commonstore.AppError) {
	opportunity, err := scanOpportunity(q.QueryRow(ctx, `
		SELECT id, company_id, created_by_user_id, title, summary, slug, description,
			type, status, moderation_status, participation_format, location_id,
			contact_email, contact_phone, cover_media_id, published_at, expires_at,
			created_at, updated_at
		FROM opportunities
		WHERE id = $1
	`, opportunityID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "opportunity not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load opportunity", nil)
	}
	if repoErr := s.hydrateOpportunity(ctx, q, opportunity); repoErr != nil {
		return nil, repoErr
	}
	return opportunity, nil
}

func (s *Store) validateCuratorOpportunityInput(current *model.Opportunity, input model.UpdateCuratorOpportunityInput, q queryable) *commonstore.AppError {
	if input.ParticipationFormat != nil && !isSupportedParticipationFormat(*input.ParticipationFormat) {
		return appErr(422, "validation_error", "unsupported participation format", nil)
	}
	if detailsBlockCount(input.VacancyDetails, input.MentorProgramDetails, input.EventDetails) > 1 {
		return appErr(422, "validation_error", "at most one non-null detail block may be sent", nil)
	}
	if input.VacancyDetails != nil && current.Type != model.OpportunityTypeInternship && current.Type != model.OpportunityTypeVacancy {
		return appErr(422, "validation_error", "detail block must match the existing opportunity type", nil)
	}
	if input.MentorProgramDetails != nil && current.Type != model.OpportunityTypeMentorProgram {
		return appErr(422, "validation_error", "detail block must match the existing opportunity type", nil)
	}
	if input.EventDetails != nil && current.Type != model.OpportunityTypeEvent {
		return appErr(422, "validation_error", "detail block must match the existing opportunity type", nil)
	}
	if repoErr := validateVacancyDetails(input.VacancyDetails); repoErr != nil {
		return repoErr
	}
	if repoErr := validateMentorProgramDetails(input.MentorProgramDetails); repoErr != nil {
		return repoErr
	}
	if repoErr := validateEventDetails(input.EventDetails); repoErr != nil {
		return repoErr
	}
	if input.ReplaceLinks {
		if repoErr := validateOpportunityLinks(input.Links); repoErr != nil {
			return repoErr
		}
	}
	if input.ReplaceMedia {
		if repoErr := validateOpportunityMedia(input.Media); repoErr != nil {
			return repoErr
		}
	}
	if input.ReplaceTagIDs {
		if repoErr := s.validateTagIDs(context.Background(), q, input.TagIDs); repoErr != nil {
			return repoErr
		}
	}
	return nil
}
