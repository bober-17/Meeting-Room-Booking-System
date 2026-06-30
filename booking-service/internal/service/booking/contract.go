package booking

import (
	"context"

	"github.com/google/uuid"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/model"
)

type BookingRepository interface {
	CreateBooking(ctx context.Context, slotID, userID uuid.UUID) (model.Booking, error)
	GetBookingByID(ctx context.Context, id uuid.UUID) (model.Booking, error)
	CancelBooking(ctx context.Context, id, userID uuid.UUID) (model.Booking, error)
	ListBookings(ctx context.Context, page, pageSize int) ([]model.Booking, int, error)
	ListUserBookings(ctx context.Context, userID uuid.UUID) ([]model.Booking, error)
	UpdateConferenceLink(ctx context.Context, bookingID uuid.UUID, link string) error
}

type SlotRepository interface {
	GetSlotByID(ctx context.Context, id uuid.UUID) (model.Slot, error)
}

type ConferenceClient interface {
	CreateLink(ctx context.Context, bookingID uuid.UUID) (string, error)
}
