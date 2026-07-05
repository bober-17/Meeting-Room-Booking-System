package booking

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/model"
)

// conferenceTimeout — максимальное время ожидания ответа от конференц-сервиса.
// Вызов некритичный: при сбое бронь сохраняется без ссылки, ошибка только логируется.
const conferenceTimeout = 2 * time.Second

type Service struct {
	bookingRepo BookingRepository
	slotRepo    SlotRepository
	confClient  ConferenceClient
	log         *slog.Logger
}

func New(bookingRepo BookingRepository, slotRepo SlotRepository, confClient ConferenceClient, log *slog.Logger) *Service {
	return &Service{
		bookingRepo: bookingRepo,
		slotRepo:    slotRepo,
		confClient:  confClient,
		log:         log,
	}
}

// Защита от двойного бронирования обеспечивается частичным уникальным индексом на уровне БД
// (уникальность slot_id WHERE status = 'active'), без явных блокировок SELECT FOR UPDATE.
// Если createConferenceLink = true — запрашивает ссылку у ConferenceClient;
// сбой конференц-сервиса не откатывает бронь, а только логируется.
func (s *Service) CreateBooking(ctx context.Context, slotID, userID uuid.UUID, createConferenceLink bool) (model.Booking, error) {
	slot, err := s.slotRepo.GetSlotByID(ctx, slotID)
	if err != nil {
		return model.Booking{}, fmt.Errorf("booking service: get slot: %w", err)
	}

	if slot.StartAt.Before(time.Now().UTC()) {
		return model.Booking{}, fmt.Errorf("booking service: %w", model.ErrSlotInPast)
	}

	booking, err := s.bookingRepo.CreateBooking(ctx, slotID, userID)
	if err != nil {
		return model.Booking{}, fmt.Errorf("booking service: create: %w", err)
	}

	if createConferenceLink {
		confCtx, cancel := context.WithTimeout(ctx, conferenceTimeout)
		defer cancel()

		link, err := s.confClient.CreateLink(confCtx, booking.ID)
		if err != nil {
			s.log.Error("booking service: create conference link", "err", err, "booking_id", booking.ID)
		} else {
			if err := s.bookingRepo.UpdateConferenceLink(context.Background(), booking.ID, link); err != nil {
				s.log.Error("booking service: update conference link", "err", err, "booking_id", booking.ID)
			} else {
				booking.ConferenceLink = &link
			}
		}
	}

	return booking, nil
}

// CancelBooking отменяет бронь, принадлежащую указанному пользователю.
// Атомарная операция: проверка владельца и смена статуса выполняются в одном UPDATE.
// Повторная отмена уже отменённой брони идемпотентна.
func (s *Service) CancelBooking(ctx context.Context, bookingID, userID uuid.UUID) (model.Booking, error) {
	booking, err := s.bookingRepo.CancelBooking(ctx, bookingID, userID)
	if err != nil {
		return model.Booking{}, fmt.Errorf("booking service: cancel: %w", err)
	}

	return booking, nil
}

// ListBookings возвращает постраничный список всех броней (только для admin).
func (s *Service) ListBookings(ctx context.Context, page, pageSize int) ([]model.Booking, int, error) {
	bookings, total, err := s.bookingRepo.ListBookings(ctx, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("booking service: list: %w", err)
	}

	return bookings, total, nil
}

// ListUserBookings возвращает только активные брони текущего пользователя на будущие слоты.
func (s *Service) ListUserBookings(ctx context.Context, userID uuid.UUID) ([]model.Booking, error) {
	bookings, err := s.bookingRepo.ListUserBookings(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("booking service: list user: %w", err)
	}

	return bookings, nil
}
