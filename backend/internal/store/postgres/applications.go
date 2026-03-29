package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

func (s *Store) ListApplicantApplications(applicantUserID string, input model.ListApplicationsInput) ([]*model.Application, int, *commonstore.AppError) {
	ctx := context.Background()
	if !s.applicantExists(ctx, s.db, applicantUserID) {
		return nil, 0, appErr(403, "forbidden", "current user is not an applicant", nil)
	}

	var status any
	if input.Status != "" {
		status = input.Status
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, opportunity_id, applicant_user_id, cover_letter, status, applied_at, updated_at
		FROM applications
		WHERE applicant_user_id = $1
			AND ($2::text IS NULL OR status::text = $2::text)
		ORDER BY applied_at DESC, id DESC
	`, applicantUserID, status)
	if err != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load applications", nil)
	}
	defer rows.Close()

	items, repoErr := s.scanApplicationList(ctx, rows)
	if repoErr != nil {
		return nil, 0, repoErr
	}
	paged, total := paginate(items, input.Page, input.PageSize)
	return paged, total, nil
}

func (s *Store) CreateApplication(applicantUserID string, input model.CreateApplicationInput) (*model.Application, *commonstore.AppError) {
	if input.OpportunityID == "" {
		return nil, appErr(422, "validation_error", "required fields are missing", nil)
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create application", nil)
	}
	defer tx.Rollback(ctx)

	if !s.applicantExists(ctx, tx, applicantUserID) {
		return nil, appErr(403, "forbidden", "only applicants can apply or opportunity is not available for applying", nil)
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
			return nil, appErr(403, "forbidden", "only applicants can apply or opportunity is not available for applying", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load opportunity", nil)
	}

	if !isOpportunityAvailableForApplying(opportunity, time.Now().UTC()) {
		return nil, appErr(403, "forbidden", "only applicants can apply or opportunity is not available for applying", nil)
	}

	now := time.Now().UTC()
	application, err := scanApplication(tx.QueryRow(ctx, `
		INSERT INTO applications (
			opportunity_id, applicant_user_id, cover_letter, status, applied_at, updated_at
		) VALUES ($1, $2, $3, 'submitted', $4, $4)
		RETURNING id, opportunity_id, applicant_user_id, cover_letter, status, applied_at, updated_at
	`, input.OpportunityID, applicantUserID, input.CoverLetter, now))
	if err != nil {
		if isUniqueViolation(err, "applications_opportunity_id_applicant_user_id_key") {
			return nil, appErr(409, "conflict", "applicant has already applied to this opportunity", nil)
		}
		return nil, appErr(500, "internal_error", "failed to create application", nil)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO application_status_history (
			application_id, old_status, new_status, changed_by_user_id, comment, created_at
		) VALUES ($1, NULL, 'submitted', $2, NULL, $3)
	`, application.ID, applicantUserID, now); err != nil {
		return nil, appErr(500, "internal_error", "failed to create application history", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to create application", nil)
	}

	return s.GetApplicantApplication(applicantUserID, application.ID)
}

func (s *Store) GetApplicantApplication(applicantUserID, applicationID string) (*model.Application, *commonstore.AppError) {
	ctx := context.Background()
	if !s.applicantExists(ctx, s.db, applicantUserID) {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}

	application, err := scanApplication(s.db.QueryRow(ctx, `
		SELECT id, opportunity_id, applicant_user_id, cover_letter, status, applied_at, updated_at
		FROM applications
		WHERE id = $1 AND applicant_user_id = $2
	`, applicationID, applicantUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "application not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load application", nil)
	}
	if repoErr := s.hydrateApplication(ctx, s.db, application); repoErr != nil {
		return nil, repoErr
	}
	return application, nil
}

func (s *Store) WithdrawApplicantApplication(applicantUserID, applicationID string) (*model.Application, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update application", nil)
	}
	defer tx.Rollback(ctx)

	if !s.applicantExists(ctx, tx, applicantUserID) {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}

	application, err := scanApplication(tx.QueryRow(ctx, `
		SELECT id, opportunity_id, applicant_user_id, cover_letter, status, applied_at, updated_at
		FROM applications
		WHERE id = $1 AND applicant_user_id = $2
		FOR UPDATE
	`, applicationID, applicantUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "application not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load application", nil)
	}

	if !canWithdrawApplicationStatus(application.Status) {
		return nil, appErr(409, "conflict", "application status transition is not allowed", nil)
	}

	now := time.Now().UTC()
	oldStatus := application.Status
	application.Status = model.ApplicationStatusWithdrawn
	application.UpdatedAt = now

	if _, err := tx.Exec(ctx, `
		UPDATE applications
		SET status = 'withdrawn', updated_at = $1
		WHERE id = $2
	`, now, application.ID); err != nil {
		return nil, appErr(500, "internal_error", "failed to update application", nil)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO application_status_history (
			application_id, old_status, new_status, changed_by_user_id, comment, created_at
		) VALUES ($1, $2, 'withdrawn', $3, NULL, $4)
	`, application.ID, oldStatus, applicantUserID, now); err != nil {
		return nil, appErr(500, "internal_error", "failed to update application history", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update application", nil)
	}

	return s.GetApplicantApplication(applicantUserID, application.ID)
}

func (s *Store) ListOpportunityApplications(actorUserID, opportunityID string, input model.ListApplicationsInput) ([]*model.Application, int, *commonstore.AppError) {
	ctx := context.Background()
	opportunity, repoErr := s.getEmployerAccessibleOpportunity(ctx, s.db, actorUserID, opportunityID)
	if repoErr != nil {
		return nil, 0, repoErr
	}

	var status any
	if input.Status != "" {
		status = input.Status
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, opportunity_id, applicant_user_id, cover_letter, status, applied_at, updated_at
		FROM applications
		WHERE opportunity_id = $1
			AND ($2::text IS NULL OR status::text = $2::text)
		ORDER BY applied_at DESC, id DESC
	`, opportunity.ID, status)
	if err != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load applications", nil)
	}
	defer rows.Close()

	items, scanErr := s.scanApplicationList(ctx, rows)
	if scanErr != nil {
		return nil, 0, scanErr
	}
	paged, total := paginate(items, input.Page, input.PageSize)
	return paged, total, nil
}

func (s *Store) UpdateApplicationStatus(actorUserID, applicationID string, input model.UpdateApplicationStatusInput) (*model.Application, *commonstore.AppError) {
	if !isEmployerSettableApplicationStatus(input.Status) {
		return nil, appErr(422, "validation_error", "invalid application status", nil)
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update application", nil)
	}
	defer tx.Rollback(ctx)

	application, err := scanApplication(tx.QueryRow(ctx, `
		SELECT id, opportunity_id, applicant_user_id, cover_letter, status, applied_at, updated_at
		FROM applications
		WHERE id = $1
		FOR UPDATE
	`, applicationID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "application not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load application", nil)
	}

	opportunity, repoErr := s.getEmployerAccessibleOpportunity(ctx, tx, actorUserID, application.OpportunityID)
	if repoErr != nil {
		if repoErr.Status == 404 {
			return nil, appErr(403, "forbidden", "current employer has no access to this application", nil)
		}
		if repoErr.Status == 403 {
			return nil, appErr(403, "forbidden", "current employer has no access to this application", nil)
		}
		return nil, repoErr
	}

	if !isAllowedEmployerApplicationStatusTransition(application.Status, input.Status) {
		return nil, appErr(409, "conflict", "requested application status transition is not allowed", nil)
	}

	now := time.Now().UTC()
	oldStatus := application.Status
	application.Status = input.Status
	application.UpdatedAt = now

	if _, err := tx.Exec(ctx, `
		UPDATE applications
		SET status = $1, updated_at = $2
		WHERE id = $3
	`, input.Status, now, application.ID); err != nil {
		return nil, appErr(500, "internal_error", "failed to update application", nil)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO application_status_history (
			application_id, old_status, new_status, changed_by_user_id, comment, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, application.ID, oldStatus, input.Status, actorUserID, input.Comment, now); err != nil {
		return nil, appErr(500, "internal_error", "failed to update application history", nil)
	}

	title, body := applicationStatusNotification(opportunity.Title, input.Status)
	if repoErr := s.createNotification(ctx, tx, createNotificationInput{
		RecipientUserID: application.ApplicantUserID,
		ActorUserID:     stringPtr(actorUserID),
		Type:            model.NotificationTypeApplicationStatus,
		SourceType:      model.NotificationSourceApplication,
		SourceID:        stringPtr(application.ID),
		CompanyID:       stringPtr(opportunity.CompanyID),
		OpportunityID:   stringPtr(opportunity.ID),
		ApplicationID:   stringPtr(application.ID),
		Title:           title,
		Body:            body,
	}); repoErr != nil {
		return nil, repoErr
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update application", nil)
	}

	return s.getEmployerApplication(actorUserID, application.ID)
}

func (s *Store) getEmployerApplication(actorUserID, applicationID string) (*model.Application, *commonstore.AppError) {
	ctx := context.Background()
	application, err := scanApplication(s.db.QueryRow(ctx, `
		SELECT id, opportunity_id, applicant_user_id, cover_letter, status, applied_at, updated_at
		FROM applications
		WHERE id = $1
	`, applicationID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "application not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load application", nil)
	}

	if _, repoErr := s.getEmployerAccessibleOpportunity(ctx, s.db, actorUserID, application.OpportunityID); repoErr != nil {
		if repoErr.Status == 404 || repoErr.Status == 403 {
			return nil, appErr(403, "forbidden", "current employer has no access to this application", nil)
		}
		return nil, repoErr
	}
	if repoErr := s.hydrateApplication(ctx, s.db, application); repoErr != nil {
		return nil, repoErr
	}
	return application, nil
}

func (s *Store) scanApplicationList(ctx context.Context, rows pgx.Rows) ([]*model.Application, *commonstore.AppError) {
	items := make([]*model.Application, 0)
	for rows.Next() {
		application, err := scanApplication(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load applications", nil)
		}
		items = append(items, application)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load applications", nil)
	}
	for _, application := range items {
		if repoErr := s.hydrateApplication(ctx, s.db, application); repoErr != nil {
			return nil, repoErr
		}
	}
	return items, nil
}

func (s *Store) hydrateApplication(ctx context.Context, q queryable, application *model.Application) *commonstore.AppError {
	opportunity, err := scanOpportunity(q.QueryRow(ctx, `
		SELECT id, company_id, created_by_user_id, title, summary, slug, description,
			type, status, moderation_status, participation_format, location_id,
			contact_email, contact_phone, cover_media_id, published_at, expires_at,
			created_at, updated_at
		FROM opportunities
		WHERE id = $1
	`, application.OpportunityID))
	if err != nil {
		return appErr(500, "internal_error", "failed to load opportunity", nil)
	}
	if repoErr := s.hydrateOpportunity(ctx, q, opportunity); repoErr != nil {
		return repoErr
	}
	application.Opportunity = opportunity

	applicant, err := s.loadApplicantPreview(ctx, q, application.ApplicantUserID)
	if err != nil {
		return appErr(500, "internal_error", "failed to load applicant", nil)
	}
	application.Applicant = applicant

	historyRows, err := q.Query(ctx, `
		SELECT id, application_id, old_status, new_status, changed_by_user_id, comment, created_at
		FROM application_status_history
		WHERE application_id = $1
		ORDER BY created_at ASC, id ASC
	`, application.ID)
	if err != nil {
		return appErr(500, "internal_error", "failed to load application history", nil)
	}
	defer historyRows.Close()

	application.History = []model.ApplicationStatusHistory{}
	for historyRows.Next() {
		item, err := scanApplicationStatusHistory(historyRows)
		if err != nil {
			return appErr(500, "internal_error", "failed to load application history", nil)
		}
		application.History = append(application.History, *item)
	}
	if historyRows.Err() != nil {
		return appErr(500, "internal_error", "failed to load application history", nil)
	}

	return nil
}

func (s *Store) loadApplicantPreview(ctx context.Context, q queryable, userID string) (*model.ApplicantPreview, error) {
	preview, err := scanApplicantPreview(q.QueryRow(ctx, `
		SELECT u.id, u.display_name, u.avatar_media_id, ap.first_name, ap.last_name, ap.middle_name,
			ap.university_name, ap.city, ap.graduation_year
		FROM users u
		JOIN applicant_profiles ap ON ap.user_id = u.id
		WHERE u.id = $1
	`, userID))
	if err != nil {
		return nil, err
	}

	tagRows, err := q.Query(ctx, `
		SELECT t.id, t.name, t.tag_type, t.is_system, t.is_active, t.created_by_user_id, t.created_at
		FROM applicant_tags at
		JOIN tags t ON t.id = at.tag_id
		WHERE at.applicant_user_id = $1
		ORDER BY t.name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()

	preview.Tags = []*model.Tag{}
	for tagRows.Next() {
		tag, err := scanTag(tagRows)
		if err != nil {
			return nil, err
		}
		preview.Tags = append(preview.Tags, tag)
	}
	if tagRows.Err() != nil {
		return nil, tagRows.Err()
	}

	return preview, nil
}

func (s *Store) getEmployerAccessibleOpportunity(ctx context.Context, q queryable, actorUserID, opportunityID string) (*model.Opportunity, *commonstore.AppError) {
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

	if _, repoErr := s.requireApprovedMembership(ctx, q, actorUserID, opportunity.CompanyID); repoErr != nil {
		return nil, appErr(403, "forbidden", "current employer has no access to applications of this opportunity", nil)
	}
	return opportunity, nil
}

func (s *Store) applicantExists(ctx context.Context, q queryable, userID string) bool {
	var exists bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM applicant_profiles WHERE user_id = $1)`, userID).Scan(&exists); err != nil {
		return false
	}
	return exists
}

func isOpportunityAvailableForApplying(opportunity *model.Opportunity, now time.Time) bool {
	if !isPublicOpportunity(opportunity, now) {
		return false
	}
	return opportunity.ExpiresAt == nil || opportunity.ExpiresAt.After(now)
}

func canWithdrawApplicationStatus(status string) bool {
	switch status {
	case model.ApplicationStatusSubmitted, model.ApplicationStatusReviewing, model.ApplicationStatusReserve:
		return true
	default:
		return false
	}
}

func isEmployerSettableApplicationStatus(status string) bool {
	switch status {
	case model.ApplicationStatusReviewing, model.ApplicationStatusReserve, model.ApplicationStatusAccepted, model.ApplicationStatusRejected:
		return true
	default:
		return false
	}
}

func isAllowedEmployerApplicationStatusTransition(from, to string) bool {
	switch from {
	case model.ApplicationStatusSubmitted:
		return to == model.ApplicationStatusReviewing || to == model.ApplicationStatusReserve || to == model.ApplicationStatusRejected
	case model.ApplicationStatusReviewing:
		return to == model.ApplicationStatusReserve || to == model.ApplicationStatusAccepted || to == model.ApplicationStatusRejected
	case model.ApplicationStatusReserve:
		return to == model.ApplicationStatusReviewing || to == model.ApplicationStatusAccepted || to == model.ApplicationStatusRejected
	default:
		return false
	}
}

func scanApplication(src scanner) (*model.Application, error) {
	var application model.Application
	var coverLetter sql.NullString
	if err := src.Scan(
		&application.ID,
		&application.OpportunityID,
		&application.ApplicantUserID,
		&coverLetter,
		&application.Status,
		&application.AppliedAt,
		&application.UpdatedAt,
	); err != nil {
		return nil, err
	}
	application.CoverLetter = nullStringPtr(coverLetter)
	return &application, nil
}

func scanApplicantPreview(src scanner) (*model.ApplicantPreview, error) {
	var preview model.ApplicantPreview
	var avatarMediaID, middleName, universityName, city sql.NullString
	var graduationYear sql.NullInt16
	if err := src.Scan(
		&preview.UserID,
		&preview.DisplayName,
		&avatarMediaID,
		&preview.FirstName,
		&preview.LastName,
		&middleName,
		&universityName,
		&city,
		&graduationYear,
	); err != nil {
		return nil, err
	}
	preview.AvatarMediaID = nullStringPtr(avatarMediaID)
	preview.MiddleName = nullStringPtr(middleName)
	preview.UniversityName = nullStringPtr(universityName)
	preview.City = nullStringPtr(city)
	preview.GraduationYear = nullInt16Ptr(graduationYear)
	return &preview, nil
}

func scanApplicationStatusHistory(src scanner) (*model.ApplicationStatusHistory, error) {
	var item model.ApplicationStatusHistory
	var oldStatus, comment sql.NullString
	if err := src.Scan(
		&item.ID,
		&item.ApplicationID,
		&oldStatus,
		&item.NewStatus,
		&item.ChangedByUserID,
		&comment,
		&item.CreatedAt,
	); err != nil {
		return nil, err
	}
	item.OldStatus = nullStringPtr(oldStatus)
	item.Comment = nullStringPtr(comment)
	return &item, nil
}
