package conference

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Client — заглушка внешнего конференц-сервиса.
// В реальном сервисе здесь был бы HTTP-клиент к стороннему API.
// Возвращает детерминированную ссылку на основе bookingID без сетевых вызовов.
type Client struct{}

func New() *Client {
	return &Client{}
}

// CreateLink возвращает ссылку на конференцию для указанной брони.
func (c *Client) CreateLink(_ context.Context, bookingID uuid.UUID) (string, error) {
	return fmt.Sprintf("https://conference.example/meeting/%s", bookingID), nil
}
