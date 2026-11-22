package oauth

import "testing"

func TestVerifyS256MatchesRFC7636Example(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	if !VerifyS256(verifier, challenge) {
		t.Fatal("RFC 7636 appendix B example did not verify")
	}

	if !ValidChallenge(challenge) {
		t.Error("RFC 7636 challenge reported invalid")
	}
}

func TestVerifyS256Rejects(t *testing.T) {
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	tests := map[string]string{
		"wrong verifier":  "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXX",
		"too short":       "short",
		"plain challenge": challenge,
		"invalid charset": "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEj+k",
		"empty":           "",
	}

	for name, verifier := range tests {
		if VerifyS256(verifier, challenge) {
			t.Errorf("%s: verified, want rejection", name)
		}
	}

	if ValidChallenge("not-a-sha256") {
		t.Error("short challenge reported valid")
	}
}
