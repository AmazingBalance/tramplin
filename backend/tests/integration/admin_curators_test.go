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

func TestAdminCuratorManagementIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	adminClient := env.newSessionClient(t)
	adminID := seedAdminCuratorSession(t, env, adminClient, fmt.Sprintf("admin-curator-%d@example.com", time.Now().UnixNano()))

	listInitial := env.requestJSONWithClient(t, adminClient, http.MethodGet, "/v1/curator/admin/curators", nil)
	assertStatus(t, listInitial, http.StatusOK)
	assertCount(t, listInitial.body, "items", 1)
	assertEqual(t, nestedStringField(t, listInitial.body, "items", 0, "id"), adminID)
	assertEqual(t, nestedStringField(t, listInitial.body, "items", 0, "role"), "curator")
	assertNestedBoolValue(t, listInitial.body, "items", 0, "isAdmin", true)

	createCurator := env.requestJSONWithClient(t, adminClient, http.MethodPost, "/v1/curator/admin/curators", map[string]any{
		"email":       fmt.Sprintf("managed-curator-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Managed Curator",
		"fullName":    "Managed Curator Full",
		"reason":      "Expand moderation team",
	})
	assertStatus(t, createCurator, http.StatusCreated)
	curatorID := stringField(t, createCurator.body, "id")
	assertEqual(t, stringField(t, createCurator.body, "role"), "curator")
	assertEqual(t, stringField(t, createCurator.body, "fullName"), "Managed Curator Full")
	assertBoolValue(t, createCurator.body, "isAdmin", false)
	assertObjectBoolValue(t, createCurator.body, "curatorProfile", "isAdmin", false)

	listAfterCreate := env.requestJSONWithClient(t, adminClient, http.MethodGet, "/v1/curator/admin/curators?q=Managed", nil)
	assertStatus(t, listAfterCreate, http.StatusOK)
	assertCount(t, listAfterCreate.body, "items", 1)
	assertEqual(t, nestedStringField(t, listAfterCreate.body, "items", 0, "id"), curatorID)

	patchCurator := env.requestJSONWithClient(t, adminClient, http.MethodPatch, "/v1/curator/admin/curators/"+curatorID, map[string]any{
		"displayName": "Managed Curator Updated",
		"fullName":    "Managed Curator Updated Full",
		"isActive":    false,
		"reason":      "Deactivate unused curator account",
	})
	assertStatus(t, patchCurator, http.StatusOK)
	assertEqual(t, stringField(t, patchCurator.body, "displayName"), "Managed Curator Updated")
	assertEqual(t, stringField(t, patchCurator.body, "fullName"), "Managed Curator Updated Full")
	assertBoolValue(t, patchCurator.body, "isActive", false)
}

func TestAdminCuratorInvariantIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	adminClient := env.newSessionClient(t)
	memberClient := env.newSessionClient(t)

	adminID := seedAdminCuratorSession(t, env, adminClient, fmt.Sprintf("invariant-admin-%d@example.com", time.Now().UnixNano()))

	createCurator := env.requestJSONWithClient(t, adminClient, http.MethodPost, "/v1/curator/admin/curators", map[string]any{
		"email":       fmt.Sprintf("non-admin-curator-%d@example.com", time.Now().UnixNano()),
		"password":    "password123",
		"displayName": "Team Curator",
		"fullName":    "Team Curator Full",
		"reason":      "Need another curator",
	})
	assertStatus(t, createCurator, http.StatusCreated)

	loginMember := env.requestJSONWithClient(t, memberClient, http.MethodPost, "/v1/auth/login", map[string]any{
		"email":    stringField(t, createCurator.body, "email"),
		"password": "password123",
	})
	assertStatus(t, loginMember, http.StatusOK)

	nonAdminList := env.requestJSONWithClient(t, memberClient, http.MethodGet, "/v1/curator/admin/curators", nil)
	assertStatus(t, nonAdminList, http.StatusForbidden)

	deactivateAdmin := env.requestJSONWithClient(t, adminClient, http.MethodPatch, "/v1/curator/admin/curators/"+adminID, map[string]any{
		"isActive": false,
		"reason":   "Should be rejected",
	})
	assertStatus(t, deactivateAdmin, http.StatusConflict)
	assertEqual(t, stringField(t, deactivateAdmin.body, "code"), "conflict")
}

func seedAdminCuratorSession(t *testing.T, env *integrationEnv, client *http.Client, email string) string {
	t.Helper()

	passwordHash, err := authplatform.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash admin curator password: %v", err)
	}

	userID := "22222222-2222-4222-8222-222222222222"
	now := time.Now().UTC()

	env.execSQL(t, `
		INSERT INTO users (
			id, email, password_hash, display_name, role, is_active, token_version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, TRUE, 0, $6, $6)
	`, userID, email, passwordHash, "Integration Admin Curator", model.UserRoleCurator, now)
	env.execSQL(t, `
		INSERT INTO curator_profiles (user_id, full_name, is_admin, created_at)
		VALUES ($1, $2, TRUE, $3)
	`, userID, "Integration Admin Curator", now)

	login := env.requestJSONWithClient(t, client, http.MethodPost, "/v1/auth/login", map[string]any{
		"email":    email,
		"password": "password123",
	})
	assertStatus(t, login, http.StatusOK)
	return userID
}
