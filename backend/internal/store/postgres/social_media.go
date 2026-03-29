package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

func (s *Store) ListApplicantSocialLinks(applicantUserID string) ([]*model.ApplicantSocialLink, *commonstore.AppError) {
	ctx := context.Background()
	if !s.applicantExists(ctx, s.db, applicantUserID) {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}
	return s.listApplicantSocialLinks(ctx, s.db, applicantUserID)
}

func (s *Store) CreateApplicantSocialLink(applicantUserID string, input model.CreateApplicantSocialLinkInput) (*model.ApplicantSocialLink, *commonstore.AppError) {
	if strings.TrimSpace(input.Platform) == "" || strings.TrimSpace(input.URL) == "" {
		return nil, appErr(422, "validation_error", "required fields are missing", nil)
	}

	ctx := context.Background()
	if !s.applicantExists(ctx, s.db, applicantUserID) {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}

	link, err := scanApplicantSocialLink(s.db.QueryRow(ctx, `
		INSERT INTO applicant_social_links (
			applicant_user_id, platform, url, is_public, created_at
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, applicant_user_id, platform, url, is_public, created_at
	`, applicantUserID, strings.TrimSpace(input.Platform), strings.TrimSpace(input.URL), input.IsPublic, time.Now().UTC()))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create applicant social link", nil)
	}
	return link, nil
}

func (s *Store) UpdateApplicantSocialLink(applicantUserID, linkID string, input model.UpdateApplicantSocialLinkInput) (*model.ApplicantSocialLink, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update applicant social link", nil)
	}
	defer tx.Rollback(ctx)

	if !s.applicantExists(ctx, tx, applicantUserID) {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}

	link, err := scanApplicantSocialLink(tx.QueryRow(ctx, `
		SELECT id, applicant_user_id, platform, url, is_public, created_at
		FROM applicant_social_links
		WHERE id = $1 AND applicant_user_id = $2
		FOR UPDATE
	`, linkID, applicantUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "social link not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load applicant social link", nil)
	}

	if input.Platform != nil {
		link.Platform = strings.TrimSpace(*input.Platform)
	}
	if input.URL != nil {
		link.URL = strings.TrimSpace(*input.URL)
	}
	if input.IsPublic != nil {
		link.IsPublic = *input.IsPublic
	}
	if link.Platform == "" || link.URL == "" {
		return nil, appErr(422, "validation_error", "required fields are missing", nil)
	}

	_, err = tx.Exec(ctx, `
		UPDATE applicant_social_links
		SET platform = $1, url = $2, is_public = $3
		WHERE id = $4
	`, link.Platform, link.URL, link.IsPublic, link.ID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update applicant social link", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update applicant social link", nil)
	}
	return link, nil
}

func (s *Store) DeleteApplicantSocialLink(applicantUserID, linkID string) *commonstore.AppError {
	ctx := context.Background()
	if !s.applicantExists(ctx, s.db, applicantUserID) {
		return appErr(403, "forbidden", "current user is not an applicant", nil)
	}

	result, err := s.db.Exec(ctx, `
		DELETE FROM applicant_social_links
		WHERE id = $1 AND applicant_user_id = $2
	`, linkID, applicantUserID)
	if err != nil {
		return appErr(500, "internal_error", "failed to delete applicant social link", nil)
	}
	if result.RowsAffected() == 0 {
		return appErr(404, "not_found", "social link not found", nil)
	}
	return nil
}

func (s *Store) ListCompanySocialLinks(actorUserID, companyID string) ([]*model.CompanySocialLink, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireApprovedMembership(ctx, s.db, actorUserID, companyID); repoErr != nil {
		return nil, repoErr
	}
	return s.listCompanySocialLinks(ctx, s.db, companyID)
}

func (s *Store) CreateCompanySocialLink(actorUserID, companyID string, input model.CreateCompanySocialLinkInput) (*model.CompanySocialLink, *commonstore.AppError) {
	if strings.TrimSpace(input.Platform) == "" || strings.TrimSpace(input.URL) == "" {
		return nil, appErr(422, "validation_error", "required fields are missing", nil)
	}

	ctx := context.Background()
	if _, repoErr := s.requireApprovedMembership(ctx, s.db, actorUserID, companyID); repoErr != nil {
		return nil, repoErr
	}

	link, err := scanCompanySocialLink(s.db.QueryRow(ctx, `
		INSERT INTO company_social_links (company_id, platform, url)
		VALUES ($1, $2, $3)
		RETURNING id, company_id, platform, url
	`, companyID, strings.TrimSpace(input.Platform), strings.TrimSpace(input.URL)))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create company social link", nil)
	}
	return link, nil
}

func (s *Store) UpdateCompanySocialLink(actorUserID, companyID, linkID string, input model.UpdateCompanySocialLinkInput) (*model.CompanySocialLink, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update company social link", nil)
	}
	defer tx.Rollback(ctx)

	if _, repoErr := s.requireApprovedMembership(ctx, tx, actorUserID, companyID); repoErr != nil {
		return nil, repoErr
	}

	link, err := scanCompanySocialLink(tx.QueryRow(ctx, `
		SELECT id, company_id, platform, url
		FROM company_social_links
		WHERE id = $1 AND company_id = $2
		FOR UPDATE
	`, linkID, companyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "company social link not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load company social link", nil)
	}

	if input.Platform != nil {
		link.Platform = strings.TrimSpace(*input.Platform)
	}
	if input.URL != nil {
		link.URL = strings.TrimSpace(*input.URL)
	}
	if link.Platform == "" || link.URL == "" {
		return nil, appErr(422, "validation_error", "required fields are missing", nil)
	}

	_, err = tx.Exec(ctx, `
		UPDATE company_social_links
		SET platform = $1, url = $2
		WHERE id = $3
	`, link.Platform, link.URL, link.ID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update company social link", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update company social link", nil)
	}
	return link, nil
}

func (s *Store) DeleteCompanySocialLink(actorUserID, companyID, linkID string) *commonstore.AppError {
	ctx := context.Background()
	if _, repoErr := s.requireApprovedMembership(ctx, s.db, actorUserID, companyID); repoErr != nil {
		return repoErr
	}

	result, err := s.db.Exec(ctx, `
		DELETE FROM company_social_links
		WHERE id = $1 AND company_id = $2
	`, linkID, companyID)
	if err != nil {
		return appErr(500, "internal_error", "failed to delete company social link", nil)
	}
	if result.RowsAffected() == 0 {
		return appErr(404, "not_found", "company social link not found", nil)
	}
	return nil
}

func (s *Store) ListCompanyMedia(actorUserID, companyID string) ([]*model.CompanyMedia, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireApprovedMembership(ctx, s.db, actorUserID, companyID); repoErr != nil {
		return nil, repoErr
	}
	return s.listCompanyMedia(ctx, s.db, companyID)
}

func (s *Store) CreateCompanyMedia(actorUserID, companyID string, input model.CreateCompanyMediaInput) (*model.CompanyMedia, *commonstore.AppError) {
	mediaFileID := strings.TrimSpace(input.MediaFileID)
	if mediaFileID == "" {
		return nil, appErr(422, "validation_error", "required fields are missing", nil)
	}

	ctx := context.Background()
	if _, repoErr := s.requireApprovedMembership(ctx, s.db, actorUserID, companyID); repoErr != nil {
		return nil, repoErr
	}

	media, err := scanCompanyMedia(s.db.QueryRow(ctx, `
		INSERT INTO company_media (company_id, media_file_id, title, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, company_id, media_file_id, title, sort_order, created_at
	`, companyID, mediaFileID, input.Title, input.SortOrder, time.Now().UTC()))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create company media entry", nil)
	}
	return media, nil
}

func (s *Store) UpdateCompanyMedia(actorUserID, companyID, mediaID string, input model.UpdateCompanyMediaInput) (*model.CompanyMedia, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update company media entry", nil)
	}
	defer tx.Rollback(ctx)

	if _, repoErr := s.requireApprovedMembership(ctx, tx, actorUserID, companyID); repoErr != nil {
		return nil, repoErr
	}

	media, err := scanCompanyMedia(tx.QueryRow(ctx, `
		SELECT id, company_id, media_file_id, title, sort_order, created_at
		FROM company_media
		WHERE id = $1 AND company_id = $2
		FOR UPDATE
	`, mediaID, companyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "company media entry not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load company media entry", nil)
	}

	if input.Title != nil {
		media.Title = input.Title
	}
	if input.SortOrder != nil {
		media.SortOrder = *input.SortOrder
	}

	_, err = tx.Exec(ctx, `
		UPDATE company_media
		SET title = $1, sort_order = $2
		WHERE id = $3
	`, media.Title, media.SortOrder, media.ID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update company media entry", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update company media entry", nil)
	}
	return media, nil
}

func (s *Store) DeleteCompanyMedia(actorUserID, companyID, mediaID string) *commonstore.AppError {
	ctx := context.Background()
	if _, repoErr := s.requireApprovedMembership(ctx, s.db, actorUserID, companyID); repoErr != nil {
		return repoErr
	}

	result, err := s.db.Exec(ctx, `
		DELETE FROM company_media
		WHERE id = $1 AND company_id = $2
	`, mediaID, companyID)
	if err != nil {
		return appErr(500, "internal_error", "failed to delete company media entry", nil)
	}
	if result.RowsAffected() == 0 {
		return appErr(404, "not_found", "company media entry not found", nil)
	}
	return nil
}

func (s *Store) hydrateCompanyPublicFields(ctx context.Context, q queryable, company *model.Company) *commonstore.AppError {
	if company == nil {
		return nil
	}

	socialLinks, repoErr := s.listCompanySocialLinks(ctx, q, company.ID)
	if repoErr != nil {
		return repoErr
	}
	company.SocialLinks = make([]model.CompanySocialLink, 0, len(socialLinks))
	for _, link := range socialLinks {
		company.SocialLinks = append(company.SocialLinks, *link)
	}

	mediaItems, repoErr := s.listCompanyMedia(ctx, q, company.ID)
	if repoErr != nil {
		return repoErr
	}
	company.Media = make([]model.CompanyMedia, 0, len(mediaItems))
	for _, media := range mediaItems {
		company.Media = append(company.Media, *media)
	}
	return nil
}

func (s *Store) listApplicantSocialLinks(ctx context.Context, q queryable, applicantUserID string) ([]*model.ApplicantSocialLink, *commonstore.AppError) {
	rows, err := q.Query(ctx, `
		SELECT id, applicant_user_id, platform, url, is_public, created_at
		FROM applicant_social_links
		WHERE applicant_user_id = $1
		ORDER BY created_at ASC, id ASC
	`, applicantUserID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load applicant social links", nil)
	}
	defer rows.Close()

	items := make([]*model.ApplicantSocialLink, 0)
	for rows.Next() {
		link, err := scanApplicantSocialLink(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load applicant social links", nil)
		}
		items = append(items, link)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load applicant social links", nil)
	}
	return items, nil
}

func (s *Store) listCompanySocialLinks(ctx context.Context, q queryable, companyID string) ([]*model.CompanySocialLink, *commonstore.AppError) {
	rows, err := q.Query(ctx, `
		SELECT id, company_id, platform, url
		FROM company_social_links
		WHERE company_id = $1
		ORDER BY id ASC
	`, companyID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load company social links", nil)
	}
	defer rows.Close()

	items := make([]*model.CompanySocialLink, 0)
	for rows.Next() {
		link, err := scanCompanySocialLink(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load company social links", nil)
		}
		items = append(items, link)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load company social links", nil)
	}
	return items, nil
}

func (s *Store) listCompanyMedia(ctx context.Context, q queryable, companyID string) ([]*model.CompanyMedia, *commonstore.AppError) {
	rows, err := q.Query(ctx, `
		SELECT id, company_id, media_file_id, title, sort_order, created_at
		FROM company_media
		WHERE company_id = $1
		ORDER BY sort_order ASC, created_at ASC, id ASC
	`, companyID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load company media", nil)
	}
	defer rows.Close()

	items := make([]*model.CompanyMedia, 0)
	for rows.Next() {
		media, err := scanCompanyMedia(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load company media", nil)
		}
		items = append(items, media)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load company media", nil)
	}
	return items, nil
}

func scanApplicantSocialLink(src scanner) (*model.ApplicantSocialLink, error) {
	var link model.ApplicantSocialLink
	if err := src.Scan(&link.ID, &link.ApplicantUserID, &link.Platform, &link.URL, &link.IsPublic, &link.CreatedAt); err != nil {
		return nil, err
	}
	return &link, nil
}

func scanCompanySocialLink(src scanner) (*model.CompanySocialLink, error) {
	var link model.CompanySocialLink
	if err := src.Scan(&link.ID, &link.CompanyID, &link.Platform, &link.URL); err != nil {
		return nil, err
	}
	return &link, nil
}

func scanCompanyMedia(src scanner) (*model.CompanyMedia, error) {
	var media model.CompanyMedia
	var title sql.NullString
	if err := src.Scan(&media.ID, &media.CompanyID, &media.MediaFileID, &title, &media.SortOrder, &media.CreatedAt); err != nil {
		return nil, err
	}
	media.Title = nullStringPtr(title)
	return &media, nil
}
