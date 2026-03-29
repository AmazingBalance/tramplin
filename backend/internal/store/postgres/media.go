package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

func (s *Store) CreatePresignedUpload(userID string, input model.CreatePresignedUploadInput) (*model.MediaFile, *commonstore.AppError) {
	if strings.TrimSpace(input.OriginalName) == "" || strings.TrimSpace(input.MimeType) == "" || input.FileSize <= 0 {
		return nil, appErr(422, "validation_error", "required fields are missing", nil)
	}
	if input.Purpose != nil && !isValidMediaPurpose(*input.Purpose) {
		return nil, appErr(422, "validation_error", "invalid media purpose", nil)
	}

	ctx := context.Background()
	now := time.Now().UTC()
	mediaID := uuid.NewString()
	objectKey := "media/" + userID + "/" + mediaID

	file, err := scanMediaFile(s.db.QueryRow(ctx, `
		INSERT INTO media_files (
			id, file_key, original_name, mime_type, file_size, uploaded_by_user_id,
			purpose, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8)
		RETURNING id, file_key, original_name, mime_type, file_size, uploaded_by_user_id,
			purpose, status, etag, completed_at, deleted_at, created_at
	`, mediaID, objectKey, strings.TrimSpace(input.OriginalName), strings.TrimSpace(input.MimeType), input.FileSize, userID, input.Purpose, now))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create upload record", nil)
	}
	return file, nil
}

func (s *Store) CompleteUpload(userID, mediaFileID string, input model.CompleteUploadInput, objectETag *string) (*model.MediaFile, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to finalize upload", nil)
	}
	defer tx.Rollback(ctx)

	file, err := scanMediaFile(tx.QueryRow(ctx, `
		SELECT id, file_key, original_name, mime_type, file_size, uploaded_by_user_id,
			purpose, status, etag, completed_at, deleted_at, created_at
		FROM media_files
		WHERE id = $1
		FOR UPDATE
	`, mediaFileID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "media file not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to finalize upload", nil)
	}
	if file.DeletedAt != nil || file.Status == model.MediaFileStatusDeleted {
		return nil, appErr(404, "not_found", "media file not found", nil)
	}
	if file.UploadedByUserID == nil || *file.UploadedByUserID != userID {
		return nil, appErr(403, "forbidden", "not enough permissions", nil)
	}

	expectedETag := normalizeETag(input.ETag)
	actualETag := normalizeETag(objectETag)

	if file.Status == model.MediaFileStatusUploaded {
		storedETag := normalizeETag(file.ETag)
		if expectedETag != nil && storedETag != nil && *expectedETag != *storedETag {
			return nil, appErr(409, "conflict", "file was not uploaded or cannot be finalized", nil)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, appErr(500, "internal_error", "failed to finalize upload", nil)
		}
		return file, nil
	}

	if actualETag == nil {
		return nil, appErr(409, "conflict", "file was not uploaded or cannot be finalized", nil)
	}
	if expectedETag != nil && *expectedETag != *actualETag {
		return nil, appErr(409, "conflict", "file was not uploaded or cannot be finalized", nil)
	}

	now := time.Now().UTC()
	file.Status = model.MediaFileStatusUploaded
	file.ETag = actualETag
	file.CompletedAt = &now

	_, err = tx.Exec(ctx, `
		UPDATE media_files
		SET status = 'uploaded', etag = $1, completed_at = $2
		WHERE id = $3
	`, file.ETag, file.CompletedAt, file.ID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to finalize upload", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to finalize upload", nil)
	}
	return file, nil
}

func (s *Store) GetMediaFile(actorUserID, mediaFileID string) (*model.MediaFile, *commonstore.AppError) {
	ctx := context.Background()
	file, err := s.loadMediaFile(ctx, s.db, mediaFileID)
	if err != nil {
		return nil, err
	}

	allowed, accessErr := s.hasMediaReadAccess(ctx, s.db, mediaFileID, actorUserID)
	if accessErr != nil {
		return nil, accessErr
	}
	if !allowed {
		return nil, appErr(403, "forbidden", "not enough permissions", nil)
	}
	return file, nil
}

func (s *Store) GetMediaFileForDownload(actorUserID *string, mediaFileID string) (*model.MediaFile, *commonstore.AppError) {
	ctx := context.Background()
	file, err := s.loadMediaFile(ctx, s.db, mediaFileID)
	if err != nil {
		return nil, err
	}
	if file.Status != model.MediaFileStatusUploaded {
		return nil, appErr(404, "not_found", "media file not found", nil)
	}

	if actorUserID != nil {
		allowed, accessErr := s.hasMediaReadAccess(ctx, s.db, mediaFileID, *actorUserID)
		if accessErr != nil {
			return nil, accessErr
		}
		if !allowed {
			return nil, appErr(403, "forbidden", "not enough permissions", nil)
		}
		return file, nil
	}

	publicAllowed, accessErr := s.hasAnonymousMediaDownloadAccess(ctx, s.db, mediaFileID)
	if accessErr != nil {
		return nil, accessErr
	}
	if !publicAllowed {
		return nil, appErr(401, "unauthorized", "authentication failed", nil)
	}
	return file, nil
}

func (s *Store) DeleteMediaFile(actorUserID, mediaFileID string) (*model.MediaFile, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to delete media file", nil)
	}
	defer tx.Rollback(ctx)

	file, err := scanMediaFile(tx.QueryRow(ctx, `
		SELECT id, file_key, original_name, mime_type, file_size, uploaded_by_user_id,
			purpose, status, etag, completed_at, deleted_at, created_at
		FROM media_files
		WHERE id = $1
		FOR UPDATE
	`, mediaFileID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "media file not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to delete media file", nil)
	}
	if file.DeletedAt != nil || file.Status == model.MediaFileStatusDeleted {
		return nil, appErr(404, "not_found", "media file not found", nil)
	}
	if file.UploadedByUserID == nil || *file.UploadedByUserID != actorUserID {
		return nil, appErr(403, "forbidden", "not enough permissions", nil)
	}

	attached, accessErr := s.isMediaAttached(ctx, tx, mediaFileID)
	if accessErr != nil {
		return nil, accessErr
	}
	if attached {
		return nil, appErr(409, "conflict", "file is still attached to an entity that forbids deletion", nil)
	}

	now := time.Now().UTC()
	file.Status = model.MediaFileStatusDeleted
	file.DeletedAt = &now

	_, err = tx.Exec(ctx, `
		UPDATE media_files
		SET status = 'deleted', deleted_at = $1
		WHERE id = $2
	`, now, mediaFileID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to delete media file", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to delete media file", nil)
	}
	return file, nil
}

func (s *Store) loadMediaFile(ctx context.Context, q queryable, mediaFileID string) (*model.MediaFile, *commonstore.AppError) {
	file, err := scanMediaFile(q.QueryRow(ctx, `
		SELECT id, file_key, original_name, mime_type, file_size, uploaded_by_user_id,
			purpose, status, etag, completed_at, deleted_at, created_at
		FROM media_files
		WHERE id = $1
	`, mediaFileID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "media file not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load media file", nil)
	}
	if file.DeletedAt != nil || file.Status == model.MediaFileStatusDeleted {
		return nil, appErr(404, "not_found", "media file not found", nil)
	}
	return file, nil
}

func (s *Store) hasMediaReadAccess(ctx context.Context, q queryable, mediaFileID, actorUserID string) (bool, *commonstore.AppError) {
	owned, err := s.exists(ctx, q, `
		SELECT 1
		FROM media_files
		WHERE id = $1 AND uploaded_by_user_id = $2
	`, mediaFileID, actorUserID)
	if err != nil {
		return false, appErr(500, "internal_error", "failed to load media access", nil)
	}
	if owned {
		return true, nil
	}

	profileOwned, err := s.exists(ctx, q, `
		SELECT 1
		FROM users
		WHERE id = $2 AND avatar_media_id = $1
		UNION ALL
		SELECT 1
		FROM applicant_profiles
		WHERE user_id = $2 AND resume_media_id = $1
		LIMIT 1
	`, mediaFileID, actorUserID)
	if err != nil {
		return false, appErr(500, "internal_error", "failed to load media access", nil)
	}
	if profileOwned {
		return true, nil
	}

	employerAccess, err := s.exists(ctx, q, `
		SELECT 1
		FROM company_memberships cm
		JOIN companies c ON c.id = cm.company_id
		WHERE cm.employer_user_id = $2
			AND cm.status = 'approved'
			AND (
				c.logo_media_id = $1
				OR c.banner_media_id = $1
				OR EXISTS (
					SELECT 1
					FROM company_media company_media_entry
					WHERE company_media_entry.company_id = c.id
						AND company_media_entry.media_file_id = $1
				)
				OR EXISTS (
					SELECT 1
					FROM opportunities o
					WHERE o.company_id = c.id
						AND (
							o.cover_media_id = $1
							OR EXISTS (
								SELECT 1
								FROM opportunity_media om
								WHERE om.opportunity_id = o.id
									AND om.media_file_id = $1
							)
						)
				)
			)
		LIMIT 1
	`, mediaFileID, actorUserID)
	if err != nil {
		return false, appErr(500, "internal_error", "failed to load media access", nil)
	}
	if employerAccess {
		return true, nil
	}

	publicAccess, accessErr := s.hasAnonymousMediaDownloadAccess(ctx, q, mediaFileID)
	if accessErr != nil {
		return false, accessErr
	}
	return publicAccess, nil
}

func (s *Store) hasAnonymousMediaDownloadAccess(ctx context.Context, q queryable, mediaFileID string) (bool, *commonstore.AppError) {
	publicCompany, err := s.exists(ctx, q, `
		SELECT 1
		FROM companies
		WHERE logo_media_id = $1 OR banner_media_id = $1
		UNION ALL
		SELECT 1
		FROM company_media
		WHERE media_file_id = $1
		LIMIT 1
	`, mediaFileID)
	if err != nil {
		return false, appErr(500, "internal_error", "failed to load media access", nil)
	}
	if publicCompany {
		return true, nil
	}

	publicOpportunity, err := s.exists(ctx, q, `
		SELECT 1
		FROM opportunities o
		WHERE (
			o.cover_media_id = $1
			OR EXISTS (
				SELECT 1
				FROM opportunity_media om
				WHERE om.opportunity_id = o.id
					AND om.media_file_id = $1
			)
		)
			AND o.moderation_status = 'approved'
			AND o.status IN ('planned', 'active')
			AND (o.published_at IS NULL OR o.published_at <= $2)
		LIMIT 1
	`, mediaFileID, time.Now().UTC())
	if err != nil {
		return false, appErr(500, "internal_error", "failed to load media access", nil)
	}
	return publicOpportunity, nil
}

func (s *Store) isMediaAttached(ctx context.Context, q queryable, mediaFileID string) (bool, *commonstore.AppError) {
	attached, err := s.exists(ctx, q, `
		SELECT 1 FROM users WHERE avatar_media_id = $1
		UNION ALL
		SELECT 1 FROM applicant_profiles WHERE resume_media_id = $1
		UNION ALL
		SELECT 1 FROM companies WHERE logo_media_id = $1 OR banner_media_id = $1
		UNION ALL
		SELECT 1 FROM company_media WHERE media_file_id = $1
		UNION ALL
		SELECT 1 FROM opportunities WHERE cover_media_id = $1
		UNION ALL
		SELECT 1 FROM opportunity_media WHERE media_file_id = $1
		LIMIT 1
	`, mediaFileID)
	if err != nil {
		return false, appErr(500, "internal_error", "failed to validate media attachments", nil)
	}
	return attached, nil
}

func (s *Store) exists(ctx context.Context, q queryable, query string, args ...any) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `SELECT EXISTS (`+query+`)`, args...).Scan(&exists)
	return exists, err
}

func scanMediaFile(src scanner) (*model.MediaFile, error) {
	var file model.MediaFile
	var originalName, mimeType, uploadedByUserID, purpose, etag sql.NullString
	var fileSize sql.NullInt64
	var completedAt, deletedAt sql.NullTime
	if err := src.Scan(
		&file.ID,
		&file.ObjectKey,
		&originalName,
		&mimeType,
		&fileSize,
		&uploadedByUserID,
		&purpose,
		&file.Status,
		&etag,
		&completedAt,
		&deletedAt,
		&file.CreatedAt,
	); err != nil {
		return nil, err
	}
	file.OriginalName = nullStringPtr(originalName)
	file.MimeType = nullStringPtr(mimeType)
	file.FileSize = nullInt64Ptr(fileSize)
	file.UploadedByUserID = nullStringPtr(uploadedByUserID)
	file.Purpose = nullStringPtr(purpose)
	file.ETag = nullStringPtr(etag)
	file.CompletedAt = nullTimePtr(completedAt)
	file.DeletedAt = nullTimePtr(deletedAt)
	return &file, nil
}

func isValidMediaPurpose(value string) bool {
	switch value {
	case model.MediaFilePurposeAvatar,
		model.MediaFilePurposeResume,
		model.MediaFilePurposeCompanyLogo,
		model.MediaFilePurposeCompanyBanner,
		model.MediaFilePurposeCompanyMedia,
		model.MediaFilePurposeOpportunityCover,
		model.MediaFilePurposeOpportunityMedia,
		model.MediaFilePurposeVerificationEvidence,
		model.MediaFilePurposeOther:
		return true
	default:
		return false
	}
}

func normalizeETag(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	normalized = strings.Trim(normalized, `"`)
	if normalized == "" {
		return nil
	}
	return &normalized
}
