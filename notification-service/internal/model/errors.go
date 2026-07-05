package model

import "errors"

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrForbidden            = errors.New("forbidden")
)
