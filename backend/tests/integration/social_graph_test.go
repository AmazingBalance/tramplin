//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestConnectionsAndRecommendationsIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	employerClient := env.newSessionClient(t)
	applicantAClient := env.newSessionClient(t)
	applicantBClient := env.newSessionClient(t)

	registerEmployer := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("graph-employer-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Graph Employer",
		"fullName":    "Graph Employer",
	})
	assertStatus(t, registerEmployer, http.StatusCreated)

	createCompany := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName": "Graph Company",
		"slug":      fmt.Sprintf("graph-company-%d", time.Now().UnixNano()),
	})
	assertStatus(t, createCompany, http.StatusCreated)
	companyID := stringField(t, createCompany.body, "id")

	createOpportunity := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/opportunities", map[string]any{
		"companyId":           companyID,
		"title":               "Graph Opportunity",
		"summary":             "Opportunity for recommendations",
		"slug":                fmt.Sprintf("graph-opportunity-%d", time.Now().UnixNano()),
		"description":         "Recommend this",
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

	registerApplicantA := env.requestJSONWithClient(t, applicantAClient, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       fmt.Sprintf("graph-applicant-a-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Applicant A",
		"firstName":   "Alice",
		"lastName":    "Graph",
	})
	assertStatus(t, registerApplicantA, http.StatusCreated)
	applicantAID := nestedObjectStringField(t, registerApplicantA.body, "user", "id")

	registerApplicantB := env.requestJSONWithClient(t, applicantBClient, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       fmt.Sprintf("graph-applicant-b-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Applicant B",
		"firstName":   "Bob",
		"lastName":    "Graph",
	})
	assertStatus(t, registerApplicantB, http.StatusCreated)
	applicantBID := nestedObjectStringField(t, registerApplicantB.body, "user", "id")

	createConnection := env.requestJSONWithClient(t, applicantAClient, http.MethodPost, "/v1/me/connections", map[string]any{
		"targetApplicantUserId": applicantBID,
		"initiatorNote":         "Let us connect",
	})
	assertStatus(t, createConnection, http.StatusCreated)
	connectionID := stringField(t, createConnection.body, "id")
	assertEqual(t, stringField(t, createConnection.body, "initiatorUserId"), applicantAID)
	assertEqual(t, nestedStringFieldFromObject(t, createConnection.body, "otherApplicant", "userId"), applicantBID)

	createConnectionAgain := env.requestJSONWithClient(t, applicantAClient, http.MethodPost, "/v1/me/connections", map[string]any{
		"targetApplicantUserId": applicantBID,
	})
	assertStatus(t, createConnectionAgain, http.StatusConflict)

	listConnectionsA := env.requestJSONWithClient(t, applicantAClient, http.MethodGet, "/v1/me/connections?status=pending", nil)
	assertStatus(t, listConnectionsA, http.StatusOK)
	assertCount(t, listConnectionsA.body, "items", 1)
	assertEqual(t, nestedStringField(t, listConnectionsA.body, "items", 0, "id"), connectionID)

	acceptConnection := env.requestJSONWithClient(t, applicantBClient, http.MethodPatch, "/v1/me/connections/"+connectionID, map[string]any{
		"status": "accepted",
	})
	assertStatus(t, acceptConnection, http.StatusOK)
	assertEqual(t, stringField(t, acceptConnection.body, "status"), "accepted")
	assertEqual(t, nestedStringFieldFromObject(t, acceptConnection.body, "otherApplicant", "userId"), applicantAID)

	listConnectionsB := env.requestJSONWithClient(t, applicantBClient, http.MethodGet, "/v1/me/connections?status=accepted", nil)
	assertStatus(t, listConnectionsB, http.StatusOK)
	assertCount(t, listConnectionsB.body, "items", 1)

	createRecommendation := env.requestJSONWithClient(t, applicantAClient, http.MethodPost, "/v1/me/recommendations", map[string]any{
		"recipientUserId": applicantBID,
		"opportunityId":   opportunityID,
		"message":         "This fits your profile",
	})
	assertStatus(t, createRecommendation, http.StatusCreated)
	recommendationID := stringField(t, createRecommendation.body, "id")
	assertEqual(t, stringField(t, createRecommendation.body, "recommenderUserId"), applicantAID)
	assertEqual(t, stringField(t, createRecommendation.body, "recipientUserId"), applicantBID)

	sentRecommendations := env.requestJSONWithClient(t, applicantAClient, http.MethodGet, "/v1/me/recommendations/sent", nil)
	assertStatus(t, sentRecommendations, http.StatusOK)
	assertCount(t, sentRecommendations.body, "items", 1)
	assertEqual(t, nestedStringField(t, sentRecommendations.body, "items", 0, "id"), recommendationID)

	receivedRecommendations := env.requestJSONWithClient(t, applicantBClient, http.MethodGet, "/v1/me/recommendations/received", nil)
	assertStatus(t, receivedRecommendations, http.StatusOK)
	assertCount(t, receivedRecommendations.body, "items", 1)
	assertEqual(t, nestedStringField(t, receivedRecommendations.body, "items", 0, "id"), recommendationID)

	notifications := env.requestJSONWithClient(t, applicantBClient, http.MethodGet, "/v1/notifications?unreadOnly=true", nil)
	assertStatus(t, notifications, http.StatusOK)
	assertCount(t, notifications.body, "items", 1)
	assertEqual(t, nestedStringField(t, notifications.body, "items", 0, "type"), "recommendation_received")
	assertEqual(t, nestedStringField(t, notifications.body, "items", 0, "sourceType"), "recommendation")
	assertEqual(t, nestedStringField(t, notifications.body, "items", 0, "sourceId"), recommendationID)
	assertEqual(t, nestedStringField(t, notifications.body, "items", 0, "opportunityId"), opportunityID)

	disableRecommendations := env.requestJSONWithClient(t, applicantBClient, http.MethodPatch, "/v1/me/applicant/privacy", map[string]any{
		"allowRecommendations": false,
	})
	assertStatus(t, disableRecommendations, http.StatusOK)

	createRecommendationAfterDisable := env.requestJSONWithClient(t, applicantAClient, http.MethodPost, "/v1/me/recommendations", map[string]any{
		"recipientUserId": applicantBID,
		"opportunityId":   opportunityID,
	})
	assertStatus(t, createRecommendationAfterDisable, http.StatusForbidden)
}

func nestedStringFieldFromObject(t *testing.T, body map[string]any, objectField string, nestedField string) string {
	t.Helper()

	object, ok := body[objectField].(map[string]any)
	if !ok {
		t.Fatalf("field %q is not an object: %v", objectField, body[objectField])
	}
	value, ok := object[nestedField].(string)
	if !ok {
		t.Fatalf("nested field %q is not a string: %v", nestedField, object[nestedField])
	}
	return value
}
