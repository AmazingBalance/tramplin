package memory

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

type AppError = commonstore.AppError

type Store struct {
	mu                      sync.RWMutex
	users                   map[string]*model.User
	userIDsByEmail          map[string]string
	applicantProfiles       map[string]*model.ApplicantProfile
	applicantPrivacy        map[string]*model.ApplicantPrivacySettings
	employerProfiles        map[string]*model.EmployerProfile
	uiSettings              map[string]*model.UISettings
	locations               map[string]*model.Location
	tags                    map[string]*model.Tag
	companies               map[string]*model.Company
	companyIDsBySlug        map[string]string
	memberships             map[string]*model.CompanyMembership
	membershipIDsByPair     map[string]string
	notificationPreferences map[string]*model.NotificationPreferences
}

func New() *Store {
	s := &Store{
		users:                   make(map[string]*model.User),
		userIDsByEmail:          make(map[string]string),
		applicantProfiles:       make(map[string]*model.ApplicantProfile),
		applicantPrivacy:        make(map[string]*model.ApplicantPrivacySettings),
		employerProfiles:        make(map[string]*model.EmployerProfile),
		uiSettings:              make(map[string]*model.UISettings),
		locations:               make(map[string]*model.Location),
		tags:                    make(map[string]*model.Tag),
		companies:               make(map[string]*model.Company),
		companyIDsBySlug:        make(map[string]string),
		memberships:             make(map[string]*model.CompanyMembership),
		membershipIDsByPair:     make(map[string]string),
		notificationPreferences: make(map[string]*model.NotificationPreferences),
	}
	s.seedTags()
	return s
}

func (s *Store) CreateApplicant(input model.RegisterApplicantInput, passwordHash string) (*model.User, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	emailKey := normalizeEmail(input.Email)
	if _, exists := s.userIDsByEmail[emailKey]; exists {
		return nil, appErr(409, "conflict", "user with this email already exists", nil)
	}

	now := time.Now().UTC()
	userID := newID()
	user := &model.User{
		ID:           userID,
		Email:        emailKey,
		PasswordHash: passwordHash,
		DisplayName:  input.DisplayName,
		Role:         model.UserRoleApplicant,
		IsActive:     true,
		TokenVersion: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	profile := &model.ApplicantProfile{
		UserID:      userID,
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		MiddleName:  input.MiddleName,
		TagIDs:      []string{},
		SocialLinks: []model.ApplicantSocialLink{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	privacy := &model.ApplicantPrivacySettings{
		ApplicantUserID:        userID,
		ProfileVisibility:      "authenticated_public",
		ResumeVisibility:       "contacts_only",
		ApplicationsVisibility: "private",
		ContactsVisibility:     "contacts_only",
		ShowCareerInterests:    true,
		AllowRecommendations:   true,
		UpdatedAt:              now,
	}

	s.users[userID] = user
	s.userIDsByEmail[emailKey] = userID
	s.applicantProfiles[userID] = profile
	s.applicantPrivacy[userID] = privacy
	return cloneUser(user), nil
}

func (s *Store) CreateEmployer(input model.RegisterEmployerInput, passwordHash string) (*model.User, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	emailKey := normalizeEmail(input.Email)
	if _, exists := s.userIDsByEmail[emailKey]; exists {
		return nil, appErr(409, "conflict", "user with this email already exists", nil)
	}

	now := time.Now().UTC()
	userID := newID()
	user := &model.User{
		ID:           userID,
		Email:        emailKey,
		PasswordHash: passwordHash,
		DisplayName:  input.DisplayName,
		Role:         model.UserRoleEmployer,
		IsActive:     true,
		TokenVersion: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	profile := &model.EmployerProfile{
		UserID:    userID,
		FullName:  input.FullName,
		JobTitle:  input.JobTitle,
		Phone:     input.Phone,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.users[userID] = user
	s.userIDsByEmail[emailKey] = userID
	s.employerProfiles[userID] = profile
	return cloneUser(user), nil
}

func (s *Store) GetUserByEmail(email string) (*model.User, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.userIDsByEmail[normalizeEmail(email)]
	if !ok {
		return nil, appErr(401, "unauthorized", "invalid credentials", nil)
	}
	return cloneUser(s.users[id]), nil
}

func (s *Store) GetUserByID(userID string) (*model.User, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[userID]
	if !ok {
		return nil, appErr(401, "unauthorized", "authentication failed", nil)
	}
	return cloneUser(user), nil
}

func (s *Store) GetCurrentCuratorProfile(userID string) (*model.CurrentCuratorProfile, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[userID]
	if !ok || user.Role != model.UserRoleCurator || !user.IsActive {
		return nil, appErr(500, "internal_error", "failed to load curator profile", nil)
	}
	return &model.CurrentCuratorProfile{IsAdmin: false}, nil
}

func (s *Store) TouchLastLogin(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if user, ok := s.users[userID]; ok {
		now := time.Now().UTC()
		user.LastLoginAt = &now
		user.UpdatedAt = now
	}
}

func (s *Store) BumpTokenVersion(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if user, ok := s.users[userID]; ok {
		user.TokenVersion++
		user.UpdatedAt = time.Now().UTC()
	}
}

func (s *Store) GetUISettings(userID string) (*model.UISettings, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if settings, ok := s.uiSettings[userID]; ok {
		return cloneUISettings(settings), nil
	}

	now := time.Now().UTC()
	settings := &model.UISettings{
		UserID:        userID,
		SettingsJSON:  map[string]any{},
		SchemaVersion: 1,
		UpdatedAt:     now,
	}
	s.uiSettings[userID] = settings
	return cloneUISettings(settings), nil
}

func (s *Store) PutUISettings(userID string, input model.PutUISettingsInput) (*model.UISettings, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	schemaVersion := 1
	if input.SchemaVersion != nil {
		schemaVersion = *input.SchemaVersion
	}

	settings := &model.UISettings{
		UserID:        userID,
		SettingsJSON:  cloneMap(input.SettingsJSON),
		SchemaVersion: schemaVersion,
		UpdatedAt:     now,
	}
	s.uiSettings[userID] = settings
	return cloneUISettings(settings), nil
}

func (s *Store) GetApplicantProfile(userID string) (*model.ApplicantProfile, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile, ok := s.applicantProfiles[userID]
	if !ok {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}
	return cloneApplicantProfile(profile), nil
}

func (s *Store) UpdateApplicantProfile(userID string, input model.UpdateApplicantProfileInput) (*model.ApplicantProfile, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, ok := s.applicantProfiles[userID]
	if !ok {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}

	if input.FirstName != nil {
		profile.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		profile.LastName = *input.LastName
	}
	if input.MiddleName != nil {
		profile.MiddleName = input.MiddleName
	}
	if input.UniversityName != nil {
		profile.UniversityName = input.UniversityName
	}
	if input.Faculty != nil {
		profile.Faculty = input.Faculty
	}
	if input.ProgramName != nil {
		profile.ProgramName = input.ProgramName
	}
	if input.StudyYear != nil {
		profile.StudyYear = input.StudyYear
	}
	if input.GraduationYear != nil {
		profile.GraduationYear = input.GraduationYear
	}
	if input.City != nil {
		profile.City = input.City
	}
	if input.About != nil {
		profile.About = input.About
	}
	if input.ResumeMediaID != nil {
		profile.ResumeMediaID = input.ResumeMediaID
	}
	profile.UpdatedAt = time.Now().UTC()
	return cloneApplicantProfile(profile), nil
}

func (s *Store) GetApplicantPrivacy(userID string) (*model.ApplicantPrivacySettings, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	privacy, ok := s.applicantPrivacy[userID]
	if !ok {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}
	return cloneApplicantPrivacy(privacy), nil
}

func (s *Store) UpdateApplicantPrivacy(userID string, input model.UpdateApplicantPrivacyInput) (*model.ApplicantPrivacySettings, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	privacy, ok := s.applicantPrivacy[userID]
	if !ok {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}
	if input.ProfileVisibility != nil {
		privacy.ProfileVisibility = *input.ProfileVisibility
	}
	if input.ResumeVisibility != nil {
		privacy.ResumeVisibility = *input.ResumeVisibility
	}
	if input.ApplicationsVisibility != nil {
		privacy.ApplicationsVisibility = *input.ApplicationsVisibility
	}
	if input.ContactsVisibility != nil {
		privacy.ContactsVisibility = *input.ContactsVisibility
	}
	if input.ShowCareerInterests != nil {
		privacy.ShowCareerInterests = *input.ShowCareerInterests
	}
	if input.AllowRecommendations != nil {
		privacy.AllowRecommendations = *input.AllowRecommendations
	}
	privacy.UpdatedAt = time.Now().UTC()
	return cloneApplicantPrivacy(privacy), nil
}

func (s *Store) ReplaceApplicantTags(userID string, tagIDs []string) ([]*model.Tag, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, ok := s.applicantProfiles[userID]
	if !ok {
		return nil, appErr(403, "forbidden", "current user is not an applicant", nil)
	}

	seen := make(map[string]struct{}, len(tagIDs))
	resolved := make([]*model.Tag, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		if _, exists := seen[tagID]; exists {
			continue
		}
		tag, exists := s.tags[tagID]
		if !exists {
			return nil, appErr(422, "validation_error", "unknown tag id", map[string]any{"tagId": tagID})
		}
		seen[tagID] = struct{}{}
		resolved = append(resolved, cloneTag(tag))
	}

	profile.TagIDs = orderedKeys(seen)
	profile.UpdatedAt = time.Now().UTC()
	sort.Slice(resolved, func(i, j int) bool {
		return resolved[i].Name < resolved[j].Name
	})
	return resolved, nil
}

func (s *Store) GetEmployerProfile(userID string) (*model.EmployerProfile, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profile, ok := s.employerProfiles[userID]
	if !ok {
		return nil, appErr(403, "forbidden", "current user is not an employer", nil)
	}
	return cloneEmployerProfile(profile), nil
}

func (s *Store) UpdateEmployerProfile(userID string, input model.UpdateEmployerProfileInput) (*model.EmployerProfile, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	profile, ok := s.employerProfiles[userID]
	if !ok {
		return nil, appErr(403, "forbidden", "current user is not an employer", nil)
	}
	if input.FullName != nil {
		profile.FullName = *input.FullName
	}
	if input.JobTitle != nil {
		profile.JobTitle = input.JobTitle
	}
	if input.Phone != nil {
		profile.Phone = input.Phone
	}
	profile.UpdatedAt = time.Now().UTC()
	return cloneEmployerProfile(profile), nil
}

func (s *Store) ListPublicTags(tagType, q string) ([]*model.Tag, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*model.Tag, 0)
	for _, tag := range s.tags {
		if !tag.IsActive {
			continue
		}
		if tagType != "" && tag.TagType != tagType {
			continue
		}
		if q != "" && !containsFold(tag.Name, q) {
			continue
		}
		items = append(items, cloneTag(tag))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func (s *Store) CreateOrReuseLocation(input model.LocationInput) (*model.Location, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, location := range s.locations {
		if sameLocation(location, input) {
			return cloneLocation(location), nil
		}
	}

	now := time.Now().UTC()
	location := &model.Location{
		ID:          newID(),
		Precision:   input.Precision,
		Country:     input.Country,
		Region:      input.Region,
		City:        input.City,
		AddressLine: input.AddressLine,
		PostalCode:  input.PostalCode,
		PlaceName:   input.PlaceName,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		CreatedAt:   now,
	}

	s.locations[location.ID] = location
	return cloneLocation(location), nil
}

func (s *Store) SearchLocations(q string, page, pageSize int) ([]*model.Location, int, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*model.Location, 0)
	for _, location := range s.locations {
		if q == "" ||
			containsFold(location.Country, q) ||
			containsFold(location.City, q) ||
			(location.Region != nil && containsFold(*location.Region, q)) ||
			(location.AddressLine != nil && containsFold(*location.AddressLine, q)) ||
			(location.PlaceName != nil && containsFold(*location.PlaceName, q)) {
			items = append(items, cloneLocation(location))
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	paged, total := paginate(items, page, pageSize)
	return paged, total, nil
}

func (s *Store) CreateCompany(actorUserID string, input model.CreateCompanyInput) (*model.Company, *model.CompanyMembership, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.employerProfiles[actorUserID]; !ok {
		return nil, nil, appErr(403, "forbidden", "current user is not an employer", nil)
	}

	if input.HeadquartersLocationID != nil && input.HeadquartersLocation != nil {
		return nil, nil, appErr(422, "validation_error", "provide either headquartersLocationId or headquartersLocation", nil)
	}

	slugKey := strings.TrimSpace(input.Slug)
	if slugKey == "" {
		return nil, nil, appErr(422, "validation_error", "slug is required", nil)
	}
	if _, exists := s.companyIDsBySlug[slugKey]; exists {
		return nil, nil, appErr(409, "conflict", "company slug already exists", nil)
	}

	headquartersLocationID, err := s.resolveLocationLocked(input.HeadquartersLocationID, input.HeadquartersLocation)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now().UTC()
	companyID := newID()
	company := &model.Company{
		ID:                     companyID,
		LegalName:              input.LegalName,
		BrandName:              input.BrandName,
		Slug:                   slugKey,
		INN:                    input.INN,
		Description:            input.Description,
		Industry:               input.Industry,
		WebsiteURL:             input.WebsiteURL,
		CorporateEmailDomain:   input.CorporateEmailDomain,
		HeadquartersLocationID: headquartersLocationID,
		LogoMediaID:            input.LogoMediaID,
		BannerMediaID:          input.BannerMediaID,
		VerificationStatus:     "pending",
		CreatedByUserID:        stringPtr(actorUserID),
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	membership := &model.CompanyMembership{
		ID:               newID(),
		CompanyID:        companyID,
		EmployerUserID:   actorUserID,
		Status:           model.CompanyMembershipApproved,
		MemberRole:       model.CompanyRoleOwner,
		IsPrimaryContact: true,
		StatusUpdatedAt:  now,
		ApprovedAt:       &now,
		UpdatedAt:        now,
		CreatedAt:        now,
	}

	s.companies[companyID] = company
	s.companyIDsBySlug[slugKey] = companyID
	s.memberships[membership.ID] = membership
	s.membershipIDsByPair[membershipKey(companyID, actorUserID)] = membership.ID
	return cloneCompany(company), cloneMembership(membership), nil
}

func (s *Store) UpdateCompany(actorUserID, companyID string, input model.UpdateCompanyInput) (*model.Company, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.requireApprovedMembershipLocked(actorUserID, companyID); err != nil {
		return nil, err
	}

	if input.HeadquartersLocationID != nil && input.HeadquartersLocation != nil {
		return nil, appErr(422, "validation_error", "provide either headquartersLocationId or headquartersLocation", nil)
	}

	company, ok := s.companies[companyID]
	if !ok {
		return nil, appErr(404, "not_found", "company not found", nil)
	}

	if input.Slug != nil && *input.Slug != company.Slug {
		if _, exists := s.companyIDsBySlug[*input.Slug]; exists {
			return nil, appErr(409, "conflict", "company slug already exists", nil)
		}
		delete(s.companyIDsBySlug, company.Slug)
		s.companyIDsBySlug[*input.Slug] = company.ID
		company.Slug = *input.Slug
	}

	if input.LegalName != nil {
		company.LegalName = *input.LegalName
	}
	if input.BrandName != nil {
		company.BrandName = input.BrandName
	}
	if input.INN != nil {
		company.INN = input.INN
	}
	if input.Description != nil {
		company.Description = input.Description
	}
	if input.Industry != nil {
		company.Industry = input.Industry
	}
	if input.WebsiteURL != nil {
		company.WebsiteURL = input.WebsiteURL
	}
	if input.CorporateEmailDomain != nil {
		company.CorporateEmailDomain = input.CorporateEmailDomain
	}
	if input.LogoMediaID != nil {
		company.LogoMediaID = input.LogoMediaID
	}
	if input.BannerMediaID != nil {
		company.BannerMediaID = input.BannerMediaID
	}

	if input.HeadquartersLocationID != nil || input.HeadquartersLocation != nil {
		locationID, err := s.resolveLocationLocked(input.HeadquartersLocationID, input.HeadquartersLocation)
		if err != nil {
			return nil, err
		}
		company.HeadquartersLocationID = locationID
	}

	company.UpdatedAt = time.Now().UTC()
	return cloneCompany(company), nil
}

func (s *Store) ListEmployerMemberships(userID string) ([]*model.CompanyMembership, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.employerProfiles[userID]; !ok {
		return nil, appErr(403, "forbidden", "current user is not an employer", nil)
	}

	items := make([]*model.CompanyMembership, 0)
	for _, membership := range s.memberships {
		if membership.EmployerUserID == userID {
			items = append(items, cloneMembership(membership))
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Store) GetCompanyForEmployer(userID, companyID string) (*model.Company, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := s.requireApprovedMembershipLocked(userID, companyID); err != nil {
		return nil, err
	}
	company, ok := s.companies[companyID]
	if !ok {
		return nil, appErr(404, "not_found", "company not found", nil)
	}
	return cloneCompany(company), nil
}

func (s *Store) ListCompanyMemberships(actorUserID, companyID, status string) ([]*model.CompanyMembership, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := s.requireApprovedMembershipLocked(actorUserID, companyID); err != nil {
		return nil, err
	}

	items := make([]*model.CompanyMembership, 0)
	for _, membership := range s.memberships {
		if membership.CompanyID != companyID {
			continue
		}
		if status != "" && membership.Status != status {
			continue
		}
		items = append(items, cloneMembership(membership))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Store) CreateCompanyMembership(actorUserID, companyID string, input model.CreateCompanyMembershipInput) (*model.CompanyMembership, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	actorMembership, err := s.requireApprovedMembershipLocked(actorUserID, companyID)
	if err != nil {
		return nil, err
	}

	targetUserID, ok := s.userIDsByEmail[normalizeEmail(input.EmployerEmail)]
	if !ok {
		return nil, appErr(404, "not_found", "target employer account not found", nil)
	}

	targetUser := s.users[targetUserID]
	if targetUser.Role != model.UserRoleEmployer {
		return nil, appErr(404, "not_found", "target employer account not found", nil)
	}

	if !canInviteIntoRole(actorMembership.MemberRole, input.MemberRole, input.IsPrimaryContact) {
		return nil, appErr(403, "forbidden", "current employer cannot invite members to this company or requested role is not allowed", nil)
	}

	key := membershipKey(companyID, targetUserID)
	if existingID, exists := s.membershipIDsByPair[key]; exists {
		existing := s.memberships[existingID]
		if existing.Status == model.CompanyMembershipPending || existing.Status == model.CompanyMembershipApproved {
			return nil, appErr(409, "conflict", "membership already exists in pending or approved state", nil)
		}

		now := time.Now().UTC()
		existing.Status = model.CompanyMembershipPending
		existing.MemberRole = input.MemberRole
		existing.IsPrimaryContact = input.IsPrimaryContact
		existing.InvitedByUserID = stringPtr(actorUserID)
		existing.StatusChangedByUserID = stringPtr(actorUserID)
		existing.StatusComment = input.Comment
		existing.StatusUpdatedAt = now
		existing.UpdatedAt = now
		return cloneMembership(existing), nil
	}

	now := time.Now().UTC()
	membership := &model.CompanyMembership{
		ID:                    newID(),
		CompanyID:             companyID,
		EmployerUserID:        targetUserID,
		InvitedByUserID:       stringPtr(actorUserID),
		Status:                model.CompanyMembershipPending,
		MemberRole:            input.MemberRole,
		IsPrimaryContact:      input.IsPrimaryContact,
		StatusChangedByUserID: stringPtr(actorUserID),
		StatusComment:         input.Comment,
		StatusUpdatedAt:       now,
		UpdatedAt:             now,
		CreatedAt:             now,
	}

	s.memberships[membership.ID] = membership
	s.membershipIDsByPair[key] = membership.ID
	return cloneMembership(membership), nil
}

func (s *Store) UpdateCompanyMembership(actorUserID, companyID, membershipID string, input model.UpdateCompanyMembershipInput) (*model.CompanyMembership, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	actorMembership, err := s.requireApprovedMembershipLocked(actorUserID, companyID)
	if err != nil {
		return nil, err
	}

	target, ok := s.memberships[membershipID]
	if !ok || target.CompanyID != companyID {
		return nil, appErr(404, "not_found", "membership not found", nil)
	}
	if target.Status != model.CompanyMembershipApproved {
		return nil, appErr(409, "conflict", "requested change would violate membership lifecycle or owner invariants", nil)
	}

	if !canEditMembership(actorMembership.MemberRole, target.MemberRole) {
		return nil, appErr(403, "forbidden", "current employer cannot change this membership", nil)
	}

	if input.MemberRole != nil {
		if !canAssignRole(actorMembership.MemberRole, target.MemberRole, *input.MemberRole) {
			return nil, appErr(403, "forbidden", "current employer cannot change this membership", nil)
		}
		if target.MemberRole == model.CompanyRoleOwner && *input.MemberRole != model.CompanyRoleOwner && s.countApprovedOwnersLocked(companyID) <= 1 {
			return nil, appErr(409, "conflict", "requested change would violate membership lifecycle or owner invariants", nil)
		}
		target.MemberRole = *input.MemberRole
	}

	if input.IsPrimaryContact != nil {
		if *input.IsPrimaryContact {
			if actorMembership.MemberRole != model.CompanyRoleOwner {
				return nil, appErr(403, "forbidden", "current employer cannot change this membership", nil)
			}
			for _, membership := range s.memberships {
				if membership.CompanyID == companyID && membership.IsPrimaryContact {
					membership.IsPrimaryContact = false
					membership.UpdatedAt = time.Now().UTC()
				}
			}
			target.IsPrimaryContact = true
		} else if target.IsPrimaryContact {
			return nil, appErr(409, "conflict", "requested change would violate membership lifecycle or owner invariants", nil)
		}
	}

	target.UpdatedAt = time.Now().UTC()
	return cloneMembership(target), nil
}

func (s *Store) ApproveCompanyMembership(actorUserID, companyID, membershipID string, comment *string) (*model.CompanyMembership, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transitionMembershipLocked(actorUserID, companyID, membershipID, model.CompanyMembershipApproved, comment)
}

func (s *Store) RejectCompanyMembership(actorUserID, companyID, membershipID string, comment string) (*model.CompanyMembership, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transitionMembershipLocked(actorUserID, companyID, membershipID, model.CompanyMembershipRejected, &comment)
}

func (s *Store) RevokeCompanyMembership(actorUserID, companyID, membershipID string, comment string) (*model.CompanyMembership, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	actorMembership, err := s.requireApprovedMembershipLocked(actorUserID, companyID)
	if err != nil {
		return nil, err
	}

	target, ok := s.memberships[membershipID]
	if !ok || target.CompanyID != companyID {
		return nil, appErr(404, "not_found", "membership not found", nil)
	}
	if target.Status != model.CompanyMembershipApproved {
		return nil, appErr(409, "conflict", "membership is not approved or revocation would remove the last approved owner", nil)
	}
	if !canFinalizeMembership(actorMembership.MemberRole, target.MemberRole) {
		return nil, appErr(403, "forbidden", "current employer cannot revoke this membership", nil)
	}
	if target.MemberRole == model.CompanyRoleOwner && s.countApprovedOwnersLocked(companyID) <= 1 {
		return nil, appErr(409, "conflict", "membership is not approved or revocation would remove the last approved owner", nil)
	}

	now := time.Now().UTC()
	target.Status = model.CompanyMembershipRevoked
	target.StatusChangedByUserID = stringPtr(actorUserID)
	target.StatusComment = &comment
	target.StatusUpdatedAt = now
	target.UpdatedAt = now
	target.IsPrimaryContact = false
	return cloneMembership(target), nil
}

func (s *Store) ListPublicCompanies(q, industry, city string, page, pageSize int) ([]*model.Company, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*model.Company, 0)
	for _, company := range s.companies {
		if q != "" && !containsFold(company.LegalName, q) && (company.BrandName == nil || !containsFold(*company.BrandName, q)) {
			continue
		}
		if industry != "" && (company.Industry == nil || !containsFold(*company.Industry, industry)) {
			continue
		}
		if city != "" {
			location := s.locationByIDLocked(company.HeadquartersLocationID)
			if location == nil || !containsFold(location.City, city) {
				continue
			}
		}
		items = append(items, cloneCompany(company))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	paged, total := paginate(items, page, pageSize)
	return paged, total
}

func (s *Store) GetPublicCompanyByID(companyID string) (*model.Company, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	company, ok := s.companies[companyID]
	if !ok {
		return nil, appErr(404, "not_found", "company not found", nil)
	}
	return cloneCompany(company), nil
}

func (s *Store) GetPublicCompanyBySlug(slug string) (*model.Company, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	companyID, ok := s.companyIDsBySlug[slug]
	if !ok {
		return nil, appErr(404, "not_found", "company not found", nil)
	}
	return cloneCompany(s.companies[companyID]), nil
}

func (s *Store) GetLocationByID(locationID *string) *model.Location {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneLocation(s.locationByIDLocked(locationID))
}

func (s *Store) GetEmployerSnapshot(userID string) (*model.User, *model.EmployerProfile) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneUser(s.users[userID]), cloneEmployerProfile(s.employerProfiles[userID])
}

func (s *Store) GetApplicantTags(userID string) ([]*model.Tag, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile := s.applicantProfiles[userID]
	if profile == nil {
		return []*model.Tag{}, nil
	}

	items := make([]*model.Tag, 0, len(profile.TagIDs))
	for _, tagID := range profile.TagIDs {
		if tag, ok := s.tags[tagID]; ok {
			items = append(items, cloneTag(tag))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (s *Store) GetNotificationPreferences(userID string) (*model.NotificationPreferences, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if preferences, ok := s.notificationPreferences[userID]; ok {
		return cloneNotificationPreferences(preferences), nil
	}

	now := time.Now().UTC()
	preferences := &model.NotificationPreferences{
		UserID:                   userID,
		InAppEnabled:             true,
		EmailEnabled:             true,
		RecommendationEnabled:    true,
		ApplicationStatusEnabled: true,
		EmployerMessagesEnabled:  true,
		SystemEnabled:            true,
		UpdatedAt:                now,
	}
	s.notificationPreferences[userID] = preferences
	return cloneNotificationPreferences(preferences), nil
}

func (s *Store) PatchNotificationPreferences(
	userID string,
	inAppEnabled, emailEnabled, recommendationEnabled, applicationStatusEnabled, employerMessagesEnabled, systemEnabled *bool,
) (*model.NotificationPreferences, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	preferences := s.notificationPreferences[userID]
	if preferences == nil {
		now := time.Now().UTC()
		preferences = &model.NotificationPreferences{
			UserID:                   userID,
			InAppEnabled:             true,
			EmailEnabled:             true,
			RecommendationEnabled:    true,
			ApplicationStatusEnabled: true,
			EmployerMessagesEnabled:  true,
			SystemEnabled:            true,
			UpdatedAt:                now,
		}
		s.notificationPreferences[userID] = preferences
	}

	if inAppEnabled != nil {
		preferences.InAppEnabled = *inAppEnabled
	}
	if emailEnabled != nil {
		preferences.EmailEnabled = *emailEnabled
	}
	if recommendationEnabled != nil {
		preferences.RecommendationEnabled = *recommendationEnabled
	}
	if applicationStatusEnabled != nil {
		preferences.ApplicationStatusEnabled = *applicationStatusEnabled
	}
	if employerMessagesEnabled != nil {
		preferences.EmployerMessagesEnabled = *employerMessagesEnabled
	}
	if systemEnabled != nil {
		preferences.SystemEnabled = *systemEnabled
	}
	preferences.UpdatedAt = time.Now().UTC()
	return cloneNotificationPreferences(preferences), nil
}

func (s *Store) resolveLocationLocked(locationID *string, input *model.LocationInput) (*string, *AppError) {
	if locationID != nil {
		if _, exists := s.locations[*locationID]; !exists {
			return nil, appErr(422, "validation_error", "unknown location id", nil)
		}
		return locationID, nil
	}

	if input == nil {
		return nil, nil
	}

	for _, location := range s.locations {
		if sameLocation(location, *input) {
			return stringPtr(location.ID), nil
		}
	}

	now := time.Now().UTC()
	location := &model.Location{
		ID:          newID(),
		Precision:   input.Precision,
		Country:     input.Country,
		Region:      input.Region,
		City:        input.City,
		AddressLine: input.AddressLine,
		PostalCode:  input.PostalCode,
		PlaceName:   input.PlaceName,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		CreatedAt:   now,
	}
	s.locations[location.ID] = location
	return stringPtr(location.ID), nil
}

func (s *Store) requireApprovedMembershipLocked(userID, companyID string) (*model.CompanyMembership, *AppError) {
	membershipID, ok := s.membershipIDsByPair[membershipKey(companyID, userID)]
	if !ok {
		return nil, appErr(403, "forbidden", "current employer has no access to this company", nil)
	}
	membership := s.memberships[membershipID]
	if membership.Status != model.CompanyMembershipApproved {
		return nil, appErr(403, "forbidden", "current employer has no access to this company", nil)
	}
	return membership, nil
}

func (s *Store) transitionMembershipLocked(actorUserID, companyID, membershipID, targetStatus string, comment *string) (*model.CompanyMembership, *AppError) {
	actorMembership, err := s.requireApprovedMembershipLocked(actorUserID, companyID)
	if err != nil {
		return nil, err
	}

	target, ok := s.memberships[membershipID]
	if !ok || target.CompanyID != companyID {
		return nil, appErr(404, "not_found", "membership not found", nil)
	}
	if target.Status != model.CompanyMembershipPending {
		if targetStatus == model.CompanyMembershipApproved {
			return nil, appErr(409, "conflict", "membership is not pending or approval would violate owner-role rules", nil)
		}
		return nil, appErr(409, "conflict", "membership is not pending or rejection would violate owner-role rules", nil)
	}
	if !canFinalizeMembership(actorMembership.MemberRole, target.MemberRole) {
		if targetStatus == model.CompanyMembershipApproved {
			return nil, appErr(403, "forbidden", "current employer cannot approve this membership", nil)
		}
		return nil, appErr(403, "forbidden", "current employer cannot reject this membership", nil)
	}

	now := time.Now().UTC()
	target.Status = targetStatus
	target.StatusChangedByUserID = stringPtr(actorUserID)
	target.StatusComment = comment
	target.StatusUpdatedAt = now
	target.UpdatedAt = now
	if targetStatus == model.CompanyMembershipApproved {
		target.ApprovedAt = &now
	}
	return cloneMembership(target), nil
}

func (s *Store) countApprovedOwnersLocked(companyID string) int {
	count := 0
	for _, membership := range s.memberships {
		if membership.CompanyID == companyID && membership.Status == model.CompanyMembershipApproved && membership.MemberRole == model.CompanyRoleOwner {
			count++
		}
	}
	return count
}

func (s *Store) locationByIDLocked(locationID *string) *model.Location {
	if locationID == nil {
		return nil
	}
	location, ok := s.locations[*locationID]
	if !ok {
		return nil
	}
	return location
}

func (s *Store) seedTags() {
	now := time.Now().UTC()
	for _, seed := range []struct {
		Name    string
		TagType string
	}{
		{Name: "Go", TagType: "technology"},
		{Name: "Python", TagType: "technology"},
		{Name: "Backend", TagType: "role"},
		{Name: "Junior", TagType: "level"},
		{Name: "Remote", TagType: "format"},
	} {
		tag := &model.Tag{
			ID:        newID(),
			Name:      seed.Name,
			TagType:   seed.TagType,
			IsSystem:  true,
			IsActive:  true,
			CreatedAt: now,
		}
		s.tags[tag.ID] = tag
	}
}

func orderedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func canInviteIntoRole(actorRole, requestedRole string, requestedPrimary bool) bool {
	if actorRole == model.CompanyRoleOwner {
		return true
	}
	if requestedPrimary || requestedRole == model.CompanyRoleOwner {
		return false
	}
	return requestedRole == model.CompanyRoleRecruiter || requestedRole == model.CompanyRoleHR || requestedRole == model.CompanyRoleManager
}

func canEditMembership(actorRole, targetRole string) bool {
	switch actorRole {
	case model.CompanyRoleOwner:
		return true
	case model.CompanyRoleManager:
		return targetRole != model.CompanyRoleOwner
	default:
		return false
	}
}

func canAssignRole(actorRole, targetRole, newRole string) bool {
	switch actorRole {
	case model.CompanyRoleOwner:
		return true
	case model.CompanyRoleManager:
		if targetRole == model.CompanyRoleOwner {
			return false
		}
		return newRole != model.CompanyRoleOwner
	default:
		return false
	}
}

func canFinalizeMembership(actorRole, targetRole string) bool {
	switch actorRole {
	case model.CompanyRoleOwner:
		return true
	case model.CompanyRoleManager:
		return targetRole != model.CompanyRoleOwner
	default:
		return false
	}
}

func sameLocation(existing *model.Location, input model.LocationInput) bool {
	return existing.Precision == input.Precision &&
		existing.Country == input.Country &&
		existing.City == input.City &&
		stringValue(existing.Region) == stringValue(input.Region) &&
		stringValue(existing.AddressLine) == stringValue(input.AddressLine) &&
		stringValue(existing.PostalCode) == stringValue(input.PostalCode) &&
		stringValue(existing.PlaceName) == stringValue(input.PlaceName) &&
		floatValue(existing.Latitude) == floatValue(input.Latitude) &&
		floatValue(existing.Longitude) == floatValue(input.Longitude)
}

func paginate[T any](items []T, page, pageSize int) ([]T, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	total := len(items)
	start := (page - 1) * pageSize
	if start >= total {
		return []T{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return items[start:end], total
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func membershipKey(companyID, employerUserID string) string {
	return companyID + ":" + employerUserID
}

func containsFold(value, needle string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(needle))
}

func appErr(status int, code, message string, details map[string]any) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Details: details}
}

func newID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return hex.EncodeToString(bytes[0:4]) + "-" +
		hex.EncodeToString(bytes[4:6]) + "-" +
		hex.EncodeToString(bytes[6:8]) + "-" +
		hex.EncodeToString(bytes[8:10]) + "-" +
		hex.EncodeToString(bytes[10:16])
}

func cloneUser(user *model.User) *model.User {
	if user == nil {
		return nil
	}
	copy := *user
	return &copy
}

func cloneApplicantProfile(profile *model.ApplicantProfile) *model.ApplicantProfile {
	if profile == nil {
		return nil
	}
	copy := *profile
	copy.TagIDs = append([]string{}, profile.TagIDs...)
	copy.SocialLinks = append([]model.ApplicantSocialLink{}, profile.SocialLinks...)
	return &copy
}

func cloneApplicantPrivacy(privacy *model.ApplicantPrivacySettings) *model.ApplicantPrivacySettings {
	if privacy == nil {
		return nil
	}
	copy := *privacy
	return &copy
}

func cloneEmployerProfile(profile *model.EmployerProfile) *model.EmployerProfile {
	if profile == nil {
		return nil
	}
	copy := *profile
	return &copy
}

func cloneUISettings(settings *model.UISettings) *model.UISettings {
	if settings == nil {
		return nil
	}
	copy := *settings
	copy.SettingsJSON = cloneMap(settings.SettingsJSON)
	return &copy
}

func cloneLocation(location *model.Location) *model.Location {
	if location == nil {
		return nil
	}
	copy := *location
	return &copy
}

func cloneTag(tag *model.Tag) *model.Tag {
	if tag == nil {
		return nil
	}
	copy := *tag
	return &copy
}

func cloneCompany(company *model.Company) *model.Company {
	if company == nil {
		return nil
	}
	copy := *company
	return &copy
}

func cloneMembership(membership *model.CompanyMembership) *model.CompanyMembership {
	if membership == nil {
		return nil
	}
	copy := *membership
	return &copy
}

func cloneNotificationPreferences(preferences *model.NotificationPreferences) *model.NotificationPreferences {
	if preferences == nil {
		return nil
	}
	copy := *preferences
	return &copy
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	copy := make(map[string]any, len(input))
	for key, value := range input {
		copy[key] = value
	}
	return copy
}

func stringPtr(value string) *string {
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func floatValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
