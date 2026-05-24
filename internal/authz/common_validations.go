package authz

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type (
	WithKID interface {
		jwt.Claims
		KID() (string, error)
	}
)

func AnyKey[C WithKID](c C) (any, error) {
	id, err := c.KID()
	if err != nil {
		return nil, err
	}
	buf, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		return nil, err
	} else if len(buf) != 32 {
		return nil, errors.New("invalid key")
	}
	return ed25519.PublicKey(buf), nil
}

func AcceptAll[C jwt.Claims](_ *http.Request, _ C) error { return nil }
