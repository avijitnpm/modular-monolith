package context

import "context"

type Membership struct {
	MembershipID   string
	OrganizationID string
}

const membershipContextKey ContextKey = "membership"

func SetMembership(ctx context.Context, m *Membership) context.Context {
	return context.WithValue(ctx, membershipContextKey, m)
}

func GetMembership(ctx context.Context) (*Membership, bool) {
	m, ok := ctx.Value(membershipContextKey).(*Membership)
	return m, ok
}
