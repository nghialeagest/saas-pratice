package authn

import "golang.org/x/crypto/bcrypt"

// Hasher provides password hashing and verification.
type Hasher interface {
	Hash(plain string) (string, error)
	Verify(plain, hash string) bool
}

type bcryptHasher struct {
	cost int
}

// NewHasher returns a bcrypt-based Hasher. Cost defaults to 12.
func NewHasher(cost int) Hasher {
	if cost <= 0 {
		cost = 12
	}
	return &bcryptHasher{cost: cost}
}

func (h *bcryptHasher) Hash(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	return string(bytes), err
}

// Verify runs in constant time. If hash has bad format, returns false without panic.
func (h *bcryptHasher) Verify(plain, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	return err == nil
}
