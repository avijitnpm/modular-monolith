package rbac

import "errors"

var (
	ErrRoleAlreadyExists  = errors.New("role already exists")
	ErrRoleNotFound       = errors.New("role not found")
	ErrUserRoleNotFound   = errors.New("user role not found")
	ErrUserRoleExists     = errors.New("user role already exists")
	ErrUnknownPermissions = errors.New("unknown permissions")
)
