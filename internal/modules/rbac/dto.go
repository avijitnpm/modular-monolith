package rbac

type PermissionResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RoleResponse struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Permissions []PermissionResponse `json:"permissions"`
}

type CreateRoleRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

type AssignUserRoleRequest struct {
	RoleID string `json:"role_id"`
}

func permissionResponse(
	permission Permission,
) PermissionResponse {

	return PermissionResponse{
		ID:   permission.ID,
		Name: permission.Name,
	}
}

func roleResponse(
	role Role,
) RoleResponse {

	permissions := make(
		[]PermissionResponse,
		0,
		len(role.Permissions),
	)

	for _, permission := range role.Permissions {
		permissions = append(
			permissions,
			permissionResponse(permission),
		)
	}

	return RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Permissions: permissions,
	}
}
