package model

import "errors"

var (
	ErrRoomNotFound       = errors.New("room not found")
	ErrSlotNotFound       = errors.New("slot not found")
	ErrBookingNotFound    = errors.New("booking not found")
	ErrSlotAlreadyBooked  = errors.New("slot already booked")
	ErrScheduleExists     = errors.New("schedule already exists")
	ErrForbidden          = errors.New("forbidden")
	ErrSlotInPast         = errors.New("slot is in the past")
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
