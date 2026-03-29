package http

import (
	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

func (s *Server) authUserResponse(user *model.User) (map[string]any, *commonstore.AppError) {
	currentUser, appErr := s.currentUserResponse(user)
	if appErr != nil {
		return nil, appErr
	}
	return map[string]any{
		"user":                 currentUser,
		"accessTokenExpiresIn": s.cfg.DefaultAccessTokenTTL,
	}, nil
}

func (s *Server) currentUserResponse(user *model.User) (map[string]any, *commonstore.AppError) {
	curatorProfile, appErr := s.currentCuratorProfile(user)
	if appErr != nil {
		return nil, appErr
	}
	return s.currentUserResponseBody(user, curatorProfile), nil
}

func (s *Server) currentUserResponseBody(user *model.User, curatorProfile *model.CurrentCuratorProfile) map[string]any {
	var curatorProfileBody any
	if curatorProfile != nil {
		curatorProfileBody = s.currentCuratorProfileResponse(curatorProfile.IsAdmin)
	}
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
		"curatorProfile":  curatorProfileBody,
	}
}

func (s *Server) currentCuratorProfile(user *model.User) (*model.CurrentCuratorProfile, *commonstore.AppError) {
	if user.Role != model.UserRoleCurator {
		return nil, nil
	}
	return s.store.GetCurrentCuratorProfile(user.ID)
}

func (s *Server) curatorAccountsResponse(items []*model.CuratorAccount) []any {
	response := make([]any, 0, len(items))
	for _, item := range items {
		response = append(response, s.curatorAccountResponse(item))
	}
	return response
}

func (s *Server) curatorAccountResponse(item *model.CuratorAccount) map[string]any {
	body := s.currentUserResponseBody(item.User, &model.CurrentCuratorProfile{IsAdmin: item.IsAdmin})
	body["role"] = model.UserRoleCurator
	body["fullName"] = item.FullName
	body["isAdmin"] = item.IsAdmin
	return body
}

func (s *Server) currentCuratorProfileResponse(isAdmin bool) map[string]any {
	return map[string]any{
		"isAdmin": isAdmin,
	}
}

func (s *Server) moderatedUsersResponse(items []*model.User) []any {
	response := make([]any, 0, len(items))
	for _, item := range items {
		response = append(response, s.moderatedUserResponse(item))
	}
	return response
}

func (s *Server) moderatedUserResponse(user *model.User) map[string]any {
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

func (s *Server) applicantProfileViewResponse(view *model.ApplicantProfileView) map[string]any {
	body := s.applicantProfileResponse(view.Profile)
	body["displayName"] = view.DisplayName
	body["avatarMediaId"] = nullableString(view.AvatarMediaID)
	body["tags"] = s.tagsResponse(view.Tags)
	body["socialLinks"] = s.applicantSocialLinksValueResponse(view.SocialLinks)
	body["access"] = s.applicantProfileAccessResponse(view.Access)
	return body
}

func (s *Server) curatorApplicantProfileViewResponse(view *model.CuratorApplicantProfileView) map[string]any {
	body := s.applicantProfileViewResponse(view.View)
	body["privacy"] = s.applicantPrivacyResponse(view.Privacy)
	return body
}

func (s *Server) applicantSocialLinksResponse(items []*model.ApplicantSocialLink) []any {
	response := make([]any, 0, len(items))
	for _, link := range items {
		response = append(response, s.applicantSocialLinkResponse(link))
	}
	return response
}

func (s *Server) applicantSocialLinksValueResponse(items []model.ApplicantSocialLink) []any {
	response := make([]any, 0, len(items))
	for i := range items {
		response = append(response, s.applicantSocialLinkResponse(&items[i]))
	}
	return response
}

func (s *Server) applicantSocialLinkResponse(link *model.ApplicantSocialLink) map[string]any {
	return map[string]any{
		"id":              link.ID,
		"applicantUserId": link.ApplicantUserID,
		"platform":        link.Platform,
		"url":             link.URL,
		"isPublic":        link.IsPublic,
		"createdAt":       timestamp(link.CreatedAt),
	}
}

func (s *Server) applicantProfileAccessResponse(access model.ApplicantProfileAccessMeta) map[string]any {
	return map[string]any{
		"canViewResume":                 access.CanViewResume,
		"canViewApplications":           access.CanViewApplications,
		"canViewContacts":               access.CanViewContacts,
		"profileVisibilityApplied":      access.ProfileVisibilityApplied,
		"resumeVisibilityApplied":       access.ResumeVisibilityApplied,
		"applicationsVisibilityApplied": access.ApplicationsVisibilityApplied,
		"contactsVisibilityApplied":     access.ContactsVisibilityApplied,
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

func (s *Server) employerProfileViewResponse(view *model.EmployerProfileView) map[string]any {
	body := s.employerProfileResponse(view.Profile)
	body["displayName"] = view.DisplayName
	body["avatarMediaId"] = nullableString(view.AvatarMediaID)
	body["companies"] = s.membershipsResponse(view.Companies)
	return body
}

func (s *Server) uiSettingsResponse(settings *model.UISettings) map[string]any {
	return map[string]any{
		"userId":        settings.UserID,
		"settingsJson":  settings.SettingsJSON,
		"schemaVersion": settings.SchemaVersion,
		"updatedAt":     timestamp(settings.UpdatedAt),
	}
}

func (s *Server) mediaFileResponse(file *model.MediaFile) map[string]any {
	return map[string]any{
		"id":               file.ID,
		"originalName":     nullableString(file.OriginalName),
		"mimeType":         nullableString(file.MimeType),
		"fileSize":         nullableInt64(file.FileSize),
		"uploadedByUserId": nullableString(file.UploadedByUserID),
		"purpose":          nullableString(file.Purpose),
		"status":           file.Status,
		"etag":             nullableString(file.ETag),
		"completedAt":      nullableTime(file.CompletedAt),
		"deletedAt":        nullableTime(file.DeletedAt),
		"createdAt":        timestamp(file.CreatedAt),
	}
}

func (s *Server) companyDetailResponse(company *model.Company) map[string]any {
	body := s.companySummaryResponse(company)
	body["bannerMediaId"] = nullableString(company.BannerMediaID)
	body["socialLinks"] = s.companySocialLinksResponse(company.SocialLinks)
	body["media"] = s.companyMediaResponse(company.Media)
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

func (s *Server) applicantPreviewResponse(preview *model.ApplicantPreview) map[string]any {
	if preview == nil {
		return map[string]any{
			"userId":         nil,
			"displayName":    nil,
			"firstName":      nil,
			"lastName":       nil,
			"middleName":     nil,
			"avatarMediaId":  nil,
			"universityName": nil,
			"city":           nil,
			"graduationYear": nil,
			"tags":           []any{},
		}
	}
	return map[string]any{
		"userId":         preview.UserID,
		"displayName":    preview.DisplayName,
		"firstName":      preview.FirstName,
		"lastName":       preview.LastName,
		"middleName":     nullableString(preview.MiddleName),
		"avatarMediaId":  nullableString(preview.AvatarMediaID),
		"universityName": nullableString(preview.UniversityName),
		"city":           nullableString(preview.City),
		"graduationYear": nullableInt(preview.GraduationYear),
		"tags":           s.tagsResponse(preview.Tags),
	}
}

func (s *Server) savedOpportunitiesResponse(items []*model.SavedOpportunity) []any {
	response := make([]any, 0, len(items))
	for _, item := range items {
		response = append(response, map[string]any{
			"applicantUserId": item.ApplicantUserID,
			"opportunityId":   item.OpportunityID,
			"createdAt":       timestamp(item.CreatedAt),
			"opportunity":     s.employerOpportunityCatalogItemResponse(item.Opportunity),
		})
	}
	return response
}

func (s *Server) companySocialLinksResponse(items []model.CompanySocialLink) []any {
	response := make([]any, 0, len(items))
	for _, link := range items {
		response = append(response, s.companySocialLinkResponse(link))
	}
	return response
}

func (s *Server) companySocialLinksPointersResponse(items []*model.CompanySocialLink) []any {
	response := make([]any, 0, len(items))
	for _, link := range items {
		response = append(response, s.companySocialLinkPointerResponse(link))
	}
	return response
}

func (s *Server) companySocialLinkResponse(link model.CompanySocialLink) map[string]any {
	return map[string]any{
		"id":        link.ID,
		"companyId": link.CompanyID,
		"platform":  link.Platform,
		"url":       link.URL,
	}
}

func (s *Server) companySocialLinkPointerResponse(link *model.CompanySocialLink) map[string]any {
	return s.companySocialLinkResponse(*link)
}

func (s *Server) savedCompaniesResponse(items []*model.SavedCompany) []any {
	response := make([]any, 0, len(items))
	for _, item := range items {
		response = append(response, map[string]any{
			"applicantUserId": item.ApplicantUserID,
			"companyId":       item.CompanyID,
			"createdAt":       timestamp(item.CreatedAt),
			"company":         s.companySummaryResponse(item.Company),
		})
	}
	return response
}

func (s *Server) connectionsResponse(items []*model.Connection) []any {
	response := make([]any, 0, len(items))
	for _, connection := range items {
		response = append(response, s.connectionResponse(connection))
	}
	return response
}

func (s *Server) connectionResponse(connection *model.Connection) map[string]any {
	return map[string]any{
		"id":              connection.ID,
		"initiatorUserId": connection.InitiatorUserID,
		"status":          connection.Status,
		"initiatorNote":   nullableString(connection.InitiatorNote),
		"respondedAt":     nullableTime(connection.RespondedAt),
		"createdAt":       timestamp(connection.CreatedAt),
		"otherApplicant":  s.applicantPreviewResponse(connection.OtherApplicant),
	}
}

func (s *Server) opportunityRecommendationsResponse(items []*model.OpportunityRecommendation) []any {
	response := make([]any, 0, len(items))
	for _, recommendation := range items {
		response = append(response, s.opportunityRecommendationResponse(recommendation))
	}
	return response
}

func (s *Server) opportunityRecommendationResponse(recommendation *model.OpportunityRecommendation) map[string]any {
	return map[string]any{
		"id":                recommendation.ID,
		"recommenderUserId": recommendation.RecommenderUserID,
		"recipientUserId":   recommendation.RecipientUserID,
		"opportunityId":     recommendation.OpportunityID,
		"message":           nullableString(recommendation.Message),
		"createdAt":         timestamp(recommendation.CreatedAt),
	}
}

func (s *Server) myApplicationsResponse(items []*model.Application) []any {
	response := make([]any, 0, len(items))
	for _, application := range items {
		response = append(response, s.applicationItemResponse(application))
	}
	return response
}

func (s *Server) employerApplicationsResponse(items []*model.Application) []any {
	response := make([]any, 0, len(items))
	for _, application := range items {
		response = append(response, s.employerApplicationItemResponse(application))
	}
	return response
}

func (s *Server) applicationItemResponse(application *model.Application) map[string]any {
	return map[string]any{
		"id":              application.ID,
		"opportunity":     s.employerOpportunityCatalogItemResponse(application.Opportunity),
		"applicantUserId": application.ApplicantUserID,
		"coverLetter":     nullableString(application.CoverLetter),
		"status":          application.Status,
		"appliedAt":       timestamp(application.AppliedAt),
		"updatedAt":       timestamp(application.UpdatedAt),
	}
}

func (s *Server) employerApplicationItemResponse(application *model.Application) map[string]any {
	body := s.applicationItemResponse(application)
	body["applicant"] = s.applicantPreviewResponse(application.Applicant)
	return body
}

func (s *Server) applicationDetailResponse(application *model.Application) map[string]any {
	body := s.applicationItemResponse(application)
	body["history"] = s.applicationHistoryResponse(application.History)
	return body
}

func (s *Server) employerApplicationDetailResponse(application *model.Application) map[string]any {
	body := s.employerApplicationItemResponse(application)
	body["history"] = s.applicationHistoryResponse(application.History)
	return body
}

func (s *Server) applicationHistoryResponse(items []model.ApplicationStatusHistory) []any {
	response := make([]any, 0, len(items))
	for _, item := range items {
		response = append(response, map[string]any{
			"id":              item.ID,
			"applicationId":   item.ApplicationID,
			"oldStatus":       nullableString(item.OldStatus),
			"newStatus":       item.NewStatus,
			"changedByUserId": item.ChangedByUserID,
			"comment":         nullableString(item.Comment),
			"createdAt":       timestamp(item.CreatedAt),
		})
	}
	return response
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

func (s *Server) companyMediaResponse(items []model.CompanyMedia) []any {
	response := make([]any, 0, len(items))
	for _, media := range items {
		response = append(response, s.companyMediaItemResponse(media))
	}
	return response
}

func (s *Server) companyMediaPointersResponse(items []*model.CompanyMedia) []any {
	response := make([]any, 0, len(items))
	for _, media := range items {
		response = append(response, s.companyMediaPointerResponse(media))
	}
	return response
}

func (s *Server) companyMediaItemResponse(media model.CompanyMedia) map[string]any {
	return map[string]any{
		"id":          media.ID,
		"companyId":   media.CompanyID,
		"mediaFileId": media.MediaFileID,
		"title":       nullableString(media.Title),
		"sortOrder":   media.SortOrder,
		"createdAt":   timestamp(media.CreatedAt),
	}
}

func (s *Server) companyMediaPointerResponse(media *model.CompanyMedia) map[string]any {
	return s.companyMediaItemResponse(*media)
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

func (s *Server) notificationsResponse(items []*model.Notification) []any {
	response := make([]any, 0, len(items))
	for _, notification := range items {
		response = append(response, s.notificationResponse(notification))
	}
	return response
}

func (s *Server) notificationResponse(notification *model.Notification) map[string]any {
	return map[string]any{
		"id":              notification.ID,
		"recipientUserId": notification.RecipientUserID,
		"actorUserId":     nullableString(notification.ActorUserID),
		"type":            notification.Type,
		"sourceType":      notification.SourceType,
		"sourceId":        nullableString(notification.SourceID),
		"companyId":       nullableString(notification.CompanyID),
		"opportunityId":   nullableString(notification.OpportunityID),
		"applicationId":   nullableString(notification.ApplicationID),
		"title":           notification.Title,
		"body":            nullableString(notification.Body),
		"isRead":          notification.IsRead,
		"readAt":          nullableTime(notification.ReadAt),
		"createdAt":       timestamp(notification.CreatedAt),
	}
}

func (s *Server) notificationCampaignsResponse(items []*model.NotificationCampaign) []any {
	response := make([]any, 0, len(items))
	for _, campaign := range items {
		response = append(response, s.notificationCampaignResponse(campaign))
	}
	return response
}

func (s *Server) notificationCampaignDetailResponse(campaign *model.NotificationCampaign) map[string]any {
	body := s.notificationCampaignResponse(campaign)
	body["recipients"] = s.notificationCampaignRecipientsResponse(campaign.Recipients)
	return body
}

func (s *Server) notificationCampaignResponse(campaign *model.NotificationCampaign) map[string]any {
	return map[string]any{
		"id":              campaign.ID,
		"companyId":       campaign.CompanyID,
		"createdByUserId": campaign.CreatedByUserID,
		"opportunityId":   nullableString(campaign.OpportunityID),
		"audienceType":    campaign.AudienceType,
		"status":          campaign.Status,
		"title":           campaign.Title,
		"body":            campaign.Body,
		"sendViaInApp":    campaign.SendViaInApp,
		"sendViaEmail":    campaign.SendViaEmail,
		"scheduledAt":     nullableTime(campaign.ScheduledAt),
		"sentAt":          nullableTime(campaign.SentAt),
		"createdAt":       timestamp(campaign.CreatedAt),
	}
}

func (s *Server) notificationCampaignRecipientsResponse(items []model.NotificationCampaignRecipient) []any {
	response := make([]any, 0, len(items))
	for _, recipient := range items {
		response = append(response, map[string]any{
			"campaignId":      recipient.CampaignID,
			"applicantUserId": recipient.ApplicantUserID,
			"applicationId":   nullableString(recipient.ApplicationID),
			"createdAt":       timestamp(recipient.CreatedAt),
		})
	}
	return response
}

func (s *Server) verificationRequestsResponse(items []*model.VerificationRequest) []any {
	response := make([]any, 0, len(items))
	for _, item := range items {
		response = append(response, s.verificationRequestResponse(item))
	}
	return response
}

func (s *Server) verificationRequestResponse(item *model.VerificationRequest) map[string]any {
	return map[string]any{
		"id":                        item.ID,
		"companyId":                 item.CompanyID,
		"submittedByUserId":         item.SubmittedByUserID,
		"method":                    item.Method,
		"status":                    item.Status,
		"companyVerificationStatus": item.CompanyVerificationStatus,
		"submittedComment":          nullableString(item.SubmittedComment),
		"reviewComment":             nullableString(item.ReviewComment),
		"reviewedByCuratorUserId":   nullableString(item.ReviewedByCuratorUserID),
		"reviewedAt":                nullableTime(item.ReviewedAt),
		"createdAt":                 timestamp(item.CreatedAt),
		"evidence":                  s.verificationEvidenceResponse(item.Evidence),
	}
}

func (s *Server) verificationEvidenceResponse(items []model.VerificationEvidence) []any {
	response := make([]any, 0, len(items))
	for _, item := range items {
		response = append(response, map[string]any{
			"id":                    item.ID,
			"verificationRequestId": item.VerificationRequestID,
			"evidenceType":          item.EvidenceType,
			"value":                 nullableString(item.Value),
			"evidenceFileId":        nullableString(item.EvidenceFileID),
			"createdAt":             timestamp(item.CreatedAt),
		})
	}
	return response
}

func (s *Server) moderationCasesResponse(items []*model.ModerationCase) []any {
	response := make([]any, 0, len(items))
	for _, item := range items {
		response = append(response, s.moderationCaseResponse(item))
	}
	return response
}

func (s *Server) moderationCaseResponse(item *model.ModerationCase) map[string]any {
	return map[string]any{
		"id":                      item.ID,
		"targetType":              item.TargetType,
		"targetId":                item.TargetID,
		"submittedByUserId":       nullableString(item.SubmittedByUserID),
		"assignedCuratorUserId":   nullableString(item.AssignedCuratorUserID),
		"resolvedByCuratorUserId": nullableString(item.ResolvedByCuratorUserID),
		"status":                  item.Status,
		"reason":                  nullableString(item.Reason),
		"createdAt":               timestamp(item.CreatedAt),
		"resolvedAt":              nullableTime(item.ResolvedAt),
		"targetPreview":           s.moderationTargetPreviewResponse(item.TargetPreview),
	}
}

func (s *Server) moderationTargetPreviewResponse(item *model.ModerationTargetPreview) any {
	if item == nil {
		return nil
	}
	return map[string]any{
		"type":     item.Type,
		"id":       item.ID,
		"title":    nullableString(item.Title),
		"subtitle": nullableString(item.Subtitle),
		"slug":     nullableString(item.Slug),
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
