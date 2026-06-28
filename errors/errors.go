// Package errors defines sentinel errors for the auther library.
package errors

import stderrors "errors"

// Role errors.
var (
	ErrRoleInvalid      = stderrors.New("grantor role is invalid")   // grantor role is nil or corrupted
	ErrGranteeInvalid   = stderrors.New("grantee role is invalid")  // grantee role is nil or corrupted
	ErrRoleInsufficient = stderrors.New("insufficient permissions") // grantor lacks the resource it tries to delegate
	ErrRoleSelfGrant    = stderrors.New("self grant is not allowed") // cannot grant to self
	ErrRoleNotFound     = stderrors.New("role not found")           // role ID does not exist
)

// User errors.
var (
	ErrUserInvalid         = stderrors.New("user is invalid")          // user object is nil or malformed
	ErrUserNotFound        = stderrors.New("user not found")          // user ID does not exist
	ErrRoleAlreadyAssigned = stderrors.New("role already assigned")   // user already has this role
	ErrRoleNotAssigned     = stderrors.New("role not assigned")       // user does not have this role
)

// Policy errors.
var (
	ErrPolicyNotFound = stderrors.New("policy not found") // policy not found in tarGrants
)

// Manager errors.
var (
	ErrAdapterRequired = stderrors.New("adapter is required") // NewManager called with nil adapter
	ErrRoleExists      = stderrors.New("role already exists")  // CreateRole with duplicate ID
	ErrUserExists      = stderrors.New("user already exists")  // CreateUser with duplicate ID
)
