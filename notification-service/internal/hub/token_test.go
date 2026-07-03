package hub

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenStore_IssueAndConsume(t *testing.T) {
	s := NewTokenStore(context.Background())
	token := s.Issue("user1")
	require.NotEmpty(t, token)

	userID, ok := s.Consume(token)
	assert.True(t, ok)
	assert.Equal(t, "user1", userID)
}

func TestTokenStore_ConsumeOnce(t *testing.T) {
	s := NewTokenStore(context.Background())
	token := s.Issue("user1")

	_, ok := s.Consume(token)
	require.True(t, ok)

	_, ok = s.Consume(token)
	assert.False(t, ok)
}

func TestTokenStore_ConsumeUnknown(t *testing.T) {
	s := NewTokenStore(context.Background())
	_, ok := s.Consume("nonexistent-token")
	assert.False(t, ok)
}

func TestTokenStore_ConsumeExpired(t *testing.T) {
	s := NewTokenStore(context.Background())

	s.mu.Lock()
	s.tokens["expired"] = tokenEntry{
		userID:    "user1",
		expiresAt: time.Now().Add(-time.Second),
	}
	s.mu.Unlock()

	_, ok := s.Consume("expired")
	assert.False(t, ok)
}

