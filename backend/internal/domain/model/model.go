package model

import "time"

const (
	UserRoleApplicant = "applicant"
	UserRoleEmployer  = "employer"
	UserRoleCurator   = "curator"

	CompanyMembershipPending  = "pending"
	CompanyMembershipApproved = "approved"
	CompanyMembershipRejected = "rejected"
	CompanyMembershipRevoked  = "revoked"

	CompanyVerificationStatusPending     = "pending"
	CompanyVerificationStatusUnderReview = "under_review"
	CompanyVerificationStatusVerified    = "verified"
	CompanyVerificationStatusRejected    = "rejected"
	CompanyVerificationStatusExpired     = "expired"

	CompanyRoleOwner     = "owner"
	CompanyRoleRecruiter = "recruiter"
	CompanyRoleHR        = "hr"
	CompanyRoleManager   = "manager"

	OpportunityTypeInternship    = "internship"
	OpportunityTypeVacancy       = "vacancy"
	OpportunityTypeMentorProgram = "mentor_program"
	OpportunityTypeEvent         = "event"

	OpportunityStatusDraft    = "draft"
	OpportunityStatusPlanned  = "planned"
	OpportunityStatusActive   = "active"
	OpportunityStatusClosed   = "closed"
	OpportunityStatusRejected = "rejected"
	OpportunityStatusArchived = "archived"

	ModerationStatusPending      = "pending"
	ModerationStatusApproved     = "approved"
	ModerationStatusRejected     = "rejected"
	ModerationStatusNeedsChanges = "needs_changes"

	ApplicationStatusSubmitted = "submitted"
	ApplicationStatusReviewing = "reviewing"
	ApplicationStatusReserve   = "reserve"
	ApplicationStatusAccepted  = "accepted"
	ApplicationStatusRejected  = "rejected"
	ApplicationStatusWithdrawn = "withdrawn"

	TagTypeTechnology     = "technology"
	TagTypeRole           = "role"
	TagTypeDomain         = "domain"
	TagTypeLevel          = "level"
	TagTypeEmploymentType = "employment_type"
	TagTypeFormat         = "format"
	TagTypeCustom         = "custom"

	ConnectionStatusPending  = "pending"
	ConnectionStatusAccepted = "accepted"
	ConnectionStatusRejected = "rejected"
	ConnectionStatusBlocked  = "blocked"

	ModerationTargetCompany      = "company"
	ModerationTargetProfile      = "profile"
	ModerationTargetOpportunity  = "opportunity"
	ModerationTargetTag          = "tag"
	ModerationTargetMedia        = "media"
	ModerationTargetVerification = "verification"

	MediaFileStatusPending  = "pending"
	MediaFileStatusUploaded = "uploaded"
	MediaFileStatusFailed   = "failed"
	MediaFileStatusDeleted  = "deleted"

	MediaFilePurposeAvatar               = "avatar"
	MediaFilePurposeResume               = "resume"
	MediaFilePurposeCompanyLogo          = "company_logo"
	MediaFilePurposeCompanyBanner        = "company_banner"
	MediaFilePurposeCompanyMedia         = "company_media"
	MediaFilePurposeOpportunityCover     = "opportunity_cover"
	MediaFilePurposeOpportunityMedia     = "opportunity_media"
	MediaFilePurposeVerificationEvidence = "verification_evidence"
	MediaFilePurposeOther                = "other"

	VerificationMethodCorporateEmail = "corporate_email"
	VerificationMethodINN            = "inn"
	VerificationMethodOfficialSite   = "official_website"
	VerificationMethodManualReview   = "manual_review"

	VerificationRequestStatusPending      = "pending"
	VerificationRequestStatusUnderReview  = "under_review"
	VerificationRequestStatusApproved     = "approved"
	VerificationRequestStatusRejected     = "rejected"
	VerificationRequestStatusNeedsChanges = "needs_changes"

	VerificationEvidenceCorporateEmail  = "corporate_email"
	VerificationEvidenceINNDoc          = "inn_doc"
	VerificationEvidenceWebsiteLink     = "website_link"
	VerificationEvidenceRegistryExtract = "registry_extract"

	NotificationTypeRecommendationReceived = "recommendation_received"
	NotificationTypeApplicationStatus      = "application_status_changed"
	NotificationTypeEmployerMessage        = "employer_message"
	NotificationTypeEmployerBroadcast      = "employer_broadcast"
	NotificationTypeSystem                 = "system"

	NotificationSourceRecommendation = "recommendation"
	NotificationSourceApplication    = "application"
	NotificationSourceCampaign       = "campaign"
	NotificationSourceSystem         = "system"

	NotificationCampaignAudienceSingleApplicant          = "single_applicant"
	NotificationCampaignAudienceSelectedApplicants       = "selected_applicants"
	NotificationCampaignAudienceAllApplicantsOpportunity = "all_applicants_of_opportunity"

	NotificationCampaignStatusDraft      = "draft"
	NotificationCampaignStatusScheduled  = "scheduled"
	NotificationCampaignStatusProcessing = "processing"
	NotificationCampaignStatusSent       = "sent"
	NotificationCampaignStatusCancelled  = "cancelled"
)

type User struct {
	ID              string
	Email           string
	PasswordHash    string
	DisplayName     string
	Role            string
	IsActive        bool
	AvatarMediaID   *string
	EmailVerifiedAt *time.Time
	LastLoginAt     *time.Time
	TokenVersion    int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CurrentCuratorProfile struct {
	IsAdmin bool
}

type CuratorAccount struct {
	User             *User
	FullName         string
	IsAdmin          bool
	ProfileCreatedAt time.Time
}

type ApplicantProfile struct {
	UserID         string
	FirstName      string
	LastName       string
	MiddleName     *string
	UniversityName *string
	Faculty        *string
	ProgramName    *string
	StudyYear      *int
	GraduationYear *int
	City           *string
	About          *string
	ResumeMediaID  *string
	TagIDs         []string
	SocialLinks    []ApplicantSocialLink
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ApplicantPreview struct {
	UserID         string
	DisplayName    string
	FirstName      string
	LastName       string
	MiddleName     *string
	AvatarMediaID  *string
	UniversityName *string
	City           *string
	GraduationYear *int
	Tags           []*Tag
}

type ApplicantPrivacySettings struct {
	ApplicantUserID        string
	ProfileVisibility      string
	ResumeVisibility       string
	ApplicationsVisibility string
	ContactsVisibility     string
	ShowCareerInterests    bool
	AllowRecommendations   bool
	UpdatedAt              time.Time
}

type ApplicantProfileAccessMeta struct {
	CanViewResume                 bool
	CanViewApplications           bool
	CanViewContacts               bool
	ProfileVisibilityApplied      string
	ResumeVisibilityApplied       string
	ApplicationsVisibilityApplied string
	ContactsVisibilityApplied     string
}

type ApplicantProfileView struct {
	Profile       *ApplicantProfile
	DisplayName   string
	AvatarMediaID *string
	Tags          []*Tag
	SocialLinks   []ApplicantSocialLink
	Access        ApplicantProfileAccessMeta
}

type CuratorApplicantProfileView struct {
	View    *ApplicantProfileView
	Privacy *ApplicantPrivacySettings
}

type ApplicantSocialLink struct {
	ID              string
	ApplicantUserID string
	Platform        string
	URL             string
	IsPublic        bool
	CreatedAt       time.Time
}

type MediaFile struct {
	ID               string
	ObjectKey        string
	OriginalName     *string
	MimeType         *string
	FileSize         *int64
	UploadedByUserID *string
	Purpose          *string
	Status           string
	ETag             *string
	CompletedAt      *time.Time
	DeletedAt        *time.Time
	CreatedAt        time.Time
}

type CompanySocialLink struct {
	ID        string
	CompanyID string
	Platform  string
	URL       string
}

type CompanyMedia struct {
	ID          string
	CompanyID   string
	MediaFileID string
	Title       *string
	SortOrder   int
	CreatedAt   time.Time
}

type VerificationEvidence struct {
	ID                    string
	VerificationRequestID string
	EvidenceType          string
	Value                 *string
	EvidenceFileID        *string
	CreatedAt             time.Time
}

type VerificationRequest struct {
	ID                        string
	CompanyID                 string
	SubmittedByUserID         string
	Method                    string
	Status                    string
	CompanyVerificationStatus string
	SubmittedComment          *string
	ReviewComment             *string
	ReviewedByCuratorUserID   *string
	ReviewedAt                *time.Time
	CreatedAt                 time.Time
	Evidence                  []VerificationEvidence
}

type ModerationTargetPreview struct {
	Type     string
	ID       string
	Title    *string
	Subtitle *string
	Slug     *string
}

type ModerationCase struct {
	ID                      string
	TargetType              string
	TargetID                string
	SubmittedByUserID       *string
	AssignedCuratorUserID   *string
	ResolvedByCuratorUserID *string
	Status                  string
	Reason                  *string
	CreatedAt               time.Time
	ResolvedAt              *time.Time
	TargetPreview           *ModerationTargetPreview
}

type EmployerProfile struct {
	UserID    string
	FullName  string
	JobTitle  *string
	Phone     *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type EmployerProfileView struct {
	Profile       *EmployerProfile
	DisplayName   string
	AvatarMediaID *string
	Companies     []*CompanyMembership
}

type UISettings struct {
	UserID        string
	SettingsJSON  map[string]any
	SchemaVersion int
	UpdatedAt     time.Time
}

type Location struct {
	ID          string
	Precision   string
	Country     string
	Region      *string
	City        string
	AddressLine *string
	PostalCode  *string
	PlaceName   *string
	Latitude    *float64
	Longitude   *float64
	CreatedAt   time.Time
}

type Tag struct {
	ID              string
	Name            string
	TagType         string
	IsSystem        bool
	IsActive        bool
	CreatedByUserID *string
	CreatedAt       time.Time
}

type Company struct {
	ID                      string
	LegalName               string
	BrandName               *string
	Slug                    string
	INN                     *string
	Description             *string
	Industry                *string
	WebsiteURL              *string
	CorporateEmailDomain    *string
	HeadquartersLocationID  *string
	LogoMediaID             *string
	BannerMediaID           *string
	VerificationStatus      string
	VerifiedAt              *time.Time
	VerifiedByCuratorUserID *string
	CreatedByUserID         *string
	CreatedAt               time.Time
	UpdatedAt               time.Time
	SocialLinks             []CompanySocialLink
	Media                   []CompanyMedia
}

type CompanyMembership struct {
	ID                    string
	CompanyID             string
	EmployerUserID        string
	InvitedByUserID       *string
	Status                string
	MemberRole            string
	IsPrimaryContact      bool
	StatusChangedByUserID *string
	StatusComment         *string
	StatusUpdatedAt       time.Time
	ApprovedAt            *time.Time
	UpdatedAt             time.Time
	CreatedAt             time.Time
}

type Opportunity struct {
	ID                   string
	CompanyID            string
	CreatedByUserID      string
	Title                string
	Summary              string
	Slug                 string
	Description          string
	Type                 string
	Status               string
	ModerationStatus     string
	ParticipationFormat  string
	LocationID           *string
	ContactEmail         *string
	ContactPhone         *string
	CoverMediaID         *string
	PublishedAt          *time.Time
	ExpiresAt            *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Company              *Company
	VacancyDetails       *OpportunityVacancyDetails
	MentorProgramDetails *OpportunityMentorProgramDetails
	EventDetails         *OpportunityEventDetails
	Tags                 []*Tag
	Links                []OpportunityLink
	Media                []OpportunityMedia
	Stats                *OpportunityStats
}

type OpportunityVacancyDetails struct {
	EmploymentType  string
	ExperienceLevel string
	SalaryFrom      *int
	SalaryTo        *int
	Currency        *string
}

type OpportunityMentorProgramDetails struct {
	StartAt            *time.Time
	EndAt              *time.Time
	SeatsCount         *int
	MentorRequirements *string
}

type OpportunityEventDetails struct {
	StartAt              time.Time
	EndAt                time.Time
	RegistrationDeadline *time.Time
	Capacity             *int
	VenueNote            *string
}

type OpportunityLink struct {
	ID            string
	OpportunityID string
	LinkType      string
	Title         string
	URL           string
	SortOrder     int
}

type OpportunityMedia struct {
	ID            string
	OpportunityID string
	MediaFileID   string
	Title         *string
	SortOrder     int
	CreatedAt     time.Time
}

type OpportunityStats struct {
	OpportunityID string
	ViewsCount    int
	UpdatedAt     time.Time
}

type Application struct {
	ID              string
	OpportunityID   string
	ApplicantUserID string
	CoverLetter     *string
	Status          string
	AppliedAt       time.Time
	UpdatedAt       time.Time
	Opportunity     *Opportunity
	Applicant       *ApplicantPreview
	History         []ApplicationStatusHistory
}

type ApplicationStatusHistory struct {
	ID              string
	ApplicationID   string
	OldStatus       *string
	NewStatus       string
	ChangedByUserID string
	Comment         *string
	CreatedAt       time.Time
}

type SavedOpportunity struct {
	ApplicantUserID string
	OpportunityID   string
	CreatedAt       time.Time
	Opportunity     *Opportunity
}

type SavedCompany struct {
	ApplicantUserID string
	CompanyID       string
	CreatedAt       time.Time
	Company         *Company
}

type Connection struct {
	ID               string
	InitiatorUserID  string
	OtherApplicantID string
	Status           string
	InitiatorNote    *string
	RespondedAt      *time.Time
	CreatedAt        time.Time
	OtherApplicant   *ApplicantPreview
}

type OpportunityRecommendation struct {
	ID                string
	RecommenderUserID string
	RecipientUserID   string
	OpportunityID     string
	Message           *string
	CreatedAt         time.Time
}

type NotificationPreferences struct {
	UserID                   string
	InAppEnabled             bool
	EmailEnabled             bool
	RecommendationEnabled    bool
	ApplicationStatusEnabled bool
	EmployerMessagesEnabled  bool
	SystemEnabled            bool
	UpdatedAt                time.Time
}

type Notification struct {
	ID              string
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
	IsRead          bool
	ReadAt          *time.Time
	CreatedAt       time.Time
}

type NotificationCampaign struct {
	ID              string
	CompanyID       string
	CreatedByUserID string
	OpportunityID   *string
	AudienceType    string
	Status          string
	Title           string
	Body            string
	SendViaInApp    bool
	SendViaEmail    bool
	ScheduledAt     *time.Time
	SentAt          *time.Time
	CreatedAt       time.Time
	Recipients      []NotificationCampaignRecipient
}

type NotificationCampaignRecipient struct {
	CampaignID      string
	ApplicantUserID string
	ApplicationID   *string
	CreatedAt       time.Time
}

type RegisterApplicantInput struct {
	Email       string
	Password    string
	DisplayName string
	FirstName   string
	LastName    string
	MiddleName  *string
}

type RegisterEmployerInput struct {
	Email       string
	Password    string
	DisplayName string
	FullName    string
	JobTitle    *string
	Phone       *string
}

type UpdateApplicantProfileInput struct {
	FirstName      *string
	LastName       *string
	MiddleName     *string
	UniversityName *string
	Faculty        *string
	ProgramName    *string
	StudyYear      *int
	GraduationYear *int
	City           *string
	About          *string
	ResumeMediaID  *string
}

type UpdateApplicantPrivacyInput struct {
	ProfileVisibility      *string
	ResumeVisibility       *string
	ApplicationsVisibility *string
	ContactsVisibility     *string
	ShowCareerInterests    *bool
	AllowRecommendations   *bool
}

type UpdateCuratorApplicantInput struct {
	Profile UpdateApplicantProfileInput
	Privacy *UpdateApplicantPrivacyInput
	Reason  string
}

type CreateApplicantSocialLinkInput struct {
	Platform string
	URL      string
	IsPublic bool
}

type CreatePresignedUploadInput struct {
	OriginalName string
	MimeType     string
	FileSize     int64
	Purpose      *string
}

type CompleteUploadInput struct {
	ETag *string
}

type UpdateApplicantSocialLinkInput struct {
	Platform *string
	URL      *string
	IsPublic *bool
}

type UpdateEmployerProfileInput struct {
	FullName *string
	JobTitle *string
	Phone    *string
}

type UpdateCuratorEmployerInput struct {
	Profile UpdateEmployerProfileInput
	Reason  string
}

type PutUISettingsInput struct {
	SettingsJSON  map[string]any
	SchemaVersion *int
}

type LocationInput struct {
	Precision   string
	Country     string
	Region      *string
	City        string
	AddressLine *string
	PostalCode  *string
	PlaceName   *string
	Latitude    *float64
	Longitude   *float64
}

type CreateCompanyInput struct {
	LegalName              string
	BrandName              *string
	Slug                   string
	INN                    *string
	Description            *string
	Industry               *string
	WebsiteURL             *string
	CorporateEmailDomain   *string
	HeadquartersLocationID *string
	LogoMediaID            *string
	BannerMediaID          *string
	HeadquartersLocation   *LocationInput
}

type UpdateCompanyInput struct {
	LegalName              *string
	BrandName              *string
	Slug                   *string
	INN                    *string
	Description            *string
	Industry               *string
	WebsiteURL             *string
	CorporateEmailDomain   *string
	HeadquartersLocationID *string
	LogoMediaID            *string
	BannerMediaID          *string
	HeadquartersLocation   *LocationInput
}

type UpdateCuratorCompanyInput struct {
	Company UpdateCompanyInput
	Reason  string
}

type CreateCompanyMembershipInput struct {
	EmployerEmail    string
	MemberRole       string
	IsPrimaryContact bool
	Comment          *string
}

type UpdateCompanyMembershipInput struct {
	MemberRole       *string
	IsPrimaryContact *bool
}

type CreateCompanySocialLinkInput struct {
	Platform string
	URL      string
}

type UpdateCompanySocialLinkInput struct {
	Platform *string
	URL      *string
}

type CreateCompanyMediaInput struct {
	MediaFileID string
	Title       *string
	SortOrder   int
}

type CreateTagInput struct {
	Name     string
	TagType  string
	IsActive bool
}

type UpdateTagInput struct {
	Name     *string
	IsActive *bool
}

type UpdateCompanyMediaInput struct {
	Title     *string
	SortOrder *int
}

type OpportunityLinkInput struct {
	LinkType  string
	Title     string
	URL       string
	SortOrder int
}

type OpportunityMediaInput struct {
	MediaFileID string
	Title       *string
	SortOrder   int
}

type CreateOpportunityInput struct {
	CompanyID            string
	Title                string
	Summary              string
	Slug                 string
	Description          string
	Type                 string
	ParticipationFormat  string
	LocationID           *string
	ContactEmail         *string
	ContactPhone         *string
	CoverMediaID         *string
	PublishedAt          *time.Time
	ExpiresAt            *time.Time
	TagIDs               []string
	VacancyDetails       *OpportunityVacancyDetails
	MentorProgramDetails *OpportunityMentorProgramDetails
	EventDetails         *OpportunityEventDetails
	Links                []OpportunityLinkInput
	Media                []OpportunityMediaInput
	Location             *LocationInput
}

type UpdateOpportunityInput struct {
	Title                *string
	Summary              *string
	Slug                 *string
	Description          *string
	Status               *string
	ParticipationFormat  *string
	LocationID           *string
	ContactEmail         *string
	ContactPhone         *string
	CoverMediaID         *string
	PublishedAt          *time.Time
	ExpiresAt            *time.Time
	TagIDs               []string
	ReplaceTagIDs        bool
	VacancyDetails       *OpportunityVacancyDetails
	MentorProgramDetails *OpportunityMentorProgramDetails
	EventDetails         *OpportunityEventDetails
	Links                []OpportunityLinkInput
	ReplaceLinks         bool
	Media                []OpportunityMediaInput
	ReplaceMedia         bool
	Location             *LocationInput
}

type UpdateCuratorOpportunityInput struct {
	Title                *string
	Summary              *string
	Slug                 *string
	Description          *string
	ParticipationFormat  *string
	LocationID           *string
	ContactEmail         *string
	ContactPhone         *string
	CoverMediaID         *string
	PublishedAt          *time.Time
	ExpiresAt            *time.Time
	TagIDs               []string
	ReplaceTagIDs        bool
	VacancyDetails       *OpportunityVacancyDetails
	MentorProgramDetails *OpportunityMentorProgramDetails
	EventDetails         *OpportunityEventDetails
	Links                []OpportunityLinkInput
	ReplaceLinks         bool
	Media                []OpportunityMediaInput
	ReplaceMedia         bool
	Location             *LocationInput
	Reason               string
}

type ListEmployerOpportunitiesInput struct {
	CompanyID        string
	Status           string
	ModerationStatus string
	Page             int
	PageSize         int
}

type GeoBounds struct {
	MinLng float64
	MinLat float64
	MaxLng float64
	MaxLat float64
}

type ListPublicOpportunitiesInput struct {
	Page                int
	PageSize            int
	Q                   string
	Type                string
	ParticipationFormat string
	CompanyID           string
	City                string
	TagIDs              []string
	EmploymentType      string
	ExperienceLevel     string
	SalaryFrom          *int
	SalaryTo            *int
	StartsAfter         *time.Time
	ExpiresAfter        *time.Time
	Sort                string
	View                string
	BBox                *GeoBounds
	Lat                 *float64
	Lng                 *float64
	RadiusKm            *float64
}

type CreateApplicationInput struct {
	OpportunityID string
	CoverLetter   *string
}

type ListApplicationsInput struct {
	Status   string
	Page     int
	PageSize int
}

type UpdateApplicationStatusInput struct {
	Status  string
	Comment *string
}

type SaveOpportunityInput struct {
	OpportunityID string
}

type SaveCompanyInput struct {
	CompanyID string
}

type CreateConnectionInput struct {
	TargetApplicantUserID string
	InitiatorNote         *string
}

type UpdateConnectionInput struct {
	Status string
}

type ListConnectionsInput struct {
	Status string
}

type ListCuratorTagsInput struct {
	Type     string
	IsActive *bool
	Page     int
	PageSize int
}

type ListCuratorUsersInput struct {
	Role     string
	IsActive *bool
	Q        string
	Page     int
	PageSize int
}

type CreateOpportunityRecommendationInput struct {
	RecipientUserID string
	OpportunityID   string
	Message         *string
}

type ListNotificationsInput struct {
	Page       int
	PageSize   int
	UnreadOnly bool
}

type CreateNotificationCampaignInput struct {
	CompanyID        string
	OpportunityID    *string
	AudienceType     string
	Title            string
	Body             string
	SendViaInApp     bool
	SendViaEmail     bool
	ScheduledAt      *time.Time
	ApplicantUserIDs []string
}

type UpdateNotificationCampaignInput struct {
	AudienceType     *string
	ReplaceAudience  bool
	OpportunityID    *string
	Title            *string
	Body             *string
	SendViaInApp     *bool
	SendViaEmail     *bool
	ScheduledAt      *time.Time
	SetScheduledAt   bool
	ApplicantUserIDs []string
}

type ListNotificationCampaignsInput struct {
	CompanyID string
	Status    string
	Page      int
	PageSize  int
}

type VerificationEvidenceInput struct {
	EvidenceType   string
	Value          *string
	EvidenceFileID *string
}

type CreateVerificationRequestInput struct {
	Method           string
	SubmittedComment *string
	Evidence         []VerificationEvidenceInput
}

type ReviewVerificationRequestInput struct {
	Status        string
	ReviewComment *string
}

type ListVerificationRequestsInput struct {
	Status   string
	Page     int
	PageSize int
}

type ListModerationCasesInput struct {
	Status     string
	TargetType string
	Page       int
	PageSize   int
}

type CreateModerationCaseInput struct {
	TargetType string
	TargetID   string
	Reason     string
}

type UpdateModerationCaseInput struct {
	SetAssignedCuratorUserID bool
	AssignedCuratorUserID    *string
	Status                   *string
	Reason                   string
}

type UpdateCuratorUserInput struct {
	DisplayName *string
	IsActive    *bool
	Reason      string
}

type CreateCuratorInput struct {
	Email       string
	Password    string
	DisplayName string
	FullName    string
	Reason      string
}

type UpdateCuratorAccountInput struct {
	DisplayName *string
	FullName    *string
	IsActive    *bool
	Reason      string
}

type ListAdminCuratorsInput struct {
	IsActive *bool
	Q        string
	Page     int
	PageSize int
}
