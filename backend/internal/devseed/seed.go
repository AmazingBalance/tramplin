package devseed

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tramplin/backend/internal/config"
	authplatform "tramplin/backend/internal/platform/auth"
	objectstoreplatform "tramplin/backend/internal/platform/objectstore"
)

const defaultPassword = "password123"

const (
	locationMoscowID            = "11111111-1111-1111-1111-111111111101"
	locationSaintPetersburgID   = "11111111-1111-1111-1111-111111111102"
	userAdminCuratorID          = "11111111-1111-1111-1111-111111111201"
	userCuratorReviewerID       = "11111111-1111-1111-1111-111111111202"
	userEmployerOwnerID         = "11111111-1111-1111-1111-111111111203"
	userEmployerRecruiterID     = "11111111-1111-1111-1111-111111111204"
	userApplicantAnnaID         = "11111111-1111-1111-1111-111111111205"
	userApplicantBorisID        = "11111111-1111-1111-1111-111111111206"
	userApplicantClaraID        = "11111111-1111-1111-1111-111111111207"
	mediaCompanyLogoID          = "11111111-1111-1111-1111-111111111301"
	mediaCompanyBannerID        = "11111111-1111-1111-1111-111111111302"
	mediaCompanyGalleryID       = "11111111-1111-1111-1111-111111111303"
	mediaAnnaAvatarID           = "11111111-1111-1111-1111-111111111304"
	mediaOpportunityCoverID     = "11111111-1111-1111-1111-111111111305"
	mediaOpportunityGalleryID   = "11111111-1111-1111-1111-111111111306"
	companySeedLabsID           = "11111111-1111-1111-1111-111111111401"
	membershipOwnerID           = "11111111-1111-1111-1111-111111111411"
	membershipRecruiterID       = "11111111-1111-1111-1111-111111111412"
	companySocialLinkedInID     = "11111111-1111-1111-1111-111111111431"
	companySocialTelegramID     = "11111111-1111-1111-1111-111111111432"
	companyMediaGalleryID       = "11111111-1111-1111-1111-111111111433"
	tagDistributedSystemsID     = "11111111-1111-1111-1111-111111111501"
	opportunityGoInternshipID   = "11111111-1111-1111-1111-111111111601"
	opportunityCareerMeetupID   = "11111111-1111-1111-1111-111111111602"
	opportunityMentorProgramID  = "11111111-1111-1111-1111-111111111603"
	opportunityGoApplyLinkID    = "11111111-1111-1111-1111-111111111611"
	opportunityMeetupInfoLinkID = "11111111-1111-1111-1111-111111111612"
	opportunityMentorInfoLinkID = "11111111-1111-1111-1111-111111111613"
	opportunityGalleryMediaID   = "11111111-1111-1111-1111-111111111621"
	applicantAnnaTelegramID     = "11111111-1111-1111-1111-111111111701"
	applicantBorisGitHubID      = "11111111-1111-1111-1111-111111111702"
	applicantClaraLinkedInID    = "11111111-1111-1111-1111-111111111703"
	applicationAnnaInternID     = "11111111-1111-1111-1111-111111111801"
	applicationBorisInternID    = "11111111-1111-1111-1111-111111111802"
	applicationAnnaSubmittedID  = "11111111-1111-1111-1111-111111111811"
	applicationAnnaReviewingID  = "11111111-1111-1111-1111-111111111812"
	applicationBorisSubmittedID = "11111111-1111-1111-1111-111111111813"
	applicationBorisReserveID   = "11111111-1111-1111-1111-111111111814"
	connectionAnnaBorisID       = "11111111-1111-1111-1111-111111111901"
	connectionAnnaClaraID       = "11111111-1111-1111-1111-111111111902"
	recommendationAnnaToBorisID = "11111111-1111-1111-1111-111111111911"
	notificationAnnaSystemID    = "11111111-1111-1111-1111-111111112001"
	notificationAnnaStatusID    = "11111111-1111-1111-1111-111111112002"
	notificationBorisRecomID    = "11111111-1111-1111-1111-111111112003"
	notificationBorisCampaignID = "11111111-1111-1111-1111-111111112004"
	notificationClaraCampaignID = "11111111-1111-1111-1111-111111112005"
	campaignInternshipID        = "11111111-1111-1111-1111-111111112101"
	verificationRequestID       = "11111111-1111-1111-1111-111111112201"
	verificationEvidenceID      = "11111111-1111-1111-1111-111111112211"
	moderationCaseID            = "11111111-1111-1111-1111-111111112301"
)

type Summary struct {
	Password               string
	MediaSeeded            bool
	AdminCuratorEmail      string
	CuratorReviewerEmail   string
	EmployerOwnerEmail     string
	EmployerRecruiterEmail string
	ApplicantEmails        []string
	CompanySlug            string
	PublicOpportunitySlug  string
	EventOpportunitySlug   string
	DraftOpportunitySlug   string
}

type seededMedia struct {
	ID               string
	FileKey          string
	OriginalName     string
	MIMEType         string
	Purpose          string
	UploadedByUserID string
	Content          []byte
	ETag             string
}

func Seed(ctx context.Context, db *pgxpool.Pool, cfg config.Config) (*Summary, error) {
	now := time.Now().UTC().Truncate(time.Second)

	passwordHash, err := authplatform.HashPassword(defaultPassword)
	if err != nil {
		return nil, fmt.Errorf("hash seed password: %w", err)
	}

	mediaByID, err := seedMediaObjects(ctx, cfg)
	if err != nil {
		return nil, err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin seed transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := seedLocations(ctx, tx, now); err != nil {
		return nil, err
	}
	if err := seedUsers(ctx, tx, now, passwordHash, mediaByID); err != nil {
		return nil, err
	}
	if err := seedProfilesAndPreferences(ctx, tx, now); err != nil {
		return nil, err
	}
	if err := seedTags(ctx, tx, now); err != nil {
		return nil, err
	}
	if err := seedMediaFiles(ctx, tx, now, mediaByID); err != nil {
		return nil, err
	}
	if err := seedCompany(ctx, tx, now, mediaByID); err != nil {
		return nil, err
	}
	if err := seedOpportunities(ctx, tx, now, mediaByID); err != nil {
		return nil, err
	}
	if err := seedApplications(ctx, tx, now); err != nil {
		return nil, err
	}
	if err := seedSavedEntities(ctx, tx, now); err != nil {
		return nil, err
	}
	if err := seedSocialGraph(ctx, tx, now); err != nil {
		return nil, err
	}
	if err := seedNotifications(ctx, tx, now); err != nil {
		return nil, err
	}
	if err := seedVerificationAndModeration(ctx, tx, now); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit seed transaction: %w", err)
	}

	return &Summary{
		Password:               defaultPassword,
		MediaSeeded:            len(mediaByID) > 0,
		AdminCuratorEmail:      "seed.admin.curator@tramplin.local",
		CuratorReviewerEmail:   "seed.curator@tramplin.local",
		EmployerOwnerEmail:     "seed.owner@tramplin.local",
		EmployerRecruiterEmail: "seed.recruiter@tramplin.local",
		ApplicantEmails: []string{
			"seed.applicant.anna@tramplin.local",
			"seed.applicant.boris@tramplin.local",
			"seed.applicant.clara@tramplin.local",
		},
		CompanySlug:           "seed-labs",
		PublicOpportunitySlug: "junior-go-backend-internship",
		EventOpportunitySlug:  "spring-career-meetup",
		DraftOpportunitySlug:  "backend-mentor-circle",
	}, nil
}

func seedMediaObjects(ctx context.Context, cfg config.Config) (map[string]seededMedia, error) {
	if !cfg.HasObjectStorageConfig() {
		return nil, nil
	}

	store, err := objectstoreplatform.NewMinIO(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open object storage for seed media: %w", err)
	}

	media := []seededMedia{
		{
			ID:               mediaCompanyLogoID,
			FileKey:          "seed/company-logo.svg",
			OriginalName:     "seed-company-logo.svg",
			MIMEType:         "image/svg+xml",
			Purpose:          "company_logo",
			UploadedByUserID: userEmployerOwnerID,
			Content:          []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256"><rect width="256" height="256" rx="48" fill="#0f766e"/><text x="50%" y="54%" dominant-baseline="middle" text-anchor="middle" font-family="Arial" font-size="88" fill="#ffffff">SL</text></svg>`),
		},
		{
			ID:               mediaCompanyBannerID,
			FileKey:          "seed/company-banner.svg",
			OriginalName:     "seed-company-banner.svg",
			MIMEType:         "image/svg+xml",
			Purpose:          "company_banner",
			UploadedByUserID: userEmployerOwnerID,
			Content:          []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1440" height="480"><defs><linearGradient id="g" x1="0" x2="1"><stop offset="0%" stop-color="#022c22"/><stop offset="100%" stop-color="#115e59"/></linearGradient></defs><rect width="1440" height="480" fill="url(#g)"/><circle cx="220" cy="140" r="84" fill="#99f6e4" fill-opacity="0.22"/><circle cx="1170" cy="340" r="120" fill="#ccfbf1" fill-opacity="0.12"/></svg>`),
		},
		{
			ID:               mediaCompanyGalleryID,
			FileKey:          "seed/company-gallery.svg",
			OriginalName:     "seed-company-gallery.svg",
			MIMEType:         "image/svg+xml",
			Purpose:          "company_media",
			UploadedByUserID: userEmployerOwnerID,
			Content:          []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1280" height="720"><rect width="1280" height="720" fill="#e2e8f0"/><rect x="80" y="80" width="1120" height="560" rx="36" fill="#ffffff"/><text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle" font-family="Arial" font-size="72" fill="#0f172a">Seed Labs Demo Workspace</text></svg>`),
		},
		{
			ID:               mediaAnnaAvatarID,
			FileKey:          "seed/anna-avatar.svg",
			OriginalName:     "seed-anna-avatar.svg",
			MIMEType:         "image/svg+xml",
			Purpose:          "avatar",
			UploadedByUserID: userApplicantAnnaID,
			Content:          []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256"><rect width="256" height="256" rx="128" fill="#f59e0b"/><text x="50%" y="54%" dominant-baseline="middle" text-anchor="middle" font-family="Arial" font-size="92" fill="#ffffff">A</text></svg>`),
		},
		{
			ID:               mediaOpportunityCoverID,
			FileKey:          "seed/go-internship-cover.svg",
			OriginalName:     "seed-go-internship-cover.svg",
			MIMEType:         "image/svg+xml",
			Purpose:          "opportunity_cover",
			UploadedByUserID: userEmployerOwnerID,
			Content:          []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1280" height="720"><rect width="1280" height="720" fill="#0f172a"/><text x="50%" y="46%" dominant-baseline="middle" text-anchor="middle" font-family="Arial" font-size="84" fill="#5eead4">Go Backend Internship</text><text x="50%" y="58%" dominant-baseline="middle" text-anchor="middle" font-family="Arial" font-size="34" fill="#e2e8f0">Public seeded opportunity</text></svg>`),
		},
		{
			ID:               mediaOpportunityGalleryID,
			FileKey:          "seed/go-internship-gallery.svg",
			OriginalName:     "seed-go-internship-gallery.svg",
			MIMEType:         "image/svg+xml",
			Purpose:          "opportunity_media",
			UploadedByUserID: userEmployerOwnerID,
			Content:          []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1280" height="720"><rect width="1280" height="720" fill="#ecfeff"/><text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle" font-family="Arial" font-size="64" fill="#155e75">Pair programming, reviews, and shipping</text></svg>`),
		},
	}

	result := make(map[string]seededMedia, len(media))
	client := &http.Client{Timeout: 30 * time.Second}
	for _, item := range media {
		upload, err := store.PresignUpload(ctx, item.FileKey, item.MIMEType, cfg.UploadURLTTL)
		if err != nil {
			return nil, fmt.Errorf("presign seed media upload %s: %w", item.FileKey, err)
		}
		req, err := http.NewRequestWithContext(ctx, upload.Method, upload.URL, bytes.NewReader(item.Content))
		if err != nil {
			return nil, fmt.Errorf("build seed media upload request %s: %w", item.FileKey, err)
		}
		req.Header.Set("Content-Type", item.MIMEType)
		for key, value := range upload.Headers {
			req.Header.Set(key, value)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("upload seed media %s: %w", item.FileKey, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			return nil, fmt.Errorf("upload seed media %s: unexpected status %d", item.FileKey, resp.StatusCode)
		}

		info, err := store.StatObject(ctx, item.FileKey)
		if err != nil {
			return nil, fmt.Errorf("stat seed media %s: %w", item.FileKey, err)
		}
		item.ETag = info.ETag
		result[item.ID] = item
	}

	return result, nil
}

func seedLocations(ctx context.Context, tx pgx.Tx, now time.Time) error {
	locations := []struct {
		id          string
		precision   string
		country     string
		region      *string
		city        string
		addressLine *string
		postalCode  *string
		placeName   *string
		latitude    *float64
		longitude   *float64
	}{
		{
			id:          locationMoscowID,
			precision:   "exact_address",
			country:     "Russia",
			region:      strptr("Moscow"),
			city:        "Moscow",
			addressLine: strptr("Tverskaya 7"),
			postalCode:  strptr("125009"),
			placeName:   strptr("Seed Labs HQ"),
			latitude:    floatptr(55.7572),
			longitude:   floatptr(37.6156),
		},
		{
			id:        locationSaintPetersburgID,
			precision: "city_only",
			country:   "Russia",
			region:    strptr("Saint Petersburg"),
			city:      "Saint Petersburg",
		},
	}

	for _, location := range locations {
		if _, err := tx.Exec(ctx, `
			INSERT INTO locations (
				id, precision, country, region, city, address_line, postal_code, place_name, latitude, longitude, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (id) DO UPDATE SET
				precision = EXCLUDED.precision,
				country = EXCLUDED.country,
				region = EXCLUDED.region,
				city = EXCLUDED.city,
				address_line = EXCLUDED.address_line,
				postal_code = EXCLUDED.postal_code,
				place_name = EXCLUDED.place_name,
				latitude = EXCLUDED.latitude,
				longitude = EXCLUDED.longitude
		`, location.id, location.precision, location.country, location.region, location.city, location.addressLine, location.postalCode, location.placeName, location.latitude, location.longitude, now); err != nil {
			return fmt.Errorf("seed location %s: %w", location.id, err)
		}
	}
	return nil
}

func seedUsers(ctx context.Context, tx pgx.Tx, now time.Time, passwordHash string, mediaByID map[string]seededMedia) error {
	users := []struct {
		id            string
		email         string
		displayName   string
		role          string
		isActive      bool
		avatarMediaID *string
	}{
		{
			id:          userAdminCuratorID,
			email:       "seed.admin.curator@tramplin.local",
			displayName: "Seed Admin Curator",
			role:        "curator",
			isActive:    true,
		},
		{
			id:          userCuratorReviewerID,
			email:       "seed.curator@tramplin.local",
			displayName: "Seed Curator",
			role:        "curator",
			isActive:    true,
		},
		{
			id:          userEmployerOwnerID,
			email:       "seed.owner@tramplin.local",
			displayName: "Seed Owner",
			role:        "employer",
			isActive:    true,
		},
		{
			id:          userEmployerRecruiterID,
			email:       "seed.recruiter@tramplin.local",
			displayName: "Seed Recruiter",
			role:        "employer",
			isActive:    true,
		},
		{
			id:            userApplicantAnnaID,
			email:         "seed.applicant.anna@tramplin.local",
			displayName:   "Anna Seed",
			role:          "applicant",
			isActive:      true,
			avatarMediaID: mediaIDPtr(mediaByID, mediaAnnaAvatarID),
		},
		{
			id:          userApplicantBorisID,
			email:       "seed.applicant.boris@tramplin.local",
			displayName: "Boris Seed",
			role:        "applicant",
			isActive:    true,
		},
		{
			id:          userApplicantClaraID,
			email:       "seed.applicant.clara@tramplin.local",
			displayName: "Clara Seed",
			role:        "applicant",
			isActive:    true,
		},
	}

	for _, user := range users {
		if _, err := tx.Exec(ctx, `
			INSERT INTO users (
				id, email, password_hash, display_name, role, email_verified_at, is_active, avatar_media_id,
				token_version, password_changed_at, last_login_at, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, $9, NULL, $10, $11)
			ON CONFLICT (id) DO UPDATE SET
				email = EXCLUDED.email,
				password_hash = EXCLUDED.password_hash,
				display_name = EXCLUDED.display_name,
				role = EXCLUDED.role,
				email_verified_at = EXCLUDED.email_verified_at,
				is_active = EXCLUDED.is_active,
				avatar_media_id = EXCLUDED.avatar_media_id,
				password_changed_at = EXCLUDED.password_changed_at,
				updated_at = EXCLUDED.updated_at
		`, user.id, user.email, passwordHash, user.displayName, user.role, now, user.isActive, user.avatarMediaID, now, now, now); err != nil {
			return fmt.Errorf("seed user %s: %w", user.email, err)
		}
	}

	curators := []struct {
		userID   string
		fullName string
		isAdmin  bool
	}{
		{userID: userAdminCuratorID, fullName: "Seed Admin Curator", isAdmin: true},
		{userID: userCuratorReviewerID, fullName: "Seed Curator Reviewer", isAdmin: false},
	}
	for _, curator := range curators {
		if _, err := tx.Exec(ctx, `
			INSERT INTO curator_profiles (user_id, full_name, is_admin, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) DO UPDATE SET
				full_name = EXCLUDED.full_name,
				is_admin = EXCLUDED.is_admin
		`, curator.userID, curator.fullName, curator.isAdmin, now); err != nil {
			return fmt.Errorf("seed curator profile %s: %w", curator.userID, err)
		}
	}

	return nil
}

func seedProfilesAndPreferences(ctx context.Context, tx pgx.Tx, now time.Time) error {
	settingsByUser := map[string]string{
		userAdminCuratorID:      `{"seeded":true,"workspace":"curation"}`,
		userCuratorReviewerID:   `{"seeded":true,"workspace":"curation"}`,
		userEmployerOwnerID:     `{"seeded":true,"workspace":"employer"}`,
		userEmployerRecruiterID: `{"seeded":true,"workspace":"employer"}`,
		userApplicantAnnaID:     `{"seeded":true,"workspace":"applicant"}`,
		userApplicantBorisID:    `{"seeded":true,"workspace":"applicant"}`,
		userApplicantClaraID:    `{"seeded":true,"workspace":"applicant"}`,
	}
	for userID, settings := range settingsByUser {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_ui_settings (user_id, settings_json, schema_version, updated_at)
			VALUES ($1, $2::jsonb, 1, $3)
			ON CONFLICT (user_id) DO UPDATE SET
				settings_json = EXCLUDED.settings_json,
				schema_version = EXCLUDED.schema_version,
				updated_at = EXCLUDED.updated_at
		`, userID, settings, now); err != nil {
			return fmt.Errorf("seed ui settings %s: %w", userID, err)
		}
	}

	applicants := []struct {
		userID          string
		firstName       string
		lastName        string
		universityName  *string
		faculty         *string
		programName     *string
		studyYear       *int
		graduationYear  *int
		city            *string
		about           *string
		profileVis      string
		resumeVis       string
		applicationsVis string
		contactsVis     string
	}{
		{
			userID:          userApplicantAnnaID,
			firstName:       "Anna",
			lastName:        "Seed",
			universityName:  strptr("OmSTU"),
			faculty:         strptr("Computer Science"),
			programName:     strptr("Software Engineering"),
			studyYear:       intptr(4),
			graduationYear:  intptr(2026),
			city:            strptr("Omsk"),
			about:           strptr("Backend-focused applicant seeded for local development."),
			profileVis:      "authenticated_public",
			resumeVis:       "employers_only",
			applicationsVis: "private",
			contactsVis:     "contacts_only",
		},
		{
			userID:          userApplicantBorisID,
			firstName:       "Boris",
			lastName:        "Seed",
			universityName:  strptr("ITMO"),
			faculty:         strptr("Information Technologies"),
			programName:     strptr("Distributed Systems"),
			studyYear:       intptr(3),
			graduationYear:  intptr(2027),
			city:            strptr("Saint Petersburg"),
			about:           strptr("Interested in internships and meetups."),
			profileVis:      "authenticated_public",
			resumeVis:       "contacts_only",
			applicationsVis: "private",
			contactsVis:     "contacts_only",
		},
		{
			userID:          userApplicantClaraID,
			firstName:       "Clara",
			lastName:        "Seed",
			universityName:  strptr("HSE"),
			faculty:         strptr("Applied Mathematics"),
			programName:     strptr("Data Science"),
			studyYear:       intptr(2),
			graduationYear:  intptr(2028),
			city:            strptr("Moscow"),
			about:           strptr("Open to mentorship and events."),
			profileVis:      "authenticated_public",
			resumeVis:       "contacts_only",
			applicationsVis: "private",
			contactsVis:     "contacts_only",
		},
	}

	for _, applicant := range applicants {
		if _, err := tx.Exec(ctx, `
			INSERT INTO applicant_profiles (
				user_id, last_name, first_name, middle_name, university_name, faculty, program_name,
				study_year, graduation_year, city, about, resume_media_id, created_at, updated_at
			) VALUES ($1, $2, $3, NULL, $4, $5, $6, $7, $8, $9, $10, NULL, $11, $12)
			ON CONFLICT (user_id) DO UPDATE SET
				last_name = EXCLUDED.last_name,
				first_name = EXCLUDED.first_name,
				middle_name = EXCLUDED.middle_name,
				university_name = EXCLUDED.university_name,
				faculty = EXCLUDED.faculty,
				program_name = EXCLUDED.program_name,
				study_year = EXCLUDED.study_year,
				graduation_year = EXCLUDED.graduation_year,
				city = EXCLUDED.city,
				about = EXCLUDED.about,
				resume_media_id = EXCLUDED.resume_media_id,
				updated_at = EXCLUDED.updated_at
		`, applicant.userID, applicant.lastName, applicant.firstName, applicant.universityName, applicant.faculty, applicant.programName, applicant.studyYear, applicant.graduationYear, applicant.city, applicant.about, now, now); err != nil {
			return fmt.Errorf("seed applicant profile %s: %w", applicant.userID, err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO applicant_privacy_settings (
				applicant_user_id, profile_visibility, resume_visibility, applications_visibility,
				contacts_visibility, show_career_interests, allow_recommendations, updated_at
			) VALUES ($1, $2, $3, $4, $5, true, true, $6)
			ON CONFLICT (applicant_user_id) DO UPDATE SET
				profile_visibility = EXCLUDED.profile_visibility,
				resume_visibility = EXCLUDED.resume_visibility,
				applications_visibility = EXCLUDED.applications_visibility,
				contacts_visibility = EXCLUDED.contacts_visibility,
				show_career_interests = EXCLUDED.show_career_interests,
				allow_recommendations = EXCLUDED.allow_recommendations,
				updated_at = EXCLUDED.updated_at
		`, applicant.userID, applicant.profileVis, applicant.resumeVis, applicant.applicationsVis, applicant.contactsVis, now); err != nil {
			return fmt.Errorf("seed applicant privacy %s: %w", applicant.userID, err)
		}
	}

	employers := []struct {
		userID   string
		fullName string
		jobTitle *string
		phone    *string
	}{
		{userID: userEmployerOwnerID, fullName: "Olga Owner", jobTitle: strptr("Head of Engineering"), phone: strptr("+7-900-100-0001")},
		{userID: userEmployerRecruiterID, fullName: "Roman Recruiter", jobTitle: strptr("Technical Recruiter"), phone: strptr("+7-900-100-0002")},
	}
	for _, employer := range employers {
		if _, err := tx.Exec(ctx, `
			INSERT INTO employer_profiles (user_id, full_name, job_title, phone, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (user_id) DO UPDATE SET
				full_name = EXCLUDED.full_name,
				job_title = EXCLUDED.job_title,
				phone = EXCLUDED.phone,
				updated_at = EXCLUDED.updated_at
		`, employer.userID, employer.fullName, employer.jobTitle, employer.phone, now, now); err != nil {
			return fmt.Errorf("seed employer profile %s: %w", employer.userID, err)
		}
	}

	preferences := []struct {
		userID                   string
		inAppEnabled             bool
		emailEnabled             bool
		recommendationEnabled    bool
		applicationStatusEnabled bool
		employerMessagesEnabled  bool
		systemEnabled            bool
	}{
		{userID: userAdminCuratorID, inAppEnabled: true, emailEnabled: true, recommendationEnabled: true, applicationStatusEnabled: true, employerMessagesEnabled: true, systemEnabled: true},
		{userID: userCuratorReviewerID, inAppEnabled: true, emailEnabled: true, recommendationEnabled: true, applicationStatusEnabled: true, employerMessagesEnabled: true, systemEnabled: true},
		{userID: userEmployerOwnerID, inAppEnabled: true, emailEnabled: true, recommendationEnabled: true, applicationStatusEnabled: true, employerMessagesEnabled: true, systemEnabled: true},
		{userID: userEmployerRecruiterID, inAppEnabled: true, emailEnabled: true, recommendationEnabled: true, applicationStatusEnabled: true, employerMessagesEnabled: true, systemEnabled: true},
		{userID: userApplicantAnnaID, inAppEnabled: true, emailEnabled: true, recommendationEnabled: true, applicationStatusEnabled: true, employerMessagesEnabled: true, systemEnabled: true},
		{userID: userApplicantBorisID, inAppEnabled: true, emailEnabled: false, recommendationEnabled: true, applicationStatusEnabled: true, employerMessagesEnabled: true, systemEnabled: true},
		{userID: userApplicantClaraID, inAppEnabled: true, emailEnabled: true, recommendationEnabled: true, applicationStatusEnabled: true, employerMessagesEnabled: true, systemEnabled: true},
	}
	for _, preference := range preferences {
		if _, err := tx.Exec(ctx, `
			INSERT INTO notification_preferences (
				user_id, in_app_enabled, email_enabled, recommendation_enabled, application_status_enabled,
				employer_messages_enabled, system_enabled, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (user_id) DO UPDATE SET
				in_app_enabled = EXCLUDED.in_app_enabled,
				email_enabled = EXCLUDED.email_enabled,
				recommendation_enabled = EXCLUDED.recommendation_enabled,
				application_status_enabled = EXCLUDED.application_status_enabled,
				employer_messages_enabled = EXCLUDED.employer_messages_enabled,
				system_enabled = EXCLUDED.system_enabled,
				updated_at = EXCLUDED.updated_at
		`, preference.userID, preference.inAppEnabled, preference.emailEnabled, preference.recommendationEnabled, preference.applicationStatusEnabled, preference.employerMessagesEnabled, preference.systemEnabled, now); err != nil {
			return fmt.Errorf("seed notification preferences %s: %w", preference.userID, err)
		}
	}

	return nil
}

func seedTags(ctx context.Context, tx pgx.Tx, now time.Time) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO tags (id, name, tag_type, is_system, is_active, created_by_user_id, created_at)
		VALUES ($1, 'Distributed Systems', 'custom', false, true, $2, $3)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			tag_type = EXCLUDED.tag_type,
			is_system = EXCLUDED.is_system,
			is_active = EXCLUDED.is_active,
			created_by_user_id = EXCLUDED.created_by_user_id
	`, tagDistributedSystemsID, userEmployerOwnerID, now); err != nil {
		return fmt.Errorf("seed custom tag: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM applicant_tags WHERE applicant_user_id = ANY($1::uuid[])`, []string{userApplicantAnnaID, userApplicantBorisID, userApplicantClaraID}); err != nil {
		return fmt.Errorf("clear applicant tags: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO applicant_tags (applicant_user_id, tag_id) VALUES
			($1, (SELECT id FROM tags WHERE name = 'Go' AND tag_type = 'technology')),
			($1, (SELECT id FROM tags WHERE name = 'Backend' AND tag_type = 'role')),
			($1, $2),
			($3, (SELECT id FROM tags WHERE name = 'Python' AND tag_type = 'technology')),
			($3, (SELECT id FROM tags WHERE name = 'Junior' AND tag_type = 'level')),
			($4, (SELECT id FROM tags WHERE name = 'Remote' AND tag_type = 'format'))
	`, userApplicantAnnaID, tagDistributedSystemsID, userApplicantBorisID, userApplicantClaraID); err != nil {
		return fmt.Errorf("seed applicant tags: %w", err)
	}

	return nil
}

func seedMediaFiles(ctx context.Context, tx pgx.Tx, now time.Time, mediaByID map[string]seededMedia) error {
	if len(mediaByID) == 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM media_files WHERE id = ANY($1::uuid[])`, []string{
			mediaCompanyLogoID,
			mediaCompanyBannerID,
			mediaCompanyGalleryID,
			mediaAnnaAvatarID,
			mediaOpportunityCoverID,
			mediaOpportunityGalleryID,
		}); err != nil {
			return fmt.Errorf("clear seed media rows: %w", err)
		}
		return nil
	}

	orderedIDs := []string{
		mediaCompanyLogoID,
		mediaCompanyBannerID,
		mediaCompanyGalleryID,
		mediaAnnaAvatarID,
		mediaOpportunityCoverID,
		mediaOpportunityGalleryID,
	}
	for _, id := range orderedIDs {
		item := mediaByID[id]
		if _, err := tx.Exec(ctx, `
			INSERT INTO media_files (
				id, file_key, original_name, mime_type, file_size, uploaded_by_user_id, purpose,
				status, etag, completed_at, deleted_at, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, 'uploaded', $8, $9, NULL, $10)
			ON CONFLICT (id) DO UPDATE SET
				file_key = EXCLUDED.file_key,
				original_name = EXCLUDED.original_name,
				mime_type = EXCLUDED.mime_type,
				file_size = EXCLUDED.file_size,
				uploaded_by_user_id = EXCLUDED.uploaded_by_user_id,
				purpose = EXCLUDED.purpose,
				status = EXCLUDED.status,
				etag = EXCLUDED.etag,
				completed_at = EXCLUDED.completed_at,
				deleted_at = EXCLUDED.deleted_at
		`, item.ID, item.FileKey, item.OriginalName, item.MIMEType, len(item.Content), item.UploadedByUserID, item.Purpose, item.ETag, now, now); err != nil {
			return fmt.Errorf("seed media file %s: %w", item.FileKey, err)
		}
	}

	return nil
}

func seedCompany(ctx context.Context, tx pgx.Tx, now time.Time, mediaByID map[string]seededMedia) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO companies (
			id, legal_name, brand_name, slug, inn, description, industry, website_url,
			corporate_email_domain, headquarters_location_id, logo_media_id, banner_media_id,
			verification_status, verified_at, verified_by_curator_user_id, created_by_user_id,
			created_at, updated_at
		) VALUES (
			$1, 'Seed Labs LLC', 'Seed Labs', 'seed-labs', '5501001200',
			'Seeded company used for local development and manual QA.',
			'Education Technology', 'https://seed-labs.example.com',
			'seed-labs.example.com', $2, $3, $4, 'verified', $5, $6, $7, $8, $9
		)
		ON CONFLICT (id) DO UPDATE SET
			legal_name = EXCLUDED.legal_name,
			brand_name = EXCLUDED.brand_name,
			slug = EXCLUDED.slug,
			inn = EXCLUDED.inn,
			description = EXCLUDED.description,
			industry = EXCLUDED.industry,
			website_url = EXCLUDED.website_url,
			corporate_email_domain = EXCLUDED.corporate_email_domain,
			headquarters_location_id = EXCLUDED.headquarters_location_id,
			logo_media_id = EXCLUDED.logo_media_id,
			banner_media_id = EXCLUDED.banner_media_id,
			verification_status = EXCLUDED.verification_status,
			verified_at = EXCLUDED.verified_at,
			verified_by_curator_user_id = EXCLUDED.verified_by_curator_user_id,
			created_by_user_id = EXCLUDED.created_by_user_id,
			updated_at = EXCLUDED.updated_at
	`, companySeedLabsID, locationMoscowID, mediaIDPtr(mediaByID, mediaCompanyLogoID), mediaIDPtr(mediaByID, mediaCompanyBannerID), now.Add(-72*time.Hour), userAdminCuratorID, userEmployerOwnerID, now.Add(-120*time.Hour), now); err != nil {
		return fmt.Errorf("seed company: %w", err)
	}

	memberships := []struct {
		id                string
		employerUserID    string
		invitedByUserID   *string
		status            string
		memberRole        string
		isPrimaryContact  bool
		statusChangedByID string
		statusComment     *string
	}{
		{
			id:                membershipOwnerID,
			employerUserID:    userEmployerOwnerID,
			status:            "approved",
			memberRole:        "owner",
			isPrimaryContact:  true,
			statusChangedByID: userEmployerOwnerID,
			statusComment:     strptr("Initial seeded owner membership."),
		},
		{
			id:                membershipRecruiterID,
			employerUserID:    userEmployerRecruiterID,
			invitedByUserID:   strptr(userEmployerOwnerID),
			status:            "approved",
			memberRole:        "recruiter",
			isPrimaryContact:  false,
			statusChangedByID: userEmployerOwnerID,
			statusComment:     strptr("Seeded approved recruiter membership."),
		},
	}
	for _, membership := range memberships {
		if _, err := tx.Exec(ctx, `
			INSERT INTO company_memberships (
				id, company_id, employer_user_id, invited_by_user_id, status, member_role,
				is_primary_contact, status_changed_by_user_id, status_comment, status_updated_at,
				approved_at, updated_at, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (id) DO UPDATE SET
				company_id = EXCLUDED.company_id,
				employer_user_id = EXCLUDED.employer_user_id,
				invited_by_user_id = EXCLUDED.invited_by_user_id,
				status = EXCLUDED.status,
				member_role = EXCLUDED.member_role,
				is_primary_contact = EXCLUDED.is_primary_contact,
				status_changed_by_user_id = EXCLUDED.status_changed_by_user_id,
				status_comment = EXCLUDED.status_comment,
				status_updated_at = EXCLUDED.status_updated_at,
				approved_at = EXCLUDED.approved_at,
				updated_at = EXCLUDED.updated_at
		`, membership.id, companySeedLabsID, membership.employerUserID, membership.invitedByUserID, membership.status, membership.memberRole, membership.isPrimaryContact, membership.statusChangedByID, membership.statusComment, now.Add(-72*time.Hour), now.Add(-72*time.Hour), now, now.Add(-96*time.Hour)); err != nil {
			return fmt.Errorf("seed company membership %s: %w", membership.id, err)
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM company_social_links WHERE company_id = $1`, companySeedLabsID); err != nil {
		return fmt.Errorf("clear company social links: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO company_social_links (id, company_id, platform, url) VALUES
			($1, $2, 'linkedin', 'https://linkedin.com/company/seed-labs'),
			($3, $2, 'telegram', 'https://t.me/seedlabs')
	`, companySocialLinkedInID, companySeedLabsID, companySocialTelegramID); err != nil {
		return fmt.Errorf("seed company social links: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM company_media WHERE company_id = $1`, companySeedLabsID); err != nil {
		return fmt.Errorf("clear company media: %w", err)
	}
	if mediaID := mediaIDPtr(mediaByID, mediaCompanyGalleryID); mediaID != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO company_media (id, company_id, media_file_id, title, sort_order, created_at)
			VALUES ($1, $2, $3, 'Seed Labs office', 0, $4)
		`, companyMediaGalleryID, companySeedLabsID, *mediaID, now); err != nil {
			return fmt.Errorf("seed company media: %w", err)
		}
	}

	return nil
}

func seedOpportunities(ctx context.Context, tx pgx.Tx, now time.Time, mediaByID map[string]seededMedia) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO opportunities (
			id, company_id, created_by_user_id, title, summary, slug, description, type, status,
			moderation_status, participation_format, location_id, contact_email, contact_phone,
			cover_media_id, published_at, expires_at, created_at, updated_at
		) VALUES
			($1, $2, $3, 'Junior Go Backend Internship', 'Public seeded internship opportunity.', 'junior-go-backend-internship',
			 'Work on the Tramplin backend, learn Go, PostgreSQL, and API design.', 'internship', 'active', 'approved', 'remote',
			 $4, 'jobs@seed-labs.example.com', '+7-900-100-0101', $5, $6, $7, $8, $9),
			($10, $2, $3, 'Spring Career Meetup', 'Public seeded event for applicant discovery.', 'spring-career-meetup',
			 'Meet employers, curators, and applicants in a seeded public event.', 'event', 'planned', 'approved', 'offline',
			 $11, 'events@seed-labs.example.com', '+7-900-100-0102', NULL, $12, NULL, $13, $14),
			($15, $2, $3, 'Backend Mentor Circle', 'Draft mentor program for employer-only views.', 'backend-mentor-circle',
			 'Internal seeded mentor program in draft status.', 'mentor_program', 'draft', 'pending', 'hybrid',
			 $4, 'mentors@seed-labs.example.com', '+7-900-100-0103', NULL, NULL, NULL, $16, $17)
		ON CONFLICT (id) DO UPDATE SET
			company_id = EXCLUDED.company_id,
			created_by_user_id = EXCLUDED.created_by_user_id,
			title = EXCLUDED.title,
			summary = EXCLUDED.summary,
			slug = EXCLUDED.slug,
			description = EXCLUDED.description,
			type = EXCLUDED.type,
			status = EXCLUDED.status,
			moderation_status = EXCLUDED.moderation_status,
			participation_format = EXCLUDED.participation_format,
			location_id = EXCLUDED.location_id,
			contact_email = EXCLUDED.contact_email,
			contact_phone = EXCLUDED.contact_phone,
			cover_media_id = EXCLUDED.cover_media_id,
			published_at = EXCLUDED.published_at,
			expires_at = EXCLUDED.expires_at,
			updated_at = EXCLUDED.updated_at
	`,
		opportunityGoInternshipID, companySeedLabsID, userEmployerOwnerID, locationMoscowID, mediaIDPtr(mediaByID, mediaOpportunityCoverID), now.Add(-48*time.Hour), now.Add(30*24*time.Hour), now.Add(-120*time.Hour), now,
		opportunityCareerMeetupID, locationSaintPetersburgID, now.Add(-24*time.Hour), now.Add(-96*time.Hour), now,
		opportunityMentorProgramID, now.Add(-12*time.Hour), now,
	); err != nil {
		return fmt.Errorf("seed opportunities: %w", err)
	}

	for _, table := range []string{
		"opportunity_vacancy_details",
		"opportunity_mentor_program_details",
		"opportunity_event_details",
		"opportunity_links",
		"opportunity_media",
		"opportunity_stats",
		"opportunity_tags",
	} {
		if _, err := tx.Exec(ctx, `DELETE FROM `+table+` WHERE opportunity_id = ANY($1::uuid[])`, []string{
			opportunityGoInternshipID,
			opportunityCareerMeetupID,
			opportunityMentorProgramID,
		}); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO opportunity_vacancy_details (
			opportunity_id, employment_type, experience_level, salary_from, salary_to, currency
		) VALUES ($1, 'full_time', 'junior', 80000, 120000, 'RUB')
	`, opportunityGoInternshipID); err != nil {
		return fmt.Errorf("seed vacancy details: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO opportunity_event_details (
			opportunity_id, start_at, end_at, registration_deadline, capacity, venue_note
		) VALUES ($1, $2, $3, $4, 180, 'Main conference hall')
	`, opportunityCareerMeetupID, now.Add(14*24*time.Hour), now.Add(14*24*time.Hour+4*time.Hour), now.Add(12*24*time.Hour)); err != nil {
		return fmt.Errorf("seed event details: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO opportunity_mentor_program_details (
			opportunity_id, start_at, end_at, seats_count, mentor_requirements
		) VALUES ($1, $2, $3, 12, 'Strong backend fundamentals and willingness to mentor junior teammates.')
	`, opportunityMentorProgramID, now.Add(21*24*time.Hour), now.Add(90*24*time.Hour)); err != nil {
		return fmt.Errorf("seed mentor details: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO opportunity_links (id, opportunity_id, link_type, title, url, sort_order) VALUES
			($1, $2, 'apply', 'Apply now', 'https://seed-labs.example.com/apply/go-internship', 0),
			($3, $4, 'registration', 'Register for event', 'https://seed-labs.example.com/events/meetup', 0),
			($5, $6, 'info', 'Program details', 'https://seed-labs.example.com/programs/mentor-circle', 0)
	`, opportunityGoApplyLinkID, opportunityGoInternshipID, opportunityMeetupInfoLinkID, opportunityCareerMeetupID, opportunityMentorInfoLinkID, opportunityMentorProgramID); err != nil {
		return fmt.Errorf("seed opportunity links: %w", err)
	}

	if mediaID := mediaIDPtr(mediaByID, mediaOpportunityGalleryID); mediaID != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO opportunity_media (id, opportunity_id, media_file_id, title, sort_order, created_at)
			VALUES ($1, $2, $3, 'Team rituals', 0, $4)
		`, opportunityGalleryMediaID, opportunityGoInternshipID, *mediaID, now); err != nil {
			return fmt.Errorf("seed opportunity media: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO opportunity_stats (opportunity_id, views_count, updated_at) VALUES
			($1, 248, $4),
			($2, 127, $4),
			($3, 43, $4)
	`, opportunityGoInternshipID, opportunityCareerMeetupID, opportunityMentorProgramID, now); err != nil {
		return fmt.Errorf("seed opportunity stats: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO opportunity_tags (opportunity_id, tag_id) VALUES
			($1, (SELECT id FROM tags WHERE name = 'Go' AND tag_type = 'technology')),
			($1, (SELECT id FROM tags WHERE name = 'Backend' AND tag_type = 'role')),
			($1, $2),
			($3, (SELECT id FROM tags WHERE name = 'Remote' AND tag_type = 'format')),
			($4, (SELECT id FROM tags WHERE name = 'Junior' AND tag_type = 'level'))
	`, opportunityGoInternshipID, tagDistributedSystemsID, opportunityCareerMeetupID, opportunityMentorProgramID); err != nil {
		return fmt.Errorf("seed opportunity tags: %w", err)
	}

	return nil
}

func seedApplications(ctx context.Context, tx pgx.Tx, now time.Time) error {
	applications := []struct {
		id              string
		opportunityID   string
		applicantUserID string
		coverLetter     *string
		status          string
		appliedAt       time.Time
	}{
		{
			id:              applicationAnnaInternID,
			opportunityID:   opportunityGoInternshipID,
			applicantUserID: userApplicantAnnaID,
			coverLetter:     strptr("I want to work on APIs and Go services."),
			status:          "reviewing",
			appliedAt:       now.Add(-36 * time.Hour),
		},
		{
			id:              applicationBorisInternID,
			opportunityID:   opportunityGoInternshipID,
			applicantUserID: userApplicantBorisID,
			coverLetter:     strptr("Interested in backend internships and team events."),
			status:          "reserve",
			appliedAt:       now.Add(-24 * time.Hour),
		},
	}
	for _, application := range applications {
		if _, err := tx.Exec(ctx, `
			INSERT INTO applications (id, opportunity_id, applicant_user_id, cover_letter, status, applied_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE SET
				opportunity_id = EXCLUDED.opportunity_id,
				applicant_user_id = EXCLUDED.applicant_user_id,
				cover_letter = EXCLUDED.cover_letter,
				status = EXCLUDED.status,
				applied_at = EXCLUDED.applied_at,
				updated_at = EXCLUDED.updated_at
		`, application.id, application.opportunityID, application.applicantUserID, application.coverLetter, application.status, application.appliedAt, now); err != nil {
			return fmt.Errorf("seed application %s: %w", application.id, err)
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM application_status_history WHERE application_id = ANY($1::uuid[])`, []string{
		applicationAnnaInternID,
		applicationBorisInternID,
	}); err != nil {
		return fmt.Errorf("clear application history: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO application_status_history (
			id, application_id, old_status, new_status, changed_by_user_id, comment, created_at
		) VALUES
			($1, $2, NULL, 'submitted', $3, 'Seeded initial submission.', $4),
			($5, $2, 'submitted', 'reviewing', $6, 'Moved to review by seeded employer.', $7),
			($8, $9, NULL, 'submitted', $10, 'Seeded initial submission.', $11),
			($12, $9, 'submitted', 'reserve', $13, 'Placed on reserve list.', $14)
	`, applicationAnnaSubmittedID, applicationAnnaInternID, userApplicantAnnaID, now.Add(-36*time.Hour),
		applicationAnnaReviewingID, userEmployerOwnerID, now.Add(-18*time.Hour),
		applicationBorisSubmittedID, applicationBorisInternID, userApplicantBorisID, now.Add(-24*time.Hour),
		applicationBorisReserveID, userEmployerOwnerID, now.Add(-12*time.Hour)); err != nil {
		return fmt.Errorf("seed application history: %w", err)
	}

	return nil
}

func seedSavedEntities(ctx context.Context, tx pgx.Tx, now time.Time) error {
	if _, err := tx.Exec(ctx, `DELETE FROM applicant_saved_opportunities WHERE applicant_user_id = ANY($1::uuid[])`, []string{
		userApplicantAnnaID,
		userApplicantBorisID,
	}); err != nil {
		return fmt.Errorf("clear saved opportunities: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM applicant_saved_companies WHERE applicant_user_id = ANY($1::uuid[])`, []string{
		userApplicantAnnaID,
		userApplicantBorisID,
	}); err != nil {
		return fmt.Errorf("clear saved companies: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO applicant_saved_opportunities (applicant_user_id, opportunity_id, created_at) VALUES
			($1, $2, $5),
			($3, $4, $6)
	`, userApplicantAnnaID, opportunityCareerMeetupID, userApplicantBorisID, opportunityGoInternshipID, now.Add(-8*time.Hour), now.Add(-6*time.Hour)); err != nil {
		return fmt.Errorf("seed saved opportunities: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO applicant_saved_companies (applicant_user_id, company_id, created_at) VALUES
			($1, $2, $3),
			($4, $2, $5)
	`, userApplicantAnnaID, companySeedLabsID, now.Add(-10*time.Hour), userApplicantBorisID, now.Add(-9*time.Hour)); err != nil {
		return fmt.Errorf("seed saved companies: %w", err)
	}

	return nil
}

func seedSocialGraph(ctx context.Context, tx pgx.Tx, now time.Time) error {
	if _, err := tx.Exec(ctx, `DELETE FROM applicant_social_links WHERE applicant_user_id = ANY($1::uuid[])`, []string{
		userApplicantAnnaID,
		userApplicantBorisID,
		userApplicantClaraID,
	}); err != nil {
		return fmt.Errorf("clear applicant social links: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO applicant_social_links (id, applicant_user_id, platform, url, is_public, created_at) VALUES
			($1, $2, 'telegram', 'https://t.me/annaseed', true, $7),
			($3, $4, 'github', 'https://github.com/borisseed', true, $8),
			($5, $6, 'linkedin', 'https://linkedin.com/in/claraseed', true, $9)
	`, applicantAnnaTelegramID, userApplicantAnnaID, applicantBorisGitHubID, userApplicantBorisID, applicantClaraLinkedInID, userApplicantClaraID, now.Add(-72*time.Hour), now.Add(-48*time.Hour), now.Add(-24*time.Hour)); err != nil {
		return fmt.Errorf("seed applicant social links: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM applicant_connections WHERE id = ANY($1::uuid[])`, []string{
		connectionAnnaBorisID,
		connectionAnnaClaraID,
	}); err != nil {
		return fmt.Errorf("clear connections: %w", err)
	}
	lowAB, highAB := orderedPair(userApplicantAnnaID, userApplicantBorisID)
	lowAC, highAC := orderedPair(userApplicantAnnaID, userApplicantClaraID)
	if _, err := tx.Exec(ctx, `
		INSERT INTO applicant_connections (
			id, applicant_low_user_id, applicant_high_user_id, initiator_user_id, status,
			initiator_note, responded_at, created_at
		) VALUES
			($1, $2, $3, $4, 'accepted', 'Seeded accepted connection.', $5, $6),
			($7, $8, $9, $10, 'pending', 'Seeded pending connection.', NULL, $11)
	`, connectionAnnaBorisID, lowAB, highAB, userApplicantAnnaID, now.Add(-30*time.Hour), now.Add(-36*time.Hour),
		connectionAnnaClaraID, lowAC, highAC, userApplicantClaraID, now.Add(-5*time.Hour)); err != nil {
		return fmt.Errorf("seed connections: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO opportunity_recommendations (
			id, recommender_user_id, recipient_user_id, opportunity_id, message, created_at
		) VALUES ($1, $2, $3, $4, 'This internship fits your backend track.', $5)
		ON CONFLICT (id) DO UPDATE SET
			recommender_user_id = EXCLUDED.recommender_user_id,
			recipient_user_id = EXCLUDED.recipient_user_id,
			opportunity_id = EXCLUDED.opportunity_id,
			message = EXCLUDED.message,
			created_at = EXCLUDED.created_at
	`, recommendationAnnaToBorisID, userApplicantAnnaID, userApplicantBorisID, opportunityGoInternshipID, now.Add(-4*time.Hour)); err != nil {
		return fmt.Errorf("seed recommendation: %w", err)
	}

	return nil
}

func seedNotifications(ctx context.Context, tx pgx.Tx, now time.Time) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO notification_campaigns (
			id, company_id, created_by_user_id, opportunity_id, audience_type, status, title, body,
			send_via_in_app, send_via_email, scheduled_at, sent_at, created_at
		) VALUES (
			$1, $2, $3, $4, 'all_applicants_of_opportunity', 'sent', 'Seed Labs internship update',
			'We are reviewing applicants this week. Watch your notifications for the next step.',
			true, false, NULL, $5, $6
		)
		ON CONFLICT (id) DO UPDATE SET
			company_id = EXCLUDED.company_id,
			created_by_user_id = EXCLUDED.created_by_user_id,
			opportunity_id = EXCLUDED.opportunity_id,
			audience_type = EXCLUDED.audience_type,
			status = EXCLUDED.status,
			title = EXCLUDED.title,
			body = EXCLUDED.body,
			send_via_in_app = EXCLUDED.send_via_in_app,
			send_via_email = EXCLUDED.send_via_email,
			scheduled_at = EXCLUDED.scheduled_at,
			sent_at = EXCLUDED.sent_at,
			created_at = EXCLUDED.created_at
	`, campaignInternshipID, companySeedLabsID, userEmployerOwnerID, opportunityGoInternshipID, now.Add(-3*time.Hour), now.Add(-5*time.Hour)); err != nil {
		return fmt.Errorf("seed notification campaign: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM notification_campaign_recipients WHERE campaign_id = $1`, campaignInternshipID); err != nil {
		return fmt.Errorf("clear campaign recipients: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO notification_campaign_recipients (
			campaign_id, applicant_user_id, application_id, created_at
		) VALUES
			($1, $2, $3, $5),
			($1, $4, NULL, $6)
	`, campaignInternshipID, userApplicantBorisID, applicationBorisInternID, userApplicantClaraID, now.Add(-5*time.Hour), now.Add(-5*time.Hour)); err != nil {
		return fmt.Errorf("seed campaign recipients: %w", err)
	}

	notifications := []struct {
		id              string
		recipientUserID string
		actorUserID     *string
		typ             string
		sourceType      string
		sourceID        *string
		companyID       *string
		opportunityID   *string
		applicationID   *string
		title           string
		body            *string
		isRead          bool
		readAt          *time.Time
		createdAt       time.Time
	}{
		{
			id:              notificationAnnaSystemID,
			recipientUserID: userApplicantAnnaID,
			typ:             "system",
			sourceType:      "system",
			title:           "Seed data loaded",
			body:            strptr("Your development workspace now contains seeded entities."),
			isRead:          false,
			createdAt:       now.Add(-2 * time.Hour),
		},
		{
			id:              notificationAnnaStatusID,
			recipientUserID: userApplicantAnnaID,
			actorUserID:     strptr(userEmployerOwnerID),
			typ:             "application_status_changed",
			sourceType:      "application",
			sourceID:        strptr(applicationAnnaInternID),
			companyID:       strptr(companySeedLabsID),
			opportunityID:   strptr(opportunityGoInternshipID),
			applicationID:   strptr(applicationAnnaInternID),
			title:           "Application moved to reviewing",
			body:            strptr("Seed Labs moved your internship application to reviewing."),
			isRead:          false,
			createdAt:       now.Add(-90 * time.Minute),
		},
		{
			id:              notificationBorisRecomID,
			recipientUserID: userApplicantBorisID,
			actorUserID:     strptr(userApplicantAnnaID),
			typ:             "recommendation_received",
			sourceType:      "recommendation",
			sourceID:        strptr(recommendationAnnaToBorisID),
			opportunityID:   strptr(opportunityGoInternshipID),
			title:           "Anna recommended an opportunity",
			body:            strptr("Anna thinks the Go internship fits your goals."),
			isRead:          true,
			readAt:          timeptr(now.Add(-55 * time.Minute)),
			createdAt:       now.Add(-70 * time.Minute),
		},
		{
			id:              notificationBorisCampaignID,
			recipientUserID: userApplicantBorisID,
			actorUserID:     strptr(userEmployerOwnerID),
			typ:             "employer_broadcast",
			sourceType:      "campaign",
			sourceID:        strptr(campaignInternshipID),
			companyID:       strptr(companySeedLabsID),
			opportunityID:   strptr(opportunityGoInternshipID),
			title:           "Seed Labs internship update",
			body:            strptr("We are reviewing applicants this week. Watch your notifications for the next step."),
			isRead:          false,
			createdAt:       now.Add(-45 * time.Minute),
		},
		{
			id:              notificationClaraCampaignID,
			recipientUserID: userApplicantClaraID,
			actorUserID:     strptr(userEmployerOwnerID),
			typ:             "employer_broadcast",
			sourceType:      "campaign",
			sourceID:        strptr(campaignInternshipID),
			companyID:       strptr(companySeedLabsID),
			opportunityID:   strptr(opportunityGoInternshipID),
			title:           "Seed Labs internship update",
			body:            strptr("We are reviewing applicants this week. Watch your notifications for the next step."),
			isRead:          false,
			createdAt:       now.Add(-45 * time.Minute),
		},
	}
	for _, notification := range notifications {
		if _, err := tx.Exec(ctx, `
			INSERT INTO notifications (
				id, recipient_user_id, actor_user_id, type, source_type, source_id, company_id,
				opportunity_id, application_id, title, body, is_read, read_at, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			ON CONFLICT (id) DO UPDATE SET
				recipient_user_id = EXCLUDED.recipient_user_id,
				actor_user_id = EXCLUDED.actor_user_id,
				type = EXCLUDED.type,
				source_type = EXCLUDED.source_type,
				source_id = EXCLUDED.source_id,
				company_id = EXCLUDED.company_id,
				opportunity_id = EXCLUDED.opportunity_id,
				application_id = EXCLUDED.application_id,
				title = EXCLUDED.title,
				body = EXCLUDED.body,
				is_read = EXCLUDED.is_read,
				read_at = EXCLUDED.read_at,
				created_at = EXCLUDED.created_at
		`, notification.id, notification.recipientUserID, notification.actorUserID, notification.typ, notification.sourceType, notification.sourceID, notification.companyID, notification.opportunityID, notification.applicationID, notification.title, notification.body, notification.isRead, notification.readAt, notification.createdAt); err != nil {
			return fmt.Errorf("seed notification %s: %w", notification.id, err)
		}
	}

	return nil
}

func seedVerificationAndModeration(ctx context.Context, tx pgx.Tx, now time.Time) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO verification_requests (
			id, company_id, submitted_by_user_id, method, status, submitted_comment, review_comment,
			reviewed_by_curator_user_id, reviewed_at, created_at
		) VALUES (
			$1, $2, $3, 'official_website', 'approved',
			'Seeded verification request for local demo data.',
			'Website ownership confirmed during seed setup.',
			$4, $5, $6
		)
		ON CONFLICT (id) DO UPDATE SET
			company_id = EXCLUDED.company_id,
			submitted_by_user_id = EXCLUDED.submitted_by_user_id,
			method = EXCLUDED.method,
			status = EXCLUDED.status,
			submitted_comment = EXCLUDED.submitted_comment,
			review_comment = EXCLUDED.review_comment,
			reviewed_by_curator_user_id = EXCLUDED.reviewed_by_curator_user_id,
			reviewed_at = EXCLUDED.reviewed_at,
			created_at = EXCLUDED.created_at
	`, verificationRequestID, companySeedLabsID, userEmployerOwnerID, userCuratorReviewerID, now.Add(-80*time.Hour), now.Add(-96*time.Hour)); err != nil {
		return fmt.Errorf("seed verification request: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM verification_evidence WHERE verification_request_id = $1`, verificationRequestID); err != nil {
		return fmt.Errorf("clear verification evidence: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO verification_evidence (
			id, verification_request_id, evidence_type, value, evidence_file_id, created_at
		) VALUES ($1, $2, 'website_link', 'https://seed-labs.example.com', NULL, $3)
	`, verificationEvidenceID, verificationRequestID, now.Add(-96*time.Hour)); err != nil {
		return fmt.Errorf("seed verification evidence: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO moderation_cases (
			id, target_type, target_id, submitted_by_user_id, assigned_curator_user_id,
			resolved_by_curator_user_id, status, reason, created_at, resolved_at
		) VALUES (
			$1, 'opportunity', $2, $3, $4, NULL, 'pending',
			'Seeded moderation case for draft mentor program review.', $5, NULL
		)
		ON CONFLICT (id) DO UPDATE SET
			target_type = EXCLUDED.target_type,
			target_id = EXCLUDED.target_id,
			submitted_by_user_id = EXCLUDED.submitted_by_user_id,
			assigned_curator_user_id = EXCLUDED.assigned_curator_user_id,
			resolved_by_curator_user_id = EXCLUDED.resolved_by_curator_user_id,
			status = EXCLUDED.status,
			reason = EXCLUDED.reason,
			created_at = EXCLUDED.created_at,
			resolved_at = EXCLUDED.resolved_at
	`, moderationCaseID, opportunityMentorProgramID, userEmployerOwnerID, userCuratorReviewerID, now.Add(-6*time.Hour)); err != nil {
		return fmt.Errorf("seed moderation case: %w", err)
	}

	return nil
}

func orderedPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

func mediaIDPtr(mediaByID map[string]seededMedia, id string) *string {
	if len(mediaByID) == 0 {
		return nil
	}
	item, ok := mediaByID[id]
	if !ok {
		return nil
	}
	return strptr(item.ID)
}

func strptr(value string) *string {
	return &value
}

func floatptr(value float64) *float64 {
	return &value
}

func intptr(value int) *int {
	return &value
}

func timeptr(value time.Time) *time.Time {
	return &value
}
