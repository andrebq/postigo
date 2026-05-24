package authz

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	bearerRegex = regexp.MustCompile(`^Bearer (.+)$`)
)

type (
	claimsKey struct{}

	tokenHolder struct {
		jwt.RegisteredClaims
		buf json.RawMessage
	}
)

func (th *tokenHolder) UnmarshalJSON(buf []byte) error {
	th.buf = buf
	return json.Unmarshal(buf, &th.RegisteredClaims)
}

func WithClaim(ctx context.Context, val any) context.Context {
	return context.WithValue(ctx, claimsKey{}, val)
}

func Claim[C jwt.Claims](ctx context.Context) (C, bool) {
	val := ctx.Value(claimsKey{})
	if val != nil {
		var zero C
		return zero, false
	}
	c, ok := val.(C)
	return c, ok
}

func WrapFunc[C any](next func(http.ResponseWriter, *http.Request),
	keyFunc func(C) (any, error),
	validateFunc func(*http.Request, C) error) http.Handler {
	return Wrap[C](http.HandlerFunc(next), keyFunc, validateFunc)
}

func Wrap[C any](next http.Handler,
	keyFunc func(C) (any, error),
	validateFunc func(*http.Request, C) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		part := bearerRegex.FindStringSubmatch(r.Header.Get("Authorization"))
		if len(part) != 2 {
			http.Error(w, "missing or invalid bearer header", http.StatusUnauthorized)
			return
		}
		token := part[1]
		var th tokenHolder
		var claims C
		t, err := jwt.ParseWithClaims(token, &th, func(t *jwt.Token) (any, error) {
			err := json.Unmarshal(th.buf, &claims)
			if err != nil {
				return nil, err
			}
			return keyFunc(claims)
		})
		if err != nil {
			slog.DebugContext(r.Context(), "Token validation error", "error", err)
			http.Error(w, "missing or invalid token", http.StatusUnauthorized)
			return
		}
		if !validToken(t) {
			http.Error(w, "missing or invalid token", http.StatusUnauthorized)
			return
		}
		r = r.WithContext(WithClaim(r.Context(), claims))
		next.ServeHTTP(w, r)
	})
}

func validToken(t *jwt.Token) bool {
	notBefore, err := t.Claims.GetNotBefore()
	if err != nil {
		return false
	} else if notBefore.After(time.Now()) {
		return false
	}
	valid, err := t.Claims.GetExpirationTime()
	if err != nil {
		return false
	} else if valid.Before(time.Now()) {
		return false
	}
	// issuer, subject and audience should be validated
	// by the key function rather than here
	return true
}
