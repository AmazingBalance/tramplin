//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestCuratorTagManagementIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	employerClient := env.newSessionClient(t)
	curatorClient := env.newSessionClient(t)

	registerEmployer := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("tag-employer-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Tag Employer",
		"fullName":    "Tag Employer",
	})
	assertStatus(t, registerEmployer, http.StatusCreated)

	employerTag := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/tags", map[string]any{
		"name":    "Campus Ambassador",
		"tagType": "custom",
	})
	assertStatus(t, employerTag, http.StatusCreated)
	employerTagID := stringField(t, employerTag.body, "id")
	assertEqual(t, stringField(t, employerTag.body, "tagType"), "custom")
	assertBoolValue(t, employerTag.body, "isActive", true)

	curatorID := seedCuratorSession(t, env, curatorClient, fmt.Sprintf("tag-curator-%d@example.com", time.Now().UnixNano()))

	listCustomTags := env.requestJSONWithClient(t, curatorClient, http.MethodGet, "/v1/curator/tags?type=custom&isActive=true", nil)
	assertStatus(t, listCustomTags, http.StatusOK)
	assertCount(t, listCustomTags.body, "items", 1)
	assertEqual(t, nestedStringField(t, listCustomTags.body, "items", 0, "id"), employerTagID)

	createCuratorTag := env.requestJSONWithClient(t, curatorClient, http.MethodPost, "/v1/curator/tags", map[string]any{
		"name":     "Rust",
		"tagType":  "technology",
		"isActive": false,
	})
	assertStatus(t, createCuratorTag, http.StatusCreated)
	curatorTagID := stringField(t, createCuratorTag.body, "id")
	assertEqual(t, stringField(t, createCuratorTag.body, "tagType"), "technology")
	assertBoolValue(t, createCuratorTag.body, "isActive", false)
	assertEqual(t, stringField(t, createCuratorTag.body, "createdByUserId"), curatorID)

	patchCuratorTag := env.requestJSONWithClient(t, curatorClient, http.MethodPatch, "/v1/curator/tags/"+curatorTagID, map[string]any{
		"name":     "RustLang",
		"isActive": true,
	})
	assertStatus(t, patchCuratorTag, http.StatusOK)
	assertEqual(t, stringField(t, patchCuratorTag.body, "name"), "RustLang")
	assertBoolValue(t, patchCuratorTag.body, "isActive", true)
}

func TestModerationCaseWorkflowIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	employerClient := env.newSessionClient(t)
	curatorClient := env.newSessionClient(t)

	registerEmployer := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("moderation-employer-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Moderation Employer",
		"fullName":    "Moderation Employer",
	})
	assertStatus(t, registerEmployer, http.StatusCreated)

	createCompany := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName": "Moderation Company",
		"slug":      fmt.Sprintf("moderation-company-%d", time.Now().UnixNano()),
	})
	assertStatus(t, createCompany, http.StatusCreated)
	companyID := stringField(t, createCompany.body, "id")

	createOpportunity := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/opportunities", map[string]any{
		"companyId":           companyID,
		"title":               "Moderated Opportunity",
		"summary":             "Needs curator review",
		"slug":                fmt.Sprintf("moderated-opportunity-%d", time.Now().UnixNano()),
		"description":         "Moderation workflow integration coverage",
		"type":                "internship",
		"participationFormat": "remote",
		"vacancyDetails": map[string]any{
			"employmentType":  "full_time",
			"experienceLevel": "junior",
		},
	})
	assertStatus(t, createOpportunity, http.StatusCreated)
	opportunityID := stringField(t, createOpportunity.body, "id")

	curatorID := seedCuratorSession(t, env, curatorClient, fmt.Sprintf("moderation-curator-%d@example.com", time.Now().UnixNano()))

	createCase := env.requestJSONWithClient(t, curatorClient, http.MethodPost, "/v1/curator/moderation-cases", map[string]any{
		"targetType": "opportunity",
		"targetId":   opportunityID,
		"reason":     "Initial moderation review",
	})
	assertStatus(t, createCase, http.StatusCreated)
	moderationCaseID := stringField(t, createCase.body, "id")
	assertEqual(t, stringField(t, createCase.body, "targetType"), "opportunity")
	assertEqual(t, nestedObjectStringField(t, createCase.body, "targetPreview", "id"), opportunityID)

	listCases := env.requestJSONWithClient(t, curatorClient, http.MethodGet, "/v1/curator/moderation-cases?targetType=opportunity&status=pending", nil)
	assertStatus(t, listCases, http.StatusOK)
	assertCount(t, listCases.body, "items", 1)
	assertEqual(t, nestedStringField(t, listCases.body, "items", 0, "id"), moderationCaseID)

	patchCase := env.requestJSONWithClient(t, curatorClient, http.MethodPatch, "/v1/curator/moderation-cases/"+moderationCaseID, map[string]any{
		"assignedCuratorUserId": curatorID,
		"status":                "approved",
		"reason":                "Opportunity content accepted",
	})
	assertStatus(t, patchCase, http.StatusOK)
	assertEqual(t, stringField(t, patchCase.body, "status"), "approved")
	assertEqual(t, stringField(t, patchCase.body, "assignedCuratorUserId"), curatorID)
	assertEqual(t, stringField(t, patchCase.body, "resolvedByCuratorUserId"), curatorID)
	_ = stringField(t, patchCase.body, "resolvedAt")

	opportunityDetail := env.requestJSONWithClient(t, employerClient, http.MethodGet, "/v1/employer/opportunities/"+opportunityID, nil)
	assertStatus(t, opportunityDetail, http.StatusOK)
	assertEqual(t, stringField(t, opportunityDetail.body, "moderationStatus"), "approved")
}
