package entitlements

import (
	"context"
	"testing"
)

type mockSubscriptions struct {
	plan   string
	status string
}

func (m *mockSubscriptions) GetSubscription(ctx context.Context, organizationID string) (string, string, error) {
	return m.plan, m.status, nil
}

type mockUsage struct {
	values map[string]int64
}

func (m *mockUsage) GetUsage(ctx context.Context, organizationID string, metric string) (int64, error) {
	return m.values[metric], nil
}

func newService(plan, status string, usage map[string]int64) *Service {
	if usage == nil {
		usage = map[string]int64{}
	}
	return NewService(
		&mockSubscriptions{plan: plan, status: status},
		&mockUsage{values: usage},
	)
}

func TestFreePlanBelowLimit(t *testing.T) {
	svc := newService("free", "active", map[string]int64{"users": 0})
	e, err := svc.CanUse(context.Background(), "org-1", "users", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !e.Allowed {
		t.Fatal("expected allowed")
	}
	if e.Limit != 1 {
		t.Fatalf("expected limit 1, got %d", e.Limit)
	}
	if e.Remaining != 1 {
		t.Fatalf("expected remaining 1, got %d", e.Remaining)
	}
}

func TestFreePlanAtLimit(t *testing.T) {
	svc := newService("free", "active", map[string]int64{"documents": 50})
	e, err := svc.CanUse(context.Background(), "org-1", "documents", 1)
	if err != nil {
		t.Fatal(err)
	}
	if e.Allowed {
		t.Fatal("expected not allowed at limit")
	}
	if e.Remaining != 0 {
		t.Fatalf("expected remaining 0, got %d", e.Remaining)
	}
}

func TestFreePlanExceedingLimit(t *testing.T) {
	svc := newService("free", "active", map[string]int64{"api_requests": 1001})
	e, err := svc.CanUse(context.Background(), "org-1", "api_requests", 1)
	if err != nil {
		t.Fatal(err)
	}
	if e.Allowed {
		t.Fatal("expected not allowed over limit")
	}
	if e.Remaining != 0 {
		t.Fatalf("expected remaining 0, got %d", e.Remaining)
	}
}

func TestProPlanLimits(t *testing.T) {
	svc := newService("pro", "active", map[string]int64{"users": 9})
	e, err := svc.CanUse(context.Background(), "org-1", "users", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !e.Allowed {
		t.Fatal("expected allowed under pro limit")
	}
	if e.Limit != 10 {
		t.Fatalf("expected limit 10, got %d", e.Limit)
	}
}

func TestProPlanAtLimit(t *testing.T) {
	svc := newService("pro", "active", map[string]int64{"users": 10})
	e, err := svc.CanUse(context.Background(), "org-1", "users", 1)
	if err != nil {
		t.Fatal(err)
	}
	if e.Allowed {
		t.Fatal("expected not allowed at pro limit")
	}
}

func TestEnterpriseUnlimited(t *testing.T) {
	svc := newService("enterprise", "active", map[string]int64{"users": 999999})
	e, err := svc.CanUse(context.Background(), "org-1", "users", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !e.Allowed {
		t.Fatal("expected allowed for enterprise")
	}
	if e.Limit != Unlimited {
		t.Fatalf("expected unlimited limit, got %d", e.Limit)
	}
	if e.Remaining != Unlimited {
		t.Fatalf("expected unlimited remaining, got %d", e.Remaining)
	}
}

func TestMissingSubscriptionDefaultsFree(t *testing.T) {
	svc := newService("", "", map[string]int64{"users": 1})
	e, err := svc.CanUse(context.Background(), "org-1", "users", 1)
	if err != nil {
		t.Fatal(err)
	}
	if e.Allowed {
		t.Fatal("expected not allowed - missing sub defaults to free, at limit")
	}
	if e.Limit != 1 {
		t.Fatalf("expected free limit 1, got %d", e.Limit)
	}
}

func TestCancelledSubscriptionDefaultsFree(t *testing.T) {
	svc := newService("pro", "cancelled", map[string]int64{"users": 5})
	e, err := svc.CanUse(context.Background(), "org-1", "users", 1)
	if err != nil {
		t.Fatal(err)
	}
	// Cancelled pro should default to free (limit 1), so 5 used > 1 limit
	if e.Allowed {
		t.Fatal("expected not allowed - cancelled defaults to free")
	}
	if e.Limit != 1 {
		t.Fatalf("expected free limit 1, got %d", e.Limit)
	}
}

func TestTrialingSubscription(t *testing.T) {
	svc := newService("pro", "trialing", map[string]int64{"users": 5})
	e, err := svc.CanUse(context.Background(), "org-1", "users", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !e.Allowed {
		t.Fatal("expected allowed - trialing uses plan's limits")
	}
	if e.Limit != 10 {
		t.Fatalf("expected pro limit 10, got %d", e.Limit)
	}
}

func TestRemainingCalculation(t *testing.T) {
	svc := newService("pro", "active", map[string]int64{"documents": 3000})
	e, err := svc.CanUse(context.Background(), "org-1", "documents", 1)
	if err != nil {
		t.Fatal(err)
	}
	if e.Remaining != 2000 {
		t.Fatalf("expected remaining 2000, got %d", e.Remaining)
	}
}

func TestGetEntitlements(t *testing.T) {
	svc := newService("free", "active", map[string]int64{"users": 0, "documents": 10})
	ents, err := svc.GetEntitlements(context.Background(), "org-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 4 {
		t.Fatalf("expected 4 entitlements, got %d", len(ents))
	}

	// Check documents entitlement
	var docs Entitlement
	for _, e := range ents {
		if e.Metric == "documents" {
			docs = e
			break
		}
	}
	if docs.Limit != 50 {
		t.Fatalf("expected documents limit 50, got %d", docs.Limit)
	}
	if docs.Used != 10 {
		t.Fatalf("expected documents used 10, got %d", docs.Used)
	}
	if docs.Remaining != 40 {
		t.Fatalf("expected documents remaining 40, got %d", docs.Remaining)
	}
	if !docs.Allowed {
		t.Fatal("expected documents allowed")
	}
}
