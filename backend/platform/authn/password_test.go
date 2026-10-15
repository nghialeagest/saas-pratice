package authn

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHashing(t *testing.T) {
	hasher := NewHasher(bcrypt.MinCost)

	password := "super-secret-password-123"

	// Test hash same password twice -> different hashes (due to random salt)
	hash1, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("expected no error on hash1, got %v", err)
	}
	hash2, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("expected no error on hash2, got %v", err)
	}

	if hash1 == hash2 {
		t.Fatalf("expected different hashes for same password, got identical hash: %s", hash1)
	}

	// Test Verify both hashes -> true
	if !hasher.Verify(password, hash1) {
		t.Errorf("expected Verify(password, hash1) to return true")
	}
	if !hasher.Verify(password, hash2) {
		t.Errorf("expected Verify(password, hash2) to return true")
	}

	// Test Verify wrong password -> false
	if hasher.Verify("wrong-password", hash1) {
		t.Errorf("expected Verify with wrong password to return false")
	}

	// Test Verify with garbage hash -> false, no panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Verify panicked on invalid input: %v", r)
		}
	}()

	if hasher.Verify(password, "not-a-valid-bcrypt-hash-garbage-value") {
		t.Errorf("expected Verify with garbage hash to return false")
	}

	// Test Verify with empty hash -> false, no panic
	if hasher.Verify(password, "") {
		t.Errorf("expected Verify with empty hash to return false")
	}
}

func TestNewHasherDefaultCost(t *testing.T) {
	hasher := NewHasher(0)
	bh, ok := hasher.(*bcryptHasher)
	if !ok {
		t.Fatalf("expected *bcryptHasher type")
	}
	if bh.cost != 12 {
		t.Errorf("expected default cost 12, got %d", bh.cost)
	}
}
