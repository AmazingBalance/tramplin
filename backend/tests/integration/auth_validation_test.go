//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestPasswordLengthValidationIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	shortApplicant := env.requestJSON(t, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       fmt.Sprintf("short-applicant-%d@example.com", time.Now().UnixNano()),
		"password":    "short",
		"displayName": "Short Applicant",
		"firstName":   "Short",
		"lastName":    "Applicant",
	})
	assertStatus(t, shortApplicant, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, shortApplicant.body, "message"), "password must be at least 8 characters")

	shortEmployer := env.requestJSON(t, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("short-employer-%d@example.com", time.Now().UnixNano()),
		"password":    "short",
		"displayName": "Short Employer",
		"fullName":    "Short Employer",
	})
	assertStatus(t, shortEmployer, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, shortEmployer.body, "message"), "password must be at least 8 characters")
}

func TestAdminCuratorPasswordLengthValidationIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	adminClient := env.newSessionClient(t)
	seedAdminCuratorSession(t, env, adminClient, fmt.Sprintf("short-admin-%d@example.com", time.Now().UnixNano()))

	shortCurator := env.requestJSONWithClient(t, adminClient, http.MethodPost, "/v1/curator/admin/curators", map[string]any{
		"email":       fmt.Sprintf("short-curator-%d@example.com", time.Now().UnixNano()),
		"password":    "short",
		"displayName": "Short Curator",
		"fullName":    "Short Curator",
		"reason":      "Should fail password validation",
	})
	assertStatus(t, shortCurator, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, shortCurator.body, "message"), "password must be at least 8 characters")
}

func TestEmailValidationIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	invalidApplicant := env.requestJSON(t, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       "not-an-email",
		"password":    "password123",
		"displayName": "Invalid Applicant",
		"firstName":   "Invalid",
		"lastName":    "Applicant",
	})
	assertStatus(t, invalidApplicant, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidApplicant.body, "message"), "invalid email")

	invalidEmployer := env.requestJSON(t, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       "not-an-email",
		"password":    "password123",
		"displayName": "Invalid Employer",
		"fullName":    "Invalid Employer",
	})
	assertStatus(t, invalidEmployer, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidEmployer.body, "message"), "invalid email")

	missingPassword := env.requestJSON(t, http.MethodPost, "/v1/auth/login", map[string]any{
		"email": "user@example.com",
	})
	assertStatus(t, missingPassword, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, missingPassword.body, "message"), "required fields are missing")

	invalidLoginEmail := env.requestJSON(t, http.MethodPost, "/v1/auth/login", map[string]any{
		"email":    "not-an-email",
		"password": "password123",
	})
	assertStatus(t, invalidLoginEmail, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidLoginEmail.body, "message"), "invalid email")

	adminClient := env.newSessionClient(t)
	seedAdminCuratorSession(t, env, adminClient, fmt.Sprintf("email-admin-%d@example.com", time.Now().UnixNano()))

	invalidCurator := env.requestJSONWithClient(t, adminClient, http.MethodPost, "/v1/curator/admin/curators", map[string]any{
		"email":       "not-an-email",
		"password":    "password123",
		"displayName": "Invalid Curator",
		"fullName":    "Invalid Curator",
		"reason":      "Validation coverage",
	})
	assertStatus(t, invalidCurator, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidCurator.body, "message"), "invalid email")

	employerClient := env.newSessionClient(t)
	registerEmployer := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("validation-owner-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Validation Employer",
		"fullName":    "Validation Employer",
	})
	assertStatus(t, registerEmployer, http.StatusCreated)

	company := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName": "Validation Company",
		"slug":      fmt.Sprintf("validation-company-%d", time.Now().UnixNano()),
	})
	assertStatus(t, company, http.StatusCreated)
	companyID := stringField(t, company.body, "id")

	invalidInvite := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies/"+companyID+"/memberships", map[string]any{
		"employerEmail": "not-an-email",
		"memberRole":    "recruiter",
	})
	assertStatus(t, invalidInvite, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidInvite.body, "message"), "invalid employerEmail")

	invalidOpportunity := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/opportunities", map[string]any{
		"companyId":           companyID,
		"title":               "Validation Opportunity",
		"summary":             "Should fail invalid contactEmail",
		"slug":                fmt.Sprintf("validation-opportunity-%d", time.Now().UnixNano()),
		"description":         "Validation coverage",
		"type":                "internship",
		"participationFormat": "remote",
		"contactEmail":        "not-an-email",
		"vacancyDetails": map[string]any{
			"employmentType":  "full_time",
			"experienceLevel": "junior",
		},
	})
	assertStatus(t, invalidOpportunity, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidOpportunity.body, "message"), "invalid contactEmail")

	validOpportunity := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/opportunities", map[string]any{
		"companyId":           companyID,
		"title":               "Validation Opportunity",
		"summary":             "Will later test patch validation",
		"slug":                fmt.Sprintf("validation-opportunity-valid-%d", time.Now().UnixNano()),
		"description":         "Validation coverage",
		"type":                "internship",
		"participationFormat": "remote",
		"vacancyDetails": map[string]any{
			"employmentType":  "full_time",
			"experienceLevel": "junior",
		},
	})
	assertStatus(t, validOpportunity, http.StatusCreated)
	opportunityID := stringField(t, validOpportunity.body, "id")

	invalidOpportunityPatch := env.requestJSONWithClient(t, employerClient, http.MethodPatch, "/v1/employer/opportunities/"+opportunityID, map[string]any{
		"contactEmail": "not-an-email",
	})
	assertStatus(t, invalidOpportunityPatch, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidOpportunityPatch.body, "message"), "invalid contactEmail")
}

func TestURIValidationIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	applicantClient := env.newSessionClient(t)
	registerApplicant := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       fmt.Sprintf("uri-applicant-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "URI Applicant",
		"firstName":   "Uri",
		"lastName":    "Applicant",
	})
	assertStatus(t, registerApplicant, http.StatusCreated)

	invalidApplicantLink := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/me/applicant/social-links", map[string]any{
		"platform": "telegram",
		"url":      "not-a-uri",
	})
	assertStatus(t, invalidApplicantLink, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidApplicantLink.body, "message"), "invalid url")

	validApplicantLink := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/me/applicant/social-links", map[string]any{
		"platform": "telegram",
		"url":      "https://t.me/uriapplicant",
	})
	assertStatus(t, validApplicantLink, http.StatusCreated)
	applicantLinkID := stringField(t, validApplicantLink.body, "id")

	invalidApplicantLinkPatch := env.requestJSONWithClient(t, applicantClient, http.MethodPatch, "/v1/me/applicant/social-links/"+applicantLinkID, map[string]any{
		"url": "not-a-uri",
	})
	assertStatus(t, invalidApplicantLinkPatch, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidApplicantLinkPatch.body, "message"), "invalid url")

	employerClient := env.newSessionClient(t)
	registerEmployer := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("uri-employer-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "URI Employer",
		"fullName":    "URI Employer",
	})
	assertStatus(t, registerEmployer, http.StatusCreated)

	invalidCompany := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName":  "URI Invalid Company",
		"slug":       fmt.Sprintf("uri-invalid-company-%d", time.Now().UnixNano()),
		"websiteUrl": "not-a-uri",
	})
	assertStatus(t, invalidCompany, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidCompany.body, "message"), "invalid websiteUrl")

	company := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName": "URI Company",
		"slug":      fmt.Sprintf("uri-company-%d", time.Now().UnixNano()),
	})
	assertStatus(t, company, http.StatusCreated)
	companyID := stringField(t, company.body, "id")

	invalidCompanyPatch := env.requestJSONWithClient(t, employerClient, http.MethodPatch, "/v1/employer/companies/"+companyID, map[string]any{
		"websiteUrl": "not-a-uri",
	})
	assertStatus(t, invalidCompanyPatch, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidCompanyPatch.body, "message"), "invalid websiteUrl")

	invalidCompanySocialLink := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies/"+companyID+"/social-links", map[string]any{
		"platform": "website",
		"url":      "not-a-uri",
	})
	assertStatus(t, invalidCompanySocialLink, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidCompanySocialLink.body, "message"), "invalid url")

	validCompanySocialLink := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies/"+companyID+"/social-links", map[string]any{
		"platform": "website",
		"url":      "https://example.com/company",
	})
	assertStatus(t, validCompanySocialLink, http.StatusCreated)
	companySocialLinkID := stringField(t, validCompanySocialLink.body, "id")

	invalidCompanySocialLinkPatch := env.requestJSONWithClient(t, employerClient, http.MethodPatch, "/v1/employer/companies/"+companyID+"/social-links/"+companySocialLinkID, map[string]any{
		"url": "not-a-uri",
	})
	assertStatus(t, invalidCompanySocialLinkPatch, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidCompanySocialLinkPatch.body, "message"), "invalid url")

	invalidOpportunity := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/opportunities", map[string]any{
		"companyId":           companyID,
		"title":               "URI Opportunity Invalid",
		"summary":             "Should fail invalid link url",
		"slug":                fmt.Sprintf("uri-opportunity-invalid-%d", time.Now().UnixNano()),
		"description":         "URI validation coverage",
		"type":                "internship",
		"participationFormat": "remote",
		"vacancyDetails": map[string]any{
			"employmentType":  "full_time",
			"experienceLevel": "junior",
		},
		"links": []map[string]any{
			{
				"linkType": "apply",
				"title":    "Apply",
				"url":      "not-a-uri",
			},
		},
	})
	assertStatus(t, invalidOpportunity, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidOpportunity.body, "message"), "invalid links.url")

	validOpportunity := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/opportunities", map[string]any{
		"companyId":           companyID,
		"title":               "URI Opportunity Valid",
		"summary":             "Will be patched with invalid link url",
		"slug":                fmt.Sprintf("uri-opportunity-valid-%d", time.Now().UnixNano()),
		"description":         "URI validation coverage",
		"type":                "internship",
		"participationFormat": "remote",
		"vacancyDetails": map[string]any{
			"employmentType":  "full_time",
			"experienceLevel": "junior",
		},
	})
	assertStatus(t, validOpportunity, http.StatusCreated)
	opportunityID := stringField(t, validOpportunity.body, "id")

	invalidOpportunityPatch := env.requestJSONWithClient(t, employerClient, http.MethodPatch, "/v1/employer/opportunities/"+opportunityID, map[string]any{
		"links": []map[string]any{
			{
				"linkType": "apply",
				"title":    "Apply",
				"url":      "not-a-uri",
			},
		},
	})
	assertStatus(t, invalidOpportunityPatch, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidOpportunityPatch.body, "message"), "invalid links.url")

	invalidVerificationRequest := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies/"+companyID+"/verification-requests", map[string]any{
		"method": "official_website",
		"evidence": []map[string]any{
			{
				"evidenceType": "website_link",
				"value":        "not-a-uri",
			},
		},
	})
	assertStatus(t, invalidVerificationRequest, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, invalidVerificationRequest.body, "message"), "invalid evidence.value")
}
