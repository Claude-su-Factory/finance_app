package auth

import (
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "super-secret-key"

func makeToken(t *testing.T, sub string, exp time.Time) string {
	t.Helper()
	tok, err := jwt.NewBuilder().
		Subject(sub).
		Expiration(exp).
		Issuer("supabase").
		Build()
	require.NoError(t, err)
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.HS256, []byte(testSecret)))
	require.NoError(t, err)
	return string(signed)
}

func TestVerify_ValidToken(t *testing.T) {
	v := NewVerifier(testSecret)
	token := makeToken(t, "user-123", time.Now().Add(time.Hour))

	uid, err := v.UserIDFromToken(token)

	assert.NoError(t, err)
	assert.Equal(t, "user-123", uid)
}

func TestVerify_ExpiredToken(t *testing.T) {
	v := NewVerifier(testSecret)
	token := makeToken(t, "user-123", time.Now().Add(-time.Hour))

	_, err := v.UserIDFromToken(token)

	assert.Error(t, err)
}

func TestVerify_WrongSecret(t *testing.T) {
	v := NewVerifier("different-secret")
	token := makeToken(t, "user-123", time.Now().Add(time.Hour))

	_, err := v.UserIDFromToken(token)

	assert.Error(t, err)
}
