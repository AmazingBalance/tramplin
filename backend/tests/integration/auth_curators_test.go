//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestCuratorAuthResponsesIncludeCuratorProfileIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	adminClient := env.newSessionClient(t)
	memberClient := env.newSessionClient(t)

	adminEmail := fmt.Sprintf("auth-admin-%d@example.com", time.Now().UnixNano())
	adminID := seedAdminCuratorSession(t, env, adminClient, adminEmail)

	adminLogin := env.requestJSONWithClient(t, adminClient, http.MethodPost, "/v1/auth/login", map[string]any{
		"email":    adminEmail,
		"password": "password123",
	})
	assertStatus(t, adminLogin, http.StatusOK)
	assertEqual(t, nestedObjectStringField(t, adminLogin.body, "user", "id"), adminID)
	assertObjectBoolValue(t, objectField(t, adminLogin.body, "user"), "curatorProfile", "isAdmin", true)

	adminMe := env.requestJSONWithClient(t, adminClient, http.MethodGet, "/v1/auth/me", nil)
	assertStatus(t, adminMe, http.StatusOK)
	assertEqual(t, stringField(t, adminMe.body, "id"), adminID)
	assertObjectBoolValue(t, adminMe.body, "curatorProfile", "isAdmin", true)

	createCurator := env.requestJSONWithClient(t, adminClient, http.MethodPost, "/v1/curator/admin/curators", map[string]any{
		"email":       fmt.Sprintf("auth-curator-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Auth Curator",
		"fullName":    "Auth Curator Full",
		"reason":      "Need another curator for auth coverage",
	})
	assertStatus(t, createCurator, http.StatusCreated)
	memberEmail := stringField(t, createCurator.body, "email")
	memberID := stringField(t, createCurator.body, "id")

	memberLogin := env.requestJSONWithClient(t, memberClient, http.MethodPost, "/v1/auth/login", map[string]any{
		"email":    memberEmail,
		"password": "password123",
	})
	assertStatus(t, memberLogin, http.StatusOK)
	assertEqual(t, nestedObjectStringField(t, memberLogin.body, "user", "id"), memberID)
	assertObjectBoolValue(t, objectField(t, memberLogin.body, "user"), "curatorProfile", "isAdmin", false)

	memberMe := env.requestJSONWithClient(t, memberClient, http.MethodGet, "/v1/auth/me", nil)
	assertStatus(t, memberMe, http.StatusOK)
	assertEqual(t, stringField(t, memberMe.body, "id"), memberID)
	assertObjectBoolValue(t, memberMe.body, "curatorProfile", "isAdmin", false)
}
