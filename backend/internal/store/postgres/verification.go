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

type curatorProfileRecord struct {
	UserID    string
	FullName  string
	IsAdmin   bool
	CreatedAt time.Time
}

func (s *Store) ListCompanyVerificationRequests(actorUserID, companyID string) ([]*model.VerificationRequest, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireApprovedMembership(ctx, s.db, actorUserID, companyID); repoErr != nil {
		return nil, repoErr
	}

	rows, err := s.db.Query(ctx, `
		SELECT vr.id, vr.company_id, vr.submitted_by_user_id, vr.method, vr.status,
			c.verification_status, vr.submitted_comment, vr.review_comment,
			vr.reviewed_by_curator_user_id, vr.reviewed_at, vr.created_at
		FROM verification_requests vr
		JOIN companies c ON c.id = vr.company_id
		WHERE vr.company_id = $1
		ORDER BY vr.created_at DESC, vr.id DESC
	`, companyID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load verification requests", nil)
	}
	defer rows.Close()

	items, repoErr := scanVerificationRequests(rows)
	if repoErr != nil {
		return nil, repoErr
	}
	if repoErr := s.hydrateVerificationRequests(ctx, s.db, items); repoErr != nil {
		return nil, repoErr
	}
	return items, nil
}

func (s *Store) CreateCompanyVerificationRequest(actorUserID, companyID string, input model.CreateVerificationRequestInput) (*model.VerificationRequest, *commonstore.AppError) {
	if repoErr := validateCreateVerificationRequestInput(input); repoErr != nil {
		return nil, repoErr
	}
	input.Method = strings.TrimSpace(input.Method)

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create verification request", nil)
	}
	defer tx.Rollback(ctx)

	if _, repoErr := s.requireApprovedMembership(ctx, tx, actorUserID, companyID); repoErr != nil {
		return nil, repoErr
	}

	var companyVerificationStatus string
	if err := tx.QueryRow(ctx, `
		SELECT verification_status::text
		FROM companies
		WHERE id = $1
		FOR UPDATE
	`, companyID).Scan(&companyVerificationStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "company not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load company", nil)
	}
	if companyVerificationStatus == model.CompanyVerificationStatusVerified {
		return nil, appErr(409, "conflict", "company is already verified", nil)
	}

	if repoErr := s.validateVerificationEvidenceFiles(ctx, tx, actorUserID, input.Evidence); repoErr != nil {
		return nil, repoErr
	}

	now := time.Now().UTC()
	var verificationRequestID string
	err = tx.QueryRow(ctx, `
		INSERT INTO verification_requests (
			company_id, submitted_by_user_id, method, status, submitted_comment, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, companyID, actorUserID, input.Method, model.VerificationRequestStatusPending, input.SubmittedComment, now).Scan(&verificationRequestID)
	if err != nil {
		if isUniqueViolation(err, "verification_requests_company_active_key") {
			return nil, appErr(409, "conflict", "company already has an active verification request", nil)
		}
		return nil, appErr(500, "internal_error", "failed to create verification request", nil)
	}

	for _, evidence := range input.Evidence {
		if _, err := tx.Exec(ctx, `
			INSERT INTO verification_evidence (
				verification_request_id, evidence_type, value, evidence_file_id, created_at
			) VALUES ($1, $2, $3, $4, $5)
		`, verificationRequestID, evidence.EvidenceType, evidence.Value, evidence.EvidenceFileID, now); err != nil {
			return nil, appErr(500, "internal_error", "failed to save verification evidence", nil)
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE companies
		SET verification_status = $2,
			verified_at = NULL,
			verified_by_curator_user_id = NULL,
			updated_at = $3
		WHERE id = $1
	`, companyID, model.CompanyVerificationStatusPending, now); err != nil {
		return nil, appErr(500, "internal_error", "failed to update company verification status", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to create verification request", nil)
	}

	request, repoErr := s.getVerificationRequest(ctx, s.db, verificationRequestID, false)
	if repoErr != nil {
		return nil, repoErr
	}
	if repoErr := s.hydrateVerificationRequests(ctx, s.db, []*model.VerificationRequest{request}); repoErr != nil {
		return nil, repoErr
	}
	return request, nil
}

func (s *Store) ListCuratorVerificationRequests(actorUserID string, input model.ListVerificationRequestsInput) ([]*model.VerificationRequest, int, *commonstore.AppError) {
	ctx := context.Background()
	if _, repoErr := s.requireCuratorProfile(ctx, s.db, actorUserID); repoErr != nil {
		return nil, 0, repoErr
	}

	var status any
	if input.Status != "" {
		status = input.Status
	}

	rows, err := s.db.Query(ctx, `
		SELECT vr.id, vr.company_id, vr.submitted_by_user_id, vr.method, vr.status,
			c.verification_status, vr.submitted_comment, vr.review_comment,
			vr.reviewed_by_curator_user_id, vr.reviewed_at, vr.created_at
		FROM verification_requests vr
		JOIN companies c ON c.id = vr.company_id
		WHERE ($1::text IS NULL OR vr.status::text = $1::text)
		ORDER BY vr.created_at DESC, vr.id DESC
	`, status)
	if err != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load verification requests", nil)
	}
	defer rows.Close()

	items, repoErr := scanVerificationRequests(rows)
	if repoErr != nil {
		return nil, 0, repoErr
	}

	paged, total := paginate(items, input.Page, input.PageSize)
	if repoErr := s.hydrateVerificationRequests(ctx, s.db, paged); repoErr != nil {
		return nil, 0, repoErr
	}
	return paged, total, nil
}

func (s *Store) ReviewVerificationRequest(actorUserID, verificationRequestID string, input model.ReviewVerificationRequestInput) (*model.VerificationRequest, *commonstore.AppError) {
	if repoErr := validateReviewVerificationRequestInput(input); repoErr != nil {
		return nil, repoErr
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to review verification request", nil)
	}
	defer tx.Rollback(ctx)

	if _, repoErr := s.requireCuratorProfile(ctx, tx, actorUserID); repoErr != nil {
		return nil, repoErr
	}

	request, repoErr := s.getVerificationRequest(ctx, tx, verificationRequestID, true)
	if repoErr != nil {
		return nil, repoErr
	}
	if request.Status != model.VerificationRequestStatusPending && request.Status != model.VerificationRequestStatusUnderReview {
		return nil, appErr(422, "validation_error", "verification request is already closed", nil)
	}

	now := time.Now().UTC()
	companyVerificationStatus := model.CompanyVerificationStatusPending
	var verifiedAt *time.Time
	var verifiedByCuratorUserID *string

	switch input.Status {
	case model.VerificationRequestStatusUnderReview:
		companyVerificationStatus = model.CompanyVerificationStatusUnderReview
	case model.VerificationRequestStatusApproved:
		companyVerificationStatus = model.CompanyVerificationStatusVerified
		verifiedAt = &now
		verifiedByCuratorUserID = stringPtr(actorUserID)
	case model.VerificationRequestStatusRejected:
		companyVerificationStatus = model.CompanyVerificationStatusRejected
	case model.VerificationRequestStatusNeedsChanges:
		companyVerificationStatus = model.CompanyVerificationStatusPending
	default:
		return nil, appErr(422, "validation_error", "invalid verification request status", nil)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE verification_requests
		SET status = $2,
			review_comment = $3,
			reviewed_by_curator_user_id = $4,
			reviewed_at = $5
		WHERE id = $1
	`, verificationRequestID, input.Status, input.ReviewComment, actorUserID, now); err != nil {
		return nil, appErr(500, "internal_error", "failed to review verification request", nil)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE companies
		SET verification_status = $2,
			verified_at = $3,
			verified_by_curator_user_id = $4,
			updated_at = $5
		WHERE id = $1
	`, request.CompanyID, companyVerificationStatus, verifiedAt, verifiedByCuratorUserID, now); err != nil {
		return nil, appErr(500, "internal_error", "failed to update company verification status", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to review verification request", nil)
	}

	request, repoErr = s.getVerificationRequest(ctx, s.db, verificationRequestID, false)
	if repoErr != nil {
		return nil, repoErr
	}
	if repoErr := s.hydrateVerificationRequests(ctx, s.db, []*model.VerificationRequest{request}); repoErr != nil {
		return nil, repoErr
	}
	return request, nil
}

func (s *Store) requireCuratorProfile(ctx context.Context, q queryable, userID string) (*curatorProfileRecord, *commonstore.AppError) {
	profile, err := scanCuratorProfile(q.QueryRow(ctx, `
		SELECT cp.user_id, cp.full_name, cp.is_admin, cp.created_at
		FROM curator_profiles cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.user_id = $1
			AND u.role = 'curator'
			AND u.is_active = TRUE
	`, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(403, "forbidden", "current user is not a curator", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load curator profile", nil)
	}
	return profile, nil
}

func (s *Store) getVerificationRequest(ctx context.Context, q queryable, verificationRequestID string, forUpdate bool) (*model.VerificationRequest, *commonstore.AppError) {
	query := `
		SELECT vr.id, vr.company_id, vr.submitted_by_user_id, vr.method, vr.status,
			c.verification_status, vr.submitted_comment, vr.review_comment,
			vr.reviewed_by_curator_user_id, vr.reviewed_at, vr.created_at
		FROM verification_requests vr
		JOIN companies c ON c.id = vr.company_id
		WHERE vr.id = $1
	`
	if forUpdate {
		query += ` FOR UPDATE OF vr, c`
	}

	request, err := scanVerificationRequest(q.QueryRow(ctx, query, verificationRequestID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "verification request not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load verification request", nil)
	}
	return request, nil
}

func (s *Store) hydrateVerificationRequests(ctx context.Context, q queryable, items []*model.VerificationRequest) *commonstore.AppError {
	for _, item := range items {
		item.Evidence = []model.VerificationEvidence{}
	}
	if len(items) == 0 {
		return nil
	}

	requestIDs := make([]string, 0, len(items))
	requestsByID := make(map[string]*model.VerificationRequest, len(items))
	for _, item := range items {
		requestIDs = append(requestIDs, item.ID)
		requestsByID[item.ID] = item
	}

	rows, err := q.Query(ctx, `
		SELECT id, verification_request_id, evidence_type, value, evidence_file_id, created_at
		FROM verification_evidence
		WHERE verification_request_id = ANY($1::uuid[])
		ORDER BY verification_request_id ASC, created_at ASC, id ASC
	`, uniqueSortedStrings(requestIDs))
	if err != nil {
		return appErr(500, "internal_error", "failed to load verification evidence", nil)
	}
	defer rows.Close()

	for rows.Next() {
		evidence, err := scanVerificationEvidence(rows)
		if err != nil {
			return appErr(500, "internal_error", "failed to load verification evidence", nil)
		}
		request := requestsByID[evidence.VerificationRequestID]
		if request == nil {
			continue
		}
		request.Evidence = append(request.Evidence, *evidence)
	}
	if rows.Err() != nil {
		return appErr(500, "internal_error", "failed to load verification evidence", nil)
	}
	return nil
}

func (s *Store) validateVerificationEvidenceFiles(ctx context.Context, q queryable, actorUserID string, evidence []model.VerificationEvidenceInput) *commonstore.AppError {
	fileIDs := make([]string, 0)
	for _, item := range evidence {
		if item.EvidenceFileID == nil || strings.TrimSpace(*item.EvidenceFileID) == "" {
			continue
		}
		fileIDs = append(fileIDs, strings.TrimSpace(*item.EvidenceFileID))
	}
	fileIDs = uniqueSortedStrings(fileIDs)
	if len(fileIDs) == 0 {
		return nil
	}

	rows, err := q.Query(ctx, `
		SELECT id
		FROM media_files
		WHERE id = ANY($1::uuid[])
			AND uploaded_by_user_id = $2
			AND status = 'uploaded'
			AND deleted_at IS NULL
	`, fileIDs, actorUserID)
	if err != nil {
		return appErr(500, "internal_error", "failed to validate verification evidence", nil)
	}
	defer rows.Close()

	validIDs := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return appErr(500, "internal_error", "failed to validate verification evidence", nil)
		}
		validIDs[id] = struct{}{}
	}
	if rows.Err() != nil {
		return appErr(500, "internal_error", "failed to validate verification evidence", nil)
	}

	for _, fileID := range fileIDs {
		if _, ok := validIDs[fileID]; !ok {
			return appErr(422, "validation_error", "unknown evidenceFileId", map[string]any{"evidenceFileId": fileID})
		}
	}
	return nil
}

func validateCreateVerificationRequestInput(input model.CreateVerificationRequestInput) *commonstore.AppError {
	if strings.TrimSpace(input.Method) == "" {
		return appErr(422, "validation_error", "method is required", nil)
	}
	if !containsStringValue([]string{
		model.VerificationMethodCorporateEmail,
		model.VerificationMethodINN,
		model.VerificationMethodOfficialSite,
		model.VerificationMethodManualReview,
	}, strings.TrimSpace(input.Method)) {
		return appErr(422, "validation_error", "invalid verification method", nil)
	}

	hasCorporateEmail := false
	hasINNDoc := false
	hasWebsiteLink := false

	for _, evidence := range input.Evidence {
		switch evidence.EvidenceType {
		case model.VerificationEvidenceCorporateEmail:
			if evidence.Value == nil || strings.TrimSpace(*evidence.Value) == "" {
				return appErr(422, "validation_error", "corporate_email evidence requires value", nil)
			}
			hasCorporateEmail = true
		case model.VerificationEvidenceWebsiteLink:
			if evidence.Value == nil || strings.TrimSpace(*evidence.Value) == "" {
				return appErr(422, "validation_error", "website_link evidence requires value", nil)
			}
			hasWebsiteLink = true
		case model.VerificationEvidenceINNDoc, model.VerificationEvidenceRegistryExtract:
			if evidence.EvidenceFileID == nil || strings.TrimSpace(*evidence.EvidenceFileID) == "" {
				return appErr(422, "validation_error", evidence.EvidenceType+" evidence requires evidenceFileId", nil)
			}
			if evidence.EvidenceType == model.VerificationEvidenceINNDoc {
				hasINNDoc = true
			}
		default:
			return appErr(422, "validation_error", "invalid verification evidence type", nil)
		}
	}

	switch input.Method {
	case model.VerificationMethodCorporateEmail:
		if !hasCorporateEmail {
			return appErr(422, "validation_error", "corporate_email evidence is required", nil)
		}
	case model.VerificationMethodINN:
		if !hasINNDoc {
			return appErr(422, "validation_error", "inn_doc evidence is required", nil)
		}
	case model.VerificationMethodOfficialSite:
		if !hasWebsiteLink {
			return appErr(422, "validation_error", "website_link evidence is required", nil)
		}
	}

	return nil
}

func validateReviewVerificationRequestInput(input model.ReviewVerificationRequestInput) *commonstore.AppError {
	if !containsStringValue([]string{
		model.VerificationRequestStatusApproved,
		model.VerificationRequestStatusUnderReview,
		model.VerificationRequestStatusRejected,
		model.VerificationRequestStatusNeedsChanges,
	}, input.Status) {
		return appErr(422, "validation_error", "invalid verification request status", nil)
	}
	if (input.Status == model.VerificationRequestStatusRejected || input.Status == model.VerificationRequestStatusNeedsChanges) &&
		(input.ReviewComment == nil || strings.TrimSpace(*input.ReviewComment) == "") {
		return appErr(422, "validation_error", "reviewComment is required for this review status", nil)
	}
	return nil
}

func containsStringValue(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func scanVerificationRequests(rows pgx.Rows) ([]*model.VerificationRequest, *commonstore.AppError) {
	items := make([]*model.VerificationRequest, 0)
	for rows.Next() {
		request, err := scanVerificationRequest(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load verification requests", nil)
		}
		items = append(items, request)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load verification requests", nil)
	}
	return items, nil
}

func scanCuratorProfile(src scanner) (*curatorProfileRecord, error) {
	var profile curatorProfileRecord
	if err := src.Scan(&profile.UserID, &profile.FullName, &profile.IsAdmin, &profile.CreatedAt); err != nil {
		return nil, err
	}
	return &profile, nil
}

func scanVerificationRequest(src scanner) (*model.VerificationRequest, error) {
	var request model.VerificationRequest
	var submittedComment, reviewComment, reviewedByCuratorUserID sql.NullString
	var reviewedAt sql.NullTime
	if err := src.Scan(
		&request.ID,
		&request.CompanyID,
		&request.SubmittedByUserID,
		&request.Method,
		&request.Status,
		&request.CompanyVerificationStatus,
		&submittedComment,
		&reviewComment,
		&reviewedByCuratorUserID,
		&reviewedAt,
		&request.CreatedAt,
	); err != nil {
		return nil, err
	}
	request.SubmittedComment = nullStringPtr(submittedComment)
	request.ReviewComment = nullStringPtr(reviewComment)
	request.ReviewedByCuratorUserID = nullStringPtr(reviewedByCuratorUserID)
	request.ReviewedAt = nullTimePtr(reviewedAt)
	request.Evidence = []model.VerificationEvidence{}
	return &request, nil
}

func scanVerificationEvidence(src scanner) (*model.VerificationEvidence, error) {
	var evidence model.VerificationEvidence
	var value, evidenceFileID sql.NullString
	if err := src.Scan(
		&evidence.ID,
		&evidence.VerificationRequestID,
		&evidence.EvidenceType,
		&value,
		&evidenceFileID,
		&evidence.CreatedAt,
	); err != nil {
		return nil, err
	}
	evidence.Value = nullStringPtr(value)
	evidence.EvidenceFileID = nullStringPtr(evidenceFileID)
	return &evidence, nil
}
