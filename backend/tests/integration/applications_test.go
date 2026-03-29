//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestApplicationWorkflowIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	employerClient := env.newSessionClient(t)
	applicantClient := env.newSessionClient(t)

	registerEmployer := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("employer-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Employer User",
		"fullName":    "Employer User",
	})
	assertStatus(t, registerEmployer, http.StatusCreated)

	company := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName": "Applications Company",
		"slug":      fmt.Sprintf("applications-company-%d", time.Now().UnixNano()),
	})
	assertStatus(t, company, http.StatusCreated)
	companyID := stringField(t, company.body, "id")

	opportunity := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/opportunities", map[string]any{
		"companyId":           companyID,
		"title":               "Backend Internship",
		"summary":             "API implementation work",
		"slug":                fmt.Sprintf("applications-opportunity-%d", time.Now().UnixNano()),
		"description":         "Implement backend features",
		"type":                "internship",
		"participationFormat": "remote",
		"vacancyDetails": map[string]any{
			"employmentType":  "full_time",
			"experienceLevel": "junior",
		},
	})
	assertStatus(t, opportunity, http.StatusCreated)
	opportunityID := stringField(t, opportunity.body, "id")

	planned := env.requestJSONWithClient(t, employerClient, http.MethodPatch, "/v1/employer/opportunities/"+opportunityID, map[string]any{
		"status": "planned",
	})
	assertStatus(t, planned, http.StatusOK)

	env.execSQL(t, `UPDATE opportunities SET moderation_status = 'approved' WHERE id = $1`, opportunityID)

	registerApplicant := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       fmt.Sprintf("applicant-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Applicant User",
		"firstName":   "Ada",
		"lastName":    "Lovelace",
	})
	assertStatus(t, registerApplicant, http.StatusCreated)

	created := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/me/applications", map[string]any{
		"opportunityId": opportunityID,
		"coverLetter":   "I would like to help with the backend.",
	})
	assertStatus(t, created, http.StatusCreated)
	applicationID := stringField(t, created.body, "id")
	assertEqual(t, stringField(t, created.body, "status"), "submitted")
	assertCount(t, created.body, "history", 1)

	duplicate := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/me/applications", map[string]any{
		"opportunityId": opportunityID,
	})
	assertStatus(t, duplicate, http.StatusConflict)

	myList := env.requestJSONWithClient(t, applicantClient, http.MethodGet, "/v1/me/applications?status=submitted", nil)
	assertStatus(t, myList, http.StatusOK)
	assertCount(t, myList.body, "items", 1)
	assertEqual(t, nestedStringField(t, myList.body, "items", 0, "id"), applicationID)

	employerList := env.requestJSONWithClient(t, employerClient, http.MethodGet, "/v1/employer/opportunities/"+opportunityID+"/applications?status=submitted", nil)
	assertStatus(t, employerList, http.StatusOK)
	assertCount(t, employerList.body, "items", 1)
	assertEqual(t, nestedObjectStringField(t, nestedObject(t, employerList.body, "items", 0), "applicant", "displayName"), "Applicant User")

	reviewing := env.requestJSONWithClient(t, employerClient, http.MethodPatch, "/v1/employer/applications/"+applicationID+"/status", map[string]any{
		"status":  "reviewing",
		"comment": "Looks promising",
	})
	assertStatus(t, reviewing, http.StatusOK)
	assertEqual(t, stringField(t, reviewing.body, "status"), "reviewing")
	assertCount(t, reviewing.body, "history", 2)

	withdrawn := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/me/applications/"+applicationID+"/withdraw", nil)
	assertStatus(t, withdrawn, http.StatusOK)
	assertEqual(t, stringField(t, withdrawn.body, "status"), "withdrawn")
	assertCount(t, withdrawn.body, "history", 3)

	invalidEmployerTransition := env.requestJSONWithClient(t, employerClient, http.MethodPatch, "/v1/employer/applications/"+applicationID+"/status", map[string]any{
		"status": "accepted",
	})
	assertStatus(t, invalidEmployerTransition, http.StatusConflict)
}

func nestedObject(t *testing.T, body map[string]any, arrayField string, index int) map[string]any {
	t.Helper()

	items, ok := body[arrayField].([]any)
	if !ok || index >= len(items) {
		t.Fatalf("array field %q missing index %d", arrayField, index)
	}
	item, ok := items[index].(map[string]any)
	if !ok {
		t.Fatalf("array field %q item %d is not an object", arrayField, index)
	}
	return item
}

func nestedObjectStringField(t *testing.T, body map[string]any, field string, nestedField string) string {
	t.Helper()

	item, ok := body[field].(map[string]any)
	if !ok {
		t.Fatalf("field %q is not an object", field)
	}
	value, ok := item[nestedField].(string)
	if !ok {
		t.Fatalf("nested field %q is not a string: %v", nestedField, item[nestedField])
	}
	return value
}
