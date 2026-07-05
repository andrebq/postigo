package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/andrebq/postigo/internal/kdb"
	"github.com/golang-jwt/jwt/v5"
)

var (
	errKeyNotSet = errors.New("key not set")
)

type (
	ed25519Key struct {
		pk  ed25519.PrivateKey
		pub ed25519.PublicKey
	}
)

func IsKeyNotSet(err error) bool {
	return errors.Is(err, errKeyNotSet)
}

func (e *ed25519Key) VerifyKey() any {
	return e.pub
}

func (e *ed25519Key) KID() string {
	return base64.StdEncoding.EncodeToString(e.pub)
}

func (e *ed25519Key) Sign(token *jwt.Token) (string, error) {
	return token.SignedString(e.pk)
}

type secretKey struct {
	ID   string `json:"id"`
	Seed []byte `json:"seed"`
}

func (s secretKey) GetID() string { return s.ID }

func LoadNodeKeyFromDB(ctx context.Context, secretsDB *kdb.DB, createNewIfNeeded bool) (KeySigner, error) {
	col, err := kdb.GetCollection[secretKey](ctx, secretsDB, "secrets")
	if err != nil {
		return nil, err
	}
	var nodeSeed secretKey
	err = col.Lookup(ctx, &nodeSeed, "node-seed")
	if errors.Is(err, sql.ErrNoRows) && createNewIfNeeded {
		var randseed [32]byte
		_, err := rand.Read(randseed[:])
		if err != nil {
			return nil, err
		}
		err = col.Put(ctx, &secretKey{
			Seed: randseed[:],
			ID:   "node-seed",
		})
		if err != nil {
			return nil, err
		}
		return NodeKeyFromSeed(randseed[:])
	} else if err != nil {
		return nil, err
	}
	return NodeKeyFromSeed(nodeSeed.Seed)
}

// Load node key from the given environment by looking up
// for the following envvar:
// POSTIGO_TOKEN_SIGN_SEED
//
// which should be either a hex or base64 encoded string
func LoadNodeKey(env func(string) string,
	setenv func(string, string) error) (KeySigner, error) {
	const envvarName = "POSTIGO_TOKEN_SIGN_SEED"
	seedtxt := env(envvarName)
	if len(seedtxt) == 0 {
		return nil, errKeyNotSet
	}
	setenv(envvarName, "")
	buf, err := hex.DecodeString(seedtxt)
	if err != nil {
		buf, err = base64.StdEncoding.DecodeString(seedtxt)
		if err != nil {
			return nil, fmt.Errorf("invalid value for envvar %v: %w", envvarName, err)
		}
	}
	return NodeKeyFromSeed(buf)
}

func NodeKeyFromSeed(seed []byte) (KeySigner, error) {
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("seed should be %v bytes long but found %v",
			ed25519.SeedSize, len(seed))
	}
	pk := ed25519.NewKeyFromSeed(seed)
	for i := range seed {
		seed[i] = 0
	}

	return &ed25519Key{pk: pk, pub: pk.Public().(ed25519.PublicKey)}, nil
}

func RandomNodeKey() (KeySigner, error) {
	var buf [32]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		return nil, err
	}
	return NodeKeyFromSeed(buf[:])
}
