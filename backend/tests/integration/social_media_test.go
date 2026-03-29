//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestApplicantSocialLinksIntegration(t *testing.T) {
	env := newIntegrationEnv(t)
	applicantClient := env.newSessionClient(t)

	registerApplicant := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       fmt.Sprintf("social-applicant-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Social Applicant",
		"firstName":   "Ada",
		"lastName":    "Lovelace",
	})
	assertStatus(t, registerApplicant, http.StatusCreated)

	createLink := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/me/applicant/social-links", map[string]any{
		"platform": "telegram",
		"url":      "https://t.me/ada",
	})
	assertStatus(t, createLink, http.StatusCreated)
	linkID := stringField(t, createLink.body, "id")
	assertBoolValue(t, createLink.body, "isPublic", true)

	listLinks := env.requestJSONWithClient(t, applicantClient, http.MethodGet, "/v1/me/applicant/social-links", nil)
	assertStatus(t, listLinks, http.StatusOK)
	assertCount(t, listLinks.body, "items", 1)
	assertEqual(t, nestedStringField(t, listLinks.body, "items", 0, "id"), linkID)
	assertNestedBoolValue(t, listLinks.body, "items", 0, "isPublic", true)

	patchLink := env.requestJSONWithClient(t, applicantClient, http.MethodPatch, "/v1/me/applicant/social-links/"+linkID, map[string]any{
		"url":      "https://t.me/ada-updated",
		"isPublic": false,
	})
	assertStatus(t, patchLink, http.StatusOK)
	assertEqual(t, stringField(t, patchLink.body, "url"), "https://t.me/ada-updated")
	assertBoolValue(t, patchLink.body, "isPublic", false)

	deleteLink := env.requestJSONWithClient(t, applicantClient, http.MethodDelete, "/v1/me/applicant/social-links/"+linkID, nil)
	assertStatus(t, deleteLink, http.StatusNoContent)

	listAfterDelete := env.requestJSONWithClient(t, applicantClient, http.MethodGet, "/v1/me/applicant/social-links", nil)
	assertStatus(t, listAfterDelete, http.StatusOK)
	assertCount(t, listAfterDelete.body, "items", 0)
}

func TestCompanySocialLinksAndMediaIntegration(t *testing.T) {
	env := newIntegrationEnv(t)
	employerClient := env.newSessionClient(t)

	registerEmployer := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("company-assets-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Assets Employer",
		"fullName":    "Assets Employer",
	})
	assertStatus(t, registerEmployer, http.StatusCreated)

	createCompany := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName": "Assets Company",
		"slug":      fmt.Sprintf("assets-company-%d", time.Now().UnixNano()),
	})
	assertStatus(t, createCompany, http.StatusCreated)
	companyID := stringField(t, createCompany.body, "id")

	createSocialLink := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies/"+companyID+"/social-links", map[string]any{
		"platform": "website",
		"url":      "https://example.com/company",
	})
	assertStatus(t, createSocialLink, http.StatusCreated)
	socialLinkID := stringField(t, createSocialLink.body, "id")

	createMedia := env.requestJSONWithClient(t, employerClient, http.MethodPost, "/v1/employer/companies/"+companyID+"/media", map[string]any{
		"mediaFileId": "11111111-1111-1111-1111-111111111111",
		"title":       "Office",
		"sortOrder":   3,
	})
	assertStatus(t, createMedia, http.StatusCreated)
	mediaID := stringField(t, createMedia.body, "id")

	listSocialLinks := env.requestJSONWithClient(t, employerClient, http.MethodGet, "/v1/employer/companies/"+companyID+"/social-links", nil)
	assertStatus(t, listSocialLinks, http.StatusOK)
	assertCount(t, listSocialLinks.body, "items", 1)
	assertEqual(t, nestedStringField(t, listSocialLinks.body, "items", 0, "id"), socialLinkID)

	listMedia := env.requestJSONWithClient(t, employerClient, http.MethodGet, "/v1/employer/companies/"+companyID+"/media", nil)
	assertStatus(t, listMedia, http.StatusOK)
	assertCount(t, listMedia.body, "items", 1)
	assertEqual(t, nestedStringField(t, listMedia.body, "items", 0, "id"), mediaID)

	employerDetail := env.requestJSONWithClient(t, employerClient, http.MethodGet, "/v1/employer/companies/"+companyID, nil)
	assertStatus(t, employerDetail, http.StatusOK)
	assertArrayFieldLen(t, employerDetail.body, "socialLinks", 1)
	assertArrayFieldLen(t, employerDetail.body, "media", 1)

	publicDetail := env.requestJSON(t, http.MethodGet, "/v1/public/companies/"+companyID, nil)
	assertStatus(t, publicDetail, http.StatusOK)
	assertArrayFieldLen(t, publicDetail.body, "socialLinks", 1)
	assertArrayFieldLen(t, publicDetail.body, "media", 1)

	patchSocialLink := env.requestJSONWithClient(t, employerClient, http.MethodPatch, "/v1/employer/companies/"+companyID+"/social-links/"+socialLinkID, map[string]any{
		"url": "https://example.com/company-updated",
	})
	assertStatus(t, patchSocialLink, http.StatusOK)
	assertEqual(t, stringField(t, patchSocialLink.body, "url"), "https://example.com/company-updated")

	patchMedia := env.requestJSONWithClient(t, employerClient, http.MethodPatch, "/v1/employer/companies/"+companyID+"/media/"+mediaID, map[string]any{
		"title":     "Updated Office",
		"sortOrder": 1,
	})
	assertStatus(t, patchMedia, http.StatusOK)
	assertEqual(t, stringField(t, patchMedia.body, "title"), "Updated Office")

	deleteSocialLink := env.requestJSONWithClient(t, employerClient, http.MethodDelete, "/v1/employer/companies/"+companyID+"/social-links/"+socialLinkID, nil)
	assertStatus(t, deleteSocialLink, http.StatusNoContent)

	deleteMedia := env.requestJSONWithClient(t, employerClient, http.MethodDelete, "/v1/employer/companies/"+companyID+"/media/"+mediaID, nil)
	assertStatus(t, deleteMedia, http.StatusNoContent)

	publicDetailAfterDelete := env.requestJSON(t, http.MethodGet, "/v1/public/companies/"+companyID, nil)
	assertStatus(t, publicDetailAfterDelete, http.StatusOK)
	assertArrayFieldLen(t, publicDetailAfterDelete.body, "socialLinks", 0)
	assertArrayFieldLen(t, publicDetailAfterDelete.body, "media", 0)
}

func assertBoolValue(t *testing.T, body map[string]any, field string, expected bool) {
	t.Helper()

	value, ok := body[field].(bool)
	if !ok {
		t.Fatalf("field %q is not a bool: %v", field, body[field])
	}
	if value != expected {
		t.Fatalf("unexpected bool value for %q: got %v want %v", field, value, expected)
	}
}

func assertNestedBoolValue(t *testing.T, body map[string]any, arrayField string, index int, nestedField string, expected bool) {
	t.Helper()

	items := arrayFieldItems(t, body, arrayField)
	item, ok := items[index].(map[string]any)
	if !ok {
		t.Fatalf("array field %q item %d is not an object", arrayField, index)
	}
	value, ok := item[nestedField].(bool)
	if !ok {
		t.Fatalf("nested field %q is not a bool: %v", nestedField, item[nestedField])
	}
	if value != expected {
		t.Fatalf("unexpected bool value for %q: got %v want %v", nestedField, value, expected)
	}
}

func assertArrayFieldLen(t *testing.T, body map[string]any, field string, expected int) {
	t.Helper()

	items := arrayFieldItems(t, body, field)
	if len(items) != expected {
		t.Fatalf("unexpected %s length: got %d want %d", field, len(items), expected)
	}
}

func arrayFieldItems(t *testing.T, body map[string]any, field string) []any {
	t.Helper()

	items, ok := body[field].([]any)
	if !ok {
		t.Fatalf("field %q is not a JSON array: %v", field, body[field])
	}
	return items
}
