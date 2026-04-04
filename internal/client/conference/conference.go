package conference

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Client — детерминированный мок ConferenceService.
// Возвращает стабильную ссылку по bookingID.
type Client struct{}

func New() *Client {
	return &Client{}
}

func (c *Client) CreateLink(_ context.Context, bookingID uuid.UUID) (string, error) {
	return fmt.Sprintf("https://conference.example/meeting/%s", bookingID), nil
}
