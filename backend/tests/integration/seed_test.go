//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"tramplin/backend/internal/config"
	"tramplin/backend/internal/devseed"
)

func TestDevSeederIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	summary, err := devseed.Seed(context.Background(), env.db, config.Config{})
	if err != nil {
		t.Fatalf("run seed: %v", err)
	}
	if _, err := devseed.Seed(context.Background(), env.db, config.Config{}); err != nil {
		t.Fatalf("rerun seed: %v", err)
	}

	assertDBCount(t, env, `SELECT count(*) FROM companies WHERE id = $1`, 1, "11111111-1111-1111-1111-111111111401")
	assertDBCount(t, env, `SELECT count(*) FROM opportunities WHERE id = ANY($1::uuid[])`, 3, []string{
		"11111111-1111-1111-1111-111111111601",
		"11111111-1111-1111-1111-111111111602",
		"11111111-1111-1111-1111-111111111603",
	})
	assertDBCount(t, env, `SELECT count(*) FROM applications WHERE id = ANY($1::uuid[])`, 2, []string{
		"11111111-1111-1111-1111-111111111801",
		"11111111-1111-1111-1111-111111111802",
	})
	assertDBCount(t, env, `SELECT count(*) FROM applicant_connections WHERE id = ANY($1::uuid[])`, 2, []string{
		"11111111-1111-1111-1111-111111111901",
		"11111111-1111-1111-1111-111111111902",
	})
	assertDBCount(t, env, `SELECT count(*) FROM notification_campaigns WHERE id = $1`, 1, "11111111-1111-1111-1111-111111112101")
	assertDBCount(t, env, `SELECT count(*) FROM verification_requests WHERE id = $1`, 1, "11111111-1111-1111-1111-111111112201")
	assertDBCount(t, env, `SELECT count(*) FROM moderation_cases WHERE id = $1`, 1, "11111111-1111-1111-1111-111111112301")

	applicantClient := env.newSessionClient(t)
	login := env.requestJSONWithClient(t, applicantClient, http.MethodPost, "/v1/auth/login", map[string]any{
		"email":    summary.ApplicantEmails[0],
		"password": summary.Password,
	})
	assertStatus(t, login, http.StatusOK)
	assertEqual(t, objectStringField(t, login.body, "user", "role"), "applicant")

	me := env.requestJSONWithClient(t, applicantClient, http.MethodGet, "/v1/auth/me", nil)
	assertStatus(t, me, http.StatusOK)
	assertEqual(t, stringField(t, me.body, "email"), summary.ApplicantEmails[0])

	applications := env.requestJSONWithClient(t, applicantClient, http.MethodGet, "/v1/me/applications?page=1&pageSize=20", nil)
	assertStatus(t, applications, http.StatusOK)
	assertCount(t, applications.body, "items", 1)

	notifications := env.requestJSONWithClient(t, applicantClient, http.MethodGet, "/v1/notifications?page=1&pageSize=20", nil)
	assertStatus(t, notifications, http.StatusOK)
	assertCount(t, notifications.body, "items", 2)

	publicCompanies := env.requestJSON(t, http.MethodGet, "/v1/public/companies?page=1&pageSize=20", nil)
	assertStatus(t, publicCompanies, http.StatusOK)
	assertCount(t, publicCompanies.body, "items", 1)

	publicOpportunities := env.requestJSON(t, http.MethodGet, "/v1/public/opportunities?page=1&pageSize=20", nil)
	assertStatus(t, publicOpportunities, http.StatusOK)
	assertCount(t, publicOpportunities.body, "items", 2)
}

func assertDBCount(t *testing.T, env *integrationEnv, query string, expected int, args ...any) {
	t.Helper()

	var actual int
	if err := env.db.QueryRow(context.Background(), query, args...).Scan(&actual); err != nil {
		t.Fatalf("query count: %v", err)
	}
	if actual != expected {
		t.Fatalf("unexpected row count: got %d want %d query=%q", actual, expected, query)
	}
}
