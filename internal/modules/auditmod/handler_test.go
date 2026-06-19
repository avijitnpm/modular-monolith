package auditmod

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/internal/repository"
)

type fakeAuditService struct {
	logs []repository.AuditLog
	err  error
}

func (f *fakeAuditService) List(_ context.Context, _ string, _, _ int) ([]repository.AuditLog, error) {
	return f.logs, f.err
}

func TestListAuditLogsReturnsEvents(t *testing.T) {
	entityID := "sub-1"
	svc := &fakeAuditService{
		logs: []repository.AuditLog{
			{
				ID:         "log-1",
				Action:     "subscription.created",
				EntityType: "subscription",
				EntityID:   &entityID,
				CreatedAt:  "2026-01-01T00:00:00Z",
				Metadata:   map[string]string{"plan": "pro"},
			},
		},
	}
	handler := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	ctx := appcontext.SetOrganizationID(req.Context(), "org-1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ListAuditLogs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Data []auditLogResponse `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 event, got %d", len(resp.Data))
	}

	if resp.Data[0].Action != "subscription.created" {
		t.Errorf("expected action subscription.created, got %q", resp.Data[0].Action)
	}

	if resp.Data[0].Metadata["plan"] != "pro" {
		t.Errorf("expected metadata plan=pro, got %q", resp.Data[0].Metadata["plan"])
	}
}

func TestListAuditLogsRejectsMissingOrganization(t *testing.T) {
	handler := NewHandler(&fakeAuditService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	rec := httptest.NewRecorder()

	handler.ListAuditLogs(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestListAuditLogsPagination(t *testing.T) {
	var captured struct {
		limit, offset int
	}

	svc := &paginationCapture{captureLimit: &captured.limit, captureOffset: &captured.offset}
	handler := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit?limit=10&offset=20", nil)
	ctx := appcontext.SetOrganizationID(req.Context(), "org-1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ListAuditLogs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if captured.limit != 10 {
		t.Errorf("expected limit=10, got %d", captured.limit)
	}

	if captured.offset != 20 {
		t.Errorf("expected offset=20, got %d", captured.offset)
	}
}

func TestListAuditLogsDefaultPagination(t *testing.T) {
	var captured struct {
		limit, offset int
	}

	svc := &paginationCapture{captureLimit: &captured.limit, captureOffset: &captured.offset}
	handler := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	ctx := appcontext.SetOrganizationID(req.Context(), "org-1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ListAuditLogs(rec, req)

	if captured.limit != 50 {
		t.Errorf("expected default limit=50, got %d", captured.limit)
	}

	if captured.offset != 0 {
		t.Errorf("expected default offset=0, got %d", captured.offset)
	}
}

type paginationCapture struct {
	captureLimit  *int
	captureOffset *int
}

func (p *paginationCapture) List(_ context.Context, _ string, limit int, offset int) ([]repository.AuditLog, error) {
	*p.captureLimit = limit
	*p.captureOffset = offset
	return nil, nil
}
