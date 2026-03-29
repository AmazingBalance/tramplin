//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestNotificationsIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	employerClient := env.newSessionClient(t)
	applicantClient := env.newSessionClient(t)

	registerEmployer := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("notifications-employer-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Notifications Employer",
		"fullName":    "Notifications Employer",
	})
	assertStatus(t, registerEmployer, http.StatusCreated)

	createCompany := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName": "Notifications Company",
		"slug":      fmt.Sprintf("notifications-company-%d", time.Now().UnixNano()),
	})
	assertStatus(t, createCompany, http.StatusCreated)
	companyID := stringField(t, createCompany.body, "id")

	createOpportunity := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/opportunities", map[string]any{
		"companyId":           companyID,
		"title":               "Notifications Opportunity",
		"summary":             "Opportunity for notifications",
		"slug":                fmt.Sprintf("notifications-opportunity-%d", time.Now().UnixNano()),
		"description":         "Use this to test notifications",
		"type":                "internship",
		"participationFormat": "remote",
		"vacancyDetails": map[string]any{
			"employmentType":  "full_time",
			"experienceLevel": "junior",
		},
	})
	assertStatus(t, createOpportunity, http.StatusCreated)
	opportunityID := stringField(t, createOpportunity.body, "id")

	planned := env.requestJSONWithClient(t, employerClient, http.MethodPatch, "/v1/employer/opportunities/"+opportunityID, map[string]any{
		"status": "planned",
	})
	assertStatus(t, planned, http.StatusOK)
	env.execSQL(t, `UPDATE opportunities SET moderation_status = 'approved' WHERE id = $1`, opportunityID)

	registerApplicant := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       fmt.Sprintf("notifications-applicant-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Notifications Applicant",
		"firstName":   "Linus",
		"lastName":    "Torvalds",
	})
	assertStatus(t, registerApplicant, http.StatusCreated)

	createApplication := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/me/applications", map[string]any{
		"opportunityId": opportunityID,
	})
	assertStatus(t, createApplication, http.StatusCreated)
	applicationID := stringField(t, createApplication.body, "id")

	reviewing := env.requestJSONWithClient(t, employerClient, http.MethodPatch, "/v1/employer/applications/"+applicationID+"/status", map[string]any{
		"status":  "reviewing",
		"comment": "Initial review",
	})
	assertStatus(t, reviewing, http.StatusOK)

	reserve := env.requestJSONWithClient(t, employerClient, http.MethodPatch, "/v1/employer/applications/"+applicationID+"/status", map[string]any{
		"status":  "reserve",
		"comment": "Reserve list",
	})
	assertStatus(t, reserve, http.StatusOK)

	listNotifications := env.requestJSONWithClient(t, applicantClient, http.MethodGet, "/v1/notifications", nil)
	assertStatus(t, listNotifications, http.StatusOK)
	assertCount(t, listNotifications.body, "items", 2)
	assertEqual(t, nestedStringField(t, listNotifications.body, "items", 0, "type"), "application_status_changed")
	assertEqual(t, nestedStringField(t, listNotifications.body, "items", 0, "sourceType"), "application")
	assertEqual(t, nestedStringField(t, listNotifications.body, "items", 0, "applicationId"), applicationID)
	assertEqual(t, nestedStringField(t, listNotifications.body, "items", 0, "opportunityId"), opportunityID)
	assertEqual(t, nestedStringField(t, listNotifications.body, "items", 0, "companyId"), companyID)
	assertNestedBoolValue(t, listNotifications.body, "items", 0, "isRead", false)

	unreadOnly := env.requestJSONWithClient(t, applicantClient, http.MethodGet, "/v1/notifications?unreadOnly=true", nil)
	assertStatus(t, unreadOnly, http.StatusOK)
	assertCount(t, unreadOnly.body, "items", 2)

	notificationID := nestedStringField(t, listNotifications.body, "items", 0, "id")
	markRead := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/notifications/"+notificationID+"/read", nil)
	assertStatus(t, markRead, http.StatusOK)
	assertEqual(t, stringField(t, markRead.body, "id"), notificationID)
	assertBoolValue(t, markRead.body, "isRead", true)

	unreadAfterSingle := env.requestJSONWithClient(t, applicantClient, http.MethodGet, "/v1/notifications?unreadOnly=true", nil)
	assertStatus(t, unreadAfterSingle, http.StatusOK)
	assertCount(t, unreadAfterSingle.body, "items", 1)

	readAll := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/notifications/read-all", nil)
	assertStatus(t, readAll, http.StatusOK)
	assertIntField(t, readAll.body, "updatedCount", 1)

	unreadAfterAll := env.requestJSONWithClient(t, applicantClient, http.MethodGet, "/v1/notifications?unreadOnly=true", nil)
	assertStatus(t, unreadAfterAll, http.StatusOK)
	assertCount(t, unreadAfterAll.body, "items", 0)
}

func assertIntField(t *testing.T, body map[string]any, field string, expected int) {
	t.Helper()

	value, ok := body[field].(float64)
	if !ok {
		t.Fatalf("field %q is not a number: %v", field, body[field])
	}
	if int(value) != expected {
		t.Fatalf("unexpected int value for %q: got %d want %d", field, int(value), expected)
	}
}
