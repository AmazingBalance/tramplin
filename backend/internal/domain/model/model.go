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

type ApplicantSocialLink struct {
	ID              string
	ApplicantUserID string
	Platform        string
	URL             string
	IsPublic        bool
	CreatedAt       time.Time
}

type EmployerProfile struct {
	UserID    string
	FullName  string
	JobTitle  *string
	Phone     *string
	CreatedAt time.Time
	UpdatedAt time.Time
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

type UpdateEmployerProfileInput struct {
	FullName *string
	JobTitle *string
	Phone    *string
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

type ListEmployerOpportunitiesInput struct {
	CompanyID        string
	Status           string
	ModerationStatus string
	Page             int
	PageSize         int
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
	BBox                string
	Lat                 *float64
	Lng                 *float64
	RadiusKm            *float64
}
