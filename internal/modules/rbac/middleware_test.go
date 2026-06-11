package rbac

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
)

func TestRequirePermissionAllowsRequest(t *testing.T) {
	checker := &fakePermissionChecker{
		allowed: true,
	}
	called := false
	handler := RequirePermission(
		checker,
		"settings.write",
	)(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		}),
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		requestWithIdentity(),
	)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}

	if checker.organizationID != "org-1" {
		t.Fatalf("expected organization id org-1, got %q", checker.organizationID)
	}

	if checker.userID != "zitadel-user-1" {
		t.Fatalf("expected user id zitadel-user-1, got %q", checker.userID)
	}

	if checker.permission != "settings.write" {
		t.Fatalf("expected permission settings.write, got %q", checker.permission)
	}
}

func TestRequirePermissionRejectsMissingPermission(t *testing.T) {
	handler := RequirePermission(
		&fakePermissionChecker{},
		"settings.write",
	)(
		http.HandlerFunc(func(
			http.ResponseWriter,
			*http.Request,
		) {
			t.Fatal("next handler should not be called")
		}),
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		requestWithIdentity(),
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestRequirePermissionRejectsMissingAuthenticatedUser(t *testing.T) {
	handler := RequirePermission(
		&fakePermissionChecker{},
		"settings.write",
	)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)
	req = req.WithContext(
		appcontext.SetOrganizationID(
			req.Context(),
			"org-1",
		),
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		req,
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
}

func TestRequirePermissionRejectsMissingTenantContext(t *testing.T) {
	handler := RequirePermission(
		&fakePermissionChecker{},
		"settings.write",
	)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)
	req = req.WithContext(
		appcontext.SetAuthenticatedUser(
			req.Context(),
			&appcontext.AuthenticatedUser{
				UserID: "zitadel-user-1",
			},
		),
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		req,
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
}

func TestRequirePermissionReturnsInternalErrorWhenCheckerFails(t *testing.T) {
	handler := RequirePermission(
		&fakePermissionChecker{
			err: errors.New("database unavailable"),
		},
		"settings.write",
	)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		requestWithIdentity(),
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
}

func requestWithIdentity() *http.Request {
	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)
	ctx := appcontext.SetAuthenticatedUser(
		req.Context(),
		&appcontext.AuthenticatedUser{
			UserID:         "zitadel-user-1",
			OrganizationID: "org-1",
			Email:          "user@example.com",
		},
	)
	ctx = appcontext.SetOrganizationID(
		ctx,
		"org-1",
	)

	return req.WithContext(ctx)
}

type fakePermissionChecker struct {
	allowed bool
	err     error

	organizationID string
	userID         string
	permission     string
}

func (f *fakePermissionChecker) UserHasPermission(
	_ context.Context,
	organizationID string,
	userID string,
	permission string,
) (bool, error) {

	f.organizationID = organizationID
	f.userID = userID
	f.permission = permission

	return f.allowed, f.err
}
