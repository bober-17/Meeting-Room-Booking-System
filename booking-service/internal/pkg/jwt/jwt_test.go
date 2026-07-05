package jwt_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgjwt "github.com/bober-17/meeting-room-booking-system/booking-service/internal/pkg/jwt"
)

const secret = "test-secret"

func TestGenerateAndParseToken_RoundTrip(t *testing.T) {
	userID := uuid.New()
	role := "admin"

	token, err := pkgjwt.GenerateToken(userID, role, secret)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	gotID, gotRole, err := pkgjwt.ParseToken(token, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, gotID)
	assert.Equal(t, role, gotRole)
}

func TestParseToken_WrongSecret_ReturnsError(t *testing.T) {
	token, err := pkgjwt.GenerateToken(uuid.New(), "user", secret)
	require.NoError(t, err)

	_, _, err = pkgjwt.ParseToken(token, "wrong-secret")
	require.Error(t, err)
}

func TestParseToken_InvalidToken_ReturnsError(t *testing.T) {
	_, _, err := pkgjwt.ParseToken("not.a.jwt", secret)
	require.Error(t, err)
}
