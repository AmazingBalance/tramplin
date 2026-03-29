package postgres

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

type resolvedNotificationCampaignRecipient struct {
	ApplicantUserID string
	ApplicationID   *string
}

func (s *Store) ListNotificationCampaigns(actorUserID string, input model.ListNotificationCampaignsInput) ([]*model.NotificationCampaign, int, *commonstore.AppError) {
	ctx := context.Background()
	if input.CompanyID != "" {
		if _, repoErr := s.requireApprovedMembership(ctx, s.db, actorUserID, input.CompanyID); repoErr != nil {
			return nil, 0, repoErr
		}
	}

	var companyID any
	if input.CompanyID != "" {
		companyID = input.CompanyID
	}
	var status any
	if input.Status != "" {
		status = input.Status
	}

	rows, err := s.db.Query(ctx, `
		SELECT nc.id, nc.company_id, nc.created_by_user_id, nc.opportunity_id, nc.audience_type,
			nc.status, nc.title, nc.body, nc.send_via_in_app, nc.send_via_email,
			nc.scheduled_at, nc.sent_at, nc.created_at
		FROM notification_campaigns nc
		JOIN company_memberships cm
			ON cm.company_id = nc.company_id
			AND cm.employer_user_id = $1
			AND cm.status = 'approved'
		WHERE ($2::uuid IS NULL OR nc.company_id = $2::uuid)
			AND ($3::text IS NULL OR nc.status::text = $3::text)
		ORDER BY nc.created_at DESC, nc.id DESC
	`, actorUserID, companyID, status)
	if err != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load notification campaigns", nil)
	}
	defer rows.Close()

	items := make([]*model.NotificationCampaign, 0)
	for rows.Next() {
		campaign, err := scanNotificationCampaign(rows)
		if err != nil {
			return nil, 0, appErr(500, "internal_error", "failed to load notification campaigns", nil)
		}
		items = append(items, campaign)
	}
	if rows.Err() != nil {
		return nil, 0, appErr(500, "internal_error", "failed to load notification campaigns", nil)
	}

	paged, total := paginate(items, input.Page, input.PageSize)
	return paged, total, nil
}

func (s *Store) CreateNotificationCampaign(actorUserID string, input model.CreateNotificationCampaignInput) (*model.NotificationCampaign, *commonstore.AppError) {
	if appErr := validateCreateNotificationCampaignInput(input); appErr != nil {
		return nil, appErr
	}

	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create notification campaign", nil)
	}
	defer tx.Rollback(ctx)

	if _, repoErr := s.requireApprovedMembership(ctx, tx, actorUserID, input.CompanyID); repoErr != nil {
		return nil, appErr(403, "forbidden", "current employer has no access to target company or audience", nil)
	}

	recipients, resolvedOpportunityID, repoErr := s.resolveNotificationCampaignAudience(ctx, tx, actorUserID, input.CompanyID, input.AudienceType, input.OpportunityID, input.ApplicantUserIDs)
	if repoErr != nil {
		return nil, repoErr
	}

	status := model.NotificationCampaignStatusDraft
	if input.ScheduledAt != nil {
		status = model.NotificationCampaignStatusScheduled
	}

	now := time.Now().UTC()
	var campaignID string
	err = tx.QueryRow(ctx, `
		INSERT INTO notification_campaigns (
			company_id, created_by_user_id, opportunity_id, audience_type, status,
			title, body, send_via_in_app, send_via_email, scheduled_at, sent_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, NULL, $11
		)
		RETURNING id
	`, input.CompanyID, actorUserID, resolvedOpportunityID, input.AudienceType, status,
		strings.TrimSpace(input.Title), strings.TrimSpace(input.Body), input.SendViaInApp, input.SendViaEmail, input.ScheduledAt, now).Scan(&campaignID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to create notification campaign", nil)
	}

	if repoErr := s.replaceNotificationCampaignRecipients(ctx, tx, campaignID, recipients, now); repoErr != nil {
		return nil, repoErr
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to create notification campaign", nil)
	}

	return s.GetNotificationCampaign(actorUserID, campaignID)
}

func (s *Store) GetNotificationCampaign(actorUserID, campaignID string) (*model.NotificationCampaign, *commonstore.AppError) {
	ctx := context.Background()
	campaign, repoErr := s.getEmployerAccessibleNotificationCampaign(ctx, s.db, actorUserID, campaignID, false)
	if repoErr != nil {
		return nil, repoErr
	}
	if repoErr := s.hydrateNotificationCampaign(ctx, s.db, campaign); repoErr != nil {
		return nil, repoErr
	}
	return campaign, nil
}

func (s *Store) UpdateNotificationCampaign(actorUserID, campaignID string, input model.UpdateNotificationCampaignInput) (*model.NotificationCampaign, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to update notification campaign", nil)
	}
	defer tx.Rollback(ctx)

	campaign, repoErr := s.getEmployerAccessibleNotificationCampaign(ctx, tx, actorUserID, campaignID, true)
	if repoErr != nil {
		return nil, repoErr
	}
	if !isEditableNotificationCampaignStatus(campaign.Status) {
		return nil, appErr(409, "conflict", "notification campaign cannot be edited in current state", nil)
	}

	if input.Title != nil {
		campaign.Title = strings.TrimSpace(*input.Title)
	}
	if input.Body != nil {
		campaign.Body = strings.TrimSpace(*input.Body)
	}
	if campaign.Title == "" || campaign.Body == "" {
		return nil, appErr(422, "validation_error", "required fields are missing", nil)
	}
	if input.SendViaInApp != nil {
		campaign.SendViaInApp = *input.SendViaInApp
	}
	if input.SendViaEmail != nil {
		campaign.SendViaEmail = *input.SendViaEmail
	}
	if !campaign.SendViaInApp && !campaign.SendViaEmail {
		return nil, appErr(422, "validation_error", "at least one delivery channel must be enabled", nil)
	}
	if input.SetScheduledAt {
		campaign.ScheduledAt = input.ScheduledAt
	}

	var recipients []resolvedNotificationCampaignRecipient
	if input.ReplaceAudience {
		if input.AudienceType == nil {
			return nil, appErr(422, "validation_error", "audienceType is required when replacing campaign audience", nil)
		}
		recipients, campaign.OpportunityID, repoErr = s.resolveNotificationCampaignAudience(
			ctx,
			tx,
			actorUserID,
			campaign.CompanyID,
			*input.AudienceType,
			input.OpportunityID,
			input.ApplicantUserIDs,
		)
		if repoErr != nil {
			return nil, repoErr
		}
		campaign.AudienceType = *input.AudienceType
	}

	if campaign.ScheduledAt != nil {
		campaign.Status = model.NotificationCampaignStatusScheduled
	} else {
		campaign.Status = model.NotificationCampaignStatusDraft
	}

	if _, err := tx.Exec(ctx, `
		UPDATE notification_campaigns
		SET opportunity_id = $1,
			audience_type = $2,
			status = $3,
			title = $4,
			body = $5,
			send_via_in_app = $6,
			send_via_email = $7,
			scheduled_at = $8
		WHERE id = $9
	`, campaign.OpportunityID, campaign.AudienceType, campaign.Status, campaign.Title, campaign.Body,
		campaign.SendViaInApp, campaign.SendViaEmail, campaign.ScheduledAt, campaign.ID); err != nil {
		return nil, appErr(500, "internal_error", "failed to update notification campaign", nil)
	}

	if input.ReplaceAudience {
		if repoErr := s.replaceNotificationCampaignRecipients(ctx, tx, campaign.ID, recipients, time.Now().UTC()); repoErr != nil {
			return nil, repoErr
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to update notification campaign", nil)
	}

	return s.GetNotificationCampaign(actorUserID, campaign.ID)
}

func (s *Store) SendNotificationCampaign(actorUserID, campaignID string) (*model.NotificationCampaign, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to send notification campaign", nil)
	}
	defer tx.Rollback(ctx)

	campaign, repoErr := s.getEmployerAccessibleNotificationCampaign(ctx, tx, actorUserID, campaignID, true)
	if repoErr != nil {
		return nil, repoErr
	}
	if !canSendNotificationCampaignStatus(campaign.Status) {
		return nil, appErr(409, "conflict", "notification campaign cannot be sent in current state", nil)
	}

	recipients, repoErr := s.listNotificationCampaignRecipients(ctx, tx, campaign.ID)
	if repoErr != nil {
		return nil, repoErr
	}
	preferences, repoErr := s.loadNotificationCampaignPreferences(ctx, tx, recipients)
	if repoErr != nil {
		return nil, repoErr
	}

	now := time.Now().UTC()
	if campaign.SendViaInApp {
		for _, recipient := range recipients {
			preference := preferences[recipient.ApplicantUserID]
			if !preference.allowInApp {
				continue
			}
			if repoErr := s.createNotification(ctx, tx, createNotificationInput{
				RecipientUserID: recipient.ApplicantUserID,
				ActorUserID:     stringPtr(actorUserID),
				Type:            model.NotificationTypeEmployerBroadcast,
				SourceType:      model.NotificationSourceCampaign,
				SourceID:        stringPtr(campaign.ID),
				CompanyID:       stringPtr(campaign.CompanyID),
				OpportunityID:   campaign.OpportunityID,
				ApplicationID:   recipient.ApplicationID,
				Title:           campaign.Title,
				Body:            stringPtr(campaign.Body),
			}); repoErr != nil {
				return nil, repoErr
			}
		}
	}

	campaign.Status = model.NotificationCampaignStatusSent
	campaign.SentAt = &now
	if _, err := tx.Exec(ctx, `
		UPDATE notification_campaigns
		SET status = 'sent', sent_at = $1
		WHERE id = $2
	`, now, campaign.ID); err != nil {
		return nil, appErr(500, "internal_error", "failed to send notification campaign", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to send notification campaign", nil)
	}

	return s.GetNotificationCampaign(actorUserID, campaign.ID)
}

func (s *Store) CancelNotificationCampaign(actorUserID, campaignID string) (*model.NotificationCampaign, *commonstore.AppError) {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to cancel notification campaign", nil)
	}
	defer tx.Rollback(ctx)

	campaign, repoErr := s.getEmployerAccessibleNotificationCampaign(ctx, tx, actorUserID, campaignID, true)
	if repoErr != nil {
		return nil, repoErr
	}
	if !canCancelNotificationCampaignStatus(campaign.Status) {
		return nil, appErr(409, "conflict", "notification campaign cannot be cancelled in current state", nil)
	}

	campaign.Status = model.NotificationCampaignStatusCancelled
	if _, err := tx.Exec(ctx, `
		UPDATE notification_campaigns
		SET status = 'cancelled'
		WHERE id = $1
	`, campaign.ID); err != nil {
		return nil, appErr(500, "internal_error", "failed to cancel notification campaign", nil)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appErr(500, "internal_error", "failed to cancel notification campaign", nil)
	}

	return s.GetNotificationCampaign(actorUserID, campaign.ID)
}

func (s *Store) getEmployerAccessibleNotificationCampaign(ctx context.Context, q queryable, actorUserID, campaignID string, forUpdate bool) (*model.NotificationCampaign, *commonstore.AppError) {
	query := `
		SELECT id, company_id, created_by_user_id, opportunity_id, audience_type,
			status, title, body, send_via_in_app, send_via_email,
			scheduled_at, sent_at, created_at
		FROM notification_campaigns
		WHERE id = $1
	`
	if forUpdate {
		query += " FOR UPDATE"
	}

	campaign, err := scanNotificationCampaign(q.QueryRow(ctx, query, campaignID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "notification campaign not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load notification campaign", nil)
	}

	if _, repoErr := s.requireApprovedMembership(ctx, q, actorUserID, campaign.CompanyID); repoErr != nil {
		return nil, appErr(403, "forbidden", "current employer has no access to this campaign", nil)
	}

	return campaign, nil
}

func (s *Store) hydrateNotificationCampaign(ctx context.Context, q queryable, campaign *model.NotificationCampaign) *commonstore.AppError {
	recipients, repoErr := s.listNotificationCampaignRecipients(ctx, q, campaign.ID)
	if repoErr != nil {
		return repoErr
	}
	campaign.Recipients = make([]model.NotificationCampaignRecipient, 0, len(recipients))
	for _, recipient := range recipients {
		campaign.Recipients = append(campaign.Recipients, *recipient)
	}
	return nil
}

func (s *Store) replaceNotificationCampaignRecipients(ctx context.Context, q queryable, campaignID string, recipients []resolvedNotificationCampaignRecipient, createdAt time.Time) *commonstore.AppError {
	if _, err := q.Exec(ctx, `DELETE FROM notification_campaign_recipients WHERE campaign_id = $1`, campaignID); err != nil {
		return appErr(500, "internal_error", "failed to update notification campaign recipients", nil)
	}
	for _, recipient := range recipients {
		if _, err := q.Exec(ctx, `
			INSERT INTO notification_campaign_recipients (
				campaign_id, applicant_user_id, application_id, created_at
			) VALUES ($1, $2, $3, $4)
		`, campaignID, recipient.ApplicantUserID, recipient.ApplicationID, createdAt); err != nil {
			return appErr(500, "internal_error", "failed to update notification campaign recipients", nil)
		}
	}
	return nil
}

func (s *Store) listNotificationCampaignRecipients(ctx context.Context, q queryable, campaignID string) ([]*model.NotificationCampaignRecipient, *commonstore.AppError) {
	rows, err := q.Query(ctx, `
		SELECT campaign_id, applicant_user_id, application_id, created_at
		FROM notification_campaign_recipients
		WHERE campaign_id = $1
		ORDER BY created_at ASC, applicant_user_id ASC
	`, campaignID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load notification campaign recipients", nil)
	}
	defer rows.Close()

	items := make([]*model.NotificationCampaignRecipient, 0)
	for rows.Next() {
		recipient, err := scanNotificationCampaignRecipient(rows)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to load notification campaign recipients", nil)
		}
		items = append(items, recipient)
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load notification campaign recipients", nil)
	}
	return items, nil
}

func (s *Store) resolveNotificationCampaignAudience(ctx context.Context, q queryable, actorUserID, companyID, audienceType string, opportunityID *string, applicantUserIDs []string) ([]resolvedNotificationCampaignRecipient, *string, *commonstore.AppError) {
	switch audienceType {
	case model.NotificationCampaignAudienceSingleApplicant:
		if opportunityID != nil && strings.TrimSpace(*opportunityID) != "" {
			return nil, nil, appErr(422, "validation_error", "opportunityId is not allowed for single_applicant audience", nil)
		}
		uniqueIDs := uniqueStringsPreserveOrder(applicantUserIDs)
		if len(uniqueIDs) != 1 {
			return nil, nil, appErr(422, "validation_error", "single_applicant audience requires exactly one applicantUserId", nil)
		}
		recipients, repoErr := s.loadCompanyAudienceRecipients(ctx, q, companyID, uniqueIDs)
		if repoErr != nil {
			return nil, nil, repoErr
		}
		return recipients, nil, nil
	case model.NotificationCampaignAudienceSelectedApplicants:
		if opportunityID != nil && strings.TrimSpace(*opportunityID) != "" {
			return nil, nil, appErr(422, "validation_error", "opportunityId is not allowed for selected_applicants audience", nil)
		}
		uniqueIDs := uniqueStringsPreserveOrder(applicantUserIDs)
		if len(uniqueIDs) == 0 {
			return nil, nil, appErr(422, "validation_error", "selected_applicants audience requires applicantUserIds", nil)
		}
		recipients, repoErr := s.loadCompanyAudienceRecipients(ctx, q, companyID, uniqueIDs)
		if repoErr != nil {
			return nil, nil, repoErr
		}
		return recipients, nil, nil
	case model.NotificationCampaignAudienceAllApplicantsOpportunity:
		if opportunityID == nil || strings.TrimSpace(*opportunityID) == "" {
			return nil, nil, appErr(422, "validation_error", "opportunityId is required for all_applicants_of_opportunity audience", nil)
		}
		if len(applicantUserIDs) > 0 {
			return nil, nil, appErr(422, "validation_error", "applicantUserIds are not allowed for all_applicants_of_opportunity audience", nil)
		}
		opportunity, repoErr := s.getEmployerAccessibleOpportunity(ctx, q, actorUserID, strings.TrimSpace(*opportunityID))
		if repoErr != nil {
			if repoErr.Status == 403 || repoErr.Status == 404 {
				return nil, nil, appErr(403, "forbidden", "current employer has no access to target company or audience", nil)
			}
			return nil, nil, repoErr
		}
		if opportunity.CompanyID != companyID {
			return nil, nil, appErr(403, "forbidden", "current employer has no access to target company or audience", nil)
		}
		recipients, repoErr := s.loadOpportunityAudienceRecipients(ctx, q, opportunity.ID)
		if repoErr != nil {
			return nil, nil, repoErr
		}
		return recipients, stringPtr(opportunity.ID), nil
	default:
		return nil, nil, appErr(422, "validation_error", "invalid audienceType", nil)
	}
}

func (s *Store) loadCompanyAudienceRecipients(ctx context.Context, q queryable, companyID string, applicantUserIDs []string) ([]resolvedNotificationCampaignRecipient, *commonstore.AppError) {
	rows, err := q.Query(ctx, `
		SELECT DISTINCT ON (a.applicant_user_id)
			a.applicant_user_id, a.id
		FROM applications a
		JOIN opportunities o ON o.id = a.opportunity_id
		WHERE o.company_id = $1
			AND a.applicant_user_id = ANY($2::uuid[])
		ORDER BY a.applicant_user_id, a.applied_at DESC, a.id DESC
	`, companyID, applicantUserIDs)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to resolve notification campaign audience", nil)
	}
	defer rows.Close()

	found := make(map[string]resolvedNotificationCampaignRecipient, len(applicantUserIDs))
	for rows.Next() {
		var applicantUserID, applicationID string
		if err := rows.Scan(&applicantUserID, &applicationID); err != nil {
			return nil, appErr(500, "internal_error", "failed to resolve notification campaign audience", nil)
		}
		found[applicantUserID] = resolvedNotificationCampaignRecipient{
			ApplicantUserID: applicantUserID,
			ApplicationID:   stringPtr(applicationID),
		}
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to resolve notification campaign audience", nil)
	}

	recipients := make([]resolvedNotificationCampaignRecipient, 0, len(applicantUserIDs))
	for _, applicantUserID := range applicantUserIDs {
		recipient, ok := found[applicantUserID]
		if !ok {
			return nil, appErr(403, "forbidden", "current employer has no access to target company or audience", nil)
		}
		recipients = append(recipients, recipient)
	}
	return recipients, nil
}

func (s *Store) loadOpportunityAudienceRecipients(ctx context.Context, q queryable, opportunityID string) ([]resolvedNotificationCampaignRecipient, *commonstore.AppError) {
	rows, err := q.Query(ctx, `
		SELECT DISTINCT ON (applicant_user_id)
			applicant_user_id, id
		FROM applications
		WHERE opportunity_id = $1
		ORDER BY applicant_user_id, applied_at DESC, id DESC
	`, opportunityID)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to resolve notification campaign audience", nil)
	}
	defer rows.Close()

	recipients := make([]resolvedNotificationCampaignRecipient, 0)
	for rows.Next() {
		var applicantUserID, applicationID string
		if err := rows.Scan(&applicantUserID, &applicationID); err != nil {
			return nil, appErr(500, "internal_error", "failed to resolve notification campaign audience", nil)
		}
		recipients = append(recipients, resolvedNotificationCampaignRecipient{
			ApplicantUserID: applicantUserID,
			ApplicationID:   stringPtr(applicationID),
		})
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to resolve notification campaign audience", nil)
	}
	return recipients, nil
}

type notificationCampaignPreference struct {
	allowInApp bool
}

func (s *Store) loadNotificationCampaignPreferences(ctx context.Context, q queryable, recipients []*model.NotificationCampaignRecipient) (map[string]notificationCampaignPreference, *commonstore.AppError) {
	userIDs := make([]string, 0, len(recipients))
	seen := map[string]struct{}{}
	for _, recipient := range recipients {
		if _, ok := seen[recipient.ApplicantUserID]; ok {
			continue
		}
		seen[recipient.ApplicantUserID] = struct{}{}
		userIDs = append(userIDs, recipient.ApplicantUserID)
	}
	sort.Strings(userIDs)
	if len(userIDs) == 0 {
		return map[string]notificationCampaignPreference{}, nil
	}

	rows, err := q.Query(ctx, `
		SELECT user_id, in_app_enabled, employer_messages_enabled
		FROM notification_preferences
		WHERE user_id = ANY($1::uuid[])
	`, userIDs)
	if err != nil {
		return nil, appErr(500, "internal_error", "failed to load notification preferences", nil)
	}
	defer rows.Close()

	preferences := make(map[string]notificationCampaignPreference, len(userIDs))
	for _, userID := range userIDs {
		preferences[userID] = notificationCampaignPreference{allowInApp: true}
	}
	for rows.Next() {
		var userID string
		var inAppEnabled, employerMessagesEnabled bool
		if err := rows.Scan(&userID, &inAppEnabled, &employerMessagesEnabled); err != nil {
			return nil, appErr(500, "internal_error", "failed to load notification preferences", nil)
		}
		preferences[userID] = notificationCampaignPreference{
			allowInApp: inAppEnabled && employerMessagesEnabled,
		}
	}
	if rows.Err() != nil {
		return nil, appErr(500, "internal_error", "failed to load notification preferences", nil)
	}
	return preferences, nil
}

func validateCreateNotificationCampaignInput(input model.CreateNotificationCampaignInput) *commonstore.AppError {
	if strings.TrimSpace(input.CompanyID) == "" || strings.TrimSpace(input.AudienceType) == "" || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Body) == "" {
		return appErr(422, "validation_error", "required fields are missing", nil)
	}
	if !input.SendViaInApp && !input.SendViaEmail {
		return appErr(422, "validation_error", "at least one delivery channel must be enabled", nil)
	}
	return nil
}

func isEditableNotificationCampaignStatus(status string) bool {
	return status == model.NotificationCampaignStatusDraft || status == model.NotificationCampaignStatusScheduled
}

func canSendNotificationCampaignStatus(status string) bool {
	return status == model.NotificationCampaignStatusDraft || status == model.NotificationCampaignStatusScheduled
}

func canCancelNotificationCampaignStatus(status string) bool {
	return status == model.NotificationCampaignStatusDraft || status == model.NotificationCampaignStatusScheduled
}

func uniqueStringsPreserveOrder(items []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func scanNotificationCampaign(src scanner) (*model.NotificationCampaign, error) {
	var campaign model.NotificationCampaign
	var opportunityID sql.NullString
	var scheduledAt, sentAt sql.NullTime
	if err := src.Scan(
		&campaign.ID,
		&campaign.CompanyID,
		&campaign.CreatedByUserID,
		&opportunityID,
		&campaign.AudienceType,
		&campaign.Status,
		&campaign.Title,
		&campaign.Body,
		&campaign.SendViaInApp,
		&campaign.SendViaEmail,
		&scheduledAt,
		&sentAt,
		&campaign.CreatedAt,
	); err != nil {
		return nil, err
	}
	campaign.OpportunityID = nullStringPtr(opportunityID)
	campaign.ScheduledAt = nullTimePtr(scheduledAt)
	campaign.SentAt = nullTimePtr(sentAt)
	return &campaign, nil
}

func scanNotificationCampaignRecipient(src scanner) (*model.NotificationCampaignRecipient, error) {
	var recipient model.NotificationCampaignRecipient
	var applicationID sql.NullString
	if err := src.Scan(&recipient.CampaignID, &recipient.ApplicantUserID, &applicationID, &recipient.CreatedAt); err != nil {
		return nil, err
	}
	recipient.ApplicationID = nullStringPtr(applicationID)
	return &recipient, nil
}
