//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestUploadLifecycleIntegration(t *testing.T) {
	env := newIntegrationEnv(t)
	runUploadLifecycleIntegration(t, env)
}

func runUploadLifecycleIntegration(t *testing.T, env *integrationEnv) {
	t.Helper()

	register := env.requestJSON(t, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       fmt.Sprintf("media-applicant-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Media Applicant",
		"firstName":   "Media",
		"lastName":    "Applicant",
	})
	assertStatus(t, register, http.StatusCreated)

	content := []byte("resume-bytes")
	presign := env.requestJSON(t, http.MethodPost, "/v1/uploads/presign", map[string]any{
		"originalName": "resume.pdf",
		"mimeType":     "application/pdf",
		"fileSize":     len(content),
		"purpose":      "resume",
	})
	assertStatus(t, presign, http.StatusOK)

	uploadURL := stringField(t, presign.body, "uploadUrl")
	mediaFileID := objectStringField(t, presign.body, "file", "id")

	putReq, err := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("build upload request: %v", err)
	}
	putReq.Header.Set("Content-Type", "application/pdf")
	putResp, err := env.client.Do(putReq)
	if err != nil {
		t.Fatalf("upload object: %v", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putResp.Body)
		t.Fatalf("upload status = %d, want %d, body = %s", putResp.StatusCode, http.StatusOK, string(body))
	}
	etag := putResp.Header.Get("ETag")

	complete := env.requestJSON(t, http.MethodPost, "/v1/uploads/"+mediaFileID+"/complete", map[string]any{
		"etag": etag,
	})
	assertStatus(t, complete, http.StatusOK)
	assertEqual(t, stringField(t, complete.body, "status"), "uploaded")

	metadata := env.requestJSON(t, http.MethodGet, "/v1/media/"+mediaFileID, nil)
	assertStatus(t, metadata, http.StatusOK)
	assertEqual(t, stringField(t, metadata.body, "mimeType"), "application/pdf")

	deleted := env.requestJSON(t, http.MethodDelete, "/v1/media/"+mediaFileID, nil)
	assertStatus(t, deleted, http.StatusNoContent)

	missing := env.requestJSON(t, http.MethodGet, "/v1/media/"+mediaFileID, nil)
	assertStatus(t, missing, http.StatusNotFound)
}

func TestPublicMediaDownloadAccessIntegration(t *testing.T) {
	env := newIntegrationEnv(t)
	runPublicMediaDownloadAccessIntegration(t, env)
}

func runPublicMediaDownloadAccessIntegration(t *testing.T, env *integrationEnv) {
	t.Helper()

	registerEmployer := env.requestJSON(t, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("media-employer-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Media Employer",
		"fullName":    "Media Employer",
	})
	assertStatus(t, registerEmployer, http.StatusCreated)

	company := env.requestJSON(t, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName": "Media Company",
		"slug":      fmt.Sprintf("media-company-%d", time.Now().UnixNano()),
	})
	assertStatus(t, company, http.StatusCreated)
	companyID := stringField(t, company.body, "id")

	content := []byte("public-company-logo")
	presign := env.requestJSON(t, http.MethodPost, "/v1/uploads/presign", map[string]any{
		"originalName": "logo.png",
		"mimeType":     "image/png",
		"fileSize":     len(content),
		"purpose":      "company_logo",
	})
	assertStatus(t, presign, http.StatusOK)
	uploadURL := stringField(t, presign.body, "uploadUrl")
	mediaFileID := objectStringField(t, presign.body, "file", "id")

	putReq, err := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("build upload request: %v", err)
	}
	putReq.Header.Set("Content-Type", "image/png")
	putResp, err := env.client.Do(putReq)
	if err != nil {
		t.Fatalf("upload object: %v", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putResp.Body)
		t.Fatalf("upload status = %d, want %d, body = %s", putResp.StatusCode, http.StatusOK, string(body))
	}

	complete := env.requestJSON(t, http.MethodPost, "/v1/uploads/"+mediaFileID+"/complete", map[string]any{
		"etag": putResp.Header.Get("ETag"),
	})
	assertStatus(t, complete, http.StatusOK)

	attach := env.requestJSON(t, http.MethodPatch, "/v1/employer/companies/"+companyID, map[string]any{
		"logoMediaId": mediaFileID,
	})
	assertStatus(t, attach, http.StatusOK)

	deleteWhileAttached := env.requestJSON(t, http.MethodDelete, "/v1/media/"+mediaFileID, nil)
	assertStatus(t, deleteWhileAttached, http.StatusConflict)

	otherClient := env.newSessionClient(t)
	registerApplicant := env.requestJSONWithClient(t, otherClient, http.MethodPost, "/v1/auth/register/applicant", map[string]any{
		"email":       fmt.Sprintf("media-viewer-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Media Viewer",
		"firstName":   "Media",
		"lastName":    "Viewer",
	})
	assertStatus(t, registerApplicant, http.StatusCreated)

	metadata := env.requestJSONWithClient(t, otherClient, http.MethodGet, "/v1/media/"+mediaFileID, nil)
	assertStatus(t, metadata, http.StatusOK)

	anonymousClient := env.newSessionClient(t)
	download := env.requestJSONWithClient(t, anonymousClient, http.MethodPost, "/v1/media/"+mediaFileID+"/download-url", nil)
	assertStatus(t, download, http.StatusOK)

	downloadReq, err := http.NewRequest(http.MethodGet, stringField(t, download.body, "downloadUrl"), nil)
	if err != nil {
		t.Fatalf("build download request: %v", err)
	}
	downloadResp, err := anonymousClient.Do(downloadReq)
	if err != nil {
		t.Fatalf("download object: %v", err)
	}
	defer downloadResp.Body.Close()
	if downloadResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(downloadResp.Body)
		t.Fatalf("download status = %d, want %d, body = %s", downloadResp.StatusCode, http.StatusOK, string(body))
	}
	body, err := io.ReadAll(downloadResp.Body)
	if err != nil {
		t.Fatalf("read download body: %v", err)
	}
	if string(body) != string(content) {
		t.Fatalf("download body = %q, want %q", string(body), string(content))
	}
}
