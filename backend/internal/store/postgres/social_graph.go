package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

type connectionRow struct {
	ID                  string
	InitiatorUserID     string
	ApplicantLowUserID  string
	ApplicantHighUserID string
	Status              string
	InitiatorNote       *string
	RespondedAt         *time.Time
	CreatedAt           time.Time
}

func (s *Store) ListConnections(applicantUserID string, input model.ListConnectionsInput) ([]*model.Connection, *commonstore.AppError) {
	ctx := context.Background()
	if !s.applicantExists(ctx, s.db, applicantUserID) {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}

	var status any
	if input.Status != "" {
		status = input.Status
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, initiator_user_id, applicant_low_user_id, applicant_high_user_id,
			status, initiator_note, responded_at, created_at
		FROM applicant_connections
		WHERE ($1 = applicant_low_user_id OR $1 = applicant_high_user_id)
			AND ($2::text IS NULL OR status::text = $2::text)
		ORDER BY created_at DESC, id DESC
	`, applicantUserID, status)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load connections", nil)
	}
	defer rows.Close()

	items := make([]*model.Connection, 0)
	for rows.Next() {
		record, err := scanConnectionRow(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load connections", nil)
		}
		connection := connectionFromRow(record, applicantUserID)
		applicant, err := s.loadApplicantPreview(ctx, s.db, connection.OtherApplicantID)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load connections", nil)
		}
		connection.OtherApplicant = applicant
		items = append(items, connection)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load connections", nil)
	}
	return items, nil
}

func (s *Store) CreateConnection(applicantUserID string, input model.CreateConnectionInput) (*model.Connection, *commonstore.AppError) {
	if strings.TrimSpace(input.TargetApplicantUserID) == "" {
		return nil, appErr(422, "validation_error", "required fields are missing", nil)
	}

	targetApplicantUserID := strings.TrimSpace(input.TargetApplicantUserID)
	if applicantUserID == targetApplicantUserID {
		return nil, appErr(403, "forbidden", "self-connection is forbidden", nil)
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create connection", nil)
	}
	defer tx.Rollback(ctx)

	if !s.applicantExists(ctx, tx, applicantUserID) || !s.applicantExists(ctx, tx, targetApplicantUserID) {
		return nil, appErr(403, "forbidden", "current user cannot create connections", nil)
	}

	lowUserID, highUserID := normalizeUserPair(applicantUserID, targetApplicantUserID)
	record, err := scanConnectionRow(tx.QueryRow(ctx, `
		INSERT INTO applicant_connections (
			applicant_low_user_id, applicant_high_user_id, initiator_user_id, status, initiator_note, created_at
		) VALUES ($1, $2, $3, 'pending', $4, $5)
		RETURNING id, initiator_user_id, applicant_low_user_id, applicant_high_user_id,
			status, initiator_note, responded_at, created_at
	`, lowUserID, highUserID, applicantUserID, input.InitiatorNote, time.Now().UTC()))
	if err != nil {
		if isUniqueViolation(err, "applicant_connections_unique_pair") {
			return nil, appErr(409, "conflict", "conflict or duplicate entity", nil)
		}
		return nil, appErr(500, "internal_error", "failed to create connection", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to create connection", nil)
	}

	connection := connectionFromRow(record, applicantUserID)
	otherApplicant, err := s.loadApplicantPreview(ctx, s.db, connection.OtherApplicantID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load connection", nil)
	}
	connection.OtherApplicant = otherApplicant
	return connection, nil
}

func (s *Store) UpdateConnection(applicantUserID, connectionID string, input model.UpdateConnectionInput) (*model.Connection, *commonstore.AppError) {
	if !isConnectionResponseStatus(input.Status) {
		return nil, appErr(422, "validation_error", "invalid connection status", nil)
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update connection", nil)
	}
	defer tx.Rollback(ctx)

	if !s.applicantExists(ctx, tx, applicantUserID) {
		return nil, appErr(403, "forbidden", "only participants of the connection can update it", nil)
	}

	record, err := scanConnectionRow(tx.QueryRow(ctx, `
		SELECT id, initiator_user_id, applicant_low_user_id, applicant_high_user_id,
			status, initiator_note, responded_at, created_at
		FROM applicant_connections
		WHERE id = $1
		FOR UPDATE
	`, connectionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "connection not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load connection", nil)
	}
	if applicantUserID != record.ApplicantLowUserID && applicantUserID != record.ApplicantHighUserID {
		return nil, appErr(403, "forbidden", "only participants of the connection can update it", nil)
	}
	if record.Status != model.ConnectionStatusPending {
		return nil, appErr(409, "conflict", "requested transition is not allowed", nil)
	}

	now := time.Now().UTC()
	record.Status = input.Status
	record.RespondedAt = &now
	if _, err := tx.Exec(ctx, `
		UPDATE applicant_connections
		SET status = $1, responded_at = $2
		WHERE id = $3
	`, input.Status, now, record.ID); err != nil {
		return nil, appErr(500, "internal_error", "failed to update connection", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update connection", nil)
	}

	connection := connectionFromRow(record, applicantUserID)
	otherApplicant, err := s.loadApplicantPreview(ctx, s.db, connection.OtherApplicantID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load connection", nil)
	}
	connection.OtherApplicant = otherApplicant
	return connection, nil
}

func (s *Store) CreateOpportunityRecommendation(applicantUserID string, input model.CreateOpportunityRecommendationInput) (*model.OpportunityRecommendation, *commonstore.AppError) {
	if strings.TrimSpace(input.RecipientUserID) == "" || strings.TrimSpace(input.OpportunityID) == "" {
		return nil, appErr(422, "validation_error", "required fields are missing", nil)
	}

	recipientUserID := strings.TrimSpace(input.RecipientUserID)
	opportunityID := strings.TrimSpace(input.OpportunityID)
	if applicantUserID == recipientUserID {
		return nil, appErr(422, "validation_error", "recipientUserId must be different from current applicant", nil)
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create recommendation", nil)
	}
	defer tx.Rollback(ctx)

	if !s.applicantExists(ctx, tx, applicantUserID) || !s.applicantExists(ctx, tx, recipientUserID) {
		return nil, appErr(422, "validation_error", "unknown applicant user id", nil)
	}

	opportunity, err := scanOpportunity(tx.QueryRow(ctx, `
		SELECT id, company_id, created_by_user_id, title, summary, slug, description,
			type, status, moderation_status, participation_format, location_id,
			contact_email, contact_phone, cover_media_id, published_at, expires_at,
			created_at, updated_at
		FROM opportunities
		WHERE id = $1
	`, opportunityID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(422, "validation_error", "unknown opportunity id", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load opportunity", nil)
	}
	if !isPublicOpportunity(opportunity, time.Now().UTC()) {
		return nil, appErr(422, "validation_error", "opportunity is not available for recommendation", nil)
	}

	var allowRecommendations bool
	if err := tx.QueryRow(ctx, `
		SELECT allow_recommendations
		FROM applicant_privacy_settings
		WHERE applicant_user_id = $1
	`, recipientUserID).Scan(&allowRecommendations); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(422, "validation_error", "unknown applicant user id", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load recipient settings", nil)
	}
	if !allowRecommendations {
		return nil, appErr(403, "forbidden", "recipient does not allow recommendations", nil)
	}

	recommendation, err := scanOpportunityRecommendation(tx.QueryRow(ctx, `
		INSERT INTO opportunity_recommendations (
			recommender_user_id, recipient_user_id, opportunity_id, message, created_at
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, recommender_user_id, recipient_user_id, opportunity_id, message, created_at
	`, applicantUserID, recipientUserID, opportunityID, input.Message, time.Now().UTC()))
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create recommendation", nil)
	}

	recommenderName, err := s.loadUserDisplayName(ctx, tx, applicantUserID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load recommendation context", nil)
	}
	title, body := recommendationNotification(opportunity.Title, recommenderName, input.Message)
	if repoErr := s.createNotification(ctx, tx, createNotificationInput{
		RecipientUserID: recipientUserID,
		ActorUserID:     stringPtr(applicantUserID),
		Type:            model.NotificationTypeRecommendationReceived,
		SourceType:      model.NotificationSourceRecommendation,
		SourceID:        stringPtr(recommendation.ID),
		CompanyID:       stringPtr(opportunity.CompanyID),
		OpportunityID:   stringPtr(opportunity.ID),
		Title:           title,
		Body:            body,
	}); repoErr != nil {
		return nil, repoErr
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to create recommendation", nil)
	}
	return recommendation, nil
}

func (s *Store) ListReceivedRecommendations(applicantUserID string) ([]*model.OpportunityRecommendation, *commonstore.AppError) {
	ctx := context.Background()
	if !s.applicantExists(ctx, s.db, applicantUserID) {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}
	return s.listRecommendations(ctx, s.db, `
		SELECT id, recommender_user_id, recipient_user_id, opportunity_id, message, created_at
		FROM opportunity_recommendations
		WHERE recipient_user_id = $1
		ORDER BY created_at DESC, id DESC
	`, applicantUserID)
}

func (s *Store) ListSentRecommendations(applicantUserID string) ([]*model.OpportunityRecommendation, *commonstore.AppError) {
	ctx := context.Background()
	if !s.applicantExists(ctx, s.db, applicantUserID) {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}
	return s.listRecommendations(ctx, s.db, `
		SELECT id, recommender_user_id, recipient_user_id, opportunity_id, message, created_at
		FROM opportunity_recommendations
		WHERE recommender_user_id = $1
		ORDER BY created_at DESC, id DESC
	`, applicantUserID)
}

func (s *Store) listRecommendations(ctx context.Context, q queryable, sqlQuery, applicantUserID string) ([]*model.OpportunityRecommendation, *commonstore.AppError) {
	rows, err := q.Query(ctx, sqlQuery, applicantUserID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load recommendations", nil)
	}
	defer rows.Close()

	items := make([]*model.OpportunityRecommendation, 0)
	for rows.Next() {
		recommendation, err := scanOpportunityRecommendation(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load recommendations", nil)
		}
		items = append(items, recommendation)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load recommendations", nil)
	}
	return items, nil
}

func (s *Store) loadUserDisplayName(ctx context.Context, q queryable, userID string) (string, error) {
	var displayName string
	err := q.QueryRow(ctx, `SELECT display_name FROM users WHERE id = $1`, userID).Scan(&displayName)
	return displayName, err
}

func normalizeUserPair(a, b string) (string, string) {
	if a <= b {
		return a, b
	}
	return b, a
}

func isConnectionResponseStatus(status string) bool {
	switch status {
	case model.ConnectionStatusAccepted, model.ConnectionStatusRejected, model.ConnectionStatusBlocked:
		return true
	default:
		return false
	}
}

func connectionFromRow(record *connectionRow, viewerUserID string) *model.Connection {
	otherApplicantID := record.ApplicantLowUserID
	if otherApplicantID == viewerUserID {
		otherApplicantID = record.ApplicantHighUserID
	}
	return &model.Connection{
		ID:               record.ID,
		InitiatorUserID:  record.InitiatorUserID,
		OtherApplicantID: otherApplicantID,
		Status:           record.Status,
		InitiatorNote:    record.InitiatorNote,
		RespondedAt:      record.RespondedAt,
		CreatedAt:        record.CreatedAt,
	}
}

func recommendationNotification(opportunityTitle, recommenderDisplayName string, message *string) (string, *string) {
	title := "New opportunity recommendation"
	text := fmt.Sprintf("%s recommended %q to you.", recommenderDisplayName, opportunityTitle)
	if message != nil && strings.TrimSpace(*message) != "" {
		text += " Message: " + strings.TrimSpace(*message)
	}
	return title, &text
}

func scanConnectionRow(src scanner) (*connectionRow, error) {
	var row connectionRow
	var initiatorNote sql.NullString
	var respondedAt sql.NullTime
	if err := src.Scan(
		&row.ID,
		&row.InitiatorUserID,
		&row.ApplicantLowUserID,
		&row.ApplicantHighUserID,
		&row.Status,
		&initiatorNote,
		&respondedAt,
		&row.CreatedAt,
	); err != nil {
		return nil, err
	}
	row.InitiatorNote = nullStringPtr(initiatorNote)
	row.RespondedAt = nullTimePtr(respondedAt)
	return &row, nil
}

func scanOpportunityRecommendation(src scanner) (*model.OpportunityRecommendation, error) {
	var recommendation model.OpportunityRecommendation
	var message sql.NullString
	if err := src.Scan(
		&recommendation.ID,
		&recommendation.RecommenderUserID,
		&recommendation.RecipientUserID,
		&recommendation.OpportunityID,
		&message,
		&recommendation.CreatedAt,
	); err != nil {
		return nil, err
	}
	recommendation.Message = nullStringPtr(message)
	return &recommendation, nil
}
