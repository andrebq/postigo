package auth

import (
	"fmt"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type (
	NodeClaims struct {
		jwt.RegisteredClaims
		KeyID    string `json:"kid"`
		Nodename string `json:"nodename"`
	}

	KeySigner interface {
		KID() string
		Sign(*jwt.Token) (string, error)
		VerifyKey() any
	}
)

func (c NodeClaims) KID() (string, error) {
	return c.KeyID, nil
}

func HasAudience(withAud interface {
	GetAudience() (jwt.ClaimStrings, error)
}, desired string) error {
	available, err := withAud.GetAudience()
	if err != nil {
		return err
	}
	if slices.Index(available, desired) >= 0 {
		return nil
	}
	return fmt.Errorf("invalid audience")
}

func DialNodeToken(ks KeySigner, nodename string, ttl time.Duration) (string, error) {
	nc := NodeClaims{}
	nc.Subject = fmt.Sprintf("/ws/dial/%v", nodename)
	nc.KeyID = ks.KID()
	nc.ExpiresAt = jwt.NewNumericDate(time.Now().Add(ttl))
	nc.NotBefore = jwt.NewNumericDate(time.Now())
	nc.Nodename = nodename
	// the issuer is the key itself
	nc.Issuer = ks.KID()
	nc.Audience = jwt.ClaimStrings{
		"dial-node",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, nc)
	return ks.Sign(token)
}

func ExposePortToken(ks KeySigner, nodename string, ttl time.Duration) (string, error) {
	nc := NodeClaims{}
	nc.Subject = fmt.Sprintf("/ws/expose/%v", nodename)
	nc.KeyID = ks.KID()
	nc.ExpiresAt = jwt.NewNumericDate(time.Now().Add(ttl))
	nc.NotBefore = jwt.NewNumericDate(time.Now())
	// each node is issued by a given kid
	// since what identifies a node is its key
	nc.Issuer = ks.KID()
	nc.Audience = jwt.ClaimStrings{
		"expose-port",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, nc)
	return ks.Sign(token)
}
