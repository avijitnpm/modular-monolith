package organizations

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avijitnpm/modular-monolith/internal/repository"
)

func TestCreateOrganizationReturnsCreatedOrganization(t *testing.T) {
	handler := NewHandler(
		&fakeService{
			organization: &repository.Organization{
				ID:             "org-row-1",
				OrganizationID: "test-org-1",
				Name:           "Acme Inc",
			},
		},
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/organizations",
		strings.NewReader(`{"zitadel_org_id":"test-org-1","name":"Acme Inc"}`),
	)
	rec := httptest.NewRecorder()

	handler.CreateOrganization(
		rec,
		req,
	)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected created, got %d", rec.Code)
	}

	body := rec.Body.String()

	if !strings.Contains(body, `"id":"org-row-1"`) {
		t.Fatalf("expected id in response, got %s", body)
	}

	if !strings.Contains(body, `"organization_id":"test-org-1"`) {
		t.Fatalf("expected organization_id in response, got %s", body)
	}

	if !strings.Contains(body, `"name":"Acme Inc"`) {
		t.Fatalf("expected name in response, got %s", body)
	}
}

func TestCreateOrganizationRejectsInvalidBody(t *testing.T) {
	handler := NewHandler(
		&fakeService{},
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/organizations",
		strings.NewReader(`{`),
	)
	rec := httptest.NewRecorder()

	handler.CreateOrganization(
		rec,
		req,
	)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", rec.Code)
	}
}

func TestCreateOrganizationRejectsValidationErrors(t *testing.T) {
	handler := NewHandler(
		&fakeService{},
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/organizations",
		strings.NewReader(`{"zitadel_org_id":"","name":""}`),
	)
	rec := httptest.NewRecorder()

	handler.CreateOrganization(
		rec,
		req,
	)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", rec.Code)
	}

	body := rec.Body.String()

	if !strings.Contains(body, `"validation failed"`) {
		t.Fatalf("expected validation failure, got %s", body)
	}
}

func TestCreateOrganizationReturnsInternalError(t *testing.T) {
	handler := NewHandler(
		&fakeService{
			err: errors.New("boom"),
		},
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/organizations",
		strings.NewReader(`{"zitadel_org_id":"test-org-1","name":"Acme Inc"}`),
	)
	rec := httptest.NewRecorder()

	handler.CreateOrganization(
		rec,
		req,
	)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected internal server error, got %d", rec.Code)
	}
}

type fakeService struct {
	organization *repository.Organization
	err          error
}

func (f *fakeService) RegisterOrganization(
	context.Context,
	string,
	string,
) (*repository.Organization, error) {

	if f.err != nil {
		return nil, f.err
	}

	return f.organization, nil
}
