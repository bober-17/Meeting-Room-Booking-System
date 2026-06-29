package events

import "time"

const TopicBookingEvents = "booking.events"

type EventType string

const (
	EventTypeBookingCreated   EventType = "booking.created"
	EventTypeBookingCancelled EventType = "booking.cancelled"
)

// BookingEvent — схема Kafka-сообщения для всех событий бронирования.
// EventID соответствует ID записи в таблице outbox booking-service — используется
// для идемпотентной обработки на стороне notification-service.
type BookingEvent struct {
	EventID    string    `json:"event_id"`
	EventType  EventType `json:"event_type"`
	OccurredAt time.Time `json:"occurred_at"`
	BookingID  string    `json:"booking_id"`
	UserID     string    `json:"user_id"`
	RoomID     string    `json:"room_id"`
	RoomName   string    `json:"room_name"`
	SlotStart  time.Time `json:"slot_start"`
	SlotEnd    time.Time `json:"slot_end"`
}
