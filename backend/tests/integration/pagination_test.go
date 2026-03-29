//go:build integration

package integration

import (
	"net/http"
	"testing"
)

func TestPaginationValidationIntegration(t *testing.T) {
	env := newIntegrationEnv(t)

	publicCompanies := env.requestJSON(t, http.MethodGet, "/v1/public/companies?page=0", nil)
	assertStatus(t, publicCompanies, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, publicCompanies.body, "code"), "validation_error")
	assertEqual(t, stringField(t, publicCompanies.body, "message"), "invalid page parameter")

	publicOpportunities := env.requestJSON(t, http.MethodGet, "/v1/public/opportunities?pageSize=101", nil)
	assertStatus(t, publicOpportunities, http.StatusUnprocessableEntity)
	assertEqual(t, stringField(t, publicOpportunities.body, "code"), "validation_error")
	assertEqual(t, stringField(t, publicOpportunities.body, "message"), "invalid pageSize parameter")
}
