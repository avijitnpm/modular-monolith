package usage

import (
	"context"
	"testing"
)

type mockStore struct {
	lastMetric string
	lastAmount int64
}

func (m *mockStore) GetUsage(ctx context.Context, organizationID string, metric string) (*UsageCounter, error) {
	return &UsageCounter{Metric: metric, Value: 10}, nil
}

func (m *mockStore) IncrementUsage(ctx context.Context, organizationID string, metric string, amount int64) (*UsageCounter, error) {
	m.lastMetric = metric
	m.lastAmount = amount
	return &UsageCounter{Metric: metric, Value: amount}, nil
}

func (m *mockStore) SetUsage(ctx context.Context, organizationID string, metric string, value int64) (*UsageCounter, error) {
	return &UsageCounter{Metric: metric, Value: value}, nil
}

func (m *mockStore) ListUsage(ctx context.Context, organizationID string) ([]UsageCounter, error) {
	return []UsageCounter{{Metric: "users", Value: 5}}, nil
}

func TestServiceIncrementMetric(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)

	counter, err := svc.IncrementMetric(context.Background(), "org-1", "custom", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.lastMetric != "custom" || store.lastAmount != 5 {
		t.Fatalf("expected metric=custom amount=5, got %s %d", store.lastMetric, store.lastAmount)
	}
	if counter.Value != 5 {
		t.Fatalf("expected value 5, got %d", counter.Value)
	}
}

func TestServiceIncrementUsers(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)

	_, err := svc.IncrementUsers(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.lastMetric != "users" || store.lastAmount != 1 {
		t.Fatalf("expected metric=users amount=1, got %s %d", store.lastMetric, store.lastAmount)
	}
}

func TestServiceIncrementDocuments(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)

	_, err := svc.IncrementDocuments(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.lastMetric != "documents" || store.lastAmount != 1 {
		t.Fatalf("expected metric=documents amount=1, got %s %d", store.lastMetric, store.lastAmount)
	}
}

func TestServiceIncrementAPIRequests(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)

	_, err := svc.IncrementAPIRequests(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.lastMetric != "api_requests" || store.lastAmount != 1 {
		t.Fatalf("expected metric=api_requests amount=1, got %s %d", store.lastMetric, store.lastAmount)
	}
}

func TestServiceIncrementStorage(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)

	_, err := svc.IncrementStorage(context.Background(), "org-1", 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.lastMetric != "storage" || store.lastAmount != 1024 {
		t.Fatalf("expected metric=storage amount=1024, got %s %d", store.lastMetric, store.lastAmount)
	}
}
