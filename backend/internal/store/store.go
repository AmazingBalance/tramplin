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
	GetCurrentCuratorProfile(userID string) (*model.CurrentCuratorProfile, *AppError)
	TouchLastLogin(userID string)
	BumpTokenVersion(userID string)
	GetUISettings(userID string) (*model.UISettings, *AppError)
	PutUISettings(userID string, input model.PutUISettingsInput) (*model.UISettings, *AppError)
	GetApplicantProfile(userID string) (*model.ApplicantProfile, *AppError)
	UpdateApplicantProfile(userID string, input model.UpdateApplicantProfileInput) (*model.ApplicantProfile, *AppError)
	GetApplicantPrivacy(userID string) (*model.ApplicantPrivacySettings, *AppError)
	UpdateApplicantPrivacy(userID string, input model.UpdateApplicantPrivacyInput) (*model.ApplicantPrivacySettings, *AppError)
	GetApplicantProfileView(viewerUserID, targetUserID string) (*model.ApplicantProfileView, *AppError)
	GetEmployerApplicantProfileView(employerUserID, targetUserID string) (*model.ApplicantProfileView, *AppError)
	CreatePresignedUpload(userID string, input model.CreatePresignedUploadInput) (*model.MediaFile, *AppError)
	CompleteUpload(userID, mediaFileID string, input model.CompleteUploadInput, objectETag *string) (*model.MediaFile, *AppError)
	GetMediaFile(actorUserID, mediaFileID string) (*model.MediaFile, *AppError)
	GetMediaFileForDownload(actorUserID *string, mediaFileID string) (*model.MediaFile, *AppError)
	DeleteMediaFile(actorUserID, mediaFileID string) (*model.MediaFile, *AppError)
	ReplaceApplicantTags(userID string, tagIDs []string) ([]*model.Tag, *AppError)
	GetApplicantTags(userID string) ([]*model.Tag, *AppError)
	ListApplicantSocialLinks(applicantUserID string) ([]*model.ApplicantSocialLink, *AppError)
	CreateApplicantSocialLink(applicantUserID string, input model.CreateApplicantSocialLinkInput) (*model.ApplicantSocialLink, *AppError)
	UpdateApplicantSocialLink(applicantUserID, linkID string, input model.UpdateApplicantSocialLinkInput) (*model.ApplicantSocialLink, *AppError)
	DeleteApplicantSocialLink(applicantUserID, linkID string) *AppError
	ListConnections(applicantUserID string, input model.ListConnectionsInput) ([]*model.Connection, *AppError)
	CreateConnection(applicantUserID string, input model.CreateConnectionInput) (*model.Connection, *AppError)
	UpdateConnection(applicantUserID, connectionID string, input model.UpdateConnectionInput) (*model.Connection, *AppError)
	CreateOpportunityRecommendation(applicantUserID string, input model.CreateOpportunityRecommendationInput) (*model.OpportunityRecommendation, *AppError)
	ListReceivedRecommendations(applicantUserID string) ([]*model.OpportunityRecommendation, *AppError)
	ListSentRecommendations(applicantUserID string) ([]*model.OpportunityRecommendation, *AppError)
	ListSavedOpportunities(applicantUserID string) ([]*model.SavedOpportunity, *AppError)
	SaveOpportunity(applicantUserID string, input model.SaveOpportunityInput) *AppError
	DeleteSavedOpportunity(applicantUserID, opportunityID string) *AppError
	ListSavedCompanies(applicantUserID string) ([]*model.SavedCompany, *AppError)
	SaveCompany(applicantUserID string, input model.SaveCompanyInput) *AppError
	DeleteSavedCompany(applicantUserID, companyID string) *AppError
	ListApplicantApplications(applicantUserID string, input model.ListApplicationsInput) ([]*model.Application, int, *AppError)
	CreateApplication(applicantUserID string, input model.CreateApplicationInput) (*model.Application, *AppError)
	GetApplicantApplication(applicantUserID, applicationID string) (*model.Application, *AppError)
	WithdrawApplicantApplication(applicantUserID, applicationID string) (*model.Application, *AppError)
	GetEmployerProfile(userID string) (*model.EmployerProfile, *AppError)
	UpdateEmployerProfile(userID string, input model.UpdateEmployerProfileInput) (*model.EmployerProfile, *AppError)
	ListCuratorUsers(actorUserID string, input model.ListCuratorUsersInput) ([]*model.User, int, *AppError)
	GetCuratorUser(actorUserID, targetUserID string) (*model.User, *AppError)
	UpdateCuratorUser(actorUserID, targetUserID string, input model.UpdateCuratorUserInput) (*model.User, *AppError)
	ListAdminCurators(actorUserID string, input model.ListAdminCuratorsInput) ([]*model.CuratorAccount, int, *AppError)
	CreateAdminCurator(actorUserID string, input model.CreateCuratorInput, passwordHash string) (*model.CuratorAccount, *AppError)
	UpdateAdminCurator(actorUserID, targetUserID string, input model.UpdateCuratorAccountInput) (*model.CuratorAccount, *AppError)
	GetCuratorApplicant(actorUserID, targetUserID string) (*model.CuratorApplicantProfileView, *AppError)
	UpdateCuratorApplicant(actorUserID, targetUserID string, input model.UpdateCuratorApplicantInput) (*model.CuratorApplicantProfileView, *AppError)
	GetCuratorEmployer(actorUserID, targetUserID string) (*model.EmployerProfileView, *AppError)
	UpdateCuratorEmployer(actorUserID, targetUserID string, input model.UpdateCuratorEmployerInput) (*model.EmployerProfileView, *AppError)
	GetCuratorCompany(actorUserID, companyID string) (*model.Company, *AppError)
	UpdateCuratorCompany(actorUserID, companyID string, input model.UpdateCuratorCompanyInput) (*model.Company, *AppError)
	GetCuratorOpportunity(actorUserID, opportunityID string) (*model.Opportunity, *AppError)
	UpdateCuratorOpportunity(actorUserID, opportunityID string, input model.UpdateCuratorOpportunityInput) (*model.Opportunity, *AppError)
	ListEmployerOpportunities(actorUserID string, input model.ListEmployerOpportunitiesInput) ([]*model.Opportunity, int, *AppError)
	CreateOpportunity(actorUserID string, input model.CreateOpportunityInput) (*model.Opportunity, *AppError)
	GetEmployerOpportunity(actorUserID, opportunityID string) (*model.Opportunity, *AppError)
	UpdateOpportunity(actorUserID, opportunityID string, input model.UpdateOpportunityInput) (*model.Opportunity, *AppError)
	ListOpportunityApplications(actorUserID, opportunityID string, input model.ListApplicationsInput) ([]*model.Application, int, *AppError)
	UpdateApplicationStatus(actorUserID, applicationID string, input model.UpdateApplicationStatusInput) (*model.Application, *AppError)
	CreateEmployerTag(actorUserID string, input model.CreateTagInput) (*model.Tag, *AppError)
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
	ListCompanyVerificationRequests(actorUserID, companyID string) ([]*model.VerificationRequest, *AppError)
	CreateCompanyVerificationRequest(actorUserID, companyID string, input model.CreateVerificationRequestInput) (*model.VerificationRequest, *AppError)
	ListCompanySocialLinks(actorUserID, companyID string) ([]*model.CompanySocialLink, *AppError)
	CreateCompanySocialLink(actorUserID, companyID string, input model.CreateCompanySocialLinkInput) (*model.CompanySocialLink, *AppError)
	UpdateCompanySocialLink(actorUserID, companyID, linkID string, input model.UpdateCompanySocialLinkInput) (*model.CompanySocialLink, *AppError)
	DeleteCompanySocialLink(actorUserID, companyID, linkID string) *AppError
	ListCompanyMedia(actorUserID, companyID string) ([]*model.CompanyMedia, *AppError)
	CreateCompanyMedia(actorUserID, companyID string, input model.CreateCompanyMediaInput) (*model.CompanyMedia, *AppError)
	UpdateCompanyMedia(actorUserID, companyID, mediaID string, input model.UpdateCompanyMediaInput) (*model.CompanyMedia, *AppError)
	DeleteCompanyMedia(actorUserID, companyID, mediaID string) *AppError
	ListPublicCompanies(q, industry, city string, page, pageSize int) ([]*model.Company, int)
	GetPublicCompanyByID(companyID string) (*model.Company, *AppError)
	GetPublicCompanyBySlug(slug string) (*model.Company, *AppError)
	GetLocationByID(locationID *string) *model.Location
	GetEmployerSnapshot(userID string) (*model.User, *model.EmployerProfile)
	GetNotificationPreferences(userID string) (*model.NotificationPreferences, *AppError)
	ListNotifications(userID string, input model.ListNotificationsInput) ([]*model.Notification, int, *AppError)
	ListCuratorTags(actorUserID string, input model.ListCuratorTagsInput) ([]*model.Tag, int, *AppError)
	CreateCuratorTag(actorUserID string, input model.CreateTagInput) (*model.Tag, *AppError)
	UpdateCuratorTag(actorUserID, tagID string, input model.UpdateTagInput) (*model.Tag, *AppError)
	ListModerationCases(actorUserID string, input model.ListModerationCasesInput) ([]*model.ModerationCase, int, *AppError)
	CreateModerationCase(actorUserID string, input model.CreateModerationCaseInput) (*model.ModerationCase, *AppError)
	UpdateModerationCase(actorUserID, moderationCaseID string, input model.UpdateModerationCaseInput) (*model.ModerationCase, *AppError)
	ListNotificationCampaigns(actorUserID string, input model.ListNotificationCampaignsInput) ([]*model.NotificationCampaign, int, *AppError)
	CreateNotificationCampaign(actorUserID string, input model.CreateNotificationCampaignInput) (*model.NotificationCampaign, *AppError)
	GetNotificationCampaign(actorUserID, campaignID string) (*model.NotificationCampaign, *AppError)
	UpdateNotificationCampaign(actorUserID, campaignID string, input model.UpdateNotificationCampaignInput) (*model.NotificationCampaign, *AppError)
	SendNotificationCampaign(actorUserID, campaignID string) (*model.NotificationCampaign, *AppError)
	CancelNotificationCampaign(actorUserID, campaignID string) (*model.NotificationCampaign, *AppError)
	ListCuratorVerificationRequests(actorUserID string, input model.ListVerificationRequestsInput) ([]*model.VerificationRequest, int, *AppError)
	ReviewVerificationRequest(actorUserID, verificationRequestID string, input model.ReviewVerificationRequestInput) (*model.VerificationRequest, *AppError)
	PatchNotificationPreferences(
		userID string,
		inAppEnabled, emailEnabled, recommendationEnabled, applicationStatusEnabled, employerMessagesEnabled, systemEnabled *bool,
	) (*model.NotificationPreferences, *AppError)
	MarkAllNotificationsRead(userID string) (int, *AppError)
	MarkNotificationRead(userID, notificationID string) (*model.Notification, *AppError)
}
