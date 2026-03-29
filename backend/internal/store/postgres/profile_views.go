package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tramplin/backend/internal/domain/model"
	commonstore "tramplin/backend/internal/store"
)

func (s *Store) GetApplicantProfileView(viewerUserID, targetUserID string) (*model.ApplicantProfileView, *commonstore.AppError) {
	ctx := context.Background()

	viewer, repoErr := s.GetUserByID(viewerUserID)
	if repoErr != nil {
		return nil, repoErr
	}

	return s.loadApplicantProfileView(ctx, s.db, viewer, targetUserID, false)
}

func (s *Store) GetEmployerApplicantProfileView(employerUserID, targetUserID string) (*model.ApplicantProfileView, *commonstore.AppError) {
	ctx := context.Background()

	viewer, repoErr := s.GetUserByID(employerUserID)
	if repoErr != nil {
		return nil, repoErr
	}
	if viewer.Role != model.UserRoleEmployer {
		return nil, appErr(403, "forbidden", "current user is not an employer", nil)
	}

	return s.loadApplicantProfileView(ctx, s.db, viewer, targetUserID, true)
}

func (s *Store) loadApplicantProfileView(ctx context.Context, q queryable, viewer *model.User, targetUserID string, allowEmployerApplicationOverride bool) (*model.ApplicantProfileView, *commonstore.AppError) {
	profile, err := scanApplicantProfile(q.QueryRow(ctx, `
		SELECT user_id, first_name, last_name, middle_name, university_name, faculty,
			program_name, study_year, graduation_year, city, about, resume_media_id,
			created_at, updated_at
		FROM applicant_profiles
		WHERE user_id = $1
	`, targetUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "applicant not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load applicant profile", nil)
	}

	preview, err := s.loadApplicantPreview(ctx, q, targetUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "applicant not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load applicant profile", nil)
	}

	privacy, err := scanApplicantPrivacy(q.QueryRow(ctx, `
		SELECT applicant_user_id, profile_visibility, resume_visibility, applications_visibility,
			contacts_visibility, show_career_interests, allow_recommendations, updated_at
		FROM applicant_privacy_settings
		WHERE applicant_user_id = $1
	`, targetUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErr(404, "not_found", "applicant not found", nil)
		}
		return nil, appErr(500, "internal_error", "failed to load applicant privacy", nil)
	}

	owner := viewer.ID == targetUserID
	isCurator := viewer.Role == model.UserRoleCurator
	isContact := false
	if !owner && viewer.Role == model.UserRoleApplicant {
		isContact, err = s.hasAcceptedConnection(ctx, q, viewer.ID, targetUserID)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to resolve applicant visibility", nil)
		}
	}

	profileVisible := owner || isCurator || visibilityAllows(privacy.ProfileVisibility, viewer.Role, isContact)
	if !profileVisible && allowEmployerApplicationOverride && viewer.Role == model.UserRoleEmployer {
		profileVisible, err = s.hasEmployerApplicantAccess(ctx, q, viewer.ID, targetUserID)
		if err != nil {
			return nil, appErr(500, "internal_error", "failed to resolve employer applicant access", nil)
		}
	}
	if !profileVisible {
		return nil, appErr(403, "forbidden", "not enough permissions", nil)
	}

	canViewResume := owner || isCurator || visibilityAllows(privacy.ResumeVisibility, viewer.Role, isContact)
	canViewApplications := owner || isCurator || visibilityAllows(privacy.ApplicationsVisibility, viewer.Role, isContact)
	canViewContacts := owner || isCurator || visibilityAllows(privacy.ContactsVisibility, viewer.Role, isContact)

	if !canViewResume {
		profile.ResumeMediaID = nil
	}

	socialLinks, repoErr := s.listApplicantSocialLinks(ctx, q, targetUserID)
	if repoErr != nil {
		return nil, repoErr
	}
	filteredSocialLinks := make([]model.ApplicantSocialLink, 0, len(socialLinks))
	if canViewContacts {
		for _, link := range socialLinks {
			if owner || isCurator || link.IsPublic {
				filteredSocialLinks = append(filteredSocialLinks, *link)
			}
		}
	}

	tags := preview.Tags
	if !privacy.ShowCareerInterests && !owner && !isCurator {
		tags = []*model.Tag{}
	}

	return &model.ApplicantProfileView{
		Profile:       profile,
		DisplayName:   preview.DisplayName,
		AvatarMediaID: preview.AvatarMediaID,
		Tags:          tags,
		SocialLinks:   filteredSocialLinks,
		Access: model.ApplicantProfileAccessMeta{
			CanViewResume:                 canViewResume,
			CanViewApplications:           canViewApplications,
			CanViewContacts:               canViewContacts,
			ProfileVisibilityApplied:      privacy.ProfileVisibility,
			ResumeVisibilityApplied:       privacy.ResumeVisibility,
			ApplicationsVisibilityApplied: privacy.ApplicationsVisibility,
			ContactsVisibilityApplied:     privacy.ContactsVisibility,
		},
	}, nil
}

func (s *Store) hasAcceptedConnection(ctx context.Context, q queryable, viewerUserID, targetUserID string) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM applicant_connections
			WHERE status = 'accepted'
				AND (
					(applicant_low_user_id = $1 AND applicant_high_user_id = $2)
					OR (applicant_low_user_id = $2 AND applicant_high_user_id = $1)
				)
		)
	`, viewerUserID, targetUserID).Scan(&exists)
	return exists, err
}

func (s *Store) hasEmployerApplicantAccess(ctx context.Context, q queryable, employerUserID, applicantUserID string) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM applications a
			JOIN opportunities o ON o.id = a.opportunity_id
			JOIN company_memberships cm ON cm.company_id = o.company_id
			WHERE a.applicant_user_id = $1
				AND cm.employer_user_id = $2
				AND cm.status = 'approved'
		)
	`, applicantUserID, employerUserID).Scan(&exists)
	return exists, err
}

func visibilityAllows(scope, viewerRole string, isContact bool) bool {
	switch scope {
	case "private":
		return false
	case "contacts_only":
		return isContact
	case "employers_only":
		return viewerRole == model.UserRoleEmployer
	case "authenticated_public":
		return viewerRole != ""
	default:
		return false
	}
}
