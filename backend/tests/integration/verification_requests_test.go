//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"tramplin/backend/internal/domain/model"
	authplatform "tramplin/backend/internal/platform/auth"
)

func TestVerificationRequestWorkflowIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	employerClient := env.newSessionClient(t)
	curatorClient := env.newSessionClient(t)

	registerEmployer := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("verification-employer-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Verification Employer",
		"fullName":    "Verification Employer",
	})
	assertStatus(t, registerEmployer, http.StatusCreated)

	createCompany := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName": "Verification Company",
		"slug":      fmt.Sprintf("verification-company-%d", time.Now().UnixNano()),
	})
	assertStatus(t, createCompany, http.StatusCreated)
	companyID := stringField(t, createCompany.body, "id")

	createRequest := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies/"+companyID+"/verification-requests", map[string]any{
		"method":           "corporate_email",
		"submittedComment": "Primary corporate mailbox is active",
		"evidence": []map[string]any{
			{
				"evidenceType": "corporate_email",
				"value":        "hr@verification.example.com",
			},
		},
	})
	assertStatus(t, createRequest, http.StatusCreated)
	verificationRequestID := stringField(t, createRequest.body, "id")
	assertEqual(t, stringField(t, createRequest.body, "status"), "pending")
	assertEqual(t, stringField(t, createRequest.body, "companyVerificationStatus"), "pending")
	assertArrayFieldLen(t, createRequest.body, "evidence", 1)
	assertEqual(t, nestedStringField(t, createRequest.body, "evidence", 0, "evidenceType"), "corporate_email")

	listCompanyRequests := env.requestJSONWithClient(t, employerClient, http.MethodGet, "/v1/employer/companies/"+companyID+"/verification-requests", nil)
	assertStatus(t, listCompanyRequests, http.StatusOK)
	assertCount(t, listCompanyRequests.body, "items", 1)
	assertEqual(t, nestedStringField(t, listCompanyRequests.body, "items", 0, "id"), verificationRequestID)

	duplicateRequest := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies/"+companyID+"/verification-requests", map[string]any{
		"method": "corporate_email",
		"evidence": []map[string]any{
			{
				"evidenceType": "corporate_email",
				"value":        "hr@verification.example.com",
			},
		},
	})
	assertStatus(t, duplicateRequest, http.StatusConflict)

	seedCuratorSession(t, env, curatorClient, fmt.Sprintf("verification-curator-%d@example.com", time.Now().UnixNano()))

	listCuratorRequests := env.requestJSONWithClient(t, curatorClient, http.MethodGet, "/v1/curator/verification-requests?status=pending", nil)
	assertStatus(t, listCuratorRequests, http.StatusOK)
	assertCount(t, listCuratorRequests.body, "items", 1)
	assertEqual(t, nestedStringField(t, listCuratorRequests.body, "items", 0, "id"), verificationRequestID)

	markUnderReview := env.requestJSONWithClient(t, curatorClient, http.MethodPatch, "/v1/curator/verification-requests/"+verificationRequestID+"/review", map[string]any{
		"status":        "under_review",
		"reviewComment": "Review in progress",
	})
	assertStatus(t, markUnderReview, http.StatusOK)
	assertEqual(t, stringField(t, markUnderReview.body, "status"), "under_review")
	assertEqual(t, stringField(t, markUnderReview.body, "companyVerificationStatus"), "under_review")

	listCompanyRequestsAfterReview := env.requestJSONWithClient(t, employerClient, http.MethodGet, "/v1/employer/companies/"+companyID+"/verification-requests", nil)
	assertStatus(t, listCompanyRequestsAfterReview, http.StatusOK)
	assertCount(t, listCompanyRequestsAfterReview.body, "items", 1)
	assertEqual(t, nestedStringField(t, listCompanyRequestsAfterReview.body, "items", 0, "status"), "under_review")
	assertEqual(t, nestedStringField(t, listCompanyRequestsAfterReview.body, "items", 0, "companyVerificationStatus"), "under_review")

	approveRequest := env.requestJSONWithClient(t, curatorClient, http.MethodPatch, "/v1/curator/verification-requests/"+verificationRequestID+"/review", map[string]any{
		"status": "approved",
	})
	assertStatus(t, approveRequest, http.StatusOK)
	assertEqual(t, stringField(t, approveRequest.body, "status"), "approved")
	assertEqual(t, stringField(t, approveRequest.body, "companyVerificationStatus"), "verified")
	_ = stringField(t, approveRequest.body, "reviewedByCuratorUserId")

	companyDetail := env.requestJSONWithClient(t, employerClient, http.MethodGet, "/v1/employer/companies/"+companyID, nil)
	assertStatus(t, companyDetail, http.StatusOK)
	assertEqual(t, stringField(t, companyDetail.body, "verificationStatus"), "verified")
	_ = stringField(t, companyDetail.body, "verifiedByCuratorUserId")
}

func seedCuratorSession(t *testing.T, env *integrationEnv, client *http.Client, email string) string {
	t.Helper()

	passwordHash, err := authplatform.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash curator password: %v", err)
	}

	userID := "11111111-1111-4111-8111-111111111111"
	now := time.Now().UTC()

	env.execSQL(t, `
		INSERT INTO users (
			id, email, password_hash, display_name, role, is_active, token_version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, TRUE, 0, $6, $6)
	`, userID, email, passwordHash, "Integration Curator", model.UserRoleCurator, now)
	env.execSQL(t, `
		INSERT INTO curator_profiles (user_id, full_name, is_admin, created_at)
		VALUES ($1, $2, FALSE, $3)
	`, userID, "Integration Curator", now)

	login := env.requestJSONWithClient(t, client, http.MethodPost, "/v1/auth/login", map[string]any{
		"email":    email,
		"password": "password123",
	})
	assertStatus(t, login, http.StatusOK)
	return userID
}
