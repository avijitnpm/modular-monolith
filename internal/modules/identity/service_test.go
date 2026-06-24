package identity

import (
	"context"
	"fmt"
	"testing"
)

type mockStore struct {
	identities map[string]*Identity
}

func newMockStore() *mockStore {
	return &mockStore{identities: make(map[string]*Identity)}
}

func (m *mockStore) GetByZitadelUserID(_ context.Context, zitadelUserID string) (*Identity, error) {
	return m.identities[zitadelUserID], nil
}

func (m *mockStore) GetByEmail(_ context.Context, email string) (*Identity, error) {
	for _, id := range m.identities {
		if id.Email == email {
			return id, nil
		}
	}
	return nil, nil
}

func (m *mockStore) Create(_ context.Context, zitadelUserID, email, name string) (*Identity, error) {
	for _, id := range m.identities {
		if id.Email == email {
			return nil, fmt.Errorf("duplicate email")
		}
	}
	id := &Identity{ID: "gen-id", ZitadelUserID: zitadelUserID, Email: email, Name: name}
	m.identities[zitadelUserID] = id
	return id, nil
}

func (m *mockStore) Update(_ context.Context, zitadelUserID, email, name string) (*Identity, error) {
	id := m.identities[zitadelUserID]
	if id == nil {
		return nil, fmt.Errorf("not found")
	}
	id.Email = email
	id.Name = name
	return id, nil
}

func (m *mockStore) GetMemberships(_ context.Context, identityID string) ([]MembershipReference, error) {
	return nil, nil
}

func TestFindOrCreate_NewIdentity(t *testing.T) {
	store := newMockStore()
	svc := NewService(store)

	got, err := svc.FindOrCreateIdentity(context.Background(), "zit-1", "a@b.com", "Alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ZitadelUserID != "zit-1" || got.Email != "a@b.com" || got.Name != "Alice" {
		t.Fatalf("unexpected identity: %+v", got)
	}
}

func TestFindOrCreate_ExistingIdentity(t *testing.T) {
	store := newMockStore()
	svc := NewService(store)

	_, _ = svc.FindOrCreateIdentity(context.Background(), "zit-1", "a@b.com", "Alice")

	got, err := svc.FindOrCreateIdentity(context.Background(), "zit-1", "a@b.com", "Alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Email != "a@b.com" {
		t.Fatalf("expected same identity, got %+v", got)
	}
}

func TestFindOrCreate_UpdateEmail(t *testing.T) {
	store := newMockStore()
	svc := NewService(store)

	_, _ = svc.FindOrCreateIdentity(context.Background(), "zit-1", "old@b.com", "Alice")

	got, err := svc.FindOrCreateIdentity(context.Background(), "zit-1", "new@b.com", "Alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Email != "new@b.com" {
		t.Fatalf("expected updated email, got %s", got.Email)
	}
}

func TestFindOrCreate_UpdateName(t *testing.T) {
	store := newMockStore()
	svc := NewService(store)

	_, _ = svc.FindOrCreateIdentity(context.Background(), "zit-1", "a@b.com", "OldName")

	got, err := svc.FindOrCreateIdentity(context.Background(), "zit-1", "a@b.com", "NewName")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "NewName" {
		t.Fatalf("expected updated name, got %s", got.Name)
	}
}

func TestFindOrCreate_DuplicateEmail(t *testing.T) {
	store := newMockStore()
	svc := NewService(store)

	_, _ = svc.FindOrCreateIdentity(context.Background(), "zit-1", "dup@b.com", "Alice")

	_, err := svc.FindOrCreateIdentity(context.Background(), "zit-2", "dup@b.com", "Bob")
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}
