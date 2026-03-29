package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tramplin/backend/internal/config"
	"tramplin/backend/internal/domain/model"
	authplatform "tramplin/backend/internal/platform/auth"
	"tramplin/backend/internal/platform/httpx"
	postgresplatform "tramplin/backend/internal/platform/postgres"
	commonstore "tramplin/backend/internal/store"
	pgstore "tramplin/backend/internal/store/postgres"
)

type Server struct {
	cfg   config.Config
	db    *pgxpool.Pool
	store commonstore.Repository
	mux   *http.ServeMux
}

type registerApplicantRequest struct {
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	DisplayName string  `json:"displayName"`
	FirstName   string  `json:"firstName"`
	LastName    string  `json:"lastName"`
	MiddleName  *string `json:"middleName"`
}

type registerEmployerRequest struct {
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	DisplayName string  `json:"displayName"`
	FullName    string  `json:"fullName"`
	JobTitle    *string `json:"jobTitle"`
	Phone       *string `json:"phone"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type putUISettingsRequest struct {
	SettingsJSON  map[string]any `json:"settingsJson"`
	SchemaVersion *int           `json:"schemaVersion"`
}

type updateApplicantProfileRequest struct {
	FirstName      *string `json:"firstName"`
	LastName       *string `json:"lastName"`
	MiddleName     *string `json:"middleName"`
	UniversityName *string `json:"universityName"`
	Faculty        *string `json:"faculty"`
	ProgramName    *string `json:"programName"`
	StudyYear      *int    `json:"studyYear"`
	GraduationYear *int    `json:"graduationYear"`
	City           *string `json:"city"`
	About          *string `json:"about"`
	ResumeMediaID  *string `json:"resumeMediaId"`
}

type updateApplicantPrivacyRequest struct {
	ProfileVisibility      *string `json:"profileVisibility"`
	ResumeVisibility       *string `json:"resumeVisibility"`
	ApplicationsVisibility *string `json:"applicationsVisibility"`
	ContactsVisibility     *string `json:"contactsVisibility"`
	ShowCareerInterests    *bool   `json:"showCareerInterests"`
	AllowRecommendations   *bool   `json:"allowRecommendations"`
}

type replaceApplicantTagsRequest struct {
	TagIDs []string `json:"tagIds"`
}

type updateEmployerProfileRequest struct {
	FullName *string `json:"fullName"`
	JobTitle *string `json:"jobTitle"`
	Phone    *string `json:"phone"`
}

type locationInputRequest struct {
	Precision   string   `json:"precision"`
	Country     string   `json:"country"`
	Region      *string  `json:"region"`
	City        string   `json:"city"`
	AddressLine *string  `json:"addressLine"`
	PostalCode  *string  `json:"postalCode"`
	PlaceName   *string  `json:"placeName"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
}

type createCompanyRequest struct {
	LegalName              string                `json:"legalName"`
	BrandName              *string               `json:"brandName"`
	Slug                   string                `json:"slug"`
	INN                    *string               `json:"inn"`
	Description            *string               `json:"description"`
	Industry               *string               `json:"industry"`
	WebsiteURL             *string               `json:"websiteUrl"`
	CorporateEmailDomain   *string               `json:"corporateEmailDomain"`
	HeadquartersLocationID *string               `json:"headquartersLocationId"`
	LogoMediaID            *string               `json:"logoMediaId"`
	BannerMediaID          *string               `json:"bannerMediaId"`
	HeadquartersLocation   *locationInputRequest `json:"headquartersLocation"`
}

type updateCompanyRequest struct {
	LegalName              *string               `json:"legalName"`
	BrandName              *string               `json:"brandName"`
	Slug                   *string               `json:"slug"`
	INN                    *string               `json:"inn"`
	Description            *string               `json:"description"`
	Industry               *string               `json:"industry"`
	WebsiteURL             *string               `json:"websiteUrl"`
	CorporateEmailDomain   *string               `json:"corporateEmailDomain"`
	HeadquartersLocationID *string               `json:"headquartersLocationId"`
	LogoMediaID            *string               `json:"logoMediaId"`
	BannerMediaID          *string               `json:"bannerMediaId"`
	HeadquartersLocation   *locationInputRequest `json:"headquartersLocation"`
}

type createCompanyMembershipRequest struct {
	EmployerEmail    string  `json:"employerEmail"`
	MemberRole       string  `json:"memberRole"`
	IsPrimaryContact bool    `json:"isPrimaryContact"`
	Comment          *string `json:"comment"`
}

type updateCompanyMembershipRequest struct {
	MemberRole       *string `json:"memberRole"`
	IsPrimaryContact *bool   `json:"isPrimaryContact"`
}

type approveCompanyMembershipRequest struct {
	Comment *string `json:"comment"`
}

type commentRequest struct {
	Comment string `json:"comment"`
}

type opportunityVacancyDetailsRequest struct {
	EmploymentType  string  `json:"employmentType"`
	ExperienceLevel string  `json:"experienceLevel"`
	SalaryFrom      *int    `json:"salaryFrom"`
	SalaryTo        *int    `json:"salaryTo"`
	Currency        *string `json:"currency"`
}

type opportunityMentorProgramDetailsRequest struct {
	StartAt            *string `json:"startAt"`
	EndAt              *string `json:"endAt"`
	SeatsCount         *int    `json:"seatsCount"`
	MentorRequirements *string `json:"mentorRequirements"`
}

type opportunityEventDetailsRequest struct {
	StartAt              string  `json:"startAt"`
	EndAt                string  `json:"endAt"`
	RegistrationDeadline *string `json:"registrationDeadline"`
	Capacity             *int    `json:"capacity"`
	VenueNote            *string `json:"venueNote"`
}

type opportunityLinkInputRequest struct {
	LinkType  string `json:"linkType"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	SortOrder *int   `json:"sortOrder"`
}

type opportunityMediaInputRequest struct {
	MediaFileID string  `json:"mediaFileId"`
	Title       *string `json:"title"`
	SortOrder   *int    `json:"sortOrder"`
}

type createOpportunityRequest struct {
	CompanyID            string                                  `json:"companyId"`
	Title                string                                  `json:"title"`
	Summary              string                                  `json:"summary"`
	Slug                 string                                  `json:"slug"`
	Description          string                                  `json:"description"`
	Type                 string                                  `json:"type"`
	ParticipationFormat  string                                  `json:"participationFormat"`
	LocationID           *string                                 `json:"locationId"`
	ContactEmail         *string                                 `json:"contactEmail"`
	ContactPhone         *string                                 `json:"contactPhone"`
	CoverMediaID         *string                                 `json:"coverMediaId"`
	PublishedAt          *string                                 `json:"publishedAt"`
	ExpiresAt            *string                                 `json:"expiresAt"`
	TagIDs               []string                                `json:"tagIds"`
	VacancyDetails       *opportunityVacancyDetailsRequest       `json:"vacancyDetails"`
	MentorProgramDetails *opportunityMentorProgramDetailsRequest `json:"mentorProgramDetails"`
	EventDetails         *opportunityEventDetailsRequest         `json:"eventDetails"`
	Links                []opportunityLinkInputRequest           `json:"links"`
	Media                []opportunityMediaInputRequest          `json:"media"`
	Location             *locationInputRequest                   `json:"location"`
}

type updateOpportunityRequest struct {
	Title                *string                                 `json:"title"`
	Summary              *string                                 `json:"summary"`
	Slug                 *string                                 `json:"slug"`
	Description          *string                                 `json:"description"`
	Status               *string                                 `json:"status"`
	ParticipationFormat  *string                                 `json:"participationFormat"`
	LocationID           *string                                 `json:"locationId"`
	ContactEmail         *string                                 `json:"contactEmail"`
	ContactPhone         *string                                 `json:"contactPhone"`
	CoverMediaID         *string                                 `json:"coverMediaId"`
	PublishedAt          *string                                 `json:"publishedAt"`
	ExpiresAt            *string                                 `json:"expiresAt"`
	TagIDs               *[]string                               `json:"tagIds"`
	VacancyDetails       *opportunityVacancyDetailsRequest       `json:"vacancyDetails"`
	MentorProgramDetails *opportunityMentorProgramDetailsRequest `json:"mentorProgramDetails"`
	EventDetails         *opportunityEventDetailsRequest         `json:"eventDetails"`
	Links                *[]opportunityLinkInputRequest          `json:"links"`
	Media                *[]opportunityMediaInputRequest         `json:"media"`
	Location             *locationInputRequest                   `json:"location"`
}

type patchNotificationPreferencesRequest struct {
	InAppEnabled             *bool `json:"inAppEnabled"`
	EmailEnabled             *bool `json:"emailEnabled"`
	RecommendationEnabled    *bool `json:"recommendationEnabled"`
	ApplicationStatusEnabled *bool `json:"applicationStatusEnabled"`
	EmployerMessagesEnabled  *bool `json:"employerMessagesEnabled"`
	SystemEnabled            *bool `json:"systemEnabled"`
}

func NewServer(cfg config.Config, db *pgxpool.Pool) http.Handler {
	server := &Server{
		cfg:   cfg,
		db:    db,
		store: pgstore.New(db),
		mux:   http.NewServeMux(),
	}
	server.registerRoutes()
	base := server.withCORS(server.mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1" {
			clone := r.Clone(r.Context())
			clone.URL.Path = "/"
			base.ServeHTTP(w, clone)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/v1/") {
			clone := r.Clone(r.Context())
			clone.URL.Path = strings.TrimPrefix(r.URL.Path, "/v1")
			base.ServeHTTP(w, clone)
			return
		}
		base.ServeHTTP(w, r)
	})
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /health/live", s.handleLive)
	s.mux.HandleFunc("GET /health/ready", s.handleReady)

	s.mux.HandleFunc("POST /auth/register/applicant", s.handleRegisterApplicant)
	s.mux.HandleFunc("POST /auth/register/employer", s.handleRegisterEmployer)
	s.mux.HandleFunc("POST /auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /auth/refresh", s.handleRefresh)
	s.mux.HandleFunc("POST /auth/logout", s.handleLogout)
	s.mux.HandleFunc("POST /auth/logout-all", s.handleLogoutAll)
	s.mux.HandleFunc("GET /auth/me", s.handleGetMe)

	s.mux.HandleFunc("GET /me/ui-settings", s.handleGetUISettings)
	s.mux.HandleFunc("PUT /me/ui-settings", s.handlePutUISettings)

	s.mux.HandleFunc("GET /me/applicant-profile", s.handleGetApplicantProfile)
	s.mux.HandleFunc("PATCH /me/applicant-profile", s.handlePatchApplicantProfile)
	s.mux.HandleFunc("GET /me/applicant/privacy", s.handleGetApplicantPrivacy)
	s.mux.HandleFunc("PATCH /me/applicant/privacy", s.handlePatchApplicantPrivacy)
	s.mux.HandleFunc("GET /me/applicant/tags", s.handleListApplicantTags)
	s.mux.HandleFunc("PUT /me/applicant/tags", s.handleReplaceApplicantTags)

	s.mux.HandleFunc("GET /employer/profile", s.handleGetEmployerProfile)
	s.mux.HandleFunc("PATCH /employer/profile", s.handlePatchEmployerProfile)

	s.mux.HandleFunc("GET /employer/companies", s.handleListEmployerCompanies)
	s.mux.HandleFunc("POST /employer/companies", s.handleCreateCompany)
	s.mux.HandleFunc("GET /employer/companies/{companyId}", s.handleGetEmployerCompany)
	s.mux.HandleFunc("PATCH /employer/companies/{companyId}", s.handlePatchEmployerCompany)

	s.mux.HandleFunc("GET /employer/companies/{companyId}/memberships", s.handleListCompanyMemberships)
	s.mux.HandleFunc("POST /employer/companies/{companyId}/memberships", s.handleCreateCompanyMembership)
	s.mux.HandleFunc("PATCH /employer/companies/{companyId}/memberships/{membershipId}", s.handlePatchCompanyMembership)
	s.mux.HandleFunc("POST /employer/companies/{companyId}/memberships/{membershipId}/approve", s.handleApproveCompanyMembership)
	s.mux.HandleFunc("POST /employer/companies/{companyId}/memberships/{membershipId}/reject", s.handleRejectCompanyMembership)
	s.mux.HandleFunc("POST /employer/companies/{companyId}/memberships/{membershipId}/revoke", s.handleRevokeCompanyMembership)
	s.mux.HandleFunc("GET /employer/opportunities", s.handleListEmployerOpportunities)
	s.mux.HandleFunc("POST /employer/opportunities", s.handleCreateOpportunity)
	s.mux.HandleFunc("GET /employer/opportunities/{opportunityId}", s.handleGetEmployerOpportunity)
	s.mux.HandleFunc("PATCH /employer/opportunities/{opportunityId}", s.handlePatchEmployerOpportunity)

	s.mux.HandleFunc("GET /public/companies", s.handleListPublicCompanies)
	s.mux.HandleFunc("GET /public/companies/{companyId}", s.handleGetPublicCompany)
	s.mux.HandleFunc("GET /public/companies/by-slug/{slug}", s.handleGetPublicCompanyBySlug)
	s.mux.HandleFunc("GET /public/tags", s.handleListPublicTags)
	s.mux.HandleFunc("GET /public/opportunities", s.handleListPublicOpportunities)
	s.mux.HandleFunc("GET /public/opportunities/{opportunityId}", s.handleGetPublicOpportunity)
	s.mux.HandleFunc("GET /public/opportunities/by-slug/{slug}", s.handleGetPublicOpportunityBySlug)

	s.mux.HandleFunc("GET /locations/search", s.handleSearchLocations)
	s.mux.HandleFunc("POST /locations", s.handleCreateLocation)

	s.mux.HandleFunc("GET /notifications", s.handleListNotifications)
	s.mux.HandleFunc("GET /notifications/preferences", s.handleGetNotificationPreferences)
	s.mux.HandleFunc("PATCH /notifications/preferences", s.handlePatchNotificationPreferences)
	s.mux.HandleFunc("POST /notifications/read-all", s.handleMarkAllNotificationsRead)
	s.mux.HandleFunc("POST /notifications/{notificationId}/read", s.handleMarkNotificationRead)

	for _, route := range []struct {
		Pattern   string
		Operation string
	}{
		{"POST /uploads/presign", "createPresignedUpload"},
		{"POST /uploads/{mediaFileId}/complete", "completeUpload"},
		{"GET /media/{mediaFileId}", "getMediaFile"},
		{"DELETE /media/{mediaFileId}", "deleteMediaFile"},
		{"POST /media/{mediaFileId}/download-url", "createMediaDownloadUrl"},
		{"GET /me/applicant/social-links", "listMyApplicantSocialLinks"},
		{"POST /me/applicant/social-links", "createMyApplicantSocialLink"},
		{"DELETE /me/applicant/social-links/{linkId}", "deleteMyApplicantSocialLink"},
		{"PATCH /me/applicant/social-links/{linkId}", "patchMyApplicantSocialLink"},
		{"GET /me/applications", "listMyApplications"},
		{"POST /me/applications", "createApplication"},
		{"GET /me/applications/{applicationId}", "getMyApplication"},
		{"POST /me/applications/{applicationId}/withdraw", "withdrawMyApplication"},
		{"GET /me/saved-opportunities", "listSavedOpportunities"},
		{"POST /me/saved-opportunities", "saveOpportunity"},
		{"DELETE /me/saved-opportunities/{opportunityId}", "deleteSavedOpportunity"},
		{"GET /me/saved-companies", "listSavedCompanies"},
		{"POST /me/saved-companies", "saveCompany"},
		{"DELETE /me/saved-companies/{companyId}", "deleteSavedCompany"},
		{"GET /me/connections", "listMyConnections"},
		{"POST /me/connections", "createConnection"},
		{"PATCH /me/connections/{connectionId}", "patchConnection"},
		{"POST /me/recommendations", "createOpportunityRecommendation"},
		{"GET /applicants/{userId}", "getApplicantProfileById"},
		{"GET /employer/applicants/{userId}", "getEmployerApplicantProfile"},
		{"GET /me/recommendations/received", "listReceivedRecommendations"},
		{"GET /me/recommendations/sent", "listSentRecommendations"},
		{"GET /employer/companies/{companyId}/verification-requests", "listCompanyVerificationRequests"},
		{"POST /employer/companies/{companyId}/verification-requests", "createCompanyVerificationRequest"},
		{"GET /employer/companies/{companyId}/social-links", "listEmployerCompanySocialLinks"},
		{"POST /employer/companies/{companyId}/social-links", "createEmployerCompanySocialLink"},
		{"PATCH /employer/companies/{companyId}/social-links/{linkId}", "patchEmployerCompanySocialLink"},
		{"DELETE /employer/companies/{companyId}/social-links/{linkId}", "deleteEmployerCompanySocialLink"},
		{"GET /employer/companies/{companyId}/media", "listEmployerCompanyMedia"},
		{"POST /employer/companies/{companyId}/media", "createEmployerCompanyMedia"},
		{"PATCH /employer/companies/{companyId}/media/{mediaId}", "patchEmployerCompanyMedia"},
		{"DELETE /employer/companies/{companyId}/media/{mediaId}", "deleteEmployerCompanyMedia"},
		{"POST /employer/opportunities/{opportunityId}/activate", "activateEmployerOpportunity"},
		{"POST /employer/opportunities/{opportunityId}/close", "closeEmployerOpportunity"},
		{"POST /employer/opportunities/{opportunityId}/archive", "archiveEmployerOpportunity"},
		{"GET /employer/opportunities/{opportunityId}/applications", "listOpportunityApplications"},
		{"PATCH /employer/applications/{applicationId}/status", "patchApplicationStatus"},
		{"GET /employer/notification-campaigns", "listNotificationCampaigns"},
		{"POST /employer/notification-campaigns", "createNotificationCampaign"},
		{"GET /employer/notification-campaigns/{campaignId}", "getNotificationCampaign"},
		{"PATCH /employer/notification-campaigns/{campaignId}", "patchNotificationCampaign"},
		{"POST /employer/notification-campaigns/{campaignId}/send", "sendNotificationCampaign"},
		{"POST /employer/notification-campaigns/{campaignId}/cancel", "cancelNotificationCampaign"},
		{"POST /employer/tags", "createEmployerTag"},
		{"GET /curator/verification-requests", "listCuratorVerificationRequests"},
		{"PATCH /curator/verification-requests/{verificationRequestId}/review", "reviewVerificationRequest"},
		{"GET /curator/moderation-cases", "listModerationCases"},
		{"POST /curator/moderation-cases", "createModerationCase"},
		{"PATCH /curator/moderation-cases/{moderationCaseId}", "patchModerationCase"},
		{"GET /curator/tags", "listCuratorTags"},
		{"POST /curator/tags", "createCuratorTag"},
		{"PATCH /curator/tags/{tagId}", "patchCuratorTag"},
		{"GET /curator/users", "listCuratorUsers"},
		{"GET /curator/users/{userId}", "getCuratorUser"},
		{"PATCH /curator/users/{userId}", "patchCuratorUser"},
		{"GET /curator/applicants/{userId}", "getCuratorApplicant"},
		{"PATCH /curator/applicants/{userId}", "patchCuratorApplicant"},
		{"GET /curator/employers/{userId}", "getCuratorEmployer"},
		{"PATCH /curator/employers/{userId}", "patchCuratorEmployer"},
		{"GET /curator/companies/{companyId}", "getCuratorCompany"},
		{"PATCH /curator/companies/{companyId}", "patchCuratorCompany"},
		{"GET /curator/opportunities/{opportunityId}", "getCuratorOpportunity"},
		{"PATCH /curator/opportunities/{opportunityId}", "patchCuratorOpportunity"},
		{"GET /curator/admin/curators", "listAdminCurators"},
		{"POST /curator/admin/curators", "createAdminCurator"},
		{"PATCH /curator/admin/curators/{userId}", "patchAdminCurator"},
	} {
		s.mux.HandleFunc(route.Pattern, s.handleNotImplemented(route.Operation))
	}
}

func (s *Server) handleLive(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "database is not configured", map[string]any{
			"check": "postgres",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.DatabasePingTimeout)
	defer cancel()

	if err := postgresplatform.Ping(ctx, s.db); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "database is not ready", map[string]any{
			"check": "postgres",
			"error": err.Error(),
		})
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"checks": map[string]any{
			"postgres": "ok",
		},
	})
}

func (s *Server) handleRegisterApplicant(w http.ResponseWriter, r *http.Request) {
	var req registerApplicantRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	if req.Email == "" || req.Password == "" || req.DisplayName == "" || req.FirstName == "" || req.LastName == "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "required fields are missing", nil)
		return
	}
	hash, err := authplatform.HashPassword(req.Password)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to hash password", nil)
		return
	}
	user, appErr := s.store.CreateApplicant(model.RegisterApplicantInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		MiddleName:  req.MiddleName,
	}, hash)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	s.writeAuthCookies(w, user)
	httpx.WriteJSON(w, http.StatusCreated, s.authUserResponse(user))
}

func (s *Server) handleRegisterEmployer(w http.ResponseWriter, r *http.Request) {
	var req registerEmployerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	if req.Email == "" || req.Password == "" || req.DisplayName == "" || req.FullName == "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "required fields are missing", nil)
		return
	}
	hash, err := authplatform.HashPassword(req.Password)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to hash password", nil)
		return
	}
	user, appErr := s.store.CreateEmployer(model.RegisterEmployerInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		FullName:    req.FullName,
		JobTitle:    req.JobTitle,
		Phone:       req.Phone,
	}, hash)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	s.writeAuthCookies(w, user)
	httpx.WriteJSON(w, http.StatusCreated, s.authUserResponse(user))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	user, appErr := s.store.GetUserByEmail(req.Email)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if !authplatform.CheckPassword(user.PasswordHash, req.Password) {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid credentials", nil)
		return
	}
	if !user.IsActive {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "account is disabled", nil)
		return
	}
	s.store.TouchLastLogin(user.ID)
	user, _ = s.store.GetUserByID(user.ID)
	s.writeAuthCookies(w, user)
	httpx.WriteJSON(w, http.StatusOK, s.authUserResponse(user))
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireRefreshUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	s.writeAuthCookies(w, user)
	httpx.WriteJSON(w, http.StatusOK, s.authUserResponse(user))
}

func (s *Server) handleLogout(w http.ResponseWriter, _ *http.Request) {
	s.clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLogoutAll(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	s.store.BumpTokenVersion(user.ID)
	s.clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.currentUserResponse(user))
}

func (s *Server) handleGetUISettings(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	settings, appErr := s.store.GetUISettings(user.ID)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.uiSettingsResponse(settings))
}

func (s *Server) handlePutUISettings(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req putUISettingsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	settings, appErr := s.store.PutUISettings(user.ID, model.PutUISettingsInput{
		SettingsJSON:  req.SettingsJSON,
		SchemaVersion: req.SchemaVersion,
	})
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.uiSettingsResponse(settings))
}

func (s *Server) handleGetApplicantProfile(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}
	profile, appErr := s.store.GetApplicantProfile(user.ID)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.applicantProfileResponse(profile))
}

func (s *Server) handlePatchApplicantProfile(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req updateApplicantProfileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	profile, appErr := s.store.UpdateApplicantProfile(user.ID, model.UpdateApplicantProfileInput{
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		MiddleName:     req.MiddleName,
		UniversityName: req.UniversityName,
		Faculty:        req.Faculty,
		ProgramName:    req.ProgramName,
		StudyYear:      req.StudyYear,
		GraduationYear: req.GraduationYear,
		City:           req.City,
		About:          req.About,
		ResumeMediaID:  req.ResumeMediaID,
	})
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.applicantProfileResponse(profile))
}

func (s *Server) handleGetApplicantPrivacy(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	privacy, appErr := s.store.GetApplicantPrivacy(user.ID)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.applicantPrivacyResponse(privacy))
}

func (s *Server) handlePatchApplicantPrivacy(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req updateApplicantPrivacyRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	privacy, appErr := s.store.UpdateApplicantPrivacy(user.ID, model.UpdateApplicantPrivacyInput{
		ProfileVisibility:      req.ProfileVisibility,
		ResumeVisibility:       req.ResumeVisibility,
		ApplicationsVisibility: req.ApplicationsVisibility,
		ContactsVisibility:     req.ContactsVisibility,
		ShowCareerInterests:    req.ShowCareerInterests,
		AllowRecommendations:   req.AllowRecommendations,
	})
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.applicantPrivacyResponse(privacy))
}

func (s *Server) handleListApplicantTags(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	tags, appErr := s.store.GetApplicantTags(user.ID)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.tagsResponse(tags),
	})
}

func (s *Server) handleReplaceApplicantTags(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req replaceApplicantTagsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	tags, appErr := s.store.ReplaceApplicantTags(user.ID, req.TagIDs)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.tagsResponse(tags),
	})
}

func (s *Server) handleGetEmployerProfile(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	profile, appErr := s.store.GetEmployerProfile(user.ID)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.employerProfileResponse(profile))
}

func (s *Server) handlePatchEmployerProfile(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req updateEmployerProfileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	profile, appErr := s.store.UpdateEmployerProfile(user.ID, model.UpdateEmployerProfileInput{
		FullName: req.FullName,
		JobTitle: req.JobTitle,
		Phone:    req.Phone,
	})
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.employerProfileResponse(profile))
}

func (s *Server) handleListEmployerCompanies(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	items, appErr := s.store.ListEmployerMemberships(user.ID)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.membershipsResponse(items),
	})
}

func (s *Server) handleCreateCompany(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req createCompanyRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	company, _, appErr := s.store.CreateCompany(user.ID, model.CreateCompanyInput{
		LegalName:              req.LegalName,
		BrandName:              req.BrandName,
		Slug:                   req.Slug,
		INN:                    req.INN,
		Description:            req.Description,
		Industry:               req.Industry,
		WebsiteURL:             req.WebsiteURL,
		CorporateEmailDomain:   req.CorporateEmailDomain,
		HeadquartersLocationID: req.HeadquartersLocationID,
		LogoMediaID:            req.LogoMediaID,
		BannerMediaID:          req.BannerMediaID,
		HeadquartersLocation:   toLocationInput(req.HeadquartersLocation),
	})
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.employerCompanyDetailResponse(company))
}

func (s *Server) handleGetEmployerCompany(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	company, appErr := s.store.GetCompanyForEmployer(user.ID, r.PathValue("companyId"))
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.employerCompanyDetailResponse(company))
}

func (s *Server) handlePatchEmployerCompany(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req updateCompanyRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	company, appErr := s.store.UpdateCompany(user.ID, r.PathValue("companyId"), model.UpdateCompanyInput{
		LegalName:              req.LegalName,
		BrandName:              req.BrandName,
		Slug:                   req.Slug,
		INN:                    req.INN,
		Description:            req.Description,
		Industry:               req.Industry,
		WebsiteURL:             req.WebsiteURL,
		CorporateEmailDomain:   req.CorporateEmailDomain,
		HeadquartersLocationID: req.HeadquartersLocationID,
		LogoMediaID:            req.LogoMediaID,
		BannerMediaID:          req.BannerMediaID,
		HeadquartersLocation:   toLocationInput(req.HeadquartersLocation),
	})
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.employerCompanyDetailResponse(company))
}

func (s *Server) handleListCompanyMemberships(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	items, appErr := s.store.ListCompanyMemberships(user.ID, r.PathValue("companyId"), r.URL.Query().Get("status"))
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.membershipsResponse(items),
	})
}

func (s *Server) handleCreateCompanyMembership(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req createCompanyMembershipRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	if req.EmployerEmail == "" || req.MemberRole == "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "required fields are missing", nil)
		return
	}
	membership, appErr := s.store.CreateCompanyMembership(user.ID, r.PathValue("companyId"), model.CreateCompanyMembershipInput{
		EmployerEmail:    req.EmployerEmail,
		MemberRole:       req.MemberRole,
		IsPrimaryContact: req.IsPrimaryContact,
		Comment:          req.Comment,
	})
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.membershipResponse(membership))
}

func (s *Server) handlePatchCompanyMembership(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req updateCompanyMembershipRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	if req.MemberRole == nil && req.IsPrimaryContact == nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "at least one field is required", nil)
		return
	}
	membership, appErr := s.store.UpdateCompanyMembership(user.ID, r.PathValue("companyId"), r.PathValue("membershipId"), model.UpdateCompanyMembershipInput{
		MemberRole:       req.MemberRole,
		IsPrimaryContact: req.IsPrimaryContact,
	})
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.membershipResponse(membership))
}

func (s *Server) handleApproveCompanyMembership(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req approveCompanyMembershipRequest
	if err := decodeOptionalJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	membership, appErr := s.store.ApproveCompanyMembership(user.ID, r.PathValue("companyId"), r.PathValue("membershipId"), req.Comment)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.membershipResponse(membership))
}

func (s *Server) handleRejectCompanyMembership(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req commentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || strings.TrimSpace(req.Comment) == "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "comment is required", nil)
		return
	}
	membership, appErr := s.store.RejectCompanyMembership(user.ID, r.PathValue("companyId"), r.PathValue("membershipId"), req.Comment)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.membershipResponse(membership))
}

func (s *Server) handleRevokeCompanyMembership(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req commentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || strings.TrimSpace(req.Comment) == "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "comment is required", nil)
		return
	}
	membership, appErr := s.store.RevokeCompanyMembership(user.ID, r.PathValue("companyId"), r.PathValue("membershipId"), req.Comment)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.membershipResponse(membership))
}

func (s *Server) handleListEmployerOpportunities(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	input, repoErr := buildListEmployerOpportunitiesInput(r)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	items, total, repoErr := s.store.ListEmployerOpportunities(user.ID, input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.employerOpportunityCatalogResponse(items),
		"meta":  paginationMeta(input.Page, input.PageSize, total),
	})
}

func (s *Server) handleCreateOpportunity(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req createOpportunityRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	input, repoErr := parseCreateOpportunityInput(req)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	opportunity, repoErr := s.store.CreateOpportunity(user.ID, input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.employerOpportunityDetailResponse(opportunity))
}

func (s *Server) handleGetEmployerOpportunity(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	opportunity, repoErr := s.store.GetEmployerOpportunity(user.ID, r.PathValue("opportunityId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.employerOpportunityDetailResponse(opportunity))
}

func (s *Server) handlePatchEmployerOpportunity(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req updateOpportunityRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	input, repoErr := parseUpdateOpportunityInput(req)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	opportunity, repoErr := s.store.UpdateOpportunity(user.ID, r.PathValue("opportunityId"), input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.employerOpportunityDetailResponse(opportunity))
}

func (s *Server) handleListPublicCompanies(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "pageSize", 20)
	items, total := s.store.ListPublicCompanies(r.URL.Query().Get("q"), r.URL.Query().Get("industry"), r.URL.Query().Get("city"), page, pageSize)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.companiesSummaryResponse(items),
		"meta":  paginationMeta(page, pageSize, total),
	})
}

func (s *Server) handleGetPublicCompany(w http.ResponseWriter, r *http.Request) {
	company, appErr := s.store.GetPublicCompanyByID(r.PathValue("companyId"))
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.companyDetailResponse(company))
}

func (s *Server) handleGetPublicCompanyBySlug(w http.ResponseWriter, r *http.Request) {
	company, appErr := s.store.GetPublicCompanyBySlug(r.PathValue("slug"))
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.companyDetailResponse(company))
}

func (s *Server) handleListPublicTags(w http.ResponseWriter, r *http.Request) {
	items, appErr := s.store.ListPublicTags(r.URL.Query().Get("type"), r.URL.Query().Get("q"))
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.tagsResponse(items),
	})
}

func (s *Server) handleListPublicOpportunities(w http.ResponseWriter, r *http.Request) {
	input, appErr := buildListPublicOpportunitiesInput(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	items, total, appErr := s.store.ListPublicOpportunities(input)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.publicOpportunityCatalogResponse(items),
		"meta":  paginationMeta(input.Page, input.PageSize, total),
	})
}

func (s *Server) handleGetPublicOpportunity(w http.ResponseWriter, r *http.Request) {
	opportunity, appErr := s.store.GetPublicOpportunityByID(r.PathValue("opportunityId"))
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.publicOpportunityDetailResponse(opportunity))
}

func (s *Server) handleGetPublicOpportunityBySlug(w http.ResponseWriter, r *http.Request) {
	opportunity, appErr := s.store.GetPublicOpportunityBySlug(r.PathValue("slug"))
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.publicOpportunityDetailResponse(opportunity))
}

func (s *Server) handleSearchLocations(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	_ = user
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "pageSize", 20)
	items, total, appErr := s.store.SearchLocations(r.URL.Query().Get("q"), page, pageSize)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.locationsResponse(items),
		"meta":  paginationMeta(page, pageSize, total),
	})
}

func (s *Server) handleCreateLocation(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	_ = user
	var req locationInputRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || req.Precision == "" || req.Country == "" || req.City == "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	location, appErr := s.store.CreateOrReuseLocation(model.LocationInput{
		Precision:   req.Precision,
		Country:     req.Country,
		Region:      req.Region,
		City:        req.City,
		AddressLine: req.AddressLine,
		PostalCode:  req.PostalCode,
		PlaceName:   req.PlaceName,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
	})
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.locationResponse(location))
}

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	_ = user
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "pageSize", 20)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": []any{},
		"meta":  paginationMeta(page, pageSize, 0),
	})
}

func (s *Server) handleGetNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	preferences, appErr := s.store.GetNotificationPreferences(user.ID)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.notificationPreferencesResponse(preferences))
}

func (s *Server) handlePatchNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req patchNotificationPreferencesRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	preferences, appErr := s.store.PatchNotificationPreferences(
		user.ID,
		req.InAppEnabled,
		req.EmailEnabled,
		req.RecommendationEnabled,
		req.ApplicationStatusEnabled,
		req.EmployerMessagesEnabled,
		req.SystemEnabled,
	)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.notificationPreferencesResponse(preferences))
}

func (s *Server) handleMarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	_ = user
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"updatedCount": 0})
}

func (s *Server) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	_ = user
	httpx.WriteError(w, http.StatusNotFound, "not_found", "notification not found", nil)
}

func (s *Server) handleNotImplemented(operation string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteError(w, http.StatusNotImplemented, "not_implemented", "operation is not implemented yet", map[string]any{
			"operationId": operation,
			"path":        r.URL.Path,
			"method":      r.Method,
		})
	}
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if s.cfg.CORSAllowOrigin == "" || s.cfg.CORSAllowOrigin == origin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With")
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,PUT,DELETE,OPTIONS")
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAccessUser(r *http.Request) (*model.User, *commonstore.AppError) {
	cookie, err := r.Cookie(s.cfg.AccessCookieName)
	if err != nil {
		return nil, &commonstore.AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "authentication failed"}
	}
	claims, err := authplatform.ParseToken(s.cfg.TokenSecret, cookie.Value, authplatform.TokenTypeAccess)
	if err != nil {
		return nil, &commonstore.AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "authentication failed"}
	}
	user, appErr := s.store.GetUserByID(claims.Subject)
	if appErr != nil {
		return nil, appErr
	}
	if user.TokenVersion != claims.TokenVersion {
		return nil, &commonstore.AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "authentication failed"}
	}
	if !user.IsActive {
		return nil, &commonstore.AppError{Status: http.StatusForbidden, Code: "forbidden", Message: "account is disabled"}
	}
	return user, nil
}

func (s *Server) requireRefreshUser(r *http.Request) (*model.User, *commonstore.AppError) {
	cookie, err := r.Cookie(s.cfg.RefreshCookieName)
	if err != nil {
		return nil, &commonstore.AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "refresh token is missing, invalid, expired, or revoked by token version change"}
	}
	claims, err := authplatform.ParseToken(s.cfg.TokenSecret, cookie.Value, authplatform.TokenTypeRefresh)
	if err != nil {
		return nil, &commonstore.AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "refresh token is missing, invalid, expired, or revoked by token version change"}
	}
	user, appErr := s.store.GetUserByID(claims.Subject)
	if appErr != nil {
		return nil, &commonstore.AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "refresh token is missing, invalid, expired, or revoked by token version change"}
	}
	if user.TokenVersion != claims.TokenVersion {
		return nil, &commonstore.AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "refresh token is missing, invalid, expired, or revoked by token version change"}
	}
	if !user.IsActive {
		return nil, &commonstore.AppError{Status: http.StatusForbidden, Code: "forbidden", Message: "account is disabled"}
	}
	return user, nil
}

func (s *Server) writeAuthCookies(w http.ResponseWriter, user *model.User) {
	access, _ := authplatform.IssueToken(s.cfg.TokenSecret, authplatform.Claims{
		Subject:      user.ID,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		Type:         authplatform.TokenTypeAccess,
		ExpiresAt:    time.Now().UTC().Add(s.cfg.AccessTokenTTL),
	})
	refresh, _ := authplatform.IssueToken(s.cfg.TokenSecret, authplatform.Claims{
		Subject:      user.ID,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		Type:         authplatform.TokenTypeRefresh,
		ExpiresAt:    time.Now().UTC().Add(s.cfg.RefreshTokenTTL),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.AccessCookieName,
		Value:    access,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Domain:   s.cfg.CookieDomain,
		MaxAge:   int(s.cfg.AccessTokenTTL.Seconds()),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.RefreshCookieName,
		Value:    refresh,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Domain:   s.cfg.CookieDomain,
		MaxAge:   int(s.cfg.RefreshTokenTTL.Seconds()),
	})
}

func (s *Server) clearAuthCookies(w http.ResponseWriter) {
	for _, name := range []string{s.cfg.AccessCookieName, s.cfg.RefreshCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   s.cfg.CookieSecure,
			SameSite: http.SameSiteLaxMode,
			Domain:   s.cfg.CookieDomain,
			MaxAge:   -1,
		})
	}
}

func (s *Server) writeAppError(w http.ResponseWriter, appErr *commonstore.AppError) {
	httpx.WriteError(w, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
}

func (s *Server) authUserResponse(user *model.User) map[string]any {
	return map[string]any{
		"user":                 s.currentUserResponse(user),
		"accessTokenExpiresIn": s.cfg.DefaultAccessTokenTTL,
	}
}

func (s *Server) currentUserResponse(user *model.User) map[string]any {
	return map[string]any{
		"id":              user.ID,
		"email":           user.Email,
		"displayName":     user.DisplayName,
		"role":            user.Role,
		"isActive":        user.IsActive,
		"avatarMediaId":   nullableString(user.AvatarMediaID),
		"emailVerifiedAt": nullableTime(user.EmailVerifiedAt),
		"lastLoginAt":     nullableTime(user.LastLoginAt),
		"createdAt":       timestamp(user.CreatedAt),
		"updatedAt":       timestamp(user.UpdatedAt),
		"curatorProfile":  nil,
	}
}

func (s *Server) applicantProfileResponse(profile *model.ApplicantProfile) map[string]any {
	return map[string]any{
		"userId":         profile.UserID,
		"firstName":      profile.FirstName,
		"lastName":       profile.LastName,
		"middleName":     nullableString(profile.MiddleName),
		"universityName": nullableString(profile.UniversityName),
		"faculty":        nullableString(profile.Faculty),
		"programName":    nullableString(profile.ProgramName),
		"studyYear":      nullableInt(profile.StudyYear),
		"graduationYear": nullableInt(profile.GraduationYear),
		"city":           nullableString(profile.City),
		"about":          nullableString(profile.About),
		"resumeMediaId":  nullableString(profile.ResumeMediaID),
		"createdAt":      timestamp(profile.CreatedAt),
		"updatedAt":      timestamp(profile.UpdatedAt),
	}
}

func (s *Server) applicantPrivacyResponse(privacy *model.ApplicantPrivacySettings) map[string]any {
	return map[string]any{
		"applicantUserId":        privacy.ApplicantUserID,
		"profileVisibility":      privacy.ProfileVisibility,
		"resumeVisibility":       privacy.ResumeVisibility,
		"applicationsVisibility": privacy.ApplicationsVisibility,
		"contactsVisibility":     privacy.ContactsVisibility,
		"showCareerInterests":    privacy.ShowCareerInterests,
		"allowRecommendations":   privacy.AllowRecommendations,
		"updatedAt":              timestamp(privacy.UpdatedAt),
	}
}

func (s *Server) employerProfileResponse(profile *model.EmployerProfile) map[string]any {
	return map[string]any{
		"userId":    profile.UserID,
		"fullName":  profile.FullName,
		"jobTitle":  nullableString(profile.JobTitle),
		"phone":     nullableString(profile.Phone),
		"createdAt": timestamp(profile.CreatedAt),
		"updatedAt": timestamp(profile.UpdatedAt),
	}
}

func (s *Server) uiSettingsResponse(settings *model.UISettings) map[string]any {
	return map[string]any{
		"userId":        settings.UserID,
		"settingsJson":  settings.SettingsJSON,
		"schemaVersion": settings.SchemaVersion,
		"updatedAt":     timestamp(settings.UpdatedAt),
	}
}

func (s *Server) companyDetailResponse(company *model.Company) map[string]any {
	body := s.companySummaryResponse(company)
	body["bannerMediaId"] = nullableString(company.BannerMediaID)
	body["socialLinks"] = []any{}
	body["media"] = []any{}
	return body
}

func (s *Server) employerCompanyDetailResponse(company *model.Company) map[string]any {
	body := s.companyDetailResponse(company)
	body["inn"] = nullableString(company.INN)
	body["corporateEmailDomain"] = nullableString(company.CorporateEmailDomain)
	body["headquartersLocationId"] = nullableString(company.HeadquartersLocationID)
	body["createdByUserId"] = nullableString(company.CreatedByUserID)
	body["verifiedAt"] = nullableTime(company.VerifiedAt)
	body["verifiedByCuratorUserId"] = nullableString(company.VerifiedByCuratorUserID)
	return body
}

func (s *Server) companySummaryResponse(company *model.Company) map[string]any {
	if company == nil {
		return map[string]any{
			"id":                   nil,
			"legalName":            nil,
			"brandName":            nil,
			"slug":                 nil,
			"description":          nil,
			"industry":             nil,
			"websiteUrl":           nil,
			"logoMediaId":          nil,
			"verificationStatus":   nil,
			"headquartersLocation": nil,
			"createdAt":            nil,
			"updatedAt":            nil,
		}
	}
	return map[string]any{
		"id":                   company.ID,
		"legalName":            company.LegalName,
		"brandName":            nullableString(company.BrandName),
		"slug":                 company.Slug,
		"description":          nullableString(company.Description),
		"industry":             nullableString(company.Industry),
		"websiteUrl":           nullableString(company.WebsiteURL),
		"logoMediaId":          nullableString(company.LogoMediaID),
		"verificationStatus":   company.VerificationStatus,
		"headquartersLocation": s.nullableLocation(company.HeadquartersLocationID),
		"createdAt":            timestamp(company.CreatedAt),
		"updatedAt":            timestamp(company.UpdatedAt),
	}
}

func (s *Server) membershipResponse(membership *model.CompanyMembership) map[string]any {
	user, profile := s.store.GetEmployerSnapshot(membership.EmployerUserID)
	company, _ := s.store.GetPublicCompanyByID(membership.CompanyID)
	return map[string]any{
		"id":                    membership.ID,
		"companyId":             membership.CompanyID,
		"employerUserId":        membership.EmployerUserID,
		"employer":              s.employerPreviewResponse(user, profile),
		"invitedByUserId":       nullableString(membership.InvitedByUserID),
		"company":               s.companySummaryResponse(company),
		"membershipStatus":      membership.Status,
		"memberRole":            membership.MemberRole,
		"isPrimaryContact":      membership.IsPrimaryContact,
		"statusChangedByUserId": nullableString(membership.StatusChangedByUserID),
		"statusComment":         nullableString(membership.StatusComment),
		"statusUpdatedAt":       timestamp(membership.StatusUpdatedAt),
		"approvedAt":            nullableTime(membership.ApprovedAt),
		"updatedAt":             timestamp(membership.UpdatedAt),
		"createdAt":             timestamp(membership.CreatedAt),
	}
}

func (s *Server) employerPreviewResponse(user *model.User, profile *model.EmployerProfile) map[string]any {
	if user == nil || profile == nil {
		return map[string]any{
			"userId":        nil,
			"email":         nil,
			"displayName":   nil,
			"fullName":      nil,
			"isActive":      false,
			"avatarMediaId": nil,
			"jobTitle":      nil,
		}
	}
	return map[string]any{
		"userId":        user.ID,
		"email":         user.Email,
		"displayName":   user.DisplayName,
		"fullName":      profile.FullName,
		"isActive":      user.IsActive,
		"avatarMediaId": nullableString(user.AvatarMediaID),
		"jobTitle":      nullableString(profile.JobTitle),
	}
}

func (s *Server) publicOpportunityCatalogResponse(items []*model.Opportunity) []any {
	response := make([]any, 0, len(items))
	for _, opportunity := range items {
		response = append(response, s.publicOpportunityCatalogItemResponse(opportunity))
	}
	return response
}

func (s *Server) employerOpportunityCatalogResponse(items []*model.Opportunity) []any {
	response := make([]any, 0, len(items))
	for _, opportunity := range items {
		response = append(response, s.employerOpportunityCatalogItemResponse(opportunity))
	}
	return response
}

func (s *Server) publicOpportunityCatalogItemResponse(opportunity *model.Opportunity) map[string]any {
	body := s.publicOpportunityFieldsResponse(opportunity)
	body["vacancyDetails"] = s.opportunityVacancyPreviewResponse(opportunity.VacancyDetails)
	body["mentorProgramDetails"] = s.opportunityMentorProgramPreviewResponse(opportunity.MentorProgramDetails)
	body["eventDetails"] = s.opportunityEventPreviewResponse(opportunity.EventDetails)
	return body
}

func (s *Server) employerOpportunityCatalogItemResponse(opportunity *model.Opportunity) map[string]any {
	body := s.publicOpportunityCatalogItemResponse(opportunity)
	body["moderationStatus"] = opportunity.ModerationStatus
	body["stats"] = s.nullableOpportunityStatsResponse(opportunity.Stats)
	return body
}

func (s *Server) publicOpportunityDetailResponse(opportunity *model.Opportunity) map[string]any {
	body := s.publicOpportunityFieldsResponse(opportunity)
	body["vacancyDetails"] = s.opportunityVacancyDetailResponse(opportunity.VacancyDetails)
	body["mentorProgramDetails"] = s.opportunityMentorProgramDetailResponse(opportunity.MentorProgramDetails)
	body["eventDetails"] = s.opportunityEventDetailResponse(opportunity.EventDetails)
	body["description"] = opportunity.Description
	body["contactEmail"] = nullableString(opportunity.ContactEmail)
	body["contactPhone"] = nullableString(opportunity.ContactPhone)
	body["media"] = s.opportunityMediaResponse(opportunity.Media)
	body["links"] = s.opportunityLinksResponse(opportunity.Links)
	return body
}

func (s *Server) employerOpportunityDetailResponse(opportunity *model.Opportunity) map[string]any {
	body := s.publicOpportunityDetailResponse(opportunity)
	body["moderationStatus"] = opportunity.ModerationStatus
	body["stats"] = s.nullableOpportunityStatsResponse(opportunity.Stats)
	return body
}

func (s *Server) publicOpportunityFieldsResponse(opportunity *model.Opportunity) map[string]any {
	return map[string]any{
		"id":                  opportunity.ID,
		"company":             s.companySummaryResponse(opportunity.Company),
		"title":               opportunity.Title,
		"summary":             opportunity.Summary,
		"slug":                opportunity.Slug,
		"type":                opportunity.Type,
		"status":              opportunity.Status,
		"participationFormat": opportunity.ParticipationFormat,
		"location":            s.nullableLocation(opportunity.LocationID),
		"coverMediaId":        nullableString(opportunity.CoverMediaID),
		"publishedAt":         nullableTime(opportunity.PublishedAt),
		"expiresAt":           nullableTime(opportunity.ExpiresAt),
		"tags":                s.tagsResponse(opportunity.Tags),
		"createdAt":           timestamp(opportunity.CreatedAt),
		"updatedAt":           timestamp(opportunity.UpdatedAt),
	}
}

func (s *Server) opportunityVacancyPreviewResponse(details *model.OpportunityVacancyDetails) any {
	if details == nil {
		return nil
	}
	return map[string]any{
		"employmentType":  details.EmploymentType,
		"experienceLevel": details.ExperienceLevel,
		"salaryFrom":      nullableInt(details.SalaryFrom),
		"salaryTo":        nullableInt(details.SalaryTo),
		"currency":        nullableString(details.Currency),
	}
}

func (s *Server) opportunityVacancyDetailResponse(details *model.OpportunityVacancyDetails) any {
	return s.opportunityVacancyPreviewResponse(details)
}

func (s *Server) opportunityMentorProgramPreviewResponse(details *model.OpportunityMentorProgramDetails) any {
	if details == nil {
		return nil
	}
	return map[string]any{
		"startAt":    nullableTime(details.StartAt),
		"endAt":      nullableTime(details.EndAt),
		"seatsCount": nullableInt(details.SeatsCount),
	}
}

func (s *Server) opportunityMentorProgramDetailResponse(details *model.OpportunityMentorProgramDetails) any {
	if details == nil {
		return nil
	}
	return map[string]any{
		"startAt":            nullableTime(details.StartAt),
		"endAt":              nullableTime(details.EndAt),
		"seatsCount":         nullableInt(details.SeatsCount),
		"mentorRequirements": nullableString(details.MentorRequirements),
	}
}

func (s *Server) opportunityEventPreviewResponse(details *model.OpportunityEventDetails) any {
	if details == nil {
		return nil
	}
	return map[string]any{
		"startAt":              timestamp(details.StartAt),
		"endAt":                timestamp(details.EndAt),
		"registrationDeadline": nullableTime(details.RegistrationDeadline),
		"capacity":             nullableInt(details.Capacity),
	}
}

func (s *Server) opportunityEventDetailResponse(details *model.OpportunityEventDetails) any {
	if details == nil {
		return nil
	}
	return map[string]any{
		"startAt":              timestamp(details.StartAt),
		"endAt":                timestamp(details.EndAt),
		"registrationDeadline": nullableTime(details.RegistrationDeadline),
		"capacity":             nullableInt(details.Capacity),
		"venueNote":            nullableString(details.VenueNote),
	}
}

func (s *Server) opportunityLinksResponse(items []model.OpportunityLink) []any {
	response := make([]any, 0, len(items))
	for _, link := range items {
		response = append(response, map[string]any{
			"id":            link.ID,
			"opportunityId": link.OpportunityID,
			"linkType":      link.LinkType,
			"title":         link.Title,
			"url":           link.URL,
			"sortOrder":     link.SortOrder,
		})
	}
	return response
}

func (s *Server) opportunityMediaResponse(items []model.OpportunityMedia) []any {
	response := make([]any, 0, len(items))
	for _, media := range items {
		response = append(response, map[string]any{
			"id":            media.ID,
			"opportunityId": media.OpportunityID,
			"mediaFileId":   media.MediaFileID,
			"title":         nullableString(media.Title),
			"sortOrder":     media.SortOrder,
			"createdAt":     timestamp(media.CreatedAt),
		})
	}
	return response
}

func (s *Server) nullableOpportunityStatsResponse(stats *model.OpportunityStats) any {
	if stats == nil {
		return nil
	}
	return map[string]any{
		"viewsCount": stats.ViewsCount,
		"updatedAt":  timestamp(stats.UpdatedAt),
	}
}

func (s *Server) notificationPreferencesResponse(preferences *model.NotificationPreferences) map[string]any {
	return map[string]any{
		"userId":                   preferences.UserID,
		"inAppEnabled":             preferences.InAppEnabled,
		"emailEnabled":             preferences.EmailEnabled,
		"recommendationEnabled":    preferences.RecommendationEnabled,
		"applicationStatusEnabled": preferences.ApplicationStatusEnabled,
		"employerMessagesEnabled":  preferences.EmployerMessagesEnabled,
		"systemEnabled":            preferences.SystemEnabled,
		"updatedAt":                timestamp(preferences.UpdatedAt),
	}
}

func (s *Server) companiesSummaryResponse(items []*model.Company) []any {
	response := make([]any, 0, len(items))
	for _, company := range items {
		response = append(response, s.companySummaryResponse(company))
	}
	return response
}

func (s *Server) tagsResponse(items []*model.Tag) []any {
	response := make([]any, 0, len(items))
	for _, tag := range items {
		response = append(response, map[string]any{
			"id":              tag.ID,
			"name":            tag.Name,
			"tagType":         tag.TagType,
			"isSystem":        tag.IsSystem,
			"isActive":        tag.IsActive,
			"createdByUserId": nullableString(tag.CreatedByUserID),
			"createdAt":       timestamp(tag.CreatedAt),
		})
	}
	return response
}

func (s *Server) locationsResponse(items []*model.Location) []any {
	response := make([]any, 0, len(items))
	for _, location := range items {
		response = append(response, s.locationResponse(location))
	}
	return response
}

func (s *Server) membershipsResponse(items []*model.CompanyMembership) []any {
	response := make([]any, 0, len(items))
	for _, membership := range items {
		response = append(response, s.membershipResponse(membership))
	}
	return response
}

func (s *Server) nullableLocation(locationID *string) any {
	location := s.store.GetLocationByID(locationID)
	if location == nil {
		return nil
	}
	return s.locationResponse(location)
}

func (s *Server) locationResponse(location *model.Location) map[string]any {
	return map[string]any{
		"id":          location.ID,
		"precision":   location.Precision,
		"country":     location.Country,
		"region":      nullableString(location.Region),
		"city":        location.City,
		"addressLine": nullableString(location.AddressLine),
		"postalCode":  nullableString(location.PostalCode),
		"placeName":   nullableString(location.PlaceName),
		"latitude":    nullableFloat(location.Latitude),
		"longitude":   nullableFloat(location.Longitude),
		"createdAt":   timestamp(location.CreatedAt),
	}
}

func buildListEmployerOpportunitiesInput(r *http.Request) (model.ListEmployerOpportunitiesInput, *commonstore.AppError) {
	input := model.ListEmployerOpportunitiesInput{
		CompanyID:        r.URL.Query().Get("companyId"),
		Status:           r.URL.Query().Get("status"),
		ModerationStatus: r.URL.Query().Get("moderationStatus"),
		Page:             queryInt(r, "page", 1),
		PageSize:         queryInt(r, "pageSize", 20),
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

func buildListPublicOpportunitiesInput(r *http.Request) (model.ListPublicOpportunitiesInput, *commonstore.AppError) {
	input := model.ListPublicOpportunitiesInput{
		Page:                queryInt(r, "page", 1),
		PageSize:            queryInt(r, "pageSize", 20),
		Q:                   r.URL.Query().Get("q"),
		Type:                r.URL.Query().Get("type"),
		ParticipationFormat: r.URL.Query().Get("participationFormat"),
		CompanyID:           r.URL.Query().Get("companyId"),
		City:                r.URL.Query().Get("city"),
		TagIDs:              r.URL.Query()["tagIds"],
		Sort:                r.URL.Query().Get("sort"),
		View:                r.URL.Query().Get("view"),
		BBox:                r.URL.Query().Get("bbox"),
	}
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
	input.Lng, err = optionalFloatQuery(r, "lng")
	if err != nil {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid lng parameter"}
	}
	input.RadiusKm, err = optionalFloatQuery(r, "radiusKm")
	if err != nil {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid radiusKm parameter"}
	}

	input.EmploymentType = r.URL.Query().Get("employmentType")
	if input.EmploymentType != "" && !containsString([]string{"full_time", "part_time", "project", "contract"}, input.EmploymentType) {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid employmentType parameter"}
	}
	input.ExperienceLevel = r.URL.Query().Get("experienceLevel")
	if input.ExperienceLevel != "" && !containsString([]string{"trainee", "junior", "middle", "senior"}, input.ExperienceLevel) {
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid experienceLevel parameter"}
	}

	if input.BBox != "" || input.Lat != nil || input.Lng != nil || input.RadiusKm != nil {
		if input.View != "map" {
			return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "map filters require view=map"}
		}
		if input.BBox != "" && (input.Lat != nil || input.Lng != nil || input.RadiusKm != nil) {
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
		return model.ListPublicOpportunitiesInput{}, &commonstore.AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "map geo filters are not implemented yet"}
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
	return model.CreateOpportunityInput{
		CompanyID:            req.CompanyID,
		Title:                req.Title,
		Summary:              req.Summary,
		Slug:                 req.Slug,
		Description:          req.Description,
		Type:                 req.Type,
		ParticipationFormat:  req.ParticipationFormat,
		LocationID:           req.LocationID,
		ContactEmail:         req.ContactEmail,
		ContactPhone:         req.ContactPhone,
		CoverMediaID:         req.CoverMediaID,
		PublishedAt:          publishedAt,
		ExpiresAt:            expiresAt,
		TagIDs:               req.TagIDs,
		VacancyDetails:       vacancyDetails,
		MentorProgramDetails: mentorDetails,
		EventDetails:         eventDetails,
		Links:                toOpportunityLinkInputs(req.Links),
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

	input := model.UpdateOpportunityInput{
		Title:                req.Title,
		Summary:              req.Summary,
		Slug:                 req.Slug,
		Description:          req.Description,
		Status:               req.Status,
		ParticipationFormat:  req.ParticipationFormat,
		LocationID:           req.LocationID,
		ContactEmail:         req.ContactEmail,
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
		input.Links = toOpportunityLinkInputs(*req.Links)
		input.ReplaceLinks = true
	}
	if req.Media != nil {
		input.Media = toOpportunityMediaInputs(*req.Media)
		input.ReplaceMedia = true
	}
	return input, nil
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

func toOpportunityLinkInputs(items []opportunityLinkInputRequest) []model.OpportunityLinkInput {
	response := make([]model.OpportunityLinkInput, 0, len(items))
	for _, item := range items {
		sortOrder := 0
		if item.SortOrder != nil {
			sortOrder = *item.SortOrder
		}
		response = append(response, model.OpportunityLinkInput{
			LinkType:  item.LinkType,
			Title:     item.Title,
			URL:       item.URL,
			SortOrder: sortOrder,
		})
	}
	return response
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

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func queryInt(r *http.Request, name string, fallback int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
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

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}

func timestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func toLocationInput(req *locationInputRequest) *model.LocationInput {
	if req == nil {
		return nil
	}
	return &model.LocationInput{
		Precision:   req.Precision,
		Country:     req.Country,
		Region:      req.Region,
		City:        req.City,
		AddressLine: req.AddressLine,
		PostalCode:  req.PostalCode,
		PlaceName:   req.PlaceName,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
	}
}
