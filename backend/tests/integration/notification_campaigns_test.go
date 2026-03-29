//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestNotificationCampaignWorkflowIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	employerClient := env.newSessionClient(t)
	applicantAClient := env.newSessionClient(t)
	applicantBClient := env.newSessionClient(t)

	registerEmployer := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("campaign-employer-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Campaign Employer",
		"fullName":    "Campaign Employer",
	})
	assertStatus(t, registerEmployer, http.StatusCreated)

	createCompany := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName": "Campaign Company",
		"slug":      fmt.Sprintf("campaign-company-%d", time.Now().UnixNano()),
	})
	assertStatus(t, createCompany, http.StatusCreated)
	companyID := stringField(t, createCompany.body, "id")

	createOpportunity := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/opportunities", map[string]any{
		"companyId":           companyID,
		"title":               "Campaign Opportunity",
		"summary":             "Audience source opportunity",
		"slug":                fmt.Sprintf("campaign-opportunity-%d", time.Now().UnixNano()),
		"description":         "Used for campaign recipients",
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
		"email":       fmt.Sprintf("campaign-applicant-a-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Applicant A",
		"firstName":   "Anna",
		"lastName":    "Audience",
	})
	assertStatus(t, registerApplicantA, http.StatusCreated)
	applicantAID := nestedObjectStringField(t, registerApplicantA.body, "user", "id")

	registerApplicantB := env.requestJSONWithClient(t, applicantBClient, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       fmt.Sprintf("campaign-applicant-b-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Applicant B",
		"firstName":   "Boris",
		"lastName":    "Audience",
	})
	assertStatus(t, registerApplicantB, http.StatusCreated)
	applicantBID := nestedObjectStringField(t, registerApplicantB.body, "user", "id")

	applyA := env.requestJSONWithClient(t, applicantAClient, http.MethodPost, "/v1/me/applications", map[string]any{
		"opportunityId": opportunityID,
	})
	assertStatus(t, applyA, http.StatusCreated)

	applyB := env.requestJSONWithClient(t, applicantBClient, http.MethodPost, "/v1/me/applications", map[string]any{
		"opportunityId": opportunityID,
	})
	assertStatus(t, applyB, http.StatusCreated)

	createCampaign := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/notification-campaigns", map[string]any{
		"companyId":        companyID,
		"audienceType":     "single_applicant",
		"title":            "Draft Campaign",
		"body":             "Hello applicant",
		"sendViaInApp":     true,
		"sendViaEmail":     false,
		"applicantUserIds": []string{applicantAID},
	})
	assertStatus(t, createCampaign, http.StatusCreated)
	campaignID := stringField(t, createCampaign.body, "id")
	assertEqual(t, stringField(t, createCampaign.body, "status"), "draft")
	assertArrayFieldLen(t, createCampaign.body, "recipients", 1)
	assertEqual(t, nestedStringField(t, createCampaign.body, "recipients", 0, "applicantUserId"), applicantAID)

	listCampaigns := env.requestJSONWithClient(t, employerClient, http.MethodGet, "/v1/employer/notification-campaigns?companyId="+companyID, nil)
	assertStatus(t, listCampaigns, http.StatusOK)
	assertCount(t, listCampaigns.body, "items", 1)
	assertEqual(t, nestedStringField(t, listCampaigns.body, "items", 0, "id"), campaignID)

	getCampaign := env.requestJSONWithClient(t, employerClient, http.MethodGet, "/v1/employer/notification-campaigns/"+campaignID, nil)
	assertStatus(t, getCampaign, http.StatusOK)
	assertArrayFieldLen(t, getCampaign.body, "recipients", 1)

	patchCampaign := env.requestJSONWithClient(t, employerClient, http.MethodPatch, "/v1/employer/notification-campaigns/"+campaignID, map[string]any{
		"audienceType":     "selected_applicants",
		"applicantUserIds": []string{applicantAID, applicantBID},
		"title":            "Updated Campaign",
		"body":             "Updated body",
		"sendViaInApp":     true,
		"sendViaEmail":     false,
		"scheduledAt":      time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339),
	})
	assertStatus(t, patchCampaign, http.StatusOK)
	assertEqual(t, stringField(t, patchCampaign.body, "status"), "scheduled")
	assertEqual(t, stringField(t, patchCampaign.body, "title"), "Updated Campaign")
	assertArrayFieldLen(t, patchCampaign.body, "recipients", 2)

	sendCampaign := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/notification-campaigns/"+campaignID+"/send", nil)
	assertStatus(t, sendCampaign, http.StatusOK)
	assertEqual(t, stringField(t, sendCampaign.body, "status"), "sent")
	_ = stringField(t, sendCampaign.body, "sentAt")
	assertArrayFieldLen(t, sendCampaign.body, "recipients", 2)

	notificationsA := env.requestJSONWithClient(t, applicantAClient, http.MethodGet, "/v1/notifications?unreadOnly=true", nil)
	assertStatus(t, notificationsA, http.StatusOK)
	assertCount(t, notificationsA.body, "items", 1)
	assertEqual(t, nestedStringField(t, notificationsA.body, "items", 0, "type"), "employer_broadcast")
	assertEqual(t, nestedStringField(t, notificationsA.body, "items", 0, "sourceType"), "campaign")
	assertEqual(t, nestedStringField(t, notificationsA.body, "items", 0, "sourceId"), campaignID)
	assertEqual(t, nestedStringField(t, notificationsA.body, "items", 0, "companyId"), companyID)
	_ = nestedStringField(t, notificationsA.body, "items", 0, "applicationId")

	notificationsB := env.requestJSONWithClient(t, applicantBClient, http.MethodGet, "/v1/notifications?unreadOnly=true", nil)
	assertStatus(t, notificationsB, http.StatusOK)
	assertCount(t, notificationsB.body, "items", 1)
	assertEqual(t, nestedStringField(t, notificationsB.body, "items", 0, "sourceId"), campaignID)

	createAllApplicantsCampaign := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/notification-campaigns", map[string]any{
		"companyId":     companyID,
		"audienceType":  "all_applicants_of_opportunity",
		"opportunityId": opportunityID,
		"title":         "All Applicants Campaign",
		"body":          "Scheduled for later",
		"sendViaInApp":  true,
		"sendViaEmail":  false,
	})
	assertStatus(t, createAllApplicantsCampaign, http.StatusCreated)
	cancelCampaignID := stringField(t, createAllApplicantsCampaign.body, "id")
	assertEqual(t, stringField(t, createAllApplicantsCampaign.body, "status"), "draft")
	assertArrayFieldLen(t, createAllApplicantsCampaign.body, "recipients", 2)

	cancelCampaign := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/notification-campaigns/"+cancelCampaignID+"/cancel", nil)
	assertStatus(t, cancelCampaign, http.StatusOK)
	assertEqual(t, stringField(t, cancelCampaign.body, "status"), "cancelled")
}
