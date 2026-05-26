package model

import "github.com/gralliry/go-auther/snapshot"

// Model 转 Snapshot

func Roles2Snapshots(src []Role) []snapshot.Role {
	out := make([]snapshot.Role, len(src))
	for i := range src {
		out[i] = snapshot.Role{ID: src[i].ID, ParentID: src[i].ParentID}
	}
	return out
}

func Users2Snapshots(src []User) []snapshot.User {
	out := make([]snapshot.User, len(src))
	for i := range src {
		out[i] = snapshot.User{ID: src[i].ID, RoleID: src[i].RoleID}
	}
	return out
}

func Grants2Snapshots(src []Grant) []snapshot.Grant {
	out := make([]snapshot.Grant, len(src))
	for i := range src {
		out[i] = snapshot.Grant{
			FromRoleID: src[i].FromRoleID,
			ToRoleID:   src[i].ToRoleID,
			Resource:   src[i].Resource,
		}
	}
	return out
}

// Snapshot 转 Model

func Roles2Models(src []snapshot.Role) []Role {
	out := make([]Role, len(src))
	for i := range src {
		out[i] = Role{ID: src[i].ID, ParentID: src[i].ParentID}
	}
	return out
}

func Users2Models(src []snapshot.User) []User {
	out := make([]User, len(src))
	for i := range src {
		out[i] = User{ID: src[i].ID, RoleID: src[i].RoleID}
	}
	return out
}

func Grants2Models(src []snapshot.Grant) []Grant {
	out := make([]Grant, len(src))
	for i := range src {
		out[i] = Grant{
			FromRoleID: src[i].FromRoleID,
			ToRoleID:   src[i].ToRoleID,
			Resource:   src[i].Resource,
		}
	}
	return out
}
