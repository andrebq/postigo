package authz

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/andrebq/postigo/internal/kdb"
	"github.com/golang-jwt/jwt/v5"
)

type (
	WithKID interface {
		jwt.Claims
		KID() (string, error)
	}

	AuthorizedKey struct {
		KID string
		// Exposes contains the list of nodenames that this
		// key is authorized to expose
		Exposes []string `json:"exposes"`

		// DialTo contains the list of nodenames that this
		// key is authorized to dial, use "*" to authorized all
		// nodes.
		//
		// An empty DialTo key indicates that this key is only for exposing
		DialTo []string `json:"dialTo"`
	}
)

func (a AuthorizedKey) GetID() string {
	return a.KID
}

// KeyFromDB returns a key lookup that checks if the KID from the token is available
// in the database.
func KeyFromDB[C WithKID](keyset *kdb.Collection[AuthorizedKey]) func(c C) (any, error) {
	return func(c C) (any, error) {
		kid, err := c.KID()
		if err != nil {
			return nil, err
		}
		var ak AuthorizedKey
		err = keyset.Lookup(context.Background(), &ak, kid)
		if err != nil {
			return nil, err
		}
		// at this point we know the key is correct,
		// so simply let it be decoded
		return AnyKey(c)
	}
}

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
