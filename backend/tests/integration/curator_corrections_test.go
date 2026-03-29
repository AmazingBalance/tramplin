//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestCuratorUserAndApplicantCorrectionIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	applicantClient := env.newSessionClient(t)
	curatorClient := env.newSessionClient(t)

	registerApplicant := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       fmt.Sprintf("curator-applicant-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Moderated Applicant",
		"firstName":   "Initial",
		"lastName":    "Applicant",
	})
	assertStatus(t, registerApplicant, http.StatusCreated)
	applicantID := nestedObjectStringField(t, registerApplicant.body, "user", "id")

	seedCuratorSession(t, env, curatorClient, fmt.Sprintf("curator-users-%d@example.com", time.Now().UnixNano()))

	listUsers := env.requestJSONWithClient(t, curatorClient, http.MethodGet, "/v1/curator/users?role=applicant&q=Moderated", nil)
	assertStatus(t, listUsers, http.StatusOK)
	assertCount(t, listUsers.body, "items", 1)
	assertEqual(t, nestedStringField(t, listUsers.body, "items", 0, "id"), applicantID)

	getUser := env.requestJSONWithClient(t, curatorClient, http.MethodGet, "/v1/curator/users/"+applicantID, nil)
	assertStatus(t, getUser, http.StatusOK)
	assertEqual(t, stringField(t, getUser.body, "role"), "applicant")
	assertEqual(t, stringField(t, getUser.body, "displayName"), "Moderated Applicant")

	patchUser := env.requestJSONWithClient(t, curatorClient, http.MethodPatch, "/v1/curator/users/"+applicantID, map[string]any{
		"displayName": "Applicant Under Review",
		"isActive":    false,
		"reason":      "Temporarily disable for moderation review",
	})
	assertStatus(t, patchUser, http.StatusOK)
	assertEqual(t, stringField(t, patchUser.body, "displayName"), "Applicant Under Review")
	assertBoolValue(t, patchUser.body, "isActive", false)

	getApplicant := env.requestJSONWithClient(t, curatorClient, http.MethodGet, "/v1/curator/applicants/"+applicantID, nil)
	assertStatus(t, getApplicant, http.StatusOK)
	assertEqual(t, stringField(t, getApplicant.body, "displayName"), "Applicant Under Review")
	assertEqual(t, nestedObjectStringField(t, getApplicant.body, "privacy", "profileVisibility"), "authenticated_public")

	patchApplicant := env.requestJSONWithClient(t, curatorClient, http.MethodPatch, "/v1/curator/applicants/"+applicantID, map[string]any{
		"firstName": "Curated",
		"city":      "Omsk",
		"privacy": map[string]any{
			"profileVisibility":   "private",
			"showCareerInterests": false,
		},
		"reason": "Correct applicant profile and tighten visibility",
	})
	assertStatus(t, patchApplicant, http.StatusOK)
	assertEqual(t, stringField(t, patchApplicant.body, "firstName"), "Curated")
	assertEqual(t, stringField(t, patchApplicant.body, "city"), "Omsk")
	assertEqual(t, nestedObjectStringField(t, patchApplicant.body, "privacy", "profileVisibility"), "private")
	assertObjectBoolValue(t, patchApplicant.body, "privacy", "showCareerInterests", false)
}

func TestCuratorEmployerCompanyOpportunityCorrectionIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	employerClient := env.newSessionClient(t)
	curatorClient := env.newSessionClient(t)

	registerEmployer := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("curator-employer-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Moderated Employer",
		"fullName":    "Moderated Employer",
	})
	assertStatus(t, registerEmployer, http.StatusCreated)
	employerID := nestedObjectStringField(t, registerEmployer.body, "user", "id")

	createCompany := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName": "Curated Company",
		"slug":      fmt.Sprintf("curated-company-%d", time.Now().UnixNano()),
	})
	assertStatus(t, createCompany, http.StatusCreated)
	companyID := stringField(t, createCompany.body, "id")

	createOpportunity := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/opportunities", map[string]any{
		"companyId":           companyID,
		"title":               "Curated Opportunity",
		"summary":             "Needs correction",
		"slug":                fmt.Sprintf("curated-opportunity-%d", time.Now().UnixNano()),
		"description":         "Curator correction coverage",
		"type":                "internship",
		"participationFormat": "remote",
		"vacancyDetails": map[string]any{
			"employmentType":  "full_time",
			"experienceLevel": "junior",
		},
	})
	assertStatus(t, createOpportunity, http.StatusCreated)
	opportunityID := stringField(t, createOpportunity.body, "id")
	env.execSQL(t, `UPDATE opportunities SET moderation_status = 'approved' WHERE id = $1`, opportunityID)

	seedCuratorSession(t, env, curatorClient, fmt.Sprintf("curator-company-%d@example.com", time.Now().UnixNano()))

	getEmployer := env.requestJSONWithClient(t, curatorClient, http.MethodGet, "/v1/curator/employers/"+employerID, nil)
	assertStatus(t, getEmployer, http.StatusOK)
	assertEqual(t, stringField(t, getEmployer.body, "displayName"), "Moderated Employer")
	assertArrayFieldLen(t, getEmployer.body, "companies", 1)

	patchEmployer := env.requestJSONWithClient(t, curatorClient, http.MethodPatch, "/v1/curator/employers/"+employerID, map[string]any{
		"fullName": "Curated Employer",
		"jobTitle": "Head of Hiring",
		"reason":   "Correct employer profile",
	})
	assertStatus(t, patchEmployer, http.StatusOK)
	assertEqual(t, stringField(t, patchEmployer.body, "fullName"), "Curated Employer")
	assertEqual(t, stringField(t, patchEmployer.body, "jobTitle"), "Head of Hiring")

	getCompany := env.requestJSONWithClient(t, curatorClient, http.MethodGet, "/v1/curator/companies/"+companyID, nil)
	assertStatus(t, getCompany, http.StatusOK)
	assertEqual(t, stringField(t, getCompany.body, "id"), companyID)

	patchCompany := env.requestJSONWithClient(t, curatorClient, http.MethodPatch, "/v1/curator/companies/"+companyID, map[string]any{
		"industry": "Education",
		"inn":      "1234567890",
		"reason":   "Correct company metadata",
	})
	assertStatus(t, patchCompany, http.StatusOK)
	assertEqual(t, stringField(t, patchCompany.body, "industry"), "Education")
	assertEqual(t, stringField(t, patchCompany.body, "inn"), "1234567890")

	getOpportunity := env.requestJSONWithClient(t, curatorClient, http.MethodGet, "/v1/curator/opportunities/"+opportunityID, nil)
	assertStatus(t, getOpportunity, http.StatusOK)
	assertEqual(t, stringField(t, getOpportunity.body, "moderationStatus"), "approved")

	patchOpportunity := env.requestJSONWithClient(t, curatorClient, http.MethodPatch, "/v1/curator/opportunities/"+opportunityID, map[string]any{
		"summary": "Curator corrected summary",
		"reason":  "Fix approved opportunity copy",
	})
	assertStatus(t, patchOpportunity, http.StatusOK)
	assertEqual(t, stringField(t, patchOpportunity.body, "summary"), "Curator corrected summary")
	assertEqual(t, stringField(t, patchOpportunity.body, "moderationStatus"), "approved")

	employerOpportunity := env.requestJSONWithClient(t, employerClient, http.MethodGet, "/v1/employer/opportunities/"+opportunityID, nil)
	assertStatus(t, employerOpportunity, http.StatusOK)
	assertEqual(t, stringField(t, employerOpportunity.body, "summary"), "Curator corrected summary")
}

func assertObjectBoolValue(t *testing.T, body map[string]any, field string, nestedField string, expected bool) {
	t.Helper()

	item, ok := body[field].(map[string]any)
	if !ok {
		t.Fatalf("field %q is not an object: %v", field, body[field])
	}
	value, ok := item[nestedField].(bool)
	if !ok {
		t.Fatalf("nested field %q is not a bool: %v", nestedField, item[nestedField])
	}
	if value != expected {
		t.Fatalf("unexpected bool value for %q: got %v want %v", nestedField, value, expected)
	}
}
