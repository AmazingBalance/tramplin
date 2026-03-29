//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestApplicantProfileVisibilityIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	applicantAClient := env.newSessionClient(t)
	applicantBClient := env.newSessionClient(t)

	registerApplicantA := env.requestJSONWithClient(t, applicantAClient, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       fmt.Sprintf("profile-a-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Applicant A",
		"firstName":   "Alice",
		"lastName":    "Visible",
	})
	assertStatus(t, registerApplicantA, http.StatusCreated)
	applicantAID := nestedObjectStringField(t, registerApplicantA.body, "user", "id")

	registerApplicantB := env.requestJSONWithClient(t, applicantBClient, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       fmt.Sprintf("profile-b-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Applicant B",
		"firstName":   "Bob",
		"lastName":    "Viewer",
	})
	assertStatus(t, registerApplicantB, http.StatusCreated)
	applicantBID := nestedObjectStringField(t, registerApplicantB.body, "user", "id")

	publicTags := env.requestJSON(t, http.MethodGet, "/v1/public/tags?q=Go", nil)
	assertStatus(t, publicTags, http.StatusOK)
	assertCount(t, publicTags.body, "items", 1)
	tagID := nestedStringField(t, publicTags.body, "items", 0, "id")

	replaceTags := env.requestJSONWithClient(t, applicantAClient, http.MethodPut, "/v1/me/applicant/tags", map[string]any{
		"tagIds": []string{tagID},
	})
	assertStatus(t, replaceTags, http.StatusOK)
	assertCount(t, replaceTags.body, "items", 1)

	createPublicLink := env.requestJSONWithClient(t, applicantAClient, http.MethodPost, "/v1/me/applicant/social-links", map[string]any{
		"platform": "telegram",
		"url":      "https://t.me/applicant-a",
		"isPublic": true,
	})
	assertStatus(t, createPublicLink, http.StatusCreated)

	createPrivateLink := env.requestJSONWithClient(t, applicantAClient, http.MethodPost, "/v1/me/applicant/social-links", map[string]any{
		"platform": "portfolio",
		"url":      "https://example.com/private-profile",
		"isPublic": false,
	})
	assertStatus(t, createPrivateLink, http.StatusCreated)

	updatePrivacy := env.requestJSONWithClient(t, applicantAClient, http.MethodPatch, "/v1/me/applicant/privacy", map[string]any{
		"profileVisibility":      "contacts_only",
		"resumeVisibility":       "contacts_only",
		"applicationsVisibility": "private",
		"contactsVisibility":     "contacts_only",
		"showCareerInterests":    false,
	})
	assertStatus(t, updatePrivacy, http.StatusOK)

	beforeConnection := env.requestJSONWithClient(t, applicantBClient, http.MethodGet, "/v1/applicants/"+applicantAID, nil)
	assertStatus(t, beforeConnection, http.StatusForbidden)

	createConnection := env.requestJSONWithClient(t, applicantAClient, http.MethodPost, "/v1/me/connections", map[string]any{
		"targetApplicantUserId": applicantBID,
	})
	assertStatus(t, createConnection, http.StatusCreated)
	connectionID := stringField(t, createConnection.body, "id")

	acceptConnection := env.requestJSONWithClient(t, applicantBClient, http.MethodPatch, "/v1/me/connections/"+connectionID, map[string]any{
		"status": "accepted",
	})
	assertStatus(t, acceptConnection, http.StatusOK)

	visibleProfile := env.requestJSONWithClient(t, applicantBClient, http.MethodGet, "/v1/applicants/"+applicantAID, nil)
	assertStatus(t, visibleProfile, http.StatusOK)
	assertEqual(t, stringField(t, visibleProfile.body, "displayName"), "Applicant A")
	assertArrayFieldLen(t, visibleProfile.body, "tags", 0)
	assertArrayFieldLen(t, visibleProfile.body, "socialLinks", 1)
	assertEqual(t, nestedStringField(t, visibleProfile.body, "socialLinks", 0, "platform"), "telegram")

	access := objectField(t, visibleProfile.body, "access")
	assertBoolValue(t, access, "canViewResume", true)
	assertBoolValue(t, access, "canViewApplications", false)
	assertBoolValue(t, access, "canViewContacts", true)
	assertEqual(t, objectStringField(t, visibleProfile.body, "access", "profileVisibilityApplied"), "contacts_only")
	assertEqual(t, objectStringField(t, visibleProfile.body, "access", "resumeVisibilityApplied"), "contacts_only")
	assertEqual(t, objectStringField(t, visibleProfile.body, "access", "applicationsVisibilityApplied"), "private")
	assertEqual(t, objectStringField(t, visibleProfile.body, "access", "contactsVisibilityApplied"), "contacts_only")
}

func TestEmployerApplicantProfileOverrideIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	employerClient := env.newSessionClient(t)
	applicantClient := env.newSessionClient(t)

	registerEmployer := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("profile-employer-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Profile Employer",
		"fullName":    "Profile Employer",
	})
	assertStatus(t, registerEmployer, http.StatusCreated)

	createCompany := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName": "Profile Access Company",
		"slug":      fmt.Sprintf("profile-access-company-%d", time.Now().UnixNano()),
	})
	assertStatus(t, createCompany, http.StatusCreated)
	companyID := stringField(t, createCompany.body, "id")

	createOpportunity := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/opportunities", map[string]any{
		"companyId":           companyID,
		"title":               "Profile Access Opportunity",
		"summary":             "Opportunity used for employer applicant visibility",
		"slug":                fmt.Sprintf("profile-access-opportunity-%d", time.Now().UnixNano()),
		"description":         "Private applicant should become visible after applying",
		"type":                "internship",
		"participationFormat": "remote",
		"vacancyDetails": map[string]any{
			"employmentType":  "full_time",
			"experienceLevel": "junior",
		},
	})
	assertStatus(t, createOpportunity, http.StatusCreated)
	opportunityID := stringField(t, createOpportunity.body, "id")

	plannedOpportunity := env.requestJSONWithClient(t, employerClient, http.MethodPatch, "/v1/employer/opportunities/"+opportunityID, map[string]any{
		"status": "planned",
	})
	assertStatus(t, plannedOpportunity, http.StatusOK)
	env.execSQL(t, `UPDATE opportunities SET moderation_status = 'approved' WHERE id = $1`, opportunityID)

	registerApplicant := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       fmt.Sprintf("profile-target-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Private Applicant",
		"firstName":   "Priya",
		"lastName":    "Private",
	})
	assertStatus(t, registerApplicant, http.StatusCreated)
	applicantID := nestedObjectStringField(t, registerApplicant.body, "user", "id")

	createLink := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/me/applicant/social-links", map[string]any{
		"platform": "telegram",
		"url":      "https://t.me/private-applicant",
		"isPublic": true,
	})
	assertStatus(t, createLink, http.StatusCreated)

	updatePrivacy := env.requestJSONWithClient(t, applicantClient, http.MethodPatch, "/v1/me/applicant/privacy", map[string]any{
		"profileVisibility":      "private",
		"resumeVisibility":       "private",
		"applicationsVisibility": "private",
		"contactsVisibility":     "private",
	})
	assertStatus(t, updatePrivacy, http.StatusOK)

	beforeApplication := env.requestJSONWithClient(t, employerClient, http.MethodGet, "/v1/employer/applicants/"+applicantID, nil)
	assertStatus(t, beforeApplication, http.StatusForbidden)

	genericBeforeApplication := env.requestJSONWithClient(t, employerClient, http.MethodGet, "/v1/applicants/"+applicantID, nil)
	assertStatus(t, genericBeforeApplication, http.StatusForbidden)

	apply := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/me/applications", map[string]any{
		"opportunityId": opportunityID,
	})
	assertStatus(t, apply, http.StatusCreated)

	employerView := env.requestJSONWithClient(t, employerClient, http.MethodGet, "/v1/employer/applicants/"+applicantID, nil)
	assertStatus(t, employerView, http.StatusOK)
	assertEqual(t, stringField(t, employerView.body, "displayName"), "Private Applicant")
	assertArrayFieldLen(t, employerView.body, "socialLinks", 0)

	access := objectField(t, employerView.body, "access")
	assertBoolValue(t, access, "canViewResume", false)
	assertBoolValue(t, access, "canViewApplications", false)
	assertBoolValue(t, access, "canViewContacts", false)
	assertEqual(t, objectStringField(t, employerView.body, "access", "profileVisibilityApplied"), "private")
	assertEqual(t, objectStringField(t, employerView.body, "access", "contactsVisibilityApplied"), "private")
}

func objectField(t *testing.T, body map[string]any, field string) map[string]any {
	t.Helper()

	value, ok := body[field].(map[string]any)
	if !ok {
		t.Fatalf("field %q is not an object: %v", field, body[field])
	}
	return value
}
