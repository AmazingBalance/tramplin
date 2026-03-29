package http

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"

	"tramplin/backend/internal/config"
	"tramplin/backend/internal/domain/model"
	authplatform "tramplin/backend/internal/platform/auth"
	"tramplin/backend/internal/platform/httpx"
	objectstoreplatform "tramplin/backend/internal/platform/objectstore"
	postgresplatform "tramplin/backend/internal/platform/postgres"
	commonstore "tramplin/backend/internal/store"
	pgstore "tramplin/backend/internal/store/postgres"
)

type Server struct {
	cfg     config.Config
	db      *pgxpool.Pool
	store   commonstore.Repository
	objects objectstoreplatform.Store
	mux     *http.ServeMux
}

type ServerOption func(*Server)

func WithObjectStore(store objectstoreplatform.Store) ServerOption {
	return func(s *Server) {
		s.objects = store
	}
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

type createPresignedUploadRequest struct {
	OriginalName string  `json:"originalName"`
	MimeType     string  `json:"mimeType"`
	FileSize     int64   `json:"fileSize"`
	Purpose      *string `json:"purpose"`
}

type completeUploadRequest struct {
	ETag *string `json:"etag"`
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

type createApplicantSocialLinkRequest struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
	IsPublic *bool  `json:"isPublic"`
}

type updateApplicantSocialLinkRequest struct {
	Platform *string `json:"platform"`
	URL      *string `json:"url"`
	IsPublic *bool   `json:"isPublic"`
}

type saveOpportunityRequest struct {
	OpportunityID string `json:"opportunityId"`
}

type saveCompanyRequest struct {
	CompanyID string `json:"companyId"`
}

type updateEmployerProfileRequest struct {
	FullName *string `json:"fullName"`
	JobTitle *string `json:"jobTitle"`
	Phone    *string `json:"phone"`
}

type updateCuratorUserRequest struct {
	DisplayName *string `json:"displayName"`
	IsActive    *bool   `json:"isActive"`
	Reason      string  `json:"reason"`
}

type updateCuratorApplicantRequest struct {
	FirstName      *string                        `json:"firstName"`
	LastName       *string                        `json:"lastName"`
	MiddleName     *string                        `json:"middleName"`
	UniversityName *string                        `json:"universityName"`
	Faculty        *string                        `json:"faculty"`
	ProgramName    *string                        `json:"programName"`
	StudyYear      *int                           `json:"studyYear"`
	GraduationYear *int                           `json:"graduationYear"`
	City           *string                        `json:"city"`
	About          *string                        `json:"about"`
	ResumeMediaID  *string                        `json:"resumeMediaId"`
	Privacy        *updateApplicantPrivacyRequest `json:"privacy"`
	Reason         string                         `json:"reason"`
}

type updateCuratorEmployerRequest struct {
	FullName *string `json:"fullName"`
	JobTitle *string `json:"jobTitle"`
	Phone    *string `json:"phone"`
	Reason   string  `json:"reason"`
}

type createCuratorRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
	FullName    string `json:"fullName"`
	Reason      string `json:"reason"`
}

type updateCuratorAccountRequest struct {
	DisplayName *string `json:"displayName"`
	FullName    *string `json:"fullName"`
	IsActive    *bool   `json:"isActive"`
	Reason      string  `json:"reason"`
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

type updateCuratorCompanyRequest struct {
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
	Reason                 string                `json:"reason"`
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

type createCompanySocialLinkRequest struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

type updateCompanySocialLinkRequest struct {
	Platform *string `json:"platform"`
	URL      *string `json:"url"`
}

type createCompanyMediaRequest struct {
	MediaFileID string  `json:"mediaFileId"`
	Title       *string `json:"title"`
	SortOrder   *int    `json:"sortOrder"`
}

type updateCompanyMediaRequest struct {
	Title     *string `json:"title"`
	SortOrder *int    `json:"sortOrder"`
}

type approveCompanyMembershipRequest struct {
	Comment *string `json:"comment"`
}

type commentRequest struct {
	Comment string `json:"comment"`
}

type createTagRequest struct {
	Name     string `json:"name"`
	TagType  string `json:"tagType"`
	IsActive *bool  `json:"isActive"`
}

type updateTagRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"isActive"`
}

type createModerationCaseRequest struct {
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
	Reason     string `json:"reason"`
}

type updateModerationCaseRequest struct {
	AssignedCuratorUserID *string `json:"assignedCuratorUserId"`
	Status                *string `json:"status"`
	Reason                string  `json:"reason"`
}

type createApplicationRequest struct {
	OpportunityID string  `json:"opportunityId"`
	CoverLetter   *string `json:"coverLetter"`
}

type updateApplicationStatusRequest struct {
	Status  string  `json:"status"`
	Comment *string `json:"comment"`
}

type createConnectionRequest struct {
	TargetApplicantUserID string  `json:"targetApplicantUserId"`
	InitiatorNote         *string `json:"initiatorNote"`
}

type updateConnectionRequest struct {
	Status string `json:"status"`
}

type createOpportunityRecommendationRequest struct {
	RecipientUserID string  `json:"recipientUserId"`
	OpportunityID   string  `json:"opportunityId"`
	Message         *string `json:"message"`
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

type updateCuratorOpportunityRequest struct {
	Title                *string                                 `json:"title"`
	Summary              *string                                 `json:"summary"`
	Slug                 *string                                 `json:"slug"`
	Description          *string                                 `json:"description"`
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
	Reason               string                                  `json:"reason"`
}

type patchNotificationPreferencesRequest struct {
	InAppEnabled             *bool `json:"inAppEnabled"`
	EmailEnabled             *bool `json:"emailEnabled"`
	RecommendationEnabled    *bool `json:"recommendationEnabled"`
	ApplicationStatusEnabled *bool `json:"applicationStatusEnabled"`
	EmployerMessagesEnabled  *bool `json:"employerMessagesEnabled"`
	SystemEnabled            *bool `json:"systemEnabled"`
}

type createNotificationCampaignRequest struct {
	CompanyID        string   `json:"companyId"`
	OpportunityID    *string  `json:"opportunityId"`
	AudienceType     string   `json:"audienceType"`
	Title            string   `json:"title"`
	Body             string   `json:"body"`
	SendViaInApp     *bool    `json:"sendViaInApp"`
	SendViaEmail     *bool    `json:"sendViaEmail"`
	ScheduledAt      *string  `json:"scheduledAt"`
	ApplicantUserIDs []string `json:"applicantUserIds"`
}

type verificationEvidenceInputRequest struct {
	EvidenceType   string  `json:"evidenceType"`
	Value          *string `json:"value"`
	EvidenceFileID *string `json:"evidenceFileId"`
}

type createVerificationRequestRequest struct {
	Method           string                             `json:"method"`
	SubmittedComment *string                            `json:"submittedComment"`
	Evidence         []verificationEvidenceInputRequest `json:"evidence"`
}

type reviewVerificationRequestRequest struct {
	Status        string  `json:"status"`
	ReviewComment *string `json:"reviewComment"`
}

func NewServer(cfg config.Config, db *pgxpool.Pool, opts ...ServerOption) http.Handler {
	server := &Server{
		cfg:   cfg,
		db:    db,
		store: pgstore.New(db),
		mux:   http.NewServeMux(),
	}
	for _, opt := range opts {
		opt(server)
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
	s.mux.HandleFunc("POST /uploads/presign", s.handleCreatePresignedUpload)
	s.mux.HandleFunc("POST /uploads/{mediaFileId}/complete", s.handleCompleteUpload)
	s.mux.HandleFunc("GET /media/{mediaFileId}", s.handleGetMediaFile)
	s.mux.HandleFunc("DELETE /media/{mediaFileId}", s.handleDeleteMediaFile)
	s.mux.HandleFunc("POST /media/{mediaFileId}/download-url", s.handleCreateMediaDownloadURL)

	s.mux.HandleFunc("GET /me/ui-settings", s.handleGetUISettings)
	s.mux.HandleFunc("PUT /me/ui-settings", s.handlePutUISettings)

	s.mux.HandleFunc("GET /me/applicant-profile", s.handleGetApplicantProfile)
	s.mux.HandleFunc("PATCH /me/applicant-profile", s.handlePatchApplicantProfile)
	s.mux.HandleFunc("GET /me/applicant/privacy", s.handleGetApplicantPrivacy)
	s.mux.HandleFunc("PATCH /me/applicant/privacy", s.handlePatchApplicantPrivacy)
	s.mux.HandleFunc("GET /me/applicant/tags", s.handleListApplicantTags)
	s.mux.HandleFunc("PUT /me/applicant/tags", s.handleReplaceApplicantTags)
	s.mux.HandleFunc("GET /me/applicant/social-links", s.handleListMyApplicantSocialLinks)
	s.mux.HandleFunc("POST /me/applicant/social-links", s.handleCreateMyApplicantSocialLink)
	s.mux.HandleFunc("PATCH /me/applicant/social-links/{linkId}", s.handlePatchMyApplicantSocialLink)
	s.mux.HandleFunc("DELETE /me/applicant/social-links/{linkId}", s.handleDeleteMyApplicantSocialLink)
	s.mux.HandleFunc("GET /applicants/{userId}", s.handleGetApplicantProfileByID)
	s.mux.HandleFunc("GET /me/saved-opportunities", s.handleListSavedOpportunities)
	s.mux.HandleFunc("POST /me/saved-opportunities", s.handleSaveOpportunity)
	s.mux.HandleFunc("DELETE /me/saved-opportunities/{opportunityId}", s.handleDeleteSavedOpportunity)
	s.mux.HandleFunc("GET /me/saved-companies", s.handleListSavedCompanies)
	s.mux.HandleFunc("POST /me/saved-companies", s.handleSaveCompany)
	s.mux.HandleFunc("DELETE /me/saved-companies/{companyId}", s.handleDeleteSavedCompany)
	s.mux.HandleFunc("GET /me/connections", s.handleListMyConnections)
	s.mux.HandleFunc("POST /me/connections", s.handleCreateConnection)
	s.mux.HandleFunc("PATCH /me/connections/{connectionId}", s.handlePatchConnection)
	s.mux.HandleFunc("POST /me/recommendations", s.handleCreateOpportunityRecommendation)
	s.mux.HandleFunc("GET /me/recommendations/received", s.handleListReceivedRecommendations)
	s.mux.HandleFunc("GET /me/recommendations/sent", s.handleListSentRecommendations)
	s.mux.HandleFunc("GET /me/applications", s.handleListMyApplications)
	s.mux.HandleFunc("POST /me/applications", s.handleCreateApplication)
	s.mux.HandleFunc("GET /me/applications/{applicationId}", s.handleGetMyApplication)
	s.mux.HandleFunc("POST /me/applications/{applicationId}/withdraw", s.handleWithdrawMyApplication)

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
	s.mux.HandleFunc("GET /employer/companies/{companyId}/social-links", s.handleListEmployerCompanySocialLinks)
	s.mux.HandleFunc("POST /employer/companies/{companyId}/social-links", s.handleCreateEmployerCompanySocialLink)
	s.mux.HandleFunc("PATCH /employer/companies/{companyId}/social-links/{linkId}", s.handlePatchEmployerCompanySocialLink)
	s.mux.HandleFunc("DELETE /employer/companies/{companyId}/social-links/{linkId}", s.handleDeleteEmployerCompanySocialLink)
	s.mux.HandleFunc("GET /employer/companies/{companyId}/media", s.handleListEmployerCompanyMedia)
	s.mux.HandleFunc("POST /employer/companies/{companyId}/media", s.handleCreateEmployerCompanyMedia)
	s.mux.HandleFunc("PATCH /employer/companies/{companyId}/media/{mediaId}", s.handlePatchEmployerCompanyMedia)
	s.mux.HandleFunc("DELETE /employer/companies/{companyId}/media/{mediaId}", s.handleDeleteEmployerCompanyMedia)
	s.mux.HandleFunc("GET /employer/companies/{companyId}/verification-requests", s.handleListCompanyVerificationRequests)
	s.mux.HandleFunc("POST /employer/companies/{companyId}/verification-requests", s.handleCreateCompanyVerificationRequest)
	s.mux.HandleFunc("GET /employer/applicants/{userId}", s.handleGetEmployerApplicantProfile)
	s.mux.HandleFunc("GET /employer/opportunities", s.handleListEmployerOpportunities)
	s.mux.HandleFunc("POST /employer/opportunities", s.handleCreateOpportunity)
	s.mux.HandleFunc("GET /employer/opportunities/{opportunityId}", s.handleGetEmployerOpportunity)
	s.mux.HandleFunc("PATCH /employer/opportunities/{opportunityId}", s.handlePatchEmployerOpportunity)
	s.mux.HandleFunc("POST /employer/opportunities/{opportunityId}/activate", s.handleActivateEmployerOpportunity)
	s.mux.HandleFunc("POST /employer/opportunities/{opportunityId}/close", s.handleCloseEmployerOpportunity)
	s.mux.HandleFunc("POST /employer/opportunities/{opportunityId}/archive", s.handleArchiveEmployerOpportunity)
	s.mux.HandleFunc("GET /employer/opportunities/{opportunityId}/applications", s.handleListOpportunityApplications)
	s.mux.HandleFunc("PATCH /employer/applications/{applicationId}/status", s.handlePatchApplicationStatus)
	s.mux.HandleFunc("GET /employer/notification-campaigns", s.handleListNotificationCampaigns)
	s.mux.HandleFunc("POST /employer/notification-campaigns", s.handleCreateNotificationCampaign)
	s.mux.HandleFunc("GET /employer/notification-campaigns/{campaignId}", s.handleGetNotificationCampaign)
	s.mux.HandleFunc("PATCH /employer/notification-campaigns/{campaignId}", s.handlePatchNotificationCampaign)
	s.mux.HandleFunc("POST /employer/notification-campaigns/{campaignId}/send", s.handleSendNotificationCampaign)
	s.mux.HandleFunc("POST /employer/notification-campaigns/{campaignId}/cancel", s.handleCancelNotificationCampaign)
	s.mux.HandleFunc("POST /employer/tags", s.handleCreateEmployerTag)

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

	s.mux.HandleFunc("GET /curator/verification-requests", s.handleListCuratorVerificationRequests)
	s.mux.HandleFunc("PATCH /curator/verification-requests/{verificationRequestId}/review", s.handleReviewVerificationRequest)
	s.mux.HandleFunc("GET /curator/moderation-cases", s.handleListModerationCases)
	s.mux.HandleFunc("POST /curator/moderation-cases", s.handleCreateModerationCase)
	s.mux.HandleFunc("PATCH /curator/moderation-cases/{moderationCaseId}", s.handlePatchModerationCase)
	s.mux.HandleFunc("GET /curator/tags", s.handleListCuratorTags)
	s.mux.HandleFunc("POST /curator/tags", s.handleCreateCuratorTag)
	s.mux.HandleFunc("PATCH /curator/tags/{tagId}", s.handlePatchCuratorTag)
	s.mux.HandleFunc("GET /curator/users", s.handleListCuratorUsers)
	s.mux.HandleFunc("GET /curator/users/{userId}", s.handleGetCuratorUser)
	s.mux.HandleFunc("PATCH /curator/users/{userId}", s.handlePatchCuratorUser)
	s.mux.HandleFunc("GET /curator/applicants/{userId}", s.handleGetCuratorApplicant)
	s.mux.HandleFunc("PATCH /curator/applicants/{userId}", s.handlePatchCuratorApplicant)
	s.mux.HandleFunc("GET /curator/employers/{userId}", s.handleGetCuratorEmployer)
	s.mux.HandleFunc("PATCH /curator/employers/{userId}", s.handlePatchCuratorEmployer)
	s.mux.HandleFunc("GET /curator/companies/{companyId}", s.handleGetCuratorCompany)
	s.mux.HandleFunc("PATCH /curator/companies/{companyId}", s.handlePatchCuratorCompany)
	s.mux.HandleFunc("GET /curator/opportunities/{opportunityId}", s.handleGetCuratorOpportunity)
	s.mux.HandleFunc("PATCH /curator/opportunities/{opportunityId}", s.handlePatchCuratorOpportunity)
	s.mux.HandleFunc("GET /curator/admin/curators", s.handleListAdminCurators)
	s.mux.HandleFunc("POST /curator/admin/curators", s.handleCreateAdminCurator)
	s.mux.HandleFunc("PATCH /curator/admin/curators/{userId}", s.handlePatchAdminCurator)

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
	email, appErr := parseRequiredEmail(req.Email, "email")
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if utf8.RuneCountInString(req.Password) < 8 {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "password must be at least 8 characters", nil)
		return
	}
	hash, err := authplatform.HashPassword(req.Password)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to hash password", nil)
		return
	}
	user, appErr := s.store.CreateApplicant(model.RegisterApplicantInput{
		Email:       email,
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
	response, appErr := s.authUserResponse(user)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, response)
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
	email, appErr := parseRequiredEmail(req.Email, "email")
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if utf8.RuneCountInString(req.Password) < 8 {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "password must be at least 8 characters", nil)
		return
	}
	hash, err := authplatform.HashPassword(req.Password)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to hash password", nil)
		return
	}
	user, appErr := s.store.CreateEmployer(model.RegisterEmployerInput{
		Email:       email,
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
	response, appErr := s.authUserResponse(user)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, response)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "required fields are missing", nil)
		return
	}
	email, appErr := parseRequiredEmail(req.Email, "email")
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	user, appErr := s.store.GetUserByEmail(email)
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
	response, appErr := s.authUserResponse(user)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireRefreshUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	s.writeAuthCookies(w, user)
	response, appErr := s.authUserResponse(user)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response)
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
	response, appErr := s.currentUserResponse(user)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreatePresignedUpload(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if s.objects == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "object storage is not configured", nil)
		return
	}

	var req createPresignedUploadRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}

	file, repoErr := s.store.CreatePresignedUpload(user.ID, model.CreatePresignedUploadInput{
		OriginalName: req.OriginalName,
		MimeType:     req.MimeType,
		FileSize:     req.FileSize,
		Purpose:      req.Purpose,
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}

	upload, err := s.objects.PresignUpload(r.Context(), file.ObjectKey, req.MimeType, s.cfg.UploadURLTTL)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create upload url", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"uploadUrl":       s.externalObjectURL(upload.URL),
		"method":          upload.Method,
		"headers":         upload.Headers,
		"uploadExpiresAt": timestamp(upload.ExpiresAt),
		"file":            s.mediaFileResponse(file),
	})
}

func (s *Server) handleCompleteUpload(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if s.objects == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "object storage is not configured", nil)
		return
	}

	var req completeUploadRequest
	if err := decodeOptionalJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}

	file, repoErr := s.store.GetMediaFile(user.ID, r.PathValue("mediaFileId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}

	info, err := s.objects.StatObject(r.Context(), file.ObjectKey)
	if err != nil {
		if errors.Is(err, objectstoreplatform.ErrObjectNotFound) {
			httpx.WriteError(w, http.StatusConflict, "conflict", "file was not uploaded or cannot be finalized", nil)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to finalize upload", nil)
		return
	}

	file, repoErr = s.store.CompleteUpload(user.ID, file.ID, model.CompleteUploadInput{
		ETag: req.ETag,
	}, &info.ETag)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, s.mediaFileResponse(file))
}

func (s *Server) handleGetMediaFile(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	file, repoErr := s.store.GetMediaFile(user.ID, r.PathValue("mediaFileId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.mediaFileResponse(file))
}

func (s *Server) handleDeleteMediaFile(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	file, repoErr := s.store.DeleteMediaFile(user.ID, r.PathValue("mediaFileId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	if s.objects != nil {
		_ = s.objects.DeleteObject(r.Context(), file.ObjectKey)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateMediaDownloadURL(w http.ResponseWriter, r *http.Request) {
	if s.objects == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "object storage is not configured", nil)
		return
	}

	var actorUserID *string
	if user := s.maybeAccessUser(r); user != nil {
		userID := user.ID
		actorUserID = &userID
	}

	file, repoErr := s.store.GetMediaFileForDownload(actorUserID, r.PathValue("mediaFileId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}

	download, err := s.objects.PresignDownload(r.Context(), file.ObjectKey, s.cfg.DownloadURLTTL)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create download url", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"downloadUrl": s.externalObjectURL(download.URL),
		"expiresAt":   timestamp(download.ExpiresAt),
	})
}

func (s *Server) externalObjectURL(raw string) string {
	if raw == "" || s.cfg.ObjectStoragePublicURL == "" {
		return raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	publicURL := s.cfg.ObjectStoragePublicURL
	if strings.Contains(publicURL, "://") {
		publicParsed, err := url.Parse(publicURL)
		if err != nil || publicParsed.Host == "" {
			return raw
		}
		parsed.Host = publicParsed.Host
		parsed.Scheme = publicParsed.Scheme
		return parsed.String()
	}

	parsed.Host = publicURL
	if s.cfg.ObjectStoragePublicSSL {
		parsed.Scheme = "https"
	} else {
		parsed.Scheme = "http"
	}
	return parsed.String()
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

func (s *Server) handleGetApplicantProfileByID(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	view, repoErr := s.store.GetApplicantProfileView(user.ID, r.PathValue("userId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.applicantProfileViewResponse(view))
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

func (s *Server) handleListMyApplicantSocialLinks(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}
	links, repoErr := s.store.ListApplicantSocialLinks(user.ID)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.applicantSocialLinksResponse(links),
	})
}

func (s *Server) handleCreateMyApplicantSocialLink(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}

	var req createApplicantSocialLinkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	linkURL, repoErr := parseRequiredURI(req.URL, "url")
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}

	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}
	link, repoErr := s.store.CreateApplicantSocialLink(user.ID, model.CreateApplicantSocialLinkInput{
		Platform: req.Platform,
		URL:      linkURL,
		IsPublic: isPublic,
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.applicantSocialLinkResponse(link))
}

func (s *Server) handlePatchMyApplicantSocialLink(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}

	var req updateApplicantSocialLinkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	if req.Platform == nil && req.URL == nil && req.IsPublic == nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "at least one field is required", nil)
		return
	}
	linkURL, repoErr := parseOptionalURI(req.URL, "url")
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}

	link, repoErr := s.store.UpdateApplicantSocialLink(user.ID, r.PathValue("linkId"), model.UpdateApplicantSocialLinkInput{
		Platform: req.Platform,
		URL:      linkURL,
		IsPublic: req.IsPublic,
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.applicantSocialLinkResponse(link))
}

func (s *Server) handleDeleteMyApplicantSocialLink(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}

	if repoErr := s.store.DeleteApplicantSocialLink(user.ID, r.PathValue("linkId")); repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListSavedOpportunities(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}
	items, repoErr := s.store.ListSavedOpportunities(user.ID)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.savedOpportunitiesResponse(items),
	})
}

func (s *Server) handleSaveOpportunity(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}

	var req saveOpportunityRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	if repoErr := s.store.SaveOpportunity(user.ID, model.SaveOpportunityInput{OpportunityID: req.OpportunityID}); repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleDeleteSavedOpportunity(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}
	if repoErr := s.store.DeleteSavedOpportunity(user.ID, r.PathValue("opportunityId")); repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListSavedCompanies(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}
	items, repoErr := s.store.ListSavedCompanies(user.ID)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.savedCompaniesResponse(items),
	})
}

func (s *Server) handleSaveCompany(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}

	var req saveCompanyRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	if repoErr := s.store.SaveCompany(user.ID, model.SaveCompanyInput{CompanyID: req.CompanyID}); repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleDeleteSavedCompany(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}
	if repoErr := s.store.DeleteSavedCompany(user.ID, r.PathValue("companyId")); repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListMyConnections(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}

	input, repoErr := buildListConnectionsInput(r)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	items, repoErr := s.store.ListConnections(user.ID, input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.connectionsResponse(items),
	})
}

func (s *Server) handleCreateConnection(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}

	var req createConnectionRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	connection, repoErr := s.store.CreateConnection(user.ID, model.CreateConnectionInput{
		TargetApplicantUserID: req.TargetApplicantUserID,
		InitiatorNote:         req.InitiatorNote,
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.connectionResponse(connection))
}

func (s *Server) handlePatchConnection(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}

	var req updateConnectionRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	if req.Status == "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "status is required", nil)
		return
	}

	connection, repoErr := s.store.UpdateConnection(user.ID, r.PathValue("connectionId"), model.UpdateConnectionInput{
		Status: req.Status,
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.connectionResponse(connection))
}

func (s *Server) handleCreateOpportunityRecommendation(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}

	var req createOpportunityRecommendationRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	recommendation, repoErr := s.store.CreateOpportunityRecommendation(user.ID, model.CreateOpportunityRecommendationInput{
		RecipientUserID: req.RecipientUserID,
		OpportunityID:   req.OpportunityID,
		Message:         req.Message,
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.opportunityRecommendationResponse(recommendation))
}

func (s *Server) handleListReceivedRecommendations(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}
	items, repoErr := s.store.ListReceivedRecommendations(user.ID)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.opportunityRecommendationsResponse(items),
	})
}

func (s *Server) handleListSentRecommendations(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}
	items, repoErr := s.store.ListSentRecommendations(user.ID)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.opportunityRecommendationsResponse(items),
	})
}

func (s *Server) handleListMyApplications(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}
	input, repoErr := buildListApplicationsInput(r)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	items, total, repoErr := s.store.ListApplicantApplications(user.ID, input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.myApplicationsResponse(items),
		"meta":  paginationMeta(input.Page, input.PageSize, total),
	})
}

func (s *Server) handleCreateApplication(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "only applicants can apply or opportunity is not available for applying", nil)
		return
	}

	var req createApplicationRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}

	application, repoErr := s.store.CreateApplication(user.ID, model.CreateApplicationInput{
		OpportunityID: req.OpportunityID,
		CoverLetter:   req.CoverLetter,
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.applicationDetailResponse(application))
}

func (s *Server) handleGetMyApplication(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}

	application, repoErr := s.store.GetApplicantApplication(user.ID, r.PathValue("applicationId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.applicationDetailResponse(application))
}

func (s *Server) handleWithdrawMyApplication(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleApplicant {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an applicant", nil)
		return
	}

	application, repoErr := s.store.WithdrawApplicantApplication(user.ID, r.PathValue("applicationId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.applicationDetailResponse(application))
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
	websiteURL, appErr := parseOptionalURI(req.WebsiteURL, "websiteUrl")
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	company, _, appErr := s.store.CreateCompany(user.ID, model.CreateCompanyInput{
		LegalName:              req.LegalName,
		BrandName:              req.BrandName,
		Slug:                   req.Slug,
		INN:                    req.INN,
		Description:            req.Description,
		Industry:               req.Industry,
		WebsiteURL:             websiteURL,
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
	websiteURL, appErr := parseOptionalURI(req.WebsiteURL, "websiteUrl")
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	company, appErr := s.store.UpdateCompany(user.ID, r.PathValue("companyId"), model.UpdateCompanyInput{
		LegalName:              req.LegalName,
		BrandName:              req.BrandName,
		Slug:                   req.Slug,
		INN:                    req.INN,
		Description:            req.Description,
		Industry:               req.Industry,
		WebsiteURL:             websiteURL,
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
	employerEmail, appErr := parseRequiredEmail(req.EmployerEmail, "employerEmail")
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	membership, appErr := s.store.CreateCompanyMembership(user.ID, r.PathValue("companyId"), model.CreateCompanyMembershipInput{
		EmployerEmail:    employerEmail,
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

func (s *Server) handleListEmployerCompanySocialLinks(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	links, repoErr := s.store.ListCompanySocialLinks(user.ID, r.PathValue("companyId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.companySocialLinksPointersResponse(links),
	})
}

func (s *Server) handleCreateEmployerCompanySocialLink(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req createCompanySocialLinkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	linkURL, repoErr := parseRequiredURI(req.URL, "url")
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	link, repoErr := s.store.CreateCompanySocialLink(user.ID, r.PathValue("companyId"), model.CreateCompanySocialLinkInput{
		Platform: req.Platform,
		URL:      linkURL,
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.companySocialLinkPointerResponse(link))
}

func (s *Server) handlePatchEmployerCompanySocialLink(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req updateCompanySocialLinkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	if req.Platform == nil && req.URL == nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "at least one field is required", nil)
		return
	}
	linkURL, repoErr := parseOptionalURI(req.URL, "url")
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}

	link, repoErr := s.store.UpdateCompanySocialLink(user.ID, r.PathValue("companyId"), r.PathValue("linkId"), model.UpdateCompanySocialLinkInput{
		Platform: req.Platform,
		URL:      linkURL,
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.companySocialLinkPointerResponse(link))
}

func (s *Server) handleDeleteEmployerCompanySocialLink(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if repoErr := s.store.DeleteCompanySocialLink(user.ID, r.PathValue("companyId"), r.PathValue("linkId")); repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListEmployerCompanyMedia(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	items, repoErr := s.store.ListCompanyMedia(user.ID, r.PathValue("companyId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.companyMediaPointersResponse(items),
	})
}

func (s *Server) handleCreateEmployerCompanyMedia(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req createCompanyMediaRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}
	media, repoErr := s.store.CreateCompanyMedia(user.ID, r.PathValue("companyId"), model.CreateCompanyMediaInput{
		MediaFileID: req.MediaFileID,
		Title:       req.Title,
		SortOrder:   sortOrder,
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.companyMediaPointerResponse(media))
}

func (s *Server) handlePatchEmployerCompanyMedia(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	var req updateCompanyMediaRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	if req.Title == nil && req.SortOrder == nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "at least one field is required", nil)
		return
	}

	media, repoErr := s.store.UpdateCompanyMedia(user.ID, r.PathValue("companyId"), r.PathValue("mediaId"), model.UpdateCompanyMediaInput{
		Title:     req.Title,
		SortOrder: req.SortOrder,
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.companyMediaPointerResponse(media))
}

func (s *Server) handleDeleteEmployerCompanyMedia(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if repoErr := s.store.DeleteCompanyMedia(user.ID, r.PathValue("companyId"), r.PathValue("mediaId")); repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetEmployerApplicantProfile(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	view, repoErr := s.store.GetEmployerApplicantProfileView(user.ID, r.PathValue("userId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.applicantProfileViewResponse(view))
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

func (s *Server) handleActivateEmployerOpportunity(w http.ResponseWriter, r *http.Request) {
	s.handleEmployerOpportunityLifecycleAction(w, r, model.OpportunityStatusActive)
}

func (s *Server) handleCloseEmployerOpportunity(w http.ResponseWriter, r *http.Request) {
	s.handleEmployerOpportunityLifecycleAction(w, r, model.OpportunityStatusClosed)
}

func (s *Server) handleArchiveEmployerOpportunity(w http.ResponseWriter, r *http.Request) {
	s.handleEmployerOpportunityLifecycleAction(w, r, model.OpportunityStatusArchived)
}

func (s *Server) handleEmployerOpportunityLifecycleAction(w http.ResponseWriter, r *http.Request, targetStatus string) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	opportunity, repoErr := s.store.UpdateOpportunity(user.ID, r.PathValue("opportunityId"), model.UpdateOpportunityInput{
		Status: &targetStatus,
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.employerOpportunityDetailResponse(opportunity))
}

func (s *Server) handleListOpportunityApplications(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleEmployer {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an employer", nil)
		return
	}

	input, repoErr := buildListApplicationsInput(r)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	items, total, repoErr := s.store.ListOpportunityApplications(user.ID, r.PathValue("opportunityId"), input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.employerApplicationsResponse(items),
		"meta":  paginationMeta(input.Page, input.PageSize, total),
	})
}

func (s *Server) handlePatchApplicationStatus(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleEmployer {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an employer", nil)
		return
	}

	var req updateApplicationStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	if req.Status == "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "status is required", nil)
		return
	}

	application, repoErr := s.store.UpdateApplicationStatus(user.ID, r.PathValue("applicationId"), model.UpdateApplicationStatusInput{
		Status:  req.Status,
		Comment: req.Comment,
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.employerApplicationDetailResponse(application))
}

func (s *Server) handleListNotificationCampaigns(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleEmployer {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an employer", nil)
		return
	}

	input, repoErr := buildListNotificationCampaignsInput(r)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	items, total, repoErr := s.store.ListNotificationCampaigns(user.ID, input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.notificationCampaignsResponse(items),
		"meta":  paginationMeta(input.Page, input.PageSize, total),
	})
}

func (s *Server) handleCreateNotificationCampaign(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleEmployer {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an employer", nil)
		return
	}

	var req createNotificationCampaignRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	input, repoErr := parseCreateNotificationCampaignInput(req)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	campaign, repoErr := s.store.CreateNotificationCampaign(user.ID, input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.notificationCampaignDetailResponse(campaign))
}

func (s *Server) handleGetNotificationCampaign(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleEmployer {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an employer", nil)
		return
	}

	campaign, repoErr := s.store.GetNotificationCampaign(user.ID, r.PathValue("campaignId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.notificationCampaignDetailResponse(campaign))
}

func (s *Server) handlePatchNotificationCampaign(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleEmployer {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an employer", nil)
		return
	}

	input, repoErr := parseUpdateNotificationCampaignInput(r)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	campaign, repoErr := s.store.UpdateNotificationCampaign(user.ID, r.PathValue("campaignId"), input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.notificationCampaignDetailResponse(campaign))
}

func (s *Server) handleSendNotificationCampaign(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleEmployer {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an employer", nil)
		return
	}

	campaign, repoErr := s.store.SendNotificationCampaign(user.ID, r.PathValue("campaignId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.notificationCampaignDetailResponse(campaign))
}

func (s *Server) handleCancelNotificationCampaign(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleEmployer {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an employer", nil)
		return
	}

	campaign, repoErr := s.store.CancelNotificationCampaign(user.ID, r.PathValue("campaignId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.notificationCampaignDetailResponse(campaign))
}

func (s *Server) handleCreateEmployerTag(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleEmployer {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an employer", nil)
		return
	}

	var req createTagRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}

	tag, repoErr := s.store.CreateEmployerTag(user.ID, parseCreateTagInput(req, true))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.tagsResponse([]*model.Tag{tag})[0])
}

func (s *Server) handleListCompanyVerificationRequests(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleEmployer {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an employer", nil)
		return
	}

	items, repoErr := s.store.ListCompanyVerificationRequests(user.ID, r.PathValue("companyId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.verificationRequestsResponse(items),
	})
}

func (s *Server) handleCreateCompanyVerificationRequest(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleEmployer {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not an employer", nil)
		return
	}

	var req createVerificationRequestRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	input, repoErr := parseCreateVerificationRequestInput(req)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}

	request, repoErr := s.store.CreateCompanyVerificationRequest(user.ID, r.PathValue("companyId"), input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.verificationRequestResponse(request))
}

func (s *Server) handleListCuratorVerificationRequests(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	input, repoErr := buildListVerificationRequestsInput(r)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	items, total, repoErr := s.store.ListCuratorVerificationRequests(user.ID, input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.verificationRequestsResponse(items),
		"meta":  paginationMeta(input.Page, input.PageSize, total),
	})
}

func (s *Server) handleReviewVerificationRequest(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	var req reviewVerificationRequestRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}

	request, repoErr := s.store.ReviewVerificationRequest(user.ID, r.PathValue("verificationRequestId"), parseReviewVerificationRequestInput(req))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.verificationRequestResponse(request))
}

func (s *Server) handleListModerationCases(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	input, repoErr := buildListModerationCasesInput(r)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	items, total, repoErr := s.store.ListModerationCases(user.ID, input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.moderationCasesResponse(items),
		"meta":  paginationMeta(input.Page, input.PageSize, total),
	})
}

func (s *Server) handleCreateModerationCase(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	var req createModerationCaseRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	item, repoErr := s.store.CreateModerationCase(user.ID, model.CreateModerationCaseInput{
		TargetType: strings.TrimSpace(req.TargetType),
		TargetID:   strings.TrimSpace(req.TargetID),
		Reason:     strings.TrimSpace(req.Reason),
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.moderationCaseResponse(item))
}

func (s *Server) handlePatchModerationCase(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	input, repoErr := parseUpdateModerationCaseInput(r)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	item, repoErr := s.store.UpdateModerationCase(user.ID, r.PathValue("moderationCaseId"), input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.moderationCaseResponse(item))
}

func (s *Server) handleListCuratorTags(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	input, repoErr := buildListCuratorTagsInput(r)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	items, total, repoErr := s.store.ListCuratorTags(user.ID, input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.tagsResponse(items),
		"meta":  paginationMeta(input.Page, input.PageSize, total),
	})
}

func (s *Server) handleCreateCuratorTag(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	var req createTagRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}

	tag, repoErr := s.store.CreateCuratorTag(user.ID, parseCreateTagInput(req, false))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.tagsResponse([]*model.Tag{tag})[0])
}

func (s *Server) handlePatchCuratorTag(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	var req updateTagRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}

	tag, repoErr := s.store.UpdateCuratorTag(user.ID, r.PathValue("tagId"), model.UpdateTagInput{
		Name:     trimStringPtr(req.Name),
		IsActive: req.IsActive,
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.tagsResponse([]*model.Tag{tag})[0])
}

func (s *Server) handleListCuratorUsers(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	input, repoErr := buildListCuratorUsersInput(r)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	items, total, repoErr := s.store.ListCuratorUsers(user.ID, input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.moderatedUsersResponse(items),
		"meta":  paginationMeta(input.Page, input.PageSize, total),
	})
}

func (s *Server) handleGetCuratorUser(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	item, repoErr := s.store.GetCuratorUser(user.ID, r.PathValue("userId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.moderatedUserResponse(item))
}

func (s *Server) handlePatchCuratorUser(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	var req updateCuratorUserRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}

	item, repoErr := s.store.UpdateCuratorUser(user.ID, r.PathValue("userId"), model.UpdateCuratorUserInput{
		DisplayName: trimStringPtr(req.DisplayName),
		IsActive:    req.IsActive,
		Reason:      strings.TrimSpace(req.Reason),
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.moderatedUserResponse(item))
}

func (s *Server) handleGetCuratorApplicant(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	item, repoErr := s.store.GetCuratorApplicant(user.ID, r.PathValue("userId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.curatorApplicantProfileViewResponse(item))
}

func (s *Server) handlePatchCuratorApplicant(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	var req updateCuratorApplicantRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}

	item, repoErr := s.store.UpdateCuratorApplicant(user.ID, r.PathValue("userId"), model.UpdateCuratorApplicantInput{
		Profile: model.UpdateApplicantProfileInput{
			FirstName:      trimStringPtr(req.FirstName),
			LastName:       trimStringPtr(req.LastName),
			MiddleName:     req.MiddleName,
			UniversityName: req.UniversityName,
			Faculty:        req.Faculty,
			ProgramName:    req.ProgramName,
			StudyYear:      req.StudyYear,
			GraduationYear: req.GraduationYear,
			City:           req.City,
			About:          req.About,
			ResumeMediaID:  trimStringPtr(req.ResumeMediaID),
		},
		Privacy: parseUpdateApplicantPrivacyInput(req.Privacy),
		Reason:  strings.TrimSpace(req.Reason),
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.curatorApplicantProfileViewResponse(item))
}

func (s *Server) handleGetCuratorEmployer(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	item, repoErr := s.store.GetCuratorEmployer(user.ID, r.PathValue("userId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.employerProfileViewResponse(item))
}

func (s *Server) handlePatchCuratorEmployer(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	var req updateCuratorEmployerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}

	item, repoErr := s.store.UpdateCuratorEmployer(user.ID, r.PathValue("userId"), model.UpdateCuratorEmployerInput{
		Profile: model.UpdateEmployerProfileInput{
			FullName: trimStringPtr(req.FullName),
			JobTitle: req.JobTitle,
			Phone:    req.Phone,
		},
		Reason: strings.TrimSpace(req.Reason),
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.employerProfileViewResponse(item))
}

func (s *Server) handleGetCuratorCompany(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	item, repoErr := s.store.GetCuratorCompany(user.ID, r.PathValue("companyId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.employerCompanyDetailResponse(item))
}

func (s *Server) handlePatchCuratorCompany(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	var req updateCuratorCompanyRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	websiteURL, repoErr := parseOptionalURI(req.WebsiteURL, "websiteUrl")
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}

	item, repoErr := s.store.UpdateCuratorCompany(user.ID, r.PathValue("companyId"), model.UpdateCuratorCompanyInput{
		Company: model.UpdateCompanyInput{
			LegalName:              trimStringPtr(req.LegalName),
			BrandName:              req.BrandName,
			Slug:                   trimStringPtr(req.Slug),
			INN:                    trimStringPtr(req.INN),
			Description:            req.Description,
			Industry:               req.Industry,
			WebsiteURL:             websiteURL,
			CorporateEmailDomain:   trimStringPtr(req.CorporateEmailDomain),
			HeadquartersLocationID: trimStringPtr(req.HeadquartersLocationID),
			LogoMediaID:            trimStringPtr(req.LogoMediaID),
			BannerMediaID:          trimStringPtr(req.BannerMediaID),
			HeadquartersLocation:   toLocationInput(req.HeadquartersLocation),
		},
		Reason: strings.TrimSpace(req.Reason),
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.employerCompanyDetailResponse(item))
}

func (s *Server) handleGetCuratorOpportunity(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	item, repoErr := s.store.GetCuratorOpportunity(user.ID, r.PathValue("opportunityId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.employerOpportunityDetailResponse(item))
}

func (s *Server) handlePatchCuratorOpportunity(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	var req updateCuratorOpportunityRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}

	input, repoErr := parseUpdateCuratorOpportunityInput(req)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	item, repoErr := s.store.UpdateCuratorOpportunity(user.ID, r.PathValue("opportunityId"), input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.employerOpportunityDetailResponse(item))
}

func (s *Server) handleListAdminCurators(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	input, repoErr := buildListAdminCuratorsInput(r)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	items, total, repoErr := s.store.ListAdminCurators(user.ID, input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.curatorAccountsResponse(items),
		"meta":  paginationMeta(input.Page, input.PageSize, total),
	})
}

func (s *Server) handleCreateAdminCurator(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	var req createCuratorRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}
	if req.Email == "" || req.Password == "" || req.DisplayName == "" || req.FullName == "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "required fields are missing", nil)
		return
	}
	email, appErr := parseRequiredEmail(req.Email, "email")
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if utf8.RuneCountInString(req.Password) < 8 {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "password must be at least 8 characters", nil)
		return
	}

	hash, err := authplatform.HashPassword(req.Password)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to hash password", nil)
		return
	}

	item, repoErr := s.store.CreateAdminCurator(user.ID, model.CreateCuratorInput{
		Email:       email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		FullName:    req.FullName,
		Reason:      strings.TrimSpace(req.Reason),
	}, hash)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s.curatorAccountResponse(item))
}

func (s *Server) handlePatchAdminCurator(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	if user.Role != model.UserRoleCurator {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "current user is not a curator", nil)
		return
	}

	var req updateCuratorAccountRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "invalid request body", nil)
		return
	}

	item, repoErr := s.store.UpdateAdminCurator(user.ID, r.PathValue("userId"), model.UpdateCuratorAccountInput{
		DisplayName: trimStringPtr(req.DisplayName),
		FullName:    trimStringPtr(req.FullName),
		IsActive:    req.IsActive,
		Reason:      strings.TrimSpace(req.Reason),
	})
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.curatorAccountResponse(item))
}

func (s *Server) handleListPublicCompanies(w http.ResponseWriter, r *http.Request) {
	page, pageSize, appErr := parsePaginationParams(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
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
	page, pageSize, appErr := parsePaginationParams(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
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

	input, repoErr := buildListNotificationsInput(r)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	items, total, repoErr := s.store.ListNotifications(user.ID, input)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": s.notificationsResponse(items),
		"meta":  paginationMeta(input.Page, input.PageSize, total),
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
	updatedCount, repoErr := s.store.MarkAllNotificationsRead(user.ID)
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"updatedCount": updatedCount})
}

func (s *Server) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	user, appErr := s.requireAccessUser(r)
	if appErr != nil {
		s.writeAppError(w, appErr)
		return
	}
	notification, repoErr := s.store.MarkNotificationRead(user.ID, r.PathValue("notificationId"))
	if repoErr != nil {
		s.writeAppError(w, repoErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.notificationResponse(notification))
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

func (s *Server) maybeAccessUser(r *http.Request) *model.User {
	user, _ := s.requireAccessUser(r)
	return user
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

func nullableInt64(value *int64) any {
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
