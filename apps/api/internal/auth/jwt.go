package auth

import (
	"errors"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type Verifier struct {
	secret []byte
}

func NewVerifier(secret string) *Verifier {
	return &Verifier{secret: []byte(secret)}
}

func (v *Verifier) UserIDFromToken(token string) (string, error) {
	parsed, err := jwt.Parse([]byte(token),
		jwt.WithKey(jwa.HS256, v.secret),
		jwt.WithValidate(true),
	)
	if err != nil {
		return "", err
	}
	sub := parsed.Subject()
	if sub == "" {
		return "", errors.New("token has no subject")
	}
	return sub, nil
}
