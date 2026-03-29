package store

import "tramplin/backend/internal/domain/model"

type AppError struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

func (e *AppError) Error() string {
	return e.Message
}

type Repository interface {
	CreateApplicant(input model.RegisterApplicantInput, passwordHash string) (*model.User, *AppError)
	CreateEmployer(input model.RegisterEmployerInput, passwordHash string) (*model.User, *AppError)
	GetUserByEmail(email string) (*model.User, *AppError)
	GetUserByID(userID string) (*model.User, *AppError)
	TouchLastLogin(userID string)
	BumpTokenVersion(userID string)
	GetUISettings(userID string) (*model.UISettings, *AppError)
	PutUISettings(userID string, input model.PutUISettingsInput) (*model.UISettings, *AppError)
	GetApplicantProfile(userID string) (*model.ApplicantProfile, *AppError)
	UpdateApplicantProfile(userID string, input model.UpdateApplicantProfileInput) (*model.ApplicantProfile, *AppError)
	GetApplicantPrivacy(userID string) (*model.ApplicantPrivacySettings, *AppError)
	UpdateApplicantPrivacy(userID string, input model.UpdateApplicantPrivacyInput) (*model.ApplicantPrivacySettings, *AppError)
	ReplaceApplicantTags(userID string, tagIDs []string) ([]*model.Tag, *AppError)
	GetApplicantTags(userID string) ([]*model.Tag, *AppError)
	GetEmployerProfile(userID string) (*model.EmployerProfile, *AppError)
	UpdateEmployerProfile(userID string, input model.UpdateEmployerProfileInput) (*model.EmployerProfile, *AppError)
	ListEmployerOpportunities(actorUserID string, input model.ListEmployerOpportunitiesInput) ([]*model.Opportunity, int, *AppError)
	CreateOpportunity(actorUserID string, input model.CreateOpportunityInput) (*model.Opportunity, *AppError)
	GetEmployerOpportunity(actorUserID, opportunityID string) (*model.Opportunity, *AppError)
	UpdateOpportunity(actorUserID, opportunityID string, input model.UpdateOpportunityInput) (*model.Opportunity, *AppError)
	ListPublicTags(tagType, q string) ([]*model.Tag, *AppError)
	ListPublicOpportunities(input model.ListPublicOpportunitiesInput) ([]*model.Opportunity, int, *AppError)
	GetPublicOpportunityByID(opportunityID string) (*model.Opportunity, *AppError)
	GetPublicOpportunityBySlug(slug string) (*model.Opportunity, *AppError)
	CreateOrReuseLocation(input model.LocationInput) (*model.Location, *AppError)
	SearchLocations(q string, page, pageSize int) ([]*model.Location, int, *AppError)
	CreateCompany(actorUserID string, input model.CreateCompanyInput) (*model.Company, *model.CompanyMembership, *AppError)
	UpdateCompany(actorUserID, companyID string, input model.UpdateCompanyInput) (*model.Company, *AppError)
	ListEmployerMemberships(userID string) ([]*model.CompanyMembership, *AppError)
	GetCompanyForEmployer(userID, companyID string) (*model.Company, *AppError)
	ListCompanyMemberships(actorUserID, companyID, status string) ([]*model.CompanyMembership, *AppError)
	CreateCompanyMembership(actorUserID, companyID string, input model.CreateCompanyMembershipInput) (*model.CompanyMembership, *AppError)
	UpdateCompanyMembership(actorUserID, companyID, membershipID string, input model.UpdateCompanyMembershipInput) (*model.CompanyMembership, *AppError)
	ApproveCompanyMembership(actorUserID, companyID, membershipID string, comment *string) (*model.CompanyMembership, *AppError)
	RejectCompanyMembership(actorUserID, companyID, membershipID string, comment string) (*model.CompanyMembership, *AppError)
	RevokeCompanyMembership(actorUserID, companyID, membershipID string, comment string) (*model.CompanyMembership, *AppError)
	ListPublicCompanies(q, industry, city string, page, pageSize int) ([]*model.Company, int)
	GetPublicCompanyByID(companyID string) (*model.Company, *AppError)
	GetPublicCompanyBySlug(slug string) (*model.Company, *AppError)
	GetLocationByID(locationID *string) *model.Location
	GetEmployerSnapshot(userID string) (*model.User, *model.EmployerProfile)
	GetNotificationPreferences(userID string) (*model.NotificationPreferences, *AppError)
	PatchNotificationPreferences(
		userID string,
		inAppEnabled, emailEnabled, recommendationEnabled, applicationStatusEnabled, employerMessagesEnabled, systemEnabled *bool,
	) (*model.NotificationPreferences, *AppError)
}
