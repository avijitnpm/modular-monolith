package organizations

type OrganizationResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
}

func organizationResponse(
	id string,
	organizationID string,
	name string,
) OrganizationResponse {

	return OrganizationResponse{
		ID:             id,
		OrganizationID: organizationID,
		Name:           name,
	}
}
