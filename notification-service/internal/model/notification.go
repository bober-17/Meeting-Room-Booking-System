package model

import "time"

type NotificationType string

const (
	TypeBookingCreated   NotificationType = "booking.created"
	TypeBookingCancelled NotificationType = "booking.cancelled"
)

type Notification struct {
	ID        string           `json:"id"`
	UserID    string           `json:"user_id"`
	Type      NotificationType `json:"type"`
	BookingID string           `json:"booking_id"`
	RoomName  string           `json:"room_name"`
	SlotStart time.Time        `json:"slot_start"`
	SlotEnd   time.Time        `json:"slot_end"`
	IsRead    bool             `json:"is_read"`
	CreatedAt time.Time        `json:"created_at"`
}
