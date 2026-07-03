package hub

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

const tokenTTL = 30 * time.Second

type tokenEntry struct {
	userID    string
	expiresAt time.Time
}

type TokenStore struct {
	mu     sync.Mutex
	tokens map[string]tokenEntry
}

func NewTokenStore(ctx context.Context) *TokenStore {
	s := &TokenStore{tokens: make(map[string]tokenEntry)}
	go s.cleanup(ctx)
	return s
}

func (s *TokenStore) cleanup(ctx context.Context) {
	ticker := time.NewTicker(tokenTTL)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			s.mu.Lock()
			for token, entry := range s.tokens {
				if now.After(entry.expiresAt) {
					delete(s.tokens, token)
				}
			}
			s.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

// Issue генерирует одноразовый токен для userID с TTL 30 секунд.
func (s *TokenStore) Issue(userID string) string {
	token := uuid.New().String()
	s.mu.Lock()
	s.tokens[token] = tokenEntry{userID: userID, expiresAt: time.Now().Add(tokenTTL)}
	s.mu.Unlock()
	return token
}

// Consume проверяет токен, удаляет его и возвращает userID.
// Возвращает false если токен не найден или истёк.
func (s *TokenStore) Consume(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.tokens[token]
	delete(s.tokens, token)
	if !ok || time.Now().After(entry.expiresAt) {
		return "", false
	}
	return entry.userID, true
}
