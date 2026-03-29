//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestSavedEntitiesIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	employerClient := env.newSessionClient(t)
	applicantClient := env.newSessionClient(t)

	registerEmployer := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("saved-employer-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Saved Employer",
		"fullName":    "Saved Employer",
	})
	assertStatus(t, registerEmployer, http.StatusCreated)

	company := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName": "Saved Company",
		"slug":      fmt.Sprintf("saved-company-%d", time.Now().UnixNano()),
	})
	assertStatus(t, company, http.StatusCreated)
	companyID := stringField(t, company.body, "id")

	opportunity := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/opportunities", map[string]any{
		"companyId":           companyID,
		"title":               "Saved Opportunity",
		"summary":             "Opportunity to save",
		"slug":                fmt.Sprintf("saved-opportunity-%d", time.Now().UnixNano()),
		"description":         "Save me",
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
		"email":       fmt.Sprintf("saved-applicant-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Saved Applicant",
		"firstName":   "Grace",
		"lastName":    "Hopper",
	})
	assertStatus(t, registerApplicant, http.StatusCreated)

	saveOpportunity := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/me/saved-opportunities", map[string]any{
		"opportunityId": opportunityID,
	})
	assertStatus(t, saveOpportunity, http.StatusCreated)

	saveOpportunityAgain := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/me/saved-opportunities", map[string]any{
		"opportunityId": opportunityID,
	})
	assertStatus(t, saveOpportunityAgain, http.StatusConflict)

	savedOpportunities := env.requestJSONWithClient(t, applicantClient, http.MethodGet, "/v1/me/saved-opportunities", nil)
	assertStatus(t, savedOpportunities, http.StatusOK)
	assertCount(t, savedOpportunities.body, "items", 1)
	assertEqual(t, nestedStringField(t, savedOpportunities.body, "items", 0, "opportunityId"), opportunityID)

	deleteSavedOpportunity := env.requestJSONWithClient(t, applicantClient, http.MethodDelete, "/v1/me/saved-opportunities/"+opportunityID, nil)
	assertStatus(t, deleteSavedOpportunity, http.StatusNoContent)

	savedOpportunitiesAfterDelete := env.requestJSONWithClient(t, applicantClient, http.MethodGet, "/v1/me/saved-opportunities", nil)
	assertStatus(t, savedOpportunitiesAfterDelete, http.StatusOK)
	assertCount(t, savedOpportunitiesAfterDelete.body, "items", 0)

	saveCompany := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/me/saved-companies", map[string]any{
		"companyId": companyID,
	})
	assertStatus(t, saveCompany, http.StatusCreated)

	saveCompanyAgain := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/me/saved-companies", map[string]any{
		"companyId": companyID,
	})
	assertStatus(t, saveCompanyAgain, http.StatusConflict)

	savedCompanies := env.requestJSONWithClient(t, applicantClient, http.MethodGet, "/v1/me/saved-companies", nil)
	assertStatus(t, savedCompanies, http.StatusOK)
	assertCount(t, savedCompanies.body, "items", 1)
	assertEqual(t, nestedStringField(t, savedCompanies.body, "items", 0, "companyId"), companyID)

	deleteSavedCompany := env.requestJSONWithClient(t, applicantClient, http.MethodDelete, "/v1/me/saved-companies/"+companyID, nil)
	assertStatus(t, deleteSavedCompany, http.StatusNoContent)

	savedCompaniesAfterDelete := env.requestJSONWithClient(t, applicantClient, http.MethodGet, "/v1/me/saved-companies", nil)
	assertStatus(t, savedCompaniesAfterDelete, http.StatusOK)
	assertCount(t, savedCompaniesAfterDelete.body, "items", 0)
}
