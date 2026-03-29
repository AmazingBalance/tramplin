package http

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

func parsePaginationParams(r *http.Request) (int, int, *commonstore.AppError) {
	page, appErr := parsePositiveIntQuery(r, "page", 1, 1, 0)
	if appErr != nil {
		return 0, 0, appErr
	}
	pageSize, appErr := parsePositiveIntQuery(r, "pageSize", 20, 1, 100)
	if appErr != nil {
		return 0, 0, appErr
	}
	return page, pageSize, nil
}

func parsePositiveIntQuery(r *http.Request, name string, fallback, min, max int) (int, *commonstore.AppError) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < min || (max > 0 && value > max) {
		return 0, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid " + name + " parameter",
		}
	}
	return value, nil
}

func parseRequiredEmail(value, field string) (string, *commonstore.AppError) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "required fields are missing",
		}
	}
	if !isValidEmail(trimmed) {
		return "", &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid " + field,
		}
	}
	return trimmed, nil
}

func parseOptionalEmail(value *string, field string) (*string, *commonstore.AppError) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" || !isValidEmail(trimmed) {
		return nil, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid " + field,
		}
	}
	return &trimmed, nil
}

func parseRequiredURI(value, field string) (string, *commonstore.AppError) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "required fields are missing",
		}
	}
	if !isValidAbsoluteURI(trimmed) {
		return "", &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid " + field,
		}
	}
	return trimmed, nil
}

func parseOptionalURI(value *string, field string) (*string, *commonstore.AppError) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" || !isValidAbsoluteURI(trimmed) {
		return nil, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid " + field,
		}
	}
	return &trimmed, nil
}

func isValidEmail(value string) bool {
	if strings.ContainsAny(value, " \t\r\n") {
		return false
	}
	addr, err := mail.ParseAddress(value)
	return err == nil && addr.Address == value
}

func isValidAbsoluteURI(value string) bool {
	if strings.ContainsAny(value, " \t\r\n") {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() {
		return false
	}
	return parsed.Host != "" || parsed.Opaque != "" || parsed.Path != ""
}

func buildListEmployerOpportunitiesInput(r *http.Request) (model.ListEmployerOpportunitiesInput, *commonstore.AppError) {
	page, pageSize, appErr := parsePaginationParams(r)
	if appErr != nil {
		return model.ListEmployerOpportunitiesInput{}, appErr
	}
	input := model.ListEmployerOpportunitiesInput{
		CompanyID:        r.URL.Query().Get("companyId"),
		Status:           r.URL.Query().Get("status"),
		ModerationStatus: r.URL.Query().Get("moderationStatus"),
		Page:             page,
		PageSize:         pageSize,
	}
	if input.Status != "" && !containsString([]string{
		model.OpportunityStatusDraft,
		model.OpportunityStatusPlanned,
		model.OpportunityStatusActive,
		model.OpportunityStatusClosed,
		model.OpportunityStatusRejected,
		model.OpportunityStatusArchived,
	}, input.Status) {
		return model.ListEmployerOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid opportunity status"}
	}
	if input.ModerationStatus != "" && !containsString([]string{
		model.ModerationStatusPending,
		model.ModerationStatusApproved,
		model.ModerationStatusRejected,
		model.ModerationStatusNeedsChanges,
	}, input.ModerationStatus) {
		return model.ListEmployerOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid moderation status"}
	}
	return input, nil
}

func buildListApplicationsInput(r *http.Request) (model.ListApplicationsInput, *commonstore.AppError) {
	page, pageSize, appErr := parsePaginationParams(r)
	if appErr != nil {
		return model.ListApplicationsInput{}, appErr
	}
	input := model.ListApplicationsInput{
		Status:   r.URL.Query().Get("status"),
		Page:     page,
		PageSize: pageSize,
	}
	if input.Status != "" && !containsString([]string{
		model.ApplicationStatusSubmitted,
		model.ApplicationStatusReviewing,
		model.ApplicationStatusReserve,
		model.ApplicationStatusAccepted,
		model.ApplicationStatusRejected,
		model.ApplicationStatusWithdrawn,
	}, input.Status) {
		return model.ListApplicationsInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid application status",
		}
	}
	return input, nil
}

func buildListConnectionsInput(r *http.Request) (model.ListConnectionsInput, *commonstore.AppError) {
	input := model.ListConnectionsInput{
		Status: r.URL.Query().Get("status"),
	}
	if input.Status != "" && !containsString([]string{
		model.ConnectionStatusPending,
		model.ConnectionStatusAccepted,
		model.ConnectionStatusRejected,
		model.ConnectionStatusBlocked,
	}, input.Status) {
		return model.ListConnectionsInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid connection status",
		}
	}
	return input, nil
}

func buildListNotificationsInput(r *http.Request) (model.ListNotificationsInput, *commonstore.AppError) {
	page, pageSize, appErr := parsePaginationParams(r)
	if appErr != nil {
		return model.ListNotificationsInput{}, appErr
	}
	input := model.ListNotificationsInput{
		Page:     page,
		PageSize: pageSize,
	}
	if raw := r.URL.Query().Get("unreadOnly"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return model.ListNotificationsInput{}, &commonstore.AppError{
				Status:  http.StatusUnprocessableEntity,
				Code:    "validation_error",
				Message: "invalid unreadOnly parameter",
			}
		}
		input.UnreadOnly = value
	}
	return input, nil
}

func buildListNotificationCampaignsInput(r *http.Request) (model.ListNotificationCampaignsInput, *commonstore.AppError) {
	page, pageSize, appErr := parsePaginationParams(r)
	if appErr != nil {
		return model.ListNotificationCampaignsInput{}, appErr
	}
	input := model.ListNotificationCampaignsInput{
		CompanyID: r.URL.Query().Get("companyId"),
		Status:    r.URL.Query().Get("status"),
		Page:      page,
		PageSize:  pageSize,
	}
	if input.Status != "" && !containsString([]string{
		model.NotificationCampaignStatusDraft,
		model.NotificationCampaignStatusScheduled,
		model.NotificationCampaignStatusProcessing,
		model.NotificationCampaignStatusSent,
		model.NotificationCampaignStatusCancelled,
	}, input.Status) {
		return model.ListNotificationCampaignsInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid notification campaign status",
		}
	}
	return input, nil
}

func buildListVerificationRequestsInput(r *http.Request) (model.ListVerificationRequestsInput, *commonstore.AppError) {
	page, pageSize, appErr := parsePaginationParams(r)
	if appErr != nil {
		return model.ListVerificationRequestsInput{}, appErr
	}
	input := model.ListVerificationRequestsInput{
		Status:   r.URL.Query().Get("status"),
		Page:     page,
		PageSize: pageSize,
	}
	if input.Status != "" && !containsString([]string{
		model.VerificationRequestStatusPending,
		model.VerificationRequestStatusUnderReview,
		model.VerificationRequestStatusApproved,
		model.VerificationRequestStatusRejected,
		model.VerificationRequestStatusNeedsChanges,
	}, input.Status) {
		return model.ListVerificationRequestsInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid verification request status",
		}
	}
	return input, nil
}

func buildListCuratorUsersInput(r *http.Request) (model.ListCuratorUsersInput, *commonstore.AppError) {
	page, pageSize, appErr := parsePaginationParams(r)
	if appErr != nil {
		return model.ListCuratorUsersInput{}, appErr
	}
	input := model.ListCuratorUsersInput{
		Role:     r.URL.Query().Get("role"),
		Q:        strings.TrimSpace(r.URL.Query().Get("q")),
		Page:     page,
		PageSize: pageSize,
	}
	if input.Role != "" && !containsString([]string{
		model.UserRoleApplicant,
		model.UserRoleEmployer,
	}, input.Role) {
		return model.ListCuratorUsersInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid moderated user role",
		}
	}
	if raw := r.URL.Query().Get("isActive"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return model.ListCuratorUsersInput{}, &commonstore.AppError{
				Status:  http.StatusUnprocessableEntity,
				Code:    "validation_error",
				Message: "invalid isActive parameter",
			}
		}
		input.IsActive = &value
	}
	return input, nil
}

func buildListAdminCuratorsInput(r *http.Request) (model.ListAdminCuratorsInput, *commonstore.AppError) {
	page, pageSize, appErr := parsePaginationParams(r)
	if appErr != nil {
		return model.ListAdminCuratorsInput{}, appErr
	}
	input := model.ListAdminCuratorsInput{
		Q:        strings.TrimSpace(r.URL.Query().Get("q")),
		Page:     page,
		PageSize: pageSize,
	}
	if raw := r.URL.Query().Get("isActive"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return model.ListAdminCuratorsInput{}, &commonstore.AppError{
				Status:  http.StatusUnprocessableEntity,
				Code:    "validation_error",
				Message: "invalid isActive parameter",
			}
		}
		input.IsActive = &value
	}
	return input, nil
}

func buildListCuratorTagsInput(r *http.Request) (model.ListCuratorTagsInput, *commonstore.AppError) {
	page, pageSize, appErr := parsePaginationParams(r)
	if appErr != nil {
		return model.ListCuratorTagsInput{}, appErr
	}
	input := model.ListCuratorTagsInput{
		Type:     r.URL.Query().Get("type"),
		Page:     page,
		PageSize: pageSize,
	}
	if input.Type != "" && !containsString([]string{
		model.TagTypeTechnology,
		model.TagTypeRole,
		model.TagTypeDomain,
		model.TagTypeLevel,
		model.TagTypeEmploymentType,
		model.TagTypeFormat,
		model.TagTypeCustom,
	}, input.Type) {
		return model.ListCuratorTagsInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid tag type",
		}
	}
	if raw := r.URL.Query().Get("isActive"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return model.ListCuratorTagsInput{}, &commonstore.AppError{
				Status:  http.StatusUnprocessableEntity,
				Code:    "validation_error",
				Message: "invalid isActive parameter",
			}
		}
		input.IsActive = &value
	}
	return input, nil
}

func buildListModerationCasesInput(r *http.Request) (model.ListModerationCasesInput, *commonstore.AppError) {
	page, pageSize, appErr := parsePaginationParams(r)
	if appErr != nil {
		return model.ListModerationCasesInput{}, appErr
	}
	input := model.ListModerationCasesInput{
		Status:     r.URL.Query().Get("status"),
		TargetType: r.URL.Query().Get("targetType"),
		Page:       page,
		PageSize:   pageSize,
	}
	if input.Status != "" && !containsString([]string{
		model.ModerationStatusPending,
		model.ModerationStatusApproved,
		model.ModerationStatusRejected,
		model.ModerationStatusNeedsChanges,
	}, input.Status) {
		return model.ListModerationCasesInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid moderation status",
		}
	}
	if input.TargetType != "" && !containsString([]string{
		model.ModerationTargetCompany,
		model.ModerationTargetProfile,
		model.ModerationTargetOpportunity,
		model.ModerationTargetTag,
		model.ModerationTargetMedia,
		model.ModerationTargetVerification,
	}, input.TargetType) {
		return model.ListModerationCasesInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid moderation target type",
		}
	}
	return input, nil
}

func buildListPublicOpportunitiesInput(r *http.Request) (model.ListPublicOpportunitiesInput, *commonstore.AppError) {
	page, pageSize, appErr := parsePaginationParams(r)
	if appErr != nil {
		return model.ListPublicOpportunitiesInput{}, appErr
	}
	input := model.ListPublicOpportunitiesInput{
		Page:                page,
		PageSize:            pageSize,
		Q:                   r.URL.Query().Get("q"),
		Type:                r.URL.Query().Get("type"),
		ParticipationFormat: r.URL.Query().Get("participationFormat"),
		CompanyID:           r.URL.Query().Get("companyId"),
		City:                r.URL.Query().Get("city"),
		TagIDs:              r.URL.Query()["tagIds"],
		Sort:                r.URL.Query().Get("sort"),
		View:                r.URL.Query().Get("view"),
	}
	rawBBox := r.URL.Query().Get("bbox")
	if input.View == "" {
		input.View = "list"
	}
	if !containsString([]string{"list", "map"}, input.View) {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid view parameter"}
	}
	if input.Type != "" && !containsString([]string{
		model.OpportunityTypeInternship,
		model.OpportunityTypeVacancy,
		model.OpportunityTypeMentorProgram,
		model.OpportunityTypeEvent,
	}, input.Type) {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid opportunity type"}
	}
	if input.ParticipationFormat != "" && !containsString([]string{"offline", "hybrid", "remote", "online"}, input.ParticipationFormat) {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid participation format"}
	}
	if input.Sort != "" && !containsString([]string{"published_at_desc", "published_at_asc", "salary_desc", "salary_asc", "starts_at_asc"}, input.Sort) {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid sort parameter"}
	}

	var err error
	input.SalaryFrom, err = optionalIntQuery(r, "salaryFrom")
	if err != nil {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid salaryFrom parameter"}
	}
	input.SalaryTo, err = optionalIntQuery(r, "salaryTo")
	if err != nil {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid salaryTo parameter"}
	}
	input.StartsAfter, err = optionalTimeQuery(r, "startsAfter")
	if err != nil {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid startsAfter parameter"}
	}
	input.ExpiresAfter, err = optionalTimeQuery(r, "expiresAfter")
	if err != nil {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid expiresAfter parameter"}
	}
	input.Lat, err = optionalFloatQuery(r, "lat")
	if err != nil {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid lat parameter"}
	}
	if input.Lat != nil && !isValidLatitude(*input.Lat) {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid lat parameter"}
	}
	input.Lng, err = optionalFloatQuery(r, "lng")
	if err != nil {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid lng parameter"}
	}
	if input.Lng != nil && !isValidLongitude(*input.Lng) {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid lng parameter"}
	}
	input.RadiusKm, err = optionalFloatQuery(r, "radiusKm")
	if err != nil {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid radiusKm parameter"}
	}
	if input.RadiusKm != nil && (!isFiniteFloat(*input.RadiusKm) || *input.RadiusKm < 0) {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid radiusKm parameter"}
	}
	if rawBBox != "" {
		input.BBox, err = parseGeoBounds(rawBBox)
		if err != nil {
			return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid bbox parameter"}
		}
	}

	input.EmploymentType = r.URL.Query().Get("employmentType")
	if input.EmploymentType != "" && !containsString([]string{"full_time", "part_time", "project", "contract"}, input.EmploymentType) {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid employmentType parameter"}
	}
	input.ExperienceLevel = r.URL.Query().Get("experienceLevel")
	if input.ExperienceLevel != "" && !containsString([]string{"trainee", "junior", "middle", "senior"}, input.ExperienceLevel) {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid experienceLevel parameter"}
	}

	if input.BBox != nil || input.Lat != nil || input.Lng != nil || input.RadiusKm != nil {
		if input.View != "map" {
			return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "map filters require view=map"}
		}
		if input.BBox != nil && (input.Lat != nil || input.Lng != nil || input.RadiusKm != nil) {
			return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "bbox cannot be combined with center and radius filters"}
		}
		if (input.Lat == nil) != (input.Lng == nil) {
			return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "lat and lng must be provided together"}
		}
		if (input.Lat != nil || input.Lng != nil) && input.RadiusKm == nil {
			return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "radiusKm is required when lat and lng are provided"}
		}
		if input.City != "" {
			return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "city cannot be combined with map geo filters"}
		}
	}

	return input, nil
}

func parseCreateNotificationCampaignInput(req createNotificationCampaignRequest) (model.CreateNotificationCampaignInput, *commonstore.AppError) {
	scheduledAt, appErr := parseOptionalRFC3339(req.ScheduledAt, "scheduledAt")
	if appErr != nil {
		return model.CreateNotificationCampaignInput{}, appErr
	}

	sendViaInApp := true
	if req.SendViaInApp != nil {
		sendViaInApp = *req.SendViaInApp
	}
	sendViaEmail := false
	if req.SendViaEmail != nil {
		sendViaEmail = *req.SendViaEmail
	}

	if !containsString([]string{
		model.NotificationCampaignAudienceSingleApplicant,
		model.NotificationCampaignAudienceSelectedApplicants,
		model.NotificationCampaignAudienceAllApplicantsOpportunity,
	}, req.AudienceType) {
		return model.CreateNotificationCampaignInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid audienceType",
		}
	}

	return model.CreateNotificationCampaignInput{
		CompanyID:        strings.TrimSpace(req.CompanyID),
		OpportunityID:    trimStringPtr(req.OpportunityID),
		AudienceType:     req.AudienceType,
		Title:            strings.TrimSpace(req.Title),
		Body:             strings.TrimSpace(req.Body),
		SendViaInApp:     sendViaInApp,
		SendViaEmail:     sendViaEmail,
		ScheduledAt:      scheduledAt,
		ApplicantUserIDs: trimStrings(req.ApplicantUserIDs),
	}, nil
}

func parseUpdateNotificationCampaignInput(r *http.Request) (model.UpdateNotificationCampaignInput, *commonstore.AppError) {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	raw := map[string]json.RawMessage{}
	if err := decoder.Decode(&raw); err != nil {
		return model.UpdateNotificationCampaignInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid request body",
		}
	}
	if decoder.More() {
		return model.UpdateNotificationCampaignInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid request body",
		}
	}

	allowedKeys := map[string]struct{}{
		"audienceType":     {},
		"opportunityId":    {},
		"title":            {},
		"body":             {},
		"sendViaInApp":     {},
		"sendViaEmail":     {},
		"scheduledAt":      {},
		"applicantUserIds": {},
	}
	for key := range raw {
		if _, ok := allowedKeys[key]; !ok {
			return model.UpdateNotificationCampaignInput{}, &commonstore.AppError{
				Status:  http.StatusUnprocessableEntity,
				Code:    "validation_error",
				Message: "invalid request body",
			}
		}
	}
	if len(raw) == 0 {
		return model.UpdateNotificationCampaignInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "at least one field is required",
		}
	}

	input := model.UpdateNotificationCampaignInput{}

	if value, ok := raw["title"]; ok {
		var title *string
		if err := json.Unmarshal(value, &title); err != nil {
			return model.UpdateNotificationCampaignInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid title"}
		}
		if title != nil {
			trimmed := strings.TrimSpace(*title)
			input.Title = &trimmed
		}
	}
	if value, ok := raw["body"]; ok {
		var body *string
		if err := json.Unmarshal(value, &body); err != nil {
			return model.UpdateNotificationCampaignInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid body"}
		}
		if body != nil {
			trimmed := strings.TrimSpace(*body)
			input.Body = &trimmed
		}
	}

	_, hasSendViaInApp := raw["sendViaInApp"]
	_, hasSendViaEmail := raw["sendViaEmail"]
	if hasSendViaInApp != hasSendViaEmail {
		return model.UpdateNotificationCampaignInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "sendViaInApp and sendViaEmail must be provided together",
		}
	}
	if hasSendViaInApp {
		var sendViaInApp bool
		if err := json.Unmarshal(raw["sendViaInApp"], &sendViaInApp); err != nil {
			return model.UpdateNotificationCampaignInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid sendViaInApp"}
		}
		var sendViaEmail bool
		if err := json.Unmarshal(raw["sendViaEmail"], &sendViaEmail); err != nil {
			return model.UpdateNotificationCampaignInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid sendViaEmail"}
		}
		input.SendViaInApp = &sendViaInApp
		input.SendViaEmail = &sendViaEmail
	}

	if value, ok := raw["scheduledAt"]; ok {
		input.SetScheduledAt = true
		var scheduledAt *string
		if err := json.Unmarshal(value, &scheduledAt); err != nil {
			return model.UpdateNotificationCampaignInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid scheduledAt"}
		}
		if scheduledAt != nil {
			parsed, appErr := parseOptionalRFC3339(scheduledAt, "scheduledAt")
			if appErr != nil {
				return model.UpdateNotificationCampaignInput{}, appErr
			}
			input.ScheduledAt = parsed
		}
	}

	_, hasAudienceType := raw["audienceType"]
	_, hasOpportunityID := raw["opportunityId"]
	_, hasApplicantUserIDs := raw["applicantUserIds"]
	if hasAudienceType || hasOpportunityID || hasApplicantUserIDs {
		input.ReplaceAudience = true
		if !hasAudienceType {
			return model.UpdateNotificationCampaignInput{}, &commonstore.AppError{
				Status:  http.StatusUnprocessableEntity,
				Code:    "validation_error",
				Message: "audienceType is required when replacing campaign audience",
			}
		}

		var audienceType *string
		if err := json.Unmarshal(raw["audienceType"], &audienceType); err != nil || audienceType == nil || !containsString([]string{
			model.NotificationCampaignAudienceSingleApplicant,
			model.NotificationCampaignAudienceSelectedApplicants,
			model.NotificationCampaignAudienceAllApplicantsOpportunity,
		}, *audienceType) {
			return model.UpdateNotificationCampaignInput{}, &commonstore.AppError{
				Status:  http.StatusUnprocessableEntity,
				Code:    "validation_error",
				Message: "invalid audienceType",
			}
		}
		input.AudienceType = audienceType

		if hasOpportunityID {
			var opportunityID *string
			if err := json.Unmarshal(raw["opportunityId"], &opportunityID); err != nil {
				return model.UpdateNotificationCampaignInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid opportunityId"}
			}
			input.OpportunityID = trimStringPtr(opportunityID)
		}

		if hasApplicantUserIDs {
			var applicantUserIDs []string
			if err := json.Unmarshal(raw["applicantUserIds"], &applicantUserIDs); err != nil {
				return model.UpdateNotificationCampaignInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid applicantUserIds"}
			}
			input.ApplicantUserIDs = trimStrings(applicantUserIDs)
		}
	}

	return input, nil
}

func parseCreateVerificationRequestInput(req createVerificationRequestRequest) (model.CreateVerificationRequestInput, *commonstore.AppError) {
	evidence := make([]model.VerificationEvidenceInput, 0, len(req.Evidence))
	for _, item := range req.Evidence {
		value := trimStringPtr(item.Value)
		if strings.TrimSpace(item.EvidenceType) == model.VerificationEvidenceWebsiteLink {
			parsedValue, appErr := parseOptionalURI(value, "evidence.value")
			if appErr != nil {
				return model.CreateVerificationRequestInput{}, appErr
			}
			value = parsedValue
		}
		evidence = append(evidence, model.VerificationEvidenceInput{
			EvidenceType:   strings.TrimSpace(item.EvidenceType),
			Value:          value,
			EvidenceFileID: trimStringPtr(item.EvidenceFileID),
		})
	}
	return model.CreateVerificationRequestInput{
		Method:           strings.TrimSpace(req.Method),
		SubmittedComment: trimStringPtr(req.SubmittedComment),
		Evidence:         evidence,
	}, nil
}

func parseReviewVerificationRequestInput(req reviewVerificationRequestRequest) model.ReviewVerificationRequestInput {
	return model.ReviewVerificationRequestInput{
		Status:        strings.TrimSpace(req.Status),
		ReviewComment: trimStringPtr(req.ReviewComment),
	}
}

func parseCreateTagInput(req createTagRequest, forceActive bool) model.CreateTagInput {
	isActive := true
	if !forceActive {
		if req.IsActive != nil {
			isActive = *req.IsActive
		}
	} else {
		isActive = true
	}
	return model.CreateTagInput{
		Name:     strings.TrimSpace(req.Name),
		TagType:  strings.TrimSpace(req.TagType),
		IsActive: isActive,
	}
}

func parseUpdateModerationCaseInput(r *http.Request) (model.UpdateModerationCaseInput, *commonstore.AppError) {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	raw := map[string]json.RawMessage{}
	if err := decoder.Decode(&raw); err != nil {
		return model.UpdateModerationCaseInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid request body",
		}
	}
	if decoder.More() {
		return model.UpdateModerationCaseInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid request body",
		}
	}

	allowedKeys := map[string]struct{}{
		"assignedCuratorUserId": {},
		"status":                {},
		"reason":                {},
	}
	for key := range raw {
		if _, ok := allowedKeys[key]; !ok {
			return model.UpdateModerationCaseInput{}, &commonstore.AppError{
				Status:  http.StatusUnprocessableEntity,
				Code:    "validation_error",
				Message: "invalid request body",
			}
		}
	}

	var req updateModerationCaseRequest
	if data, err := json.Marshal(raw); err != nil || json.Unmarshal(data, &req) != nil {
		return model.UpdateModerationCaseInput{}, &commonstore.AppError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_error",
			Message: "invalid request body",
		}
	}

	input := model.UpdateModerationCaseInput{
		Reason: strings.TrimSpace(req.Reason),
		Status: trimStringPtr(req.Status),
	}
	if _, ok := raw["assignedCuratorUserId"]; ok {
		input.SetAssignedCuratorUserID = true
		input.AssignedCuratorUserID = trimStringPtr(req.AssignedCuratorUserID)
	}
	return input, nil
}

func parseCreateOpportunityInput(req createOpportunityRequest) (model.CreateOpportunityInput, *commonstore.AppError) {
	publishedAt, appErr := parseOptionalRFC3339(req.PublishedAt, "publishedAt")
	if appErr != nil {
		return model.CreateOpportunityInput{}, appErr
	}
	expiresAt, appErr := parseOptionalRFC3339(req.ExpiresAt, "expiresAt")
	if appErr != nil {
		return model.CreateOpportunityInput{}, appErr
	}
	vacancyDetails := toOpportunityVacancyDetails(req.VacancyDetails)
	mentorDetails, appErr := toOpportunityMentorProgramDetails(req.MentorProgramDetails)
	if appErr != nil {
		return model.CreateOpportunityInput{}, appErr
	}
	eventDetails, appErr := toOpportunityEventDetails(req.EventDetails)
	if appErr != nil {
		return model.CreateOpportunityInput{}, appErr
	}
	contactEmail, appErr := parseOptionalEmail(req.ContactEmail, "contactEmail")
	if appErr != nil {
		return model.CreateOpportunityInput{}, appErr
	}
	links, appErr := toOpportunityLinkInputs(req.Links)
	if appErr != nil {
		return model.CreateOpportunityInput{}, appErr
	}
	return model.CreateOpportunityInput{
		CompanyID:            req.CompanyID,
		Title:                req.Title,
		Summary:              req.Summary,
		Slug:                 req.Slug,
		Description:          req.Description,
		Type:                 req.Type,
		ParticipationFormat:  req.ParticipationFormat,
		LocationID:           req.LocationID,
		ContactEmail:         contactEmail,
		ContactPhone:         req.ContactPhone,
		CoverMediaID:         req.CoverMediaID,
		PublishedAt:          publishedAt,
		ExpiresAt:            expiresAt,
		TagIDs:               req.TagIDs,
		VacancyDetails:       vacancyDetails,
		MentorProgramDetails: mentorDetails,
		EventDetails:         eventDetails,
		Links:                links,
		Media:                toOpportunityMediaInputs(req.Media),
		Location:             toLocationInput(req.Location),
	}, nil
}

func parseUpdateOpportunityInput(req updateOpportunityRequest) (model.UpdateOpportunityInput, *commonstore.AppError) {
	publishedAt, appErr := parseOptionalRFC3339(req.PublishedAt, "publishedAt")
	if appErr != nil {
		return model.UpdateOpportunityInput{}, appErr
	}
	expiresAt, appErr := parseOptionalRFC3339(req.ExpiresAt, "expiresAt")
	if appErr != nil {
		return model.UpdateOpportunityInput{}, appErr
	}
	vacancyDetails := toOpportunityVacancyDetails(req.VacancyDetails)
	mentorDetails, appErr := toOpportunityMentorProgramDetails(req.MentorProgramDetails)
	if appErr != nil {
		return model.UpdateOpportunityInput{}, appErr
	}
	eventDetails, appErr := toOpportunityEventDetails(req.EventDetails)
	if appErr != nil {
		return model.UpdateOpportunityInput{}, appErr
	}
	contactEmail, appErr := parseOptionalEmail(req.ContactEmail, "contactEmail")
	if appErr != nil {
		return model.UpdateOpportunityInput{}, appErr
	}

	input := model.UpdateOpportunityInput{
		Title:                req.Title,
		Summary:              req.Summary,
		Slug:                 req.Slug,
		Description:          req.Description,
		Status:               req.Status,
		ParticipationFormat:  req.ParticipationFormat,
		LocationID:           req.LocationID,
		ContactEmail:         contactEmail,
		ContactPhone:         req.ContactPhone,
		CoverMediaID:         req.CoverMediaID,
		PublishedAt:          publishedAt,
		ExpiresAt:            expiresAt,
		VacancyDetails:       vacancyDetails,
		MentorProgramDetails: mentorDetails,
		EventDetails:         eventDetails,
		Location:             toLocationInput(req.Location),
	}
	if req.TagIDs != nil {
		input.TagIDs = *req.TagIDs
		input.ReplaceTagIDs = true
	}
	if req.Links != nil {
		input.Links, appErr = toOpportunityLinkInputs(*req.Links)
		if appErr != nil {
			return model.UpdateOpportunityInput{}, appErr
		}
		input.ReplaceLinks = true
	}
	if req.Media != nil {
		input.Media = toOpportunityMediaInputs(*req.Media)
		input.ReplaceMedia = true
	}
	return input, nil
}

func parseUpdateCuratorOpportunityInput(req updateCuratorOpportunityRequest) (model.UpdateCuratorOpportunityInput, *commonstore.AppError) {
	publishedAt, appErr := parseOptionalRFC3339(req.PublishedAt, "publishedAt")
	if appErr != nil {
		return model.UpdateCuratorOpportunityInput{}, appErr
	}
	expiresAt, appErr := parseOptionalRFC3339(req.ExpiresAt, "expiresAt")
	if appErr != nil {
		return model.UpdateCuratorOpportunityInput{}, appErr
	}
	vacancyDetails := toOpportunityVacancyDetails(req.VacancyDetails)
	mentorDetails, appErr := toOpportunityMentorProgramDetails(req.MentorProgramDetails)
	if appErr != nil {
		return model.UpdateCuratorOpportunityInput{}, appErr
	}
	eventDetails, appErr := toOpportunityEventDetails(req.EventDetails)
	if appErr != nil {
		return model.UpdateCuratorOpportunityInput{}, appErr
	}
	contactEmail, appErr := parseOptionalEmail(req.ContactEmail, "contactEmail")
	if appErr != nil {
		return model.UpdateCuratorOpportunityInput{}, appErr
	}

	input := model.UpdateCuratorOpportunityInput{
		Title:                trimStringPtr(req.Title),
		Summary:              trimStringPtr(req.Summary),
		Slug:                 trimStringPtr(req.Slug),
		Description:          req.Description,
		ParticipationFormat:  trimStringPtr(req.ParticipationFormat),
		LocationID:           trimStringPtr(req.LocationID),
		ContactEmail:         contactEmail,
		ContactPhone:         req.ContactPhone,
		CoverMediaID:         trimStringPtr(req.CoverMediaID),
		PublishedAt:          publishedAt,
		ExpiresAt:            expiresAt,
		VacancyDetails:       vacancyDetails,
		MentorProgramDetails: mentorDetails,
		EventDetails:         eventDetails,
		Location:             toLocationInput(req.Location),
		Reason:               strings.TrimSpace(req.Reason),
	}
	if req.TagIDs != nil {
		input.TagIDs = trimStrings(*req.TagIDs)
		input.ReplaceTagIDs = true
	}
	if req.Links != nil {
		input.Links, appErr = toOpportunityLinkInputs(*req.Links)
		if appErr != nil {
			return model.UpdateCuratorOpportunityInput{}, appErr
		}
		input.ReplaceLinks = true
	}
	if req.Media != nil {
		input.Media = toOpportunityMediaInputs(*req.Media)
		input.ReplaceMedia = true
	}
	return input, nil
}

func parseUpdateApplicantPrivacyInput(req *updateApplicantPrivacyRequest) *model.UpdateApplicantPrivacyInput {
	if req == nil {
		return nil
	}
	return &model.UpdateApplicantPrivacyInput{
		ProfileVisibility:      trimStringPtr(req.ProfileVisibility),
		ResumeVisibility:       trimStringPtr(req.ResumeVisibility),
		ApplicationsVisibility: trimStringPtr(req.ApplicationsVisibility),
		ContactsVisibility:     trimStringPtr(req.ContactsVisibility),
		ShowCareerInterests:    req.ShowCareerInterests,
		AllowRecommendations:   req.AllowRecommendations,
	}
}

func toOpportunityVacancyDetails(req *opportunityVacancyDetailsRequest) *model.OpportunityVacancyDetails {
	if req == nil {
		return nil
	}
	return &model.OpportunityVacancyDetails{
		EmploymentType:  req.EmploymentType,
		ExperienceLevel: req.ExperienceLevel,
		SalaryFrom:      req.SalaryFrom,
		SalaryTo:        req.SalaryTo,
		Currency:        req.Currency,
	}
}

func toOpportunityMentorProgramDetails(req *opportunityMentorProgramDetailsRequest) (*model.OpportunityMentorProgramDetails, *commonstore.AppError) {
	if req == nil {
		return nil, nil
	}
	startAt, appErr := parseOptionalRFC3339(req.StartAt, "mentorProgramDetails.startAt")
	if appErr != nil {
		return nil, appErr
	}
	endAt, appErr := parseOptionalRFC3339(req.EndAt, "mentorProgramDetails.endAt")
	if appErr != nil {
		return nil, appErr
	}
	return &model.OpportunityMentorProgramDetails{
		StartAt:            startAt,
		EndAt:              endAt,
		SeatsCount:         req.SeatsCount,
		MentorRequirements: req.MentorRequirements,
	}, nil
}

func toOpportunityEventDetails(req *opportunityEventDetailsRequest) (*model.OpportunityEventDetails, *commonstore.AppError) {
	if req == nil {
		return nil, nil
	}
	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		return nil, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid eventDetails.startAt"}
	}
	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		return nil, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid eventDetails.endAt"}
	}
	registrationDeadline, appErr := parseOptionalRFC3339(req.RegistrationDeadline, "eventDetails.registrationDeadline")
	if appErr != nil {
		return nil, appErr
	}
	return &model.OpportunityEventDetails{
		StartAt:              startAt.UTC(),
		EndAt:                endAt.UTC(),
		RegistrationDeadline: registrationDeadline,
		Capacity:             req.Capacity,
		VenueNote:            req.VenueNote,
	}, nil
}

func toOpportunityLinkInputs(items []opportunityLinkInputRequest) ([]model.OpportunityLinkInput, *commonstore.AppError) {
	response := make([]model.OpportunityLinkInput, 0, len(items))
	for _, item := range items {
		linkURL, appErr := parseRequiredURI(item.URL, "links.url")
		if appErr != nil {
			return nil, appErr
		}
		sortOrder := 0
		if item.SortOrder != nil {
			sortOrder = *item.SortOrder
		}
		response = append(response, model.OpportunityLinkInput{
			LinkType:  item.LinkType,
			Title:     item.Title,
			URL:       linkURL,
			SortOrder: sortOrder,
		})
	}
	return response, nil
}

func toOpportunityMediaInputs(items []opportunityMediaInputRequest) []model.OpportunityMediaInput {
	response := make([]model.OpportunityMediaInput, 0, len(items))
	for _, item := range items {
		sortOrder := 0
		if item.SortOrder != nil {
			sortOrder = *item.SortOrder
		}
		response = append(response, model.OpportunityMediaInput{
			MediaFileID: item.MediaFileID,
			Title:       item.Title,
			SortOrder:   sortOrder,
		})
	}
	return response
}

func optionalIntQuery(r *http.Request, name string) (*int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func optionalFloatQuery(r *http.Request, name string) (*float64, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func parseGeoBounds(raw string) (*model.GeoBounds, error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return nil, strconv.ErrSyntax
	}

	values := make([]float64, 0, 4)
	for _, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil || !isFiniteFloat(value) {
			return nil, strconv.ErrSyntax
		}
		values = append(values, value)
	}

	bounds := &model.GeoBounds{
		MinLng: values[0],
		MinLat: values[1],
		MaxLng: values[2],
		MaxLat: values[3],
	}
	if !isValidLongitude(bounds.MinLng) || !isValidLongitude(bounds.MaxLng) ||
		!isValidLatitude(bounds.MinLat) || !isValidLatitude(bounds.MaxLat) ||
		bounds.MinLng > bounds.MaxLng || bounds.MinLat > bounds.MaxLat {
		return nil, strconv.ErrSyntax
	}
	return bounds, nil
}

func isFiniteFloat(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func isValidLatitude(value float64) bool {
	return isFiniteFloat(value) && value >= -90 && value <= 90
}

func isValidLongitude(value float64) bool {
	return isFiniteFloat(value) && value >= -180 && value <= 180
}

func optionalTimeQuery(r *http.Request, name string) (*time.Time, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	value = value.UTC()
	return &value, nil
}

func parseOptionalRFC3339(value *string, field string) (*time.Time, *commonstore.AppError) {
	if value == nil {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return nil, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid " + field}
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func trimStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func trimStrings(items []string) []string {
	response := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		response = append(response, item)
	}
	return response
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func paginationMeta(page, pageSize, totalItems int) map[string]any {
	totalPages := 0
	if totalItems > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(pageSize)))
	}
	return map[string]any{
		"page":       page,
		"pageSize":   pageSize,
		"totalItems": totalItems,
		"totalPages": totalPages,
	}
}

func decodeOptionalJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}
