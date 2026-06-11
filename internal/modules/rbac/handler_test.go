package rbac

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBootstrapRBACBootstrapsOrganization(t *testing.T) {
	store := &fakeStore{}
	handler := NewHandler(
		NewService(
			store,
			&fakeAuditLogger{},
		),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/bootstrap-rbac",
		strings.NewReader(`{"organization_id":" test-org-1 "}`),
	)
	recorder := httptest.NewRecorder()

	handler.BootstrapRBAC(
		recorder,
		req,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	if store.bootstrapOrganizationID != "test-org-1" {
		t.Fatalf(
			"expected organization test-org-1, got %q",
			store.bootstrapOrganizationID,
		)
	}

	if !strings.Contains(
		recorder.Body.String(),
		`"bootstrapped":true`,
	) {
		t.Fatalf("expected bootstrap response, got %s", recorder.Body.String())
	}
}

func TestBootstrapRBACRejectsInvalidBody(t *testing.T) {
	handler := NewHandler(
		NewService(
			&fakeStore{},
			&fakeAuditLogger{},
		),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/bootstrap-rbac",
		strings.NewReader(`{`),
	)
	recorder := httptest.NewRecorder()

	handler.BootstrapRBAC(
		recorder,
		req,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestBootstrapRBACRejectsMissingOrganizationID(t *testing.T) {
	handler := NewHandler(
		NewService(
			&fakeStore{},
			&fakeAuditLogger{},
		),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/bootstrap-rbac",
		strings.NewReader(`{"organization_id":" "}`),
	)
	recorder := httptest.NewRecorder()

	handler.BootstrapRBAC(
		recorder,
		req,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestBootstrapRBACReturnsInternalError(t *testing.T) {
	handler := NewHandler(
		NewService(
			&fakeStore{
				bootstrapErr: errors.New("bootstrap failed"),
			},
			&fakeAuditLogger{},
		),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/bootstrap-rbac",
		strings.NewReader(`{"organization_id":"test-org-1"}`),
	)
	recorder := httptest.NewRecorder()

	handler.BootstrapRBAC(
		recorder,
		req,
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", recorder.Code)
	}
}
