package encryption_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/tahminator/fin/internal/encryption"
)

var testKey = []byte("0123456789abcdefghijklmnopqrstuv")

func newCipher(t *testing.T, key []byte) *encryption.Cipher {
	t.Helper()

	c, err := encryption.New(key)
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func TestCipherRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
	}{
		{name: "access token", plaintext: "access-sandbox-123"},
		{name: "empty string", plaintext: ""},
		{name: "unicode", plaintext: "clé secrète ✓"},
	}

	c := newCipher(t, testKey)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sealed := c.Encrypt(tt.plaintext)

			opened, err := c.Decrypt(sealed)
			if err != nil {
				t.Fatal(err)
			}

			if opened != tt.plaintext {
				t.Fatalf("got %q, want %q", opened, tt.plaintext)
			}
		})
	}
}

func TestCipherEncryptIsNondeterministic(t *testing.T) {
	c := newCipher(t, testKey)

	first := c.Encrypt("access-sandbox-123")
	second := c.Encrypt("access-sandbox-123")

	if first == second {
		t.Fatalf("expected different ciphertexts, both were %q", first)
	}
}

func TestCipherEncryptDoesNotLeakPlaintext(t *testing.T) {
	c := newCipher(t, testKey)

	sealed, err := base64.StdEncoding.DecodeString(c.Encrypt("access-sandbox-123"))
	if err != nil {
		t.Fatal(err)
	}

	if string(sealed) == "access-sandbox-123" {
		t.Fatal("ciphertext equals plaintext")
	}
}

func TestCipherDecryptRejectsTamperedCiphertext(t *testing.T) {
	c := newCipher(t, testKey)

	sealed, err := base64.StdEncoding.DecodeString(c.Encrypt("access-sandbox-123"))
	if err != nil {
		t.Fatal(err)
	}

	for i := range sealed {
		tampered := append([]byte{}, sealed...)
		tampered[i] ^= 0x01

		if _, err := c.Decrypt(base64.StdEncoding.EncodeToString(tampered)); err == nil {
			t.Fatalf("flipping byte %d decrypted successfully", i)
		}
	}
}

func TestCipherDecryptRejectsTruncatedCiphertext(t *testing.T) {
	c := newCipher(t, testKey)

	sealed, err := base64.StdEncoding.DecodeString(c.Encrypt("access-sandbox-123"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := c.Decrypt(base64.StdEncoding.EncodeToString(sealed[:len(sealed)-1])); err == nil {
		t.Fatal("expected error for truncated ciphertext")
	}

	if _, err := c.Decrypt(""); err == nil {
		t.Fatal("expected error for empty ciphertext")
	}
}

func TestCipherDecryptRejectsWrongKey(t *testing.T) {
	sealed := newCipher(t, testKey).Encrypt("access-sandbox-123")
	other := newCipher(t, []byte("vutsrqponmlkjihgfedcba9876543210"))

	if _, err := other.Decrypt(sealed); err == nil {
		t.Fatal("expected error when decrypting with a different key")
	}
}

func TestCipherDecryptRejectsInvalidBase64(t *testing.T) {
	c := newCipher(t, testKey)

	if _, err := c.Decrypt("not base64!!"); err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestNewRejectsKeysThatAreNot32Bytes(t *testing.T) {
	for _, size := range []int{0, 16, 24, 31, 33} {
		t.Run(fmt.Sprintf("%d bytes", size), func(t *testing.T) {
			c, err := encryption.New(make([]byte, size))
			if err == nil {
				t.Fatalf("expected error for %d byte key", size)
			}

			if c != nil {
				t.Fatalf("expected nil cipher for %d byte key", size)
			}
		})
	}
}
