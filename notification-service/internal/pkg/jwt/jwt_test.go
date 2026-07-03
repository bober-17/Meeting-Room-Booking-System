package jwt

import (
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret"

// makeToken — тестовый хелпер: notification-service не имеет GenerateToken.
func makeToken(userID uuid.UUID, role, secret string, expiry time.Time) string {
	claims := &Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			ExpiresAt: gojwt.NewNumericDate(expiry),
		},
		UserID: userID,
		Role:   role,
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func TestParseToken_Valid(t *testing.T) {
	userID := uuid.New()
	token := makeToken(userID, "admin", testSecret, time.Now().Add(time.Hour))

	gotID, gotRole, err := ParseToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, userID, gotID)
	assert.Equal(t, "admin", gotRole)
}

func TestParseToken_WrongSecret(t *testing.T) {
	token := makeToken(uuid.New(), "user", testSecret, time.Now().Add(time.Hour))

	_, _, err := ParseToken(token, "wrong-secret")
	require.Error(t, err)
}

func TestParseToken_Expired(t *testing.T) {
	token := makeToken(uuid.New(), "user", testSecret, time.Now().Add(-time.Second))

	_, _, err := ParseToken(token, testSecret)
	require.Error(t, err)
}

func TestParseToken_Malformed(t *testing.T) {
	_, _, err := ParseToken("not.a.jwt", testSecret)
	require.Error(t, err)
}

func TestParseToken_WrongAlgorithm(t *testing.T) {
	// RS256 нельзя подписать без RSA-ключа — строим заголовок вручную.
	// golang-jwt вызовет keyfunc с Method=*SigningMethodRSA, guard !ok вернёт ошибку.
	// eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9 = base64url({"alg":"RS256","typ":"JWT"})
	header := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9"
	payload := "eyJ1c2VyX2lkIjoiMDAwMDAwMDAtMDAwMC0wMDAwLTAwMDAtMDAwMDAwMDAwMDAxIn0"
	fakeToken := header + "." + payload + ".invalidsignature"

	_, _, err := ParseToken(fakeToken, testSecret)
	require.Error(t, err)
}
