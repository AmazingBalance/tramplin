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

func (s *Store) CreateEmployerTag(actorUserID string, input model.CreateTagInput) (*model.Tag, *commonstore.AppError) {
	ctx := context.Background()
	if !s.employerExists(ctx, s.db, actorUserID) {
		return nil, appErr(403, "forbidden", "current user is not an employer", nil)
	}
	if repoErr := validateCreateTagInput(input); repoErr != nil {
		return nil, repoErr
	}
	if input.TagType != model.TagTypeCustom {
		return nil, appErr(403, "forbidden", "requested tag type is not allowed for employers", nil)
	}
	return s.createTag(ctx, s.db, actorUserID, model.CreateTagInput{
		Name:     input.Name,
		TagType:  input.TagType,
		IsActive: true,
	})
}

func (s *Store) ListCuratorTags(actorUserID string, input model.ListCuratorTagsInput) ([]*model.Tag, int, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireCuratorProfile(ctx, s.db, actorUserID); repoErr != nil {
		return nil, 0, repoErr
	}

	var tagType any
	if input.Type != "" {
		tagType = input.Type
	}
	var isActive any
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, name, tag_type, is_system, is_active, created_by_user_id, created_at
		FROM tags
		WHERE ($1::text IS NULL OR tag_type::text = $1::text)
			AND ($2::boolean IS NULL OR is_active = $2::boolean)
		ORDER BY name ASC, created_at DESC, id ASC
	`, tagType, isActive)
	if err != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load tags", nil)
	}
	defer rows.Close()

	items := make([]*model.Tag, 0)
	for rows.Next() {
		tag, err := scanTag(rows)
		if err != nil {
			return nil, 0, appErr(500, "internal_error", "failed to load tags", nil)
		}
		items = append(items, tag)
	}
	if rows.Err() != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load tags", nil)
	}

	paged, total := paginate(items, input.Page, input.PageSize)
	return paged, total, nil
}

func (s *Store) CreateCuratorTag(actorUserID string, input model.CreateTagInput) (*model.Tag, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireCuratorProfile(ctx, s.db, actorUserID); repoErr != nil {
		return nil, repoErr
	}
	if repoErr := validateCreateTagInput(input); repoErr != nil {
		return nil, repoErr
	}
	return s.createTag(ctx, s.db, actorUserID, input)
}

func (s *Store) UpdateCuratorTag(actorUserID, tagID string, input model.UpdateTagInput) (*model.Tag, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireCuratorProfile(ctx, s.db, actorUserID); repoErr != nil {
		return nil, repoErr
	}
	if input.Name == nil && input.IsActive == nil {
		return nil, appErr(422, "validation_error", "at least one field is required", nil)
	}

	tag, err := scanTag(s.db.QueryRow(ctx, `
		SELECT id, name, tag_type, is_system, is_active, created_by_user_id, created_at
		FROM tags
		WHERE id = $1
	`, tagID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "tag not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load tag", nil)
	}

	if input.Name != nil {
		tag.Name = strings.TrimSpace(*input.Name)
	}
	if input.IsActive != nil {
		tag.IsActive = *input.IsActive
	}
	if strings.TrimSpace(tag.Name) == "" {
		return nil, appErr(422, "validation_error", "name is required", nil)
	}

	updated, err := scanTag(s.db.QueryRow(ctx, `
		UPDATE tags
		SET name = $2,
			is_active = $3
		WHERE id = $1
		RETURNING id, name, tag_type, is_system, is_active, created_by_user_id, created_at
	`, tagID, tag.Name, tag.IsActive))
	if err != nil {
		if isUniqueViolation(err, "tags_name_tag_type_key") {
			return nil, appErr(409, "conflict", "conflict or duplicate entity", nil)
		}
		return nil, appErr(500, "internal_error", "failed to update tag", nil)
	}
	return updated, nil
}

func (s *Store) ListModerationCases(actorUserID string, input model.ListModerationCasesInput) ([]*model.ModerationCase, int, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireCuratorProfile(ctx, s.db, actorUserID); repoErr != nil {
		return nil, 0, repoErr
	}

	var status any
	if input.Status != "" {
		status = input.Status
	}
	var targetType any
	if input.TargetType != "" {
		targetType = input.TargetType
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, target_type, target_id, submitted_by_user_id, assigned_curator_user_id,
			resolved_by_curator_user_id, status, reason, created_at, resolved_at
		FROM moderation_cases
		WHERE ($1::text IS NULL OR status::text = $1::text)
			AND ($2::text IS NULL OR target_type::text = $2::text)
		ORDER BY created_at DESC, id DESC
	`, status, targetType)
	if err != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load moderation cases", nil)
	}
	defer rows.Close()

	items, repoErr := scanModerationCases(rows)
	if repoErr != nil {
		return nil, 0, repoErr
	}
	paged, total := paginate(items, input.Page, input.PageSize)
	if repoErr := s.hydrateModerationCasePreviews(ctx, s.db, paged); repoErr != nil {
		return nil, 0, repoErr
	}
	return paged, total, nil
}

func (s *Store) CreateModerationCase(actorUserID string, input model.CreateModerationCaseInput) (*model.ModerationCase, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireCuratorProfile(ctx, s.db, actorUserID); repoErr != nil {
		return nil, repoErr
	}
	if repoErr := validateCreateModerationCaseInput(input); repoErr != nil {
		return nil, repoErr
	}
	if repoErr := s.ensureModerationTargetExists(ctx, s.db, input.TargetType, input.TargetID); repoErr != nil {
		return nil, repoErr
	}

	now := time.Now().UTC()
	item, err := scanModerationCase(s.db.QueryRow(ctx, `
		INSERT INTO moderation_cases (
			target_type, target_id, submitted_by_user_id, status, reason, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, target_type, target_id, submitted_by_user_id, assigned_curator_user_id,
			resolved_by_curator_user_id, status, reason, created_at, resolved_at
	`, input.TargetType, input.TargetID, actorUserID, model.ModerationStatusPending, strings.TrimSpace(input.Reason), now))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create moderation case", nil)
	}
	if repoErr := s.hydrateModerationCasePreviews(ctx, s.db, []*model.ModerationCase{item}); repoErr != nil {
		return nil, repoErr
	}
	return item, nil
}

func (s *Store) UpdateModerationCase(actorUserID, moderationCaseID string, input model.UpdateModerationCaseInput) (*model.ModerationCase, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update moderation case", nil)
	}
	defer tx.Rollback(ctx)

	if _, repoErr := s.requireCuratorProfile(ctx, tx, actorUserID); repoErr != nil {
		return nil, repoErr
	}
	if repoErr := validateUpdateModerationCaseInput(input); repoErr != nil {
		return nil, repoErr
	}

	item, err := scanModerationCase(tx.QueryRow(ctx, `
		SELECT id, target_type, target_id, submitted_by_user_id, assigned_curator_user_id,
			resolved_by_curator_user_id, status, reason, created_at, resolved_at
		FROM moderation_cases
		WHERE id = $1
		FOR UPDATE
	`, moderationCaseID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "moderation case not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load moderation case", nil)
	}

	if input.SetAssignedCuratorUserID {
		if input.AssignedCuratorUserID != nil && strings.TrimSpace(*input.AssignedCuratorUserID) != "" {
			if _, repoErr := s.requireCuratorProfile(ctx, tx, *input.AssignedCuratorUserID); repoErr != nil {
				if repoErr.Code == "forbidden" {
					return nil, appErr(422, "validation_error", "assignedCuratorUserId must reference an existing curator", nil)
				}
				return nil, repoErr
			}
			item.AssignedCuratorUserID = input.AssignedCuratorUserID
		} else {
			item.AssignedCuratorUserID = nil
		}
	}

	item.Reason = stringPtr(strings.TrimSpace(input.Reason))

	if input.Status != nil {
		item.Status = *input.Status
		switch item.Status {
		case model.ModerationStatusPending:
			item.ResolvedByCuratorUserID = nil
			item.ResolvedAt = nil
		case model.ModerationStatusApproved, model.ModerationStatusRejected, model.ModerationStatusNeedsChanges:
			now := time.Now().UTC()
			item.ResolvedByCuratorUserID = stringPtr(actorUserID)
			item.ResolvedAt = &now
		}
		if repoErr := s.applyModerationOutcomeToTarget(ctx, tx, item.TargetType, item.TargetID, item.Status); repoErr != nil {
			return nil, repoErr
		}
	}

	updated, err := scanModerationCase(tx.QueryRow(ctx, `
		UPDATE moderation_cases
		SET assigned_curator_user_id = $2,
			resolved_by_curator_user_id = $3,
			status = $4,
			reason = $5,
			resolved_at = $6
		WHERE id = $1
		RETURNING id, target_type, target_id, submitted_by_user_id, assigned_curator_user_id,
			resolved_by_curator_user_id, status, reason, created_at, resolved_at
	`, moderationCaseID, item.AssignedCuratorUserID, item.ResolvedByCuratorUserID, item.Status, item.Reason, item.ResolvedAt))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update moderation case", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update moderation case", nil)
	}
	if repoErr := s.hydrateModerationCasePreviews(ctx, s.db, []*model.ModerationCase{updated}); repoErr != nil {
		return nil, repoErr
	}
	return updated, nil
}

func (s *Store) createTag(ctx context.Context, q queryable, actorUserID string, input model.CreateTagInput) (*model.Tag, *commonstore.AppError) {
	now := time.Now().UTC()
	tag, err := scanTag(q.QueryRow(ctx, `
		INSERT INTO tags (name, tag_type, is_system, is_active, created_by_user_id, created_at)
		VALUES ($1, $2, FALSE, $3, $4, $5)
		RETURNING id, name, tag_type, is_system, is_active, created_by_user_id, created_at
	`, strings.TrimSpace(input.Name), input.TagType, input.IsActive, actorUserID, now))
	if err != nil {
		if isUniqueViolation(err, "tags_name_tag_type_key") {
			return nil, appErr(409, "conflict", "conflict or duplicate entity", nil)
		}
		return nil, appErr(500, "internal_error", "failed to create tag", nil)
	}
	return tag, nil
}

func (s *Store) ensureModerationTargetExists(ctx context.Context, q queryable, targetType, targetID string) *commonstore.AppError {
	query := ""
	switch targetType {
	case model.ModerationTargetCompany:
		query = `SELECT EXISTS (SELECT 1 FROM companies WHERE id = $1)`
	case model.ModerationTargetProfile:
		query = `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`
	case model.ModerationTargetOpportunity:
		query = `SELECT EXISTS (SELECT 1 FROM opportunities WHERE id = $1)`
	case model.ModerationTargetTag:
		query = `SELECT EXISTS (SELECT 1 FROM tags WHERE id = $1)`
	case model.ModerationTargetMedia:
		query = `SELECT EXISTS (SELECT 1 FROM media_files WHERE id = $1)`
	case model.ModerationTargetVerification:
		query = `SELECT EXISTS (SELECT 1 FROM verification_requests WHERE id = $1)`
	default:
		return appErr(422, "validation_error", "invalid moderation target type", nil)
	}

	var exists bool
	if err := q.QueryRow(ctx, query, targetID).Scan(&exists); err != nil {
		return appErr(500, "internal_error", "failed to validate moderation target", nil)
	}
	if !exists {
		return appErr(422, "validation_error", "unknown moderation target", nil)
	}
	return nil
}

func (s *Store) applyModerationOutcomeToTarget(ctx context.Context, q queryable, targetType, targetID, status string) *commonstore.AppError {
	if targetType != model.ModerationTargetOpportunity {
		return nil
	}
	if _, err := q.Exec(ctx, `
		UPDATE opportunities
		SET moderation_status = $2,
			updated_at = $3
		WHERE id = $1
	`, targetID, status, time.Now().UTC()); err != nil {
		return appErr(500, "internal_error", "failed to update moderated target", nil)
	}
	return nil
}

func (s *Store) hydrateModerationCasePreviews(ctx context.Context, q queryable, items []*model.ModerationCase) *commonstore.AppError {
	for _, item := range items {
		preview, repoErr := s.loadModerationTargetPreview(ctx, q, item.TargetType, item.TargetID)
		if repoErr != nil {
			return repoErr
		}
		item.TargetPreview = preview
	}
	return nil
}

func (s *Store) loadModerationTargetPreview(ctx context.Context, q queryable, targetType, targetID string) (*model.ModerationTargetPreview, *commonstore.AppError) {
	preview := &model.ModerationTargetPreview{
		Type: targetType,
		ID:   targetID,
	}

	switch targetType {
	case model.ModerationTargetCompany:
		var title string
		var subtitle, slug sql.NullString
		err := q.QueryRow(ctx, `
			SELECT legal_name, brand_name, slug
			FROM companies
			WHERE id = $1
		`, targetID).Scan(&title, &subtitle, &slug)
		if err == nil {
			preview.Title = stringPtr(title)
			preview.Subtitle = nullStringPtr(subtitle)
			preview.Slug = nullStringPtr(slug)
		}
	case model.ModerationTargetProfile:
		var title string
		var subtitle sql.NullString
		err := q.QueryRow(ctx, `
			SELECT display_name, email
			FROM users
			WHERE id = $1
		`, targetID).Scan(&title, &subtitle)
		if err == nil {
			preview.Title = stringPtr(title)
			preview.Subtitle = nullStringPtr(subtitle)
		}
	case model.ModerationTargetOpportunity:
		var title string
		var subtitle, slug sql.NullString
		err := q.QueryRow(ctx, `
			SELECT title, summary, slug
			FROM opportunities
			WHERE id = $1
		`, targetID).Scan(&title, &subtitle, &slug)
		if err == nil {
			preview.Title = stringPtr(title)
			preview.Subtitle = nullStringPtr(subtitle)
			preview.Slug = nullStringPtr(slug)
		}
	case model.ModerationTargetTag:
		var title string
		var subtitle sql.NullString
		err := q.QueryRow(ctx, `
			SELECT name, tag_type::text
			FROM tags
			WHERE id = $1
		`, targetID).Scan(&title, &subtitle)
		if err == nil {
			preview.Title = stringPtr(title)
			preview.Subtitle = nullStringPtr(subtitle)
		}
	case model.ModerationTargetMedia:
		var title, subtitle sql.NullString
		err := q.QueryRow(ctx, `
			SELECT original_name, mime_type
			FROM media_files
			WHERE id = $1
		`, targetID).Scan(&title, &subtitle)
		if err == nil {
			preview.Title = nullStringPtr(title)
			preview.Subtitle = nullStringPtr(subtitle)
		}
	case model.ModerationTargetVerification:
		var title string
		var subtitle sql.NullString
		err := q.QueryRow(ctx, `
			SELECT c.legal_name, vr.method::text || ':' || vr.status::text
			FROM verification_requests vr
			JOIN companies c ON c.id = vr.company_id
			WHERE vr.id = $1
		`, targetID).Scan(&title, &subtitle)
		if err == nil {
			preview.Title = stringPtr(title)
			preview.Subtitle = nullStringPtr(subtitle)
		}
	}
	return preview, nil
}

func validateCreateTagInput(input model.CreateTagInput) *commonstore.AppError {
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.TagType) == "" {
		return appErr(422, "validation_error", "required fields are missing", nil)
	}
	if !isValidTagType(input.TagType) {
		return appErr(422, "validation_error", "invalid tag type", nil)
	}
	return nil
}

func validateCreateModerationCaseInput(input model.CreateModerationCaseInput) *commonstore.AppError {
	if strings.TrimSpace(input.TargetType) == "" || strings.TrimSpace(input.TargetID) == "" || strings.TrimSpace(input.Reason) == "" {
		return appErr(422, "validation_error", "required fields are missing", nil)
	}
	if !isValidModerationTargetType(input.TargetType) {
		return appErr(422, "validation_error", "invalid moderation target type", nil)
	}
	return nil
}

func validateUpdateModerationCaseInput(input model.UpdateModerationCaseInput) *commonstore.AppError {
	if strings.TrimSpace(input.Reason) == "" {
		return appErr(422, "validation_error", "reason is required", nil)
	}
	if input.Status != nil && !isValidModerationStatus(*input.Status) {
		return appErr(422, "validation_error", "invalid moderation status", nil)
	}
	return nil
}

func isValidTagType(value string) bool {
	switch value {
	case model.TagTypeTechnology, model.TagTypeRole, model.TagTypeDomain, model.TagTypeLevel, model.TagTypeEmploymentType, model.TagTypeFormat, model.TagTypeCustom:
		return true
	default:
		return false
	}
}

func isValidModerationTargetType(value string) bool {
	switch value {
	case model.ModerationTargetCompany, model.ModerationTargetProfile, model.ModerationTargetOpportunity, model.ModerationTargetTag, model.ModerationTargetMedia, model.ModerationTargetVerification:
		return true
	default:
		return false
	}
}

func isValidModerationStatus(value string) bool {
	switch value {
	case model.ModerationStatusPending, model.ModerationStatusApproved, model.ModerationStatusRejected, model.ModerationStatusNeedsChanges:
		return true
	default:
		return false
	}
}

func scanModerationCases(rows pgx.Rows) ([]*model.ModerationCase, *commonstore.AppError) {
	items := make([]*model.ModerationCase, 0)
	for rows.Next() {
		item, err := scanModerationCase(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load moderation cases", nil)
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load moderation cases", nil)
	}
	return items, nil
}

func scanModerationCase(src scanner) (*model.ModerationCase, error) {
	var item model.ModerationCase
	var submittedBy, assignedCurator, resolvedBy, reason sql.NullString
	var resolvedAt sql.NullTime
	if err := src.Scan(
		&item.ID,
		&item.TargetType,
		&item.TargetID,
		&submittedBy,
		&assignedCurator,
		&resolvedBy,
		&item.Status,
		&reason,
		&item.CreatedAt,
		&resolvedAt,
	); err != nil {
		return nil, err
	}
	item.SubmittedByUserID = nullStringPtr(submittedBy)
	item.AssignedCuratorUserID = nullStringPtr(assignedCurator)
	item.ResolvedByCuratorUserID = nullStringPtr(resolvedBy)
	item.Reason = nullStringPtr(reason)
	item.ResolvedAt = nullTimePtr(resolvedAt)
	return &item, nil
}
