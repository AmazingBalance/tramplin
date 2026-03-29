package postgres

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

func (s *Store) ListEmployerOpportunities(actorUserID string, input model.ListEmployerOpportunitiesInput) ([]*model.Opportunity, int, *commonstore.AppError) {
	ctx := context.Background()
	var companyID any
	if input.CompanyID != "" {
		companyID = input.CompanyID
	}
	var status any
	if input.Status != "" {
		status = input.Status
	}
	var moderationStatus any
	if input.ModerationStatus != "" {
		moderationStatus = input.ModerationStatus
	}
	rows, err := s.db.Query(ctx, `
		SELECT o.id, o.company_id, o.created_by_user_id, o.title, o.summary, o.slug, o.description,
			o.type, o.status, o.moderation_status, o.participation_format, o.location_id,
			o.contact_email, o.contact_phone, o.cover_media_id, o.published_at, o.expires_at,
			o.created_at, o.updated_at
		FROM opportunities o
		JOIN company_memberships cm
			ON cm.company_id = o.company_id
			AND cm.employer_user_id = $1
			AND cm.status = 'approved'
		WHERE ($2::uuid IS NULL OR o.company_id = $2::uuid)
			AND ($3::text IS NULL OR o.status::text = $3::text)
			AND ($4::text IS NULL OR o.moderation_status::text = $4::text)
		ORDER BY o.created_at DESC
	`, actorUserID, companyID, status, moderationStatus)
	if err != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load opportunities", nil)
	}
	defer rows.Close()

	items, appErr := s.scanOpportunityList(ctx, rows)
	if appErr != nil {
		return nil, 0, appErr
	}
	paged, total := paginate(items, input.Page, input.PageSize)
	return paged, total, nil
}

func (s *Store) CreateOpportunity(actorUserID string, input model.CreateOpportunityInput) (*model.Opportunity, *commonstore.AppError) {
	if appErr := validateCreateOpportunityInput(input); appErr != nil {
		return nil, appErr
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create opportunity", nil)
	}
	defer tx.Rollback(ctx)

	if _, repoErr := s.requireApprovedMembership(ctx, tx, actorUserID, input.CompanyID); repoErr != nil {
		return nil, appErr(403, "forbidden", "current employer cannot create opportunities for specified company", nil)
	}

	locationID, repoErr := s.resolveLocation(ctx, tx, input.LocationID, input.Location)
	if repoErr != nil {
		return nil, repoErr
	}
	if repoErr := s.validateTagIDs(ctx, tx, input.TagIDs); repoErr != nil {
		return nil, repoErr
	}

	now := time.Now().UTC()
	var opportunityID string
	err = tx.QueryRow(ctx, `
		INSERT INTO opportunities (
			company_id, created_by_user_id, title, summary, slug, description, type, status,
			moderation_status, participation_format, location_id, contact_email, contact_phone,
			cover_media_id, published_at, expires_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, 'draft', 'pending', $8, $9, $10, $11, $12, $13, $14, $15, $15
		)
		RETURNING id
	`, input.CompanyID, actorUserID, input.Title, input.Summary, input.Slug, input.Description,
		input.Type, input.ParticipationFormat, locationID, input.ContactEmail, input.ContactPhone,
		input.CoverMediaID, input.PublishedAt, input.ExpiresAt, now).Scan(&opportunityID)
	if err != nil {
		if isUniqueViolation(err, "opportunities_slug_key") {
			return nil, appErr(409, "conflict", "opportunity slug already exists or requested lifecycle action is not allowed", nil)
		}
		return nil, appErr(500, "internal_error", "failed to create opportunity", nil)
	}

	if appErr := s.replaceOpportunityDetails(ctx, tx, opportunityID, input.Type, input.VacancyDetails, input.MentorProgramDetails, input.EventDetails); appErr != nil {
		return nil, appErr
	}
	if appErr := s.replaceOpportunityTags(ctx, tx, opportunityID, input.TagIDs); appErr != nil {
		return nil, appErr
	}
	if appErr := s.replaceOpportunityLinks(ctx, tx, opportunityID, input.Links); appErr != nil {
		return nil, appErr
	}
	if appErr := s.replaceOpportunityMedia(ctx, tx, opportunityID, input.Media); appErr != nil {
		return nil, appErr
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO opportunity_stats (opportunity_id, views_count, updated_at)
		VALUES ($1, 0, $2)
	`, opportunityID, now); err != nil {
		return nil, appErr(500, "internal_error", "failed to initialize opportunity stats", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to create opportunity", nil)
	}
	return s.GetEmployerOpportunity(actorUserID, opportunityID)
}

func (s *Store) GetEmployerOpportunity(actorUserID, opportunityID string) (*model.Opportunity, *commonstore.AppError) {
	ctx := context.Background()
	opportunity, err := scanOpportunity(s.db.QueryRow(ctx, `
		SELECT o.id, o.company_id, o.created_by_user_id, o.title, o.summary, o.slug, o.description,
			o.type, o.status, o.moderation_status, o.participation_format, o.location_id,
			o.contact_email, o.contact_phone, o.cover_media_id, o.published_at, o.expires_at,
			o.created_at, o.updated_at
		FROM opportunities o
		JOIN company_memberships cm
			ON cm.company_id = o.company_id
			AND cm.employer_user_id = $1
			AND cm.status = 'approved'
		WHERE o.id = $2
	`, actorUserID, opportunityID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "opportunity not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load opportunity", nil)
	}
	if repoErr := s.hydrateOpportunity(ctx, s.db, opportunity); repoErr != nil {
		return nil, repoErr
	}
	return opportunity, nil
}

func (s *Store) UpdateOpportunity(actorUserID, opportunityID string, input model.UpdateOpportunityInput) (*model.Opportunity, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update opportunity", nil)
	}
	defer tx.Rollback(ctx)

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

	if _, repoErr := s.requireApprovedMembership(ctx, tx, actorUserID, opportunity.CompanyID); repoErr != nil {
		return nil, appErr(403, "forbidden", "current employer has no access to this opportunity", nil)
	}
	if input.LocationID != nil && input.Location != nil {
		return nil, appErr(422, "validation_error", "provide either locationId or location", nil)
	}
	if appErr := validateUpdateOpportunityInput(opportunity, input); appErr != nil {
		return nil, appErr
	}
	if input.ReplaceTagIDs {
		if repoErr := s.validateTagIDs(ctx, tx, input.TagIDs); repoErr != nil {
			return nil, repoErr
		}
	}

	if input.Title != nil {
		opportunity.Title = *input.Title
	}
	if input.Summary != nil {
		opportunity.Summary = *input.Summary
	}
	if input.Slug != nil {
		opportunity.Slug = *input.Slug
	}
	if input.Description != nil {
		opportunity.Description = *input.Description
	}
	if input.Status != nil {
		opportunity.Status = *input.Status
	}
	if input.ParticipationFormat != nil {
		opportunity.ParticipationFormat = *input.ParticipationFormat
	}
	if input.ContactEmail != nil {
		opportunity.ContactEmail = input.ContactEmail
	}
	if input.ContactPhone != nil {
		opportunity.ContactPhone = input.ContactPhone
	}
	if input.CoverMediaID != nil {
		opportunity.CoverMediaID = input.CoverMediaID
	}
	if input.PublishedAt != nil {
		opportunity.PublishedAt = input.PublishedAt
	}
	if input.ExpiresAt != nil {
		opportunity.ExpiresAt = input.ExpiresAt
	}
	if input.LocationID != nil || input.Location != nil {
		locationID, repoErr := s.resolveLocation(ctx, tx, input.LocationID, input.Location)
		if repoErr != nil {
			return nil, repoErr
		}
		opportunity.LocationID = locationID
	}
	opportunity.UpdatedAt = time.Now().UTC()

	_, err = tx.Exec(ctx, `
		UPDATE opportunities
		SET title = $1, summary = $2, slug = $3, description = $4, status = $5,
			participation_format = $6, location_id = $7, contact_email = $8,
			contact_phone = $9, cover_media_id = $10, published_at = $11, expires_at = $12,
			updated_at = $13
		WHERE id = $14
	`, opportunity.Title, opportunity.Summary, opportunity.Slug, opportunity.Description,
		opportunity.Status, opportunity.ParticipationFormat, opportunity.LocationID,
		opportunity.ContactEmail, opportunity.ContactPhone, opportunity.CoverMediaID,
		opportunity.PublishedAt, opportunity.ExpiresAt, opportunity.UpdatedAt, opportunity.ID)
	if err != nil {
		if isUniqueViolation(err, "opportunities_slug_key") {
			return nil, appErr(409, "conflict", "requested opportunity status transition is not allowed", nil)
		}
		return nil, appErr(500, "internal_error", "failed to update opportunity", nil)
	}

	if input.VacancyDetails != nil || input.MentorProgramDetails != nil || input.EventDetails != nil {
		if appErr := s.replaceOpportunityDetails(ctx, tx, opportunity.ID, opportunity.Type, input.VacancyDetails, input.MentorProgramDetails, input.EventDetails); appErr != nil {
			return nil, appErr
		}
	}
	if input.ReplaceTagIDs {
		if appErr := s.replaceOpportunityTags(ctx, tx, opportunity.ID, input.TagIDs); appErr != nil {
			return nil, appErr
		}
	}
	if input.ReplaceLinks {
		if appErr := s.replaceOpportunityLinks(ctx, tx, opportunity.ID, input.Links); appErr != nil {
			return nil, appErr
		}
	}
	if input.ReplaceMedia {
		if appErr := s.replaceOpportunityMedia(ctx, tx, opportunity.ID, input.Media); appErr != nil {
			return nil, appErr
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update opportunity", nil)
	}
	return s.GetEmployerOpportunity(actorUserID, opportunity.ID)
}

func (s *Store) ListPublicOpportunities(input model.ListPublicOpportunitiesInput) ([]*model.Opportunity, int, *commonstore.AppError) {
	ctx := context.Background()
	rows, err := s.db.Query(ctx, `
		SELECT id, company_id, created_by_user_id, title, summary, slug, description,
			type, status, moderation_status, participation_format, location_id,
			contact_email, contact_phone, cover_media_id, published_at, expires_at,
			created_at, updated_at
		FROM opportunities
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load opportunities", nil)
	}
	defer rows.Close()

	items, appErr := s.scanOpportunityList(ctx, rows)
	if appErr != nil {
		return nil, 0, appErr
	}

	filtered := s.filterPublicOpportunities(items, input)
	sortPublicOpportunities(filtered, input.Sort)
	paged, total := paginate(filtered, input.Page, input.PageSize)
	return paged, total, nil
}

func (s *Store) GetPublicOpportunityByID(opportunityID string) (*model.Opportunity, *commonstore.AppError) {
	return s.getPublicOpportunity(`
		SELECT id, company_id, created_by_user_id, title, summary, slug, description,
			type, status, moderation_status, participation_format, location_id,
			contact_email, contact_phone, cover_media_id, published_at, expires_at,
			created_at, updated_at
		FROM opportunities
		WHERE id = $1
	`, opportunityID)
}

func (s *Store) GetPublicOpportunityBySlug(slug string) (*model.Opportunity, *commonstore.AppError) {
	return s.getPublicOpportunity(`
		SELECT id, company_id, created_by_user_id, title, summary, slug, description,
			type, status, moderation_status, participation_format, location_id,
			contact_email, contact_phone, cover_media_id, published_at, expires_at,
			created_at, updated_at
		FROM opportunities
		WHERE slug = $1
	`, slug)
}

func (s *Store) getPublicOpportunity(query, value string) (*model.Opportunity, *commonstore.AppError) {
	ctx := context.Background()
	opportunity, err := scanOpportunity(s.db.QueryRow(ctx, query, value))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "opportunity not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load opportunity", nil)
	}
	if repoErr := s.hydrateOpportunity(ctx, s.db, opportunity); repoErr != nil {
		return nil, repoErr
	}
	if !isPublicOpportunity(opportunity, time.Now().UTC()) {
		return nil, appErr(404, "not_found", "opportunity not found", nil)
	}
	return opportunity, nil
}

func (s *Store) scanOpportunityList(ctx context.Context, rows pgx.Rows) ([]*model.Opportunity, *commonstore.AppError) {
	items := make([]*model.Opportunity, 0)
	for rows.Next() {
		opportunity, err := scanOpportunity(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load opportunities", nil)
		}
		items = append(items, opportunity)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load opportunities", nil)
	}
	for _, opportunity := range items {
		if repoErr := s.hydrateOpportunity(ctx, s.db, opportunity); repoErr != nil {
			return nil, repoErr
		}
	}
	return items, nil
}

func (s *Store) hydrateOpportunity(ctx context.Context, q queryable, opportunity *model.Opportunity) *commonstore.AppError {
	company, err := scanCompany(q.QueryRow(ctx, `
		SELECT id, legal_name, brand_name, slug, inn, description, industry, website_url,
			corporate_email_domain, headquarters_location_id, logo_media_id, banner_media_id,
			verification_status, verified_at, verified_by_curator_user_id, created_by_user_id,
			created_at, updated_at
		FROM companies
		WHERE id = $1
	`, opportunity.CompanyID))
	if err != nil {
		return appErr(500, "internal_error", "failed to load company", nil)
	}
	opportunity.Company = company

	vacancy, _ := scanOpportunityVacancy(q.QueryRow(ctx, `
		SELECT opportunity_id, employment_type, experience_level, salary_from, salary_to, currency
		FROM opportunity_vacancy_details
		WHERE opportunity_id = $1
	`, opportunity.ID))
	opportunity.VacancyDetails = vacancy

	mentor, _ := scanOpportunityMentorProgram(q.QueryRow(ctx, `
		SELECT opportunity_id, start_at, end_at, seats_count, mentor_requirements
		FROM opportunity_mentor_program_details
		WHERE opportunity_id = $1
	`, opportunity.ID))
	opportunity.MentorProgramDetails = mentor

	event, _ := scanOpportunityEvent(q.QueryRow(ctx, `
		SELECT opportunity_id, start_at, end_at, registration_deadline, capacity, venue_note
		FROM opportunity_event_details
		WHERE opportunity_id = $1
	`, opportunity.ID))
	opportunity.EventDetails = event

	tagRows, err := q.Query(ctx, `
		SELECT t.id, t.name, t.tag_type, t.is_system, t.is_active, t.created_by_user_id, t.created_at
		FROM opportunity_tags ot
		JOIN tags t ON t.id = ot.tag_id
		WHERE ot.opportunity_id = $1
		ORDER BY t.name ASC
	`, opportunity.ID)
	if err != nil {
		return appErr(500, "internal_error", "failed to load opportunity tags", nil)
	}
	defer tagRows.Close()
	opportunity.Tags = []*model.Tag{}
	for tagRows.Next() {
		tag, err := scanTag(tagRows)
		if err != nil {
			return appErr(500, "internal_error", "failed to load opportunity tags", nil)
		}
		opportunity.Tags = append(opportunity.Tags, tag)
	}

	linkRows, err := q.Query(ctx, `
		SELECT id, opportunity_id, link_type, title, url, sort_order
		FROM opportunity_links
		WHERE opportunity_id = $1
		ORDER BY sort_order ASC, id ASC
	`, opportunity.ID)
	if err != nil {
		return appErr(500, "internal_error", "failed to load opportunity links", nil)
	}
	defer linkRows.Close()
	opportunity.Links = []model.OpportunityLink{}
	for linkRows.Next() {
		link, err := scanOpportunityLink(linkRows)
		if err != nil {
			return appErr(500, "internal_error", "failed to load opportunity links", nil)
		}
		opportunity.Links = append(opportunity.Links, *link)
	}

	mediaRows, err := q.Query(ctx, `
		SELECT id, opportunity_id, media_file_id, title, sort_order, created_at
		FROM opportunity_media
		WHERE opportunity_id = $1
		ORDER BY sort_order ASC, id ASC
	`, opportunity.ID)
	if err != nil {
		return appErr(500, "internal_error", "failed to load opportunity media", nil)
	}
	defer mediaRows.Close()
	opportunity.Media = []model.OpportunityMedia{}
	for mediaRows.Next() {
		media, err := scanOpportunityMedia(mediaRows)
		if err != nil {
			return appErr(500, "internal_error", "failed to load opportunity media", nil)
		}
		opportunity.Media = append(opportunity.Media, *media)
	}

	stats, _ := scanOpportunityStats(q.QueryRow(ctx, `
		SELECT opportunity_id, views_count, updated_at
		FROM opportunity_stats
		WHERE opportunity_id = $1
	`, opportunity.ID))
	opportunity.Stats = stats
	return nil
}

func (s *Store) replaceOpportunityDetails(ctx context.Context, q queryable, opportunityID, opportunityType string, vacancy *model.OpportunityVacancyDetails, mentor *model.OpportunityMentorProgramDetails, event *model.OpportunityEventDetails) *commonstore.AppError {
	if _, err := q.Exec(ctx, `DELETE FROM opportunity_vacancy_details WHERE opportunity_id = $1`, opportunityID); err != nil {
		return appErr(500, "internal_error", "failed to update opportunity details", nil)
	}
	if _, err := q.Exec(ctx, `DELETE FROM opportunity_mentor_program_details WHERE opportunity_id = $1`, opportunityID); err != nil {
		return appErr(500, "internal_error", "failed to update opportunity details", nil)
	}
	if _, err := q.Exec(ctx, `DELETE FROM opportunity_event_details WHERE opportunity_id = $1`, opportunityID); err != nil {
		return appErr(500, "internal_error", "failed to update opportunity details", nil)
	}

	switch opportunityType {
	case model.OpportunityTypeInternship, model.OpportunityTypeVacancy:
		if vacancy == nil {
			return appErr(422, "validation_error", "vacancyDetails are required for this opportunity type", nil)
		}
		if _, err := q.Exec(ctx, `
			INSERT INTO opportunity_vacancy_details (
				opportunity_id, employment_type, experience_level, salary_from, salary_to, currency
			) VALUES ($1, $2, $3, $4, $5, $6)
		`, opportunityID, vacancy.EmploymentType, vacancy.ExperienceLevel, vacancy.SalaryFrom, vacancy.SalaryTo, vacancy.Currency); err != nil {
			return appErr(500, "internal_error", "failed to save vacancy details", nil)
		}
	case model.OpportunityTypeMentorProgram:
		if mentor == nil {
			return appErr(422, "validation_error", "mentorProgramDetails are required for this opportunity type", nil)
		}
		if _, err := q.Exec(ctx, `
			INSERT INTO opportunity_mentor_program_details (
				opportunity_id, start_at, end_at, seats_count, mentor_requirements
			) VALUES ($1, $2, $3, $4, $5)
		`, opportunityID, mentor.StartAt, mentor.EndAt, mentor.SeatsCount, mentor.MentorRequirements); err != nil {
			return appErr(500, "internal_error", "failed to save mentor program details", nil)
		}
	case model.OpportunityTypeEvent:
		if event == nil {
			return appErr(422, "validation_error", "eventDetails are required for this opportunity type", nil)
		}
		if _, err := q.Exec(ctx, `
			INSERT INTO opportunity_event_details (
				opportunity_id, start_at, end_at, registration_deadline, capacity, venue_note
			) VALUES ($1, $2, $3, $4, $5, $6)
		`, opportunityID, event.StartAt, event.EndAt, event.RegistrationDeadline, event.Capacity, event.VenueNote); err != nil {
			return appErr(500, "internal_error", "failed to save event details", nil)
		}
	default:
		return appErr(422, "validation_error", "unsupported opportunity type", nil)
	}
	return nil
}

func (s *Store) replaceOpportunityTags(ctx context.Context, q queryable, opportunityID string, tagIDs []string) *commonstore.AppError {
	if _, err := q.Exec(ctx, `DELETE FROM opportunity_tags WHERE opportunity_id = $1`, opportunityID); err != nil {
		return appErr(500, "internal_error", "failed to update opportunity tags", nil)
	}
	for _, tagID := range uniqueSortedStrings(tagIDs) {
		if _, err := q.Exec(ctx, `
			INSERT INTO opportunity_tags (opportunity_id, tag_id)
			VALUES ($1, $2)
		`, opportunityID, tagID); err != nil {
			return appErr(500, "internal_error", "failed to update opportunity tags", nil)
		}
	}
	return nil
}

func (s *Store) replaceOpportunityLinks(ctx context.Context, q queryable, opportunityID string, links []model.OpportunityLinkInput) *commonstore.AppError {
	if _, err := q.Exec(ctx, `DELETE FROM opportunity_links WHERE opportunity_id = $1`, opportunityID); err != nil {
		return appErr(500, "internal_error", "failed to update opportunity links", nil)
	}
	for _, link := range links {
		if _, err := q.Exec(ctx, `
			INSERT INTO opportunity_links (opportunity_id, link_type, title, url, sort_order)
			VALUES ($1, $2, $3, $4, $5)
		`, opportunityID, link.LinkType, link.Title, link.URL, link.SortOrder); err != nil {
			return appErr(500, "internal_error", "failed to update opportunity links", nil)
		}
	}
	return nil
}

func (s *Store) replaceOpportunityMedia(ctx context.Context, q queryable, opportunityID string, media []model.OpportunityMediaInput) *commonstore.AppError {
	if _, err := q.Exec(ctx, `DELETE FROM opportunity_media WHERE opportunity_id = $1`, opportunityID); err != nil {
		return appErr(500, "internal_error", "failed to update opportunity media", nil)
	}
	now := time.Now().UTC()
	for _, item := range media {
		if _, err := q.Exec(ctx, `
			INSERT INTO opportunity_media (opportunity_id, media_file_id, title, sort_order, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`, opportunityID, item.MediaFileID, item.Title, item.SortOrder, now); err != nil {
			return appErr(500, "internal_error", "failed to update opportunity media", nil)
		}
	}
	return nil
}

func (s *Store) validateTagIDs(ctx context.Context, q queryable, tagIDs []string) *commonstore.AppError {
	for _, tagID := range uniqueSortedStrings(tagIDs) {
		var exists bool
		if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tags WHERE id = $1)`, tagID).Scan(&exists); err != nil {
			return appErr(500, "internal_error", "failed to validate tag ids", nil)
		}
		if !exists {
			return appErr(422, "validation_error", "unknown tag id", map[string]any{"tagId": tagID})
		}
	}
	return nil
}

func validateCreateOpportunityInput(input model.CreateOpportunityInput) *commonstore.AppError {
	if input.CompanyID == "" || input.Title == "" || input.Summary == "" || input.Slug == "" || input.Description == "" || input.Type == "" || input.ParticipationFormat == "" {
		return appErr(422, "validation_error", "required fields are missing", nil)
	}
	if input.LocationID != nil && input.Location != nil {
		return appErr(422, "validation_error", "provide either locationId or location", nil)
	}
	if !isSupportedParticipationFormat(input.ParticipationFormat) {
		return appErr(422, "validation_error", "unsupported participation format", nil)
	}
	switch input.Type {
	case model.OpportunityTypeInternship, model.OpportunityTypeVacancy:
		if input.VacancyDetails == nil || input.MentorProgramDetails != nil || input.EventDetails != nil {
			return appErr(422, "validation_error", "detail block must match opportunity type", nil)
		}
	case model.OpportunityTypeMentorProgram:
		if input.VacancyDetails != nil || input.MentorProgramDetails == nil || input.EventDetails != nil {
			return appErr(422, "validation_error", "detail block must match opportunity type", nil)
		}
	case model.OpportunityTypeEvent:
		if input.VacancyDetails != nil || input.MentorProgramDetails != nil || input.EventDetails == nil {
			return appErr(422, "validation_error", "detail block must match opportunity type", nil)
		}
	default:
		return appErr(422, "validation_error", "unsupported opportunity type", nil)
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
	if repoErr := validateOpportunityLinks(input.Links); repoErr != nil {
		return repoErr
	}
	if repoErr := validateOpportunityMedia(input.Media); repoErr != nil {
		return repoErr
	}
	return nil
}

func validateUpdateOpportunityInput(current *model.Opportunity, input model.UpdateOpportunityInput) *commonstore.AppError {
	contentChange := input.Title != nil || input.Summary != nil || input.Slug != nil || input.Description != nil ||
		input.ParticipationFormat != nil || input.LocationID != nil || input.Location != nil ||
		input.ContactEmail != nil || input.ContactPhone != nil || input.CoverMediaID != nil ||
		input.PublishedAt != nil || input.ExpiresAt != nil || input.ReplaceTagIDs || input.ReplaceLinks ||
		input.ReplaceMedia || input.VacancyDetails != nil || input.MentorProgramDetails != nil || input.EventDetails != nil

	if current.ModerationStatus == model.ModerationStatusApproved && contentChange {
		return appErr(409, "conflict", "requested opportunity status transition is not allowed", nil)
	}
	if input.ParticipationFormat != nil && !isSupportedParticipationFormat(*input.ParticipationFormat) {
		return appErr(422, "validation_error", "unsupported participation format", nil)
	}
	if input.Status != nil && !isSupportedOpportunityStatus(*input.Status) {
		return appErr(422, "validation_error", "unsupported opportunity status", nil)
	}
	if input.Status != nil && !isAllowedOpportunityStatusTransition(current.Status, *input.Status) {
		return appErr(409, "conflict", "requested opportunity status transition is not allowed", nil)
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
	return nil
}

func isAllowedOpportunityStatusTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case model.OpportunityStatusDraft:
		return to == model.OpportunityStatusPlanned || to == model.OpportunityStatusArchived
	case model.OpportunityStatusPlanned:
		return to == model.OpportunityStatusActive || to == model.OpportunityStatusArchived
	case model.OpportunityStatusActive:
		return to == model.OpportunityStatusClosed || to == model.OpportunityStatusArchived
	case model.OpportunityStatusClosed:
		return to == model.OpportunityStatusArchived
	case model.OpportunityStatusRejected:
		return to == model.OpportunityStatusDraft || to == model.OpportunityStatusArchived
	default:
		return false
	}
}

func isPublicOpportunity(opportunity *model.Opportunity, now time.Time) bool {
	if opportunity.ModerationStatus != model.ModerationStatusApproved {
		return false
	}
	if opportunity.Status != model.OpportunityStatusPlanned && opportunity.Status != model.OpportunityStatusActive {
		return false
	}
	return opportunity.PublishedAt == nil || !opportunity.PublishedAt.After(now)
}

func (s *Store) filterPublicOpportunities(items []*model.Opportunity, input model.ListPublicOpportunitiesInput) []*model.Opportunity {
	now := time.Now().UTC()
	filtered := make([]*model.Opportunity, 0, len(items))
	for _, item := range items {
		if !isPublicOpportunity(item, now) {
			continue
		}
		if input.Q != "" && !containsFold(item.Title, input.Q) && !containsFold(item.Summary, input.Q) && !containsFold(item.Description, input.Q) {
			continue
		}
		if input.Type != "" && item.Type != input.Type {
			continue
		}
		if input.ParticipationFormat != "" && item.ParticipationFormat != input.ParticipationFormat {
			continue
		}
		if input.CompanyID != "" && item.CompanyID != input.CompanyID {
			continue
		}
		if input.City != "" && !s.opportunityMatchesCity(item, input.City) {
			continue
		}
		if input.BBox != nil && !s.opportunityMatchesBounds(item, input.BBox) {
			continue
		}
		if input.Lat != nil && input.Lng != nil && input.RadiusKm != nil && !s.opportunityMatchesRadius(item, *input.Lat, *input.Lng, *input.RadiusKm) {
			continue
		}
		if len(input.TagIDs) > 0 && !opportunityMatchesTags(item, input.TagIDs) {
			continue
		}
		if input.EmploymentType != "" {
			if item.VacancyDetails == nil || item.VacancyDetails.EmploymentType != input.EmploymentType {
				continue
			}
		}
		if input.ExperienceLevel != "" {
			if item.VacancyDetails == nil || item.VacancyDetails.ExperienceLevel != input.ExperienceLevel {
				continue
			}
		}
		if input.SalaryFrom != nil {
			if item.VacancyDetails == nil || item.VacancyDetails.SalaryFrom == nil || *item.VacancyDetails.SalaryFrom < *input.SalaryFrom {
				continue
			}
		}
		if input.SalaryTo != nil {
			if item.VacancyDetails == nil || item.VacancyDetails.SalaryTo == nil || *item.VacancyDetails.SalaryTo > *input.SalaryTo {
				continue
			}
		}
		if input.StartsAfter != nil {
			start := opportunityStartAt(item)
			if start == nil || start.Before(*input.StartsAfter) {
				continue
			}
		}
		if input.ExpiresAfter != nil {
			if item.ExpiresAt == nil || item.ExpiresAt.Before(*input.ExpiresAfter) {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func sortPublicOpportunities(items []*model.Opportunity, sortBy string) {
	switch sortBy {
	case "published_at_asc":
		sort.Slice(items, func(i, j int) bool {
			return compareTimePtrAsc(items[i].PublishedAt, items[j].PublishedAt, items[i].CreatedAt, items[j].CreatedAt)
		})
	case "salary_desc":
		sort.Slice(items, func(i, j int) bool {
			return compareSalary(items[i], items[j], true)
		})
	case "salary_asc":
		sort.Slice(items, func(i, j int) bool {
			return compareSalary(items[i], items[j], false)
		})
	case "starts_at_asc":
		sort.Slice(items, func(i, j int) bool {
			return compareStartAt(items[i], items[j])
		})
	default:
		sort.Slice(items, func(i, j int) bool {
			return compareTimePtrDesc(items[i].PublishedAt, items[j].PublishedAt, items[i].CreatedAt, items[j].CreatedAt)
		})
	}
}

func opportunityMatchesTags(opportunity *model.Opportunity, tagIDs []string) bool {
	needed := make(map[string]struct{}, len(tagIDs))
	for _, tagID := range tagIDs {
		needed[tagID] = struct{}{}
	}
	for _, tag := range opportunity.Tags {
		if _, ok := needed[tag.ID]; ok {
			return true
		}
	}
	return false
}

func (s *Store) opportunityMatchesCity(opportunity *model.Opportunity, city string) bool {
	location := s.opportunityLocation(opportunity)
	if location == nil {
		return false
	}
	return containsFold(location.City, city)
}

func (s *Store) opportunityMatchesBounds(opportunity *model.Opportunity, bounds *model.GeoBounds) bool {
	location := s.opportunityLocation(opportunity)
	if location == nil || location.Latitude == nil || location.Longitude == nil {
		return false
	}
	return *location.Longitude >= bounds.MinLng &&
		*location.Longitude <= bounds.MaxLng &&
		*location.Latitude >= bounds.MinLat &&
		*location.Latitude <= bounds.MaxLat
}

func (s *Store) opportunityMatchesRadius(opportunity *model.Opportunity, lat, lng, radiusKm float64) bool {
	location := s.opportunityLocation(opportunity)
	if location == nil || location.Latitude == nil || location.Longitude == nil {
		return false
	}
	return haversineDistanceKm(lat, lng, *location.Latitude, *location.Longitude) <= radiusKm
}

func (s *Store) opportunityLocation(opportunity *model.Opportunity) *model.Location {
	if opportunity.LocationID == nil {
		return nil
	}
	return s.GetLocationByID(opportunity.LocationID)
}

func opportunityStartAt(opportunity *model.Opportunity) *time.Time {
	if opportunity.MentorProgramDetails != nil {
		return opportunity.MentorProgramDetails.StartAt
	}
	if opportunity.EventDetails != nil {
		return &opportunity.EventDetails.StartAt
	}
	return nil
}

func compareTimePtrDesc(left, right *time.Time, leftFallback, rightFallback time.Time) bool {
	if left == nil && right == nil {
		return leftFallback.After(rightFallback)
	}
	if left == nil {
		return false
	}
	if right == nil {
		return true
	}
	if left.Equal(*right) {
		return leftFallback.After(rightFallback)
	}
	return left.After(*right)
}

func compareTimePtrAsc(left, right *time.Time, leftFallback, rightFallback time.Time) bool {
	if left == nil && right == nil {
		return leftFallback.After(rightFallback)
	}
	if left == nil {
		return true
	}
	if right == nil {
		return false
	}
	if left.Equal(*right) {
		return leftFallback.After(rightFallback)
	}
	return left.Before(*right)
}

func compareSalary(left, right *model.Opportunity, descending bool) bool {
	leftSalary, leftOk := opportunitySalaryValue(left)
	rightSalary, rightOk := opportunitySalaryValue(right)
	if leftOk != rightOk {
		return leftOk
	}
	if !leftOk && !rightOk {
		return compareTimePtrDesc(left.PublishedAt, right.PublishedAt, left.CreatedAt, right.CreatedAt)
	}
	if leftSalary == rightSalary {
		return compareTimePtrDesc(left.PublishedAt, right.PublishedAt, left.CreatedAt, right.CreatedAt)
	}
	if descending {
		return leftSalary > rightSalary
	}
	return leftSalary < rightSalary
}

func compareStartAt(left, right *model.Opportunity) bool {
	leftStart := opportunityStartAt(left)
	rightStart := opportunityStartAt(right)
	if leftStart == nil && rightStart == nil {
		return compareTimePtrDesc(left.PublishedAt, right.PublishedAt, left.CreatedAt, right.CreatedAt)
	}
	if leftStart == nil {
		return false
	}
	if rightStart == nil {
		return true
	}
	if leftStart.Equal(*rightStart) {
		return compareTimePtrDesc(left.PublishedAt, right.PublishedAt, left.CreatedAt, right.CreatedAt)
	}
	return leftStart.Before(*rightStart)
}

func opportunitySalaryValue(opportunity *model.Opportunity) (int, bool) {
	if opportunity.VacancyDetails == nil {
		return 0, false
	}
	if opportunity.VacancyDetails.SalaryTo != nil {
		return *opportunity.VacancyDetails.SalaryTo, true
	}
	if opportunity.VacancyDetails.SalaryFrom != nil {
		return *opportunity.VacancyDetails.SalaryFrom, true
	}
	return 0, false
}

func containsFold(value, needle string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(needle))
}

func haversineDistanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0

	lat1Rad := lat1 * math.Pi / 180
	lng1Rad := lng1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lng2Rad := lng2 * math.Pi / 180

	dLat := lat2Rad - lat1Rad
	dLng := lng2Rad - lng1Rad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func detailsBlockCount(vacancy *model.OpportunityVacancyDetails, mentor *model.OpportunityMentorProgramDetails, event *model.OpportunityEventDetails) int {
	count := 0
	if vacancy != nil {
		count++
	}
	if mentor != nil {
		count++
	}
	if event != nil {
		count++
	}
	return count
}

func validateVacancyDetails(details *model.OpportunityVacancyDetails) *commonstore.AppError {
	if details == nil {
		return nil
	}
	if !isSupportedEmploymentType(details.EmploymentType) {
		return appErr(422, "validation_error", "unsupported employment type", nil)
	}
	if !isSupportedExperienceLevel(details.ExperienceLevel) {
		return appErr(422, "validation_error", "unsupported experience level", nil)
	}
	return nil
}

func validateMentorProgramDetails(details *model.OpportunityMentorProgramDetails) *commonstore.AppError {
	_ = details
	return nil
}

func validateEventDetails(details *model.OpportunityEventDetails) *commonstore.AppError {
	if details == nil {
		return nil
	}
	if details.StartAt.IsZero() || details.EndAt.IsZero() {
		return appErr(422, "validation_error", "eventDetails.startAt and eventDetails.endAt are required", nil)
	}
	return nil
}

func validateOpportunityLinks(links []model.OpportunityLinkInput) *commonstore.AppError {
	for _, link := range links {
		if link.Title == "" || link.URL == "" {
			return appErr(422, "validation_error", "opportunity links require title and url", nil)
		}
		if !isSupportedLinkType(link.LinkType) {
			return appErr(422, "validation_error", "unsupported link type", nil)
		}
	}
	return nil
}

func validateOpportunityMedia(media []model.OpportunityMediaInput) *commonstore.AppError {
	for _, item := range media {
		if item.MediaFileID == "" {
			return appErr(422, "validation_error", "opportunity media requires mediaFileId", nil)
		}
	}
	return nil
}

func isSupportedParticipationFormat(value string) bool {
	switch value {
	case "offline", "hybrid", "remote", "online":
		return true
	default:
		return false
	}
}

func isSupportedEmploymentType(value string) bool {
	switch value {
	case "full_time", "part_time", "project", "contract":
		return true
	default:
		return false
	}
}

func isSupportedExperienceLevel(value string) bool {
	switch value {
	case "trainee", "junior", "middle", "senior":
		return true
	default:
		return false
	}
}

func isSupportedOpportunityStatus(value string) bool {
	switch value {
	case model.OpportunityStatusDraft, model.OpportunityStatusPlanned, model.OpportunityStatusActive, model.OpportunityStatusClosed, model.OpportunityStatusRejected, model.OpportunityStatusArchived:
		return true
	default:
		return false
	}
}

func isSupportedLinkType(value string) bool {
	switch value {
	case "apply", "info", "registration", "social", "other":
		return true
	default:
		return false
	}
}

func scanOpportunity(src scanner) (*model.Opportunity, error) {
	var opportunity model.Opportunity
	var locationID, contactEmail, contactPhone, coverMediaID sql.NullString
	var publishedAt, expiresAt sql.NullTime
	if err := src.Scan(
		&opportunity.ID,
		&opportunity.CompanyID,
		&opportunity.CreatedByUserID,
		&opportunity.Title,
		&opportunity.Summary,
		&opportunity.Slug,
		&opportunity.Description,
		&opportunity.Type,
		&opportunity.Status,
		&opportunity.ModerationStatus,
		&opportunity.ParticipationFormat,
		&locationID,
		&contactEmail,
		&contactPhone,
		&coverMediaID,
		&publishedAt,
		&expiresAt,
		&opportunity.CreatedAt,
		&opportunity.UpdatedAt,
	); err != nil {
		return nil, err
	}
	opportunity.LocationID = nullStringPtr(locationID)
	opportunity.ContactEmail = nullStringPtr(contactEmail)
	opportunity.ContactPhone = nullStringPtr(contactPhone)
	opportunity.CoverMediaID = nullStringPtr(coverMediaID)
	opportunity.PublishedAt = nullTimePtr(publishedAt)
	opportunity.ExpiresAt = nullTimePtr(expiresAt)
	return &opportunity, nil
}

func scanOpportunityVacancy(src scanner) (*model.OpportunityVacancyDetails, error) {
	var opportunityID string
	var details model.OpportunityVacancyDetails
	var salaryFrom, salaryTo sql.NullInt32
	var currency sql.NullString
	err := src.Scan(&opportunityID, &details.EmploymentType, &details.ExperienceLevel, &salaryFrom, &salaryTo, &currency)
	if err != nil {
		return nil, err
	}
	details.SalaryFrom = nullInt32Ptr(salaryFrom)
	details.SalaryTo = nullInt32Ptr(salaryTo)
	details.Currency = nullStringPtr(currency)
	return &details, nil
}

func scanOpportunityMentorProgram(src scanner) (*model.OpportunityMentorProgramDetails, error) {
	var opportunityID string
	var details model.OpportunityMentorProgramDetails
	var startAt, endAt sql.NullTime
	var seatsCount sql.NullInt32
	var mentorRequirements sql.NullString
	err := src.Scan(&opportunityID, &startAt, &endAt, &seatsCount, &mentorRequirements)
	if err != nil {
		return nil, err
	}
	details.StartAt = nullTimePtr(startAt)
	details.EndAt = nullTimePtr(endAt)
	details.SeatsCount = nullInt32Ptr(seatsCount)
	details.MentorRequirements = nullStringPtr(mentorRequirements)
	return &details, nil
}

func scanOpportunityEvent(src scanner) (*model.OpportunityEventDetails, error) {
	var opportunityID string
	var details model.OpportunityEventDetails
	var registrationDeadline sql.NullTime
	var capacity sql.NullInt32
	var venueNote sql.NullString
	err := src.Scan(&opportunityID, &details.StartAt, &details.EndAt, &registrationDeadline, &capacity, &venueNote)
	if err != nil {
		return nil, err
	}
	details.RegistrationDeadline = nullTimePtr(registrationDeadline)
	details.Capacity = nullInt32Ptr(capacity)
	details.VenueNote = nullStringPtr(venueNote)
	return &details, nil
}

func scanOpportunityLink(src scanner) (*model.OpportunityLink, error) {
	var link model.OpportunityLink
	if err := src.Scan(&link.ID, &link.OpportunityID, &link.LinkType, &link.Title, &link.URL, &link.SortOrder); err != nil {
		return nil, err
	}
	return &link, nil
}

func scanOpportunityMedia(src scanner) (*model.OpportunityMedia, error) {
	var media model.OpportunityMedia
	var title sql.NullString
	if err := src.Scan(&media.ID, &media.OpportunityID, &media.MediaFileID, &title, &media.SortOrder, &media.CreatedAt); err != nil {
		return nil, err
	}
	media.Title = nullStringPtr(title)
	return &media, nil
}

func scanOpportunityStats(src scanner) (*model.OpportunityStats, error) {
	var stats model.OpportunityStats
	if err := src.Scan(&stats.OpportunityID, &stats.ViewsCount, &stats.UpdatedAt); err != nil {
		return nil, err
	}
	return &stats, nil
}

func nullInt32Ptr(value sql.NullInt32) *int {
	if !value.Valid {
		return nil
	}
	number := int(value.Int32)
	return &number
}
