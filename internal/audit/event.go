package audit

type Event struct {
	OrganizationID string

	UserID string

	Action string

	EntityType string

	EntityID string
}
