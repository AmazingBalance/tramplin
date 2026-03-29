package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

type createNotificationInput struct {
	RecipientUserID string
	ActorUserID     *string
	Type            string
	SourceType      string
	SourceID        *string
	CompanyID       *string
	OpportunityID   *string
	ApplicationID   *string
	Title           string
	Body            *string
}

func (s *Store) ListNotifications(userID string, input model.ListNotificationsInput) ([]*model.Notification, int, *commonstore.AppError) {
	ctx := context.Background()

	rows, err := s.db.Query(ctx, `
		SELECT id, recipient_user_id, actor_user_id, type, source_type, source_id,
			company_id, opportunity_id, application_id, title, body, is_read, read_at, created_at
		FROM notifications
		WHERE recipient_user_id = $1
			AND ($2 = FALSE OR is_read = FALSE)
		ORDER BY created_at DESC, id DESC
	`, userID, input.UnreadOnly)
	if err != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load notifications", nil)
	}
	defer rows.Close()

	items := make([]*model.Notification, 0)
	for rows.Next() {
		notification, err := scanNotification(rows)
		if err != nil {
			return nil, 0, appErr(500, "internal_error", "failed to load notifications", nil)
		}
		items = append(items, notification)
	}
	if rows.Err() != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load notifications", nil)
	}

	paged, total := paginate(items, input.Page, input.PageSize)
	return paged, total, nil
}

func (s *Store) MarkAllNotificationsRead(userID string) (int, *commonstore.AppError) {
	ctx := context.Background()
	now := time.Now().UTC()

	tag, err := s.db.Exec(ctx, `
		UPDATE notifications
		SET is_read = TRUE, read_at = $1
		WHERE recipient_user_id = $2 AND is_read = FALSE
	`, now, userID)
	if err != nil {
		return 0, appErr(500, "internal_error", "failed to mark notifications as read", nil)
	}
	return int(tag.RowsAffected()), nil
}

func (s *Store) MarkNotificationRead(userID, notificationID string) (*model.Notification, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to mark notification as read", nil)
	}
	defer tx.Rollback(ctx)

	notification, err := scanNotification(tx.QueryRow(ctx, `
		SELECT id, recipient_user_id, actor_user_id, type, source_type, source_id,
			company_id, opportunity_id, application_id, title, body, is_read, read_at, created_at
		FROM notifications
		WHERE id = $1 AND recipient_user_id = $2
		FOR UPDATE
	`, notificationID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "notification not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load notification", nil)
	}

	if !notification.IsRead {
		now := time.Now().UTC()
		notification.IsRead = true
		notification.ReadAt = &now
		if _, err := tx.Exec(ctx, `
			UPDATE notifications
			SET is_read = TRUE, read_at = $1
			WHERE id = $2
		`, now, notification.ID); err != nil {
			return nil, appErr(500, "internal_error", "failed to mark notification as read", nil)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to mark notification as read", nil)
	}
	return notification, nil
}

func (s *Store) createNotification(ctx context.Context, q queryable, input createNotificationInput) *commonstore.AppError {
	if input.RecipientUserID == "" || input.Type == "" || input.SourceType == "" || input.Title == "" {
		return appErr(500, "internal_error", "failed to create notification", nil)
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO notifications (
			recipient_user_id, actor_user_id, type, source_type, source_id, company_id,
			opportunity_id, application_id, title, body, is_read, read_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, FALSE, NULL, $11)
	`, input.RecipientUserID, input.ActorUserID, input.Type, input.SourceType, input.SourceID, input.CompanyID,
		input.OpportunityID, input.ApplicationID, input.Title, input.Body, time.Now().UTC()); err != nil {
		return appErr(500, "internal_error", "failed to create notification", nil)
	}
	return nil
}

func applicationStatusNotification(opportunityTitle, status string) (string, *string) {
	title := "Application status updated"
	body := fmt.Sprintf("Your application for %q is now %s.", opportunityTitle, status)
	return title, &body
}

func scanNotification(src scanner) (*model.Notification, error) {
	var notification model.Notification
	var actorUserID, sourceID, companyID, opportunityID, applicationID, body sql.NullString
	var readAt sql.NullTime
	if err := src.Scan(
		&notification.ID,
		&notification.RecipientUserID,
		&actorUserID,
		&notification.Type,
		&notification.SourceType,
		&sourceID,
		&companyID,
		&opportunityID,
		&applicationID,
		&notification.Title,
		&body,
		&notification.IsRead,
		&readAt,
		&notification.CreatedAt,
	); err != nil {
		return nil, err
	}
	notification.ActorUserID = nullStringPtr(actorUserID)
	notification.SourceID = nullStringPtr(sourceID)
	notification.CompanyID = nullStringPtr(companyID)
	notification.OpportunityID = nullStringPtr(opportunityID)
	notification.ApplicationID = nullStringPtr(applicationID)
	notification.Body = nullStringPtr(body)
	notification.ReadAt = nullTimePtr(readAt)
	return &notification, nil
}
