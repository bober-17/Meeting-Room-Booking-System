package model

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

func (r Role) Valid() bool {
	return r == RoleAdmin || r == RoleUser
}

type User struct {
	ID        uuid.UUID
	Email     string
	Password  *string
	Role      Role
	CreatedAt time.Time
}
