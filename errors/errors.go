// Package errors defines sentinel errors for the auther library.
package errors

import stderrors "errors"

// Role errors.
var (
	ErrRoleInvalid      = stderrors.New("grantor role is invalid")
	ErrGranteeInvalid   = stderrors.New("grantee role is invalid")
	ErrRoleInsufficient = stderrors.New("insufficient permissions")
	ErrRoleSelfGrant    = stderrors.New("self grant is not allowed")
	ErrRoleNotFound     = stderrors.New("role not found")
)

// User errors.
var (
	ErrUserInvalid         = stderrors.New("user is invalid")
	ErrUserNotFound        = stderrors.New("user not found")
	ErrRoleAlreadyAssigned = stderrors.New("role already assigned")
	ErrRoleNotAssigned     = stderrors.New("role not assigned")
)

// Policy errors.
var (
	ErrPolicyNotFound = stderrors.New("policy not found")
)

// Manager errors.
var (
	ErrAdapterRequired = stderrors.New("adapter is required")
	ErrRoleExists      = stderrors.New("role already exists")
	ErrUserExists      = stderrors.New("user already exists")
)
