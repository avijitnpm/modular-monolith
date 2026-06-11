package organizations

import "github.com/avijitnpm/modular-monolith/pkg/validator"

type CreateOrganizationRequest struct {
	ZitadelOrgID string `json:"zitadel_org_id"`
	Name         string `json:"name"`
}

func (r *CreateOrganizationRequest) Validate() validator.ValidationErrors {

	v := validator.New()

	validator.Required(
		v,
		"zitadel_org_id",
		r.ZitadelOrgID,
	)

	validator.Required(
		v,
		"name",
		r.Name,
	)

	if v.Valid() {
		return nil
	}

	return v.Errors
}
