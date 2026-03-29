package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

func (s *Store) ListSavedOpportunities(applicantUserID string) ([]*model.SavedOpportunity, *commonstore.AppError) {
	ctx := context.Background()
	if !s.applicantExists(ctx, s.db, applicantUserID) {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}

	rows, err := s.db.Query(ctx, `
		SELECT applicant_user_id, opportunity_id, created_at
		FROM applicant_saved_opportunities
		WHERE applicant_user_id = $1
		ORDER BY created_at DESC, opportunity_id ASC
	`, applicantUserID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load saved opportunities", nil)
	}
	defer rows.Close()

	items := make([]*model.SavedOpportunity, 0)
	for rows.Next() {
		item, err := scanSavedOpportunity(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load saved opportunities", nil)
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load saved opportunities", nil)
	}
	for _, item := range items {
		opportunity, err := scanOpportunity(s.db.QueryRow(ctx, `
			SELECT id, company_id, created_by_user_id, title, summary, slug, description,
				type, status, moderation_status, participation_format, location_id,
				contact_email, contact_phone, cover_media_id, published_at, expires_at,
				created_at, updated_at
			FROM opportunities
			WHERE id = $1
		`, item.OpportunityID))
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load saved opportunities", nil)
		}
		if repoErr := s.hydrateOpportunity(ctx, s.db, opportunity); repoErr != nil {
			return nil, repoErr
		}
		item.Opportunity = opportunity
	}
	return items, nil
}

func (s *Store) SaveOpportunity(applicantUserID string, input model.SaveOpportunityInput) *commonstore.AppError {
	if input.OpportunityID == "" {
		return appErr(422, "validation_error", "required fields are missing", nil)
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return appErr(500, "internal_error", "failed to save opportunity", nil)
	}
	defer tx.Rollback(ctx)

	if !s.applicantExists(ctx, tx, applicantUserID) {
		return appErr(403, "forbidden", "current user is not an applicant", nil)
	}

	opportunity, err := scanOpportunity(tx.QueryRow(ctx, `
		SELECT id, company_id, created_by_user_id, title, summary, slug, description,
			type, status, moderation_status, participation_format, location_id,
			contact_email, contact_phone, cover_media_id, published_at, expires_at,
			created_at, updated_at
		FROM opportunities
		WHERE id = $1
	`, input.OpportunityID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appErr(404, "not_found", "opportunity not found", nil)
		}
		return appErr(500, "internal_error", "failed to load opportunity", nil)
	}

	if !isPublicOpportunity(opportunity, time.Now().UTC()) {
		return appErr(403, "forbidden", "opportunity is not available", nil)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO applicant_saved_opportunities (applicant_user_id, opportunity_id, created_at)
		VALUES ($1, $2, $3)
	`, applicantUserID, input.OpportunityID, time.Now().UTC()); err != nil {
		if isUniqueViolation(err, "applicant_saved_opportunities_pkey") {
			return appErr(409, "conflict", "conflict or duplicate entity", nil)
		}
		return appErr(500, "internal_error", "failed to save opportunity", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return appErr(500, "internal_error", "failed to save opportunity", nil)
	}
	return nil
}

func (s *Store) DeleteSavedOpportunity(applicantUserID, opportunityID string) *commonstore.AppError {
	ctx := context.Background()
	if !s.applicantExists(ctx, s.db, applicantUserID) {
		return appErr(403, "forbidden", "current user is not an applicant", nil)
	}
	if _, err := s.db.Exec(ctx, `
		DELETE FROM applicant_saved_opportunities
		WHERE applicant_user_id = $1 AND opportunity_id = $2
	`, applicantUserID, opportunityID); err != nil {
		return appErr(500, "internal_error", "failed to delete saved opportunity", nil)
	}
	return nil
}

func (s *Store) ListSavedCompanies(applicantUserID string) ([]*model.SavedCompany, *commonstore.AppError) {
	ctx := context.Background()
	if !s.applicantExists(ctx, s.db, applicantUserID) {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}

	rows, err := s.db.Query(ctx, `
		SELECT applicant_user_id, company_id, created_at
		FROM applicant_saved_companies
		WHERE applicant_user_id = $1
		ORDER BY created_at DESC, company_id ASC
	`, applicantUserID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load saved companies", nil)
	}
	defer rows.Close()

	items := make([]*model.SavedCompany, 0)
	for rows.Next() {
		item, err := scanSavedCompany(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load saved companies", nil)
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load saved companies", nil)
	}
	for _, item := range items {
		company, err := scanCompany(s.db.QueryRow(ctx, `
			SELECT id, legal_name, brand_name, slug, inn, description, industry, website_url,
				corporate_email_domain, headquarters_location_id, logo_media_id, banner_media_id,
				verification_status, verified_at, verified_by_curator_user_id, created_by_user_id,
				created_at, updated_at
			FROM companies
			WHERE id = $1
		`, item.CompanyID))
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load saved companies", nil)
		}
		item.Company = company
	}
	return items, nil
}

func (s *Store) SaveCompany(applicantUserID string, input model.SaveCompanyInput) *commonstore.AppError {
	if input.CompanyID == "" {
		return appErr(422, "validation_error", "required fields are missing", nil)
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return appErr(500, "internal_error", "failed to save company", nil)
	}
	defer tx.Rollback(ctx)

	if !s.applicantExists(ctx, tx, applicantUserID) {
		return appErr(403, "forbidden", "current user is not an applicant", nil)
	}

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM companies WHERE id = $1)`, input.CompanyID).Scan(&exists); err != nil {
		return appErr(500, "internal_error", "failed to load company", nil)
	}
	if !exists {
		return appErr(404, "not_found", "company not found", nil)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO applicant_saved_companies (applicant_user_id, company_id, created_at)
		VALUES ($1, $2, $3)
	`, applicantUserID, input.CompanyID, time.Now().UTC()); err != nil {
		if isUniqueViolation(err, "applicant_saved_companies_pkey") {
			return appErr(409, "conflict", "conflict or duplicate entity", nil)
		}
		return appErr(500, "internal_error", "failed to save company", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return appErr(500, "internal_error", "failed to save company", nil)
	}
	return nil
}

func (s *Store) DeleteSavedCompany(applicantUserID, companyID string) *commonstore.AppError {
	ctx := context.Background()
	if !s.applicantExists(ctx, s.db, applicantUserID) {
		return appErr(403, "forbidden", "current user is not an applicant", nil)
	}
	if _, err := s.db.Exec(ctx, `
		DELETE FROM applicant_saved_companies
		WHERE applicant_user_id = $1 AND company_id = $2
	`, applicantUserID, companyID); err != nil {
		return appErr(500, "internal_error", "failed to delete saved company", nil)
	}
	return nil
}

func scanSavedOpportunity(src scanner) (*model.SavedOpportunity, error) {
	var item model.SavedOpportunity
	if err := src.Scan(&item.ApplicantUserID, &item.OpportunityID, &item.CreatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func scanSavedCompany(src scanner) (*model.SavedCompany, error) {
	var item model.SavedCompany
	if err := src.Scan(&item.ApplicantUserID, &item.CompanyID, &item.CreatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}
