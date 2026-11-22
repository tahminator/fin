package oauth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
)

const (
	minVerifierLength = 43
	maxVerifierLength = 128
)

func ValidChallenge(challenge string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(challenge)

	return err == nil && len(decoded) == sha256.Size
}

func VerifyS256(verifier, challenge string) bool {
	if len(verifier) < minVerifierLength || len(verifier) > maxVerifierLength || !unreserved(verifier) {
		return false
	}

	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])

	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

func unreserved(value string) bool {
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r == '-' || r == '.' || r == '_' || r == '~':
		default:
			return false
		}
	}

	return true
}
