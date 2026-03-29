//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tramplin/backend/internal/config"
	httptransport "tramplin/backend/internal/transport/http"
)

func TestOpportunityLifecycleIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	register := env.requestJSON(t, http.MethodPost, "/v1/auth/register/employer", map[string]any{
		"email":       fmt.Sprintf("opps-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Integration Employer",
		"fullName":    "Integration Employer",
	})
	assertStatus(t, register, http.StatusCreated)

	company := env.requestJSON(t, http.MethodPost, "/v1/employer/companies", map[string]any{
		"legalName": "Integration Company",
		"slug":      fmt.Sprintf("integration-company-%d", time.Now().UnixNano()),
	})
	assertStatus(t, company, http.StatusCreated)
	companyID := stringField(t, company.body, "id")

	createOpportunity := env.requestJSON(t, http.MethodPost, "/v1/employer/opportunities", map[string]any{
		"companyId":           companyID,
		"title":               "Go Internship",
		"summary":             "Backend internship",
		"slug":                fmt.Sprintf("go-internship-%d", time.Now().UnixNano()),
		"description":         "Build backend APIs",
		"type":                "internship",
		"participationFormat": "remote",
		"location": map[string]any{
			"precision": "city_only",
			"country":   "Russia",
			"city":      "Omsk",
		},
		"vacancyDetails": map[string]any{
			"employmentType":  "full_time",
			"experienceLevel": "junior",
			"salaryFrom":      1000,
			"salaryTo":        1500,
			"currency":        "USD",
		},
		"links": []map[string]any{
			{
				"linkType": "apply",
				"title":    "Apply",
				"url":      "https://example.com/apply",
			},
		},
	})
	assertStatus(t, createOpportunity, http.StatusCreated)
	opportunityID := stringField(t, createOpportunity.body, "id")
	slug := stringField(t, createOpportunity.body, "slug")

	listEmployer := env.requestJSON(t, http.MethodGet, "/v1/employer/opportunities", nil)
	assertStatus(t, listEmployer, http.StatusOK)
	assertCount(t, listEmployer.body, "items", 1)
	assertEqual(t, nestedStringField(t, listEmployer.body, "items", 0, "status"), "draft")

	patchOpportunity := env.requestJSON(t, http.MethodPatch, "/v1/employer/opportunities/"+opportunityID, map[string]any{
		"summary": "Backend internship updated",
		"status":  "planned",
	})
	assertStatus(t, patchOpportunity, http.StatusOK)
	assertEqual(t, stringField(t, patchOpportunity.body, "status"), "planned")
	assertEqual(t, stringField(t, patchOpportunity.body, "summary"), "Backend internship updated")

	publicBeforeApproval := env.requestJSON(t, http.MethodGet, "/v1/public/opportunities", nil)
	assertStatus(t, publicBeforeApproval, http.StatusOK)
	assertCount(t, publicBeforeApproval.body, "items", 0)

	env.execSQL(t, `UPDATE opportunities SET moderation_status = 'approved' WHERE id = $1`, opportunityID)

	publicList := env.requestJSON(t, http.MethodGet, "/v1/public/opportunities?city=Omsk", nil)
	assertStatus(t, publicList, http.StatusOK)
	assertCount(t, publicList.body, "items", 1)
	assertEqual(t, nestedStringField(t, publicList.body, "items", 0, "id"), opportunityID)

	publicDetail := env.requestJSON(t, http.MethodGet, "/v1/public/opportunities/"+opportunityID, nil)
	assertStatus(t, publicDetail, http.StatusOK)
	assertEqual(t, stringField(t, publicDetail.body, "slug"), slug)
	assertEqual(t, stringField(t, publicDetail.body, "summary"), "Backend internship updated")

	publicBySlug := env.requestJSON(t, http.MethodGet, "/v1/public/opportunities/by-slug/"+slug, nil)
	assertStatus(t, publicBySlug, http.StatusOK)
	assertEqual(t, stringField(t, publicBySlug.body, "id"), opportunityID)
}

func TestPublicOpportunityQueryValidationIntegration(t *testing.T) {
	env := newIntegrationEnv(t)
	response := env.requestJSON(t, http.MethodGet, "/v1/public/opportunities?view=map&lat=55.75&lng=37.62&radiusKm=10", nil)

	assertStatus(t, response, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, response.body, "code"), "validation_error")
	assertEqual(t, stringField(t, response.body, "message"), "map geo filters are not implemented yet")
}

type integrationEnv struct {
	server *httptest.Server
	client *http.Client
	db     *pgxpool.Pool
	admin  *pgxpool.Pool
	schema string
}

type jsonResponse struct {
	status int
	body   map[string]any
}

func newIntegrationEnv(t *testing.T) *integrationEnv {
	t.Helper()

	backendRoot := backendRoot(t)
	loadDotEnvIfPresent(t, filepath.Join(backendRoot, ".env"))

	cfg := config.Load()
	ctx := context.Background()

	admin, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("open admin pool: %v", err)
	}

	schema := fmt.Sprintf("itest_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+quoteIdent(schema)); err != nil {
		admin.Close()
		t.Fatalf("create schema %s: %v", schema, err)
	}

	testDB, err := openSchemaPool(ctx, cfg.DatabaseURL, schema)
	if err != nil {
		admin.Exec(ctx, `DROP SCHEMA IF EXISTS `+quoteIdent(schema)+` CASCADE`)
		admin.Close()
		t.Fatalf("open schema pool: %v", err)
	}

	applyMigrations(t, testDB, filepath.Join(backendRoot, "db", "migrations"))

	server := httptest.NewServer(httptransport.NewServer(cfg, testDB))
	jar, err := cookiejar.New(nil)
	if err != nil {
		server.Close()
		testDB.Close()
		admin.Exec(ctx, `DROP SCHEMA IF EXISTS `+quoteIdent(schema)+` CASCADE`)
		admin.Close()
		t.Fatalf("create cookie jar: %v", err)
	}

	client := server.Client()
	client.Jar = jar

	env := &integrationEnv{
		server: server,
		client: client,
		db:     testDB,
		admin:  admin,
		schema: schema,
	}

	t.Cleanup(func() {
		server.Close()
		testDB.Close()
		admin.Exec(context.Background(), `DROP SCHEMA IF EXISTS `+quoteIdent(schema)+` CASCADE`)
		admin.Close()
	})

	return env
}

func (e *integrationEnv) requestJSON(t *testing.T, method, path string, payload any) jsonResponse {
	t.Helper()

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, e.server.URL+path, body)
	if err != nil {
		t.Fatalf("build request %s %s: %v", method, path, err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatalf("do request %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	result := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("decode response body: %v\nbody: %s", err, string(data))
		}
	}

	return jsonResponse{
		status: resp.StatusCode,
		body:   result,
	}
}

func (e *integrationEnv) execSQL(t *testing.T, query string, args ...any) {
	t.Helper()

	if _, err := e.db.Exec(context.Background(), query, args...); err != nil {
		t.Fatalf("exec sql: %v", err)
	}
}

func openSchemaPool(ctx context.Context, databaseURL, schema string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	return pgxpool.NewWithConfig(ctx, cfg)
}

func applyMigrations(t *testing.T, db *pgxpool.Pool, migrationsDir string) {
	t.Helper()

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}

	names := make([]string, 0, len(files))
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			names = append(names, file.Name())
		}
	}
	slices.Sort(names)

	for _, name := range names {
		path := filepath.Join(migrationsDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		sql := strings.ReplaceAll(string(content), "CREATE EXTENSION IF NOT EXISTS pgcrypto;\n", "")
		if _, err := db.Exec(context.Background(), sql); err != nil {
			t.Fatalf("apply migration %s: %v", name, err)
		}
	}
}

func backendRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func loadDotEnvIfPresent(t *testing.T, path string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("read .env: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		t.Setenv(key, value)
	}
}

func quoteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func assertStatus(t *testing.T, response jsonResponse, expected int) {
	t.Helper()
	if response.status != expected {
		t.Fatalf("unexpected status: got %d want %d body=%v", response.status, expected, response.body)
	}
}

func assertCount(t *testing.T, body map[string]any, field string, expected int) {
	t.Helper()

	items, ok := body[field].([]any)
	if !ok {
		t.Fatalf("field %q is not a JSON array: %v", field, body[field])
	}
	if len(items) != expected {
		t.Fatalf("unexpected %s length: got %d want %d", field, len(items), expected)
	}
}

func assertEqual(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("unexpected value: got %q want %q", got, want)
	}
}

func stringField(t *testing.T, body map[string]any, field string) string {
	t.Helper()

	value, ok := body[field].(string)
	if !ok {
		t.Fatalf("field %q is not a string: %v", field, body[field])
	}
	return value
}

func nestedStringField(t *testing.T, body map[string]any, arrayField string, index int, nestedField string) string {
	t.Helper()

	items, ok := body[arrayField].([]any)
	if !ok || index >= len(items) {
		t.Fatalf("array field %q missing index %d", arrayField, index)
	}
	item, ok := items[index].(map[string]any)
	if !ok {
		t.Fatalf("array field %q item %d is not an object", arrayField, index)
	}
	value, ok := item[nestedField].(string)
	if !ok {
		t.Fatalf("nested field %q is not a string: %v", nestedField, item[nestedField])
	}
	return value
}
