package slot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/model"
)

const slotDuration = 30 * time.Minute

type Service struct {
	roomRepo     RoomRepository
	scheduleRepo ScheduleRepository
	slotRepo     SlotRepository
	log          *slog.Logger
}

func New(roomRepo RoomRepository, scheduleRepo ScheduleRepository, slotRepo SlotRepository, log *slog.Logger) *Service {
	return &Service{
		roomRepo:     roomRepo,
		scheduleRepo: scheduleRepo,
		slotRepo:     slotRepo,
		log:          log,
	}
}

// ListAvailableSlots возвращает свободные 30-минутные слоты для комнаты на указанную дату.
// Слоты генерируются лениво при первом запросе: если для данной даты их ещё нет в БД,
// они вставляются пакетно через INSERT ... ON CONFLICT DO NOTHING, а затем возвращаются только незабронированные.
// Если у комнаты нет расписания или дата вне рабочих дней — возвращается пустой список.
func (s *Service) ListAvailableSlots(ctx context.Context, roomID uuid.UUID, date time.Time) ([]model.Slot, error) {
	exists, err := s.roomRepo.RoomExists(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("slot service: check room: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("slot service: %w", model.ErrRoomNotFound)
	}

	schedule, err := s.scheduleRepo.GetScheduleByRoomID(ctx, roomID)
	if err != nil {
		if errors.Is(err, model.ErrScheduleNotFound) {
			return []model.Slot{}, nil
		}
		return nil, fmt.Errorf("slot service: get schedule: %w", err)
	}

	if !isDayInSchedule(date, schedule.DaysOfWeek) {
		return []model.Slot{}, nil
	}

	slots := generateSlots(roomID, date, schedule.StartTime, schedule.EndTime)

	if err := s.slotRepo.EnsureSlots(ctx, slots); err != nil {
		return nil, fmt.Errorf("slot service: ensure slots: %w", err)
	}

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.AddDate(0, 0, 1)

	available, err := s.slotRepo.ListAvailableByRoomAndDate(ctx, roomID, dayStart, dayEnd)
	if err != nil {
		return nil, fmt.Errorf("slot service: list available: %w", err)
	}

	return available, nil
}

// isDayInSchedule проверяет, входит ли день недели даты в расписание.
// Go: Sunday=0..Saturday=6; ISO (API): Monday=1..Sunday=7.
func isDayInSchedule(date time.Time, daysOfWeek []int) bool {
	isoDay := int(date.Weekday())
	if isoDay == 0 {
		isoDay = 7
	}

	return slices.Contains(daysOfWeek, isoDay)
}

// generateSlots создаёт список 30-минутных слотов для заданной комнаты и даты.
// startTime и endTime — строки формата "HH:MM".
func generateSlots(roomID uuid.UUID, date time.Time, startTime, endTime string) []model.Slot {
	startT, err := time.Parse("15:04", startTime)
	if err != nil {
		return nil
	}

	endT, err := time.Parse("15:04", endTime)
	if err != nil {
		return nil
	}

	windowStart := time.Date(date.Year(), date.Month(), date.Day(), startT.Hour(), startT.Minute(), 0, 0, time.UTC)
	windowEnd := time.Date(date.Year(), date.Month(), date.Day(), endT.Hour(), endT.Minute(), 0, 0, time.UTC)

	var slots []model.Slot
	for cur := windowStart; ; cur = cur.Add(slotDuration) {
		slotEnd := cur.Add(slotDuration)
		if slotEnd.After(windowEnd) {
			break
		}
		slots = append(slots, model.Slot{
			ID:      uuid.New(),
			RoomID:  roomID,
			StartAt: cur,
			EndAt:   slotEnd,
		})
	}

	return slots
}
