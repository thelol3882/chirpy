package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret-do-not-use-in-production"

func makeToken(t *testing.T, userID uuid.UUID, secret string, expiresIn time.Duration) string {
	t.Helper()

	token, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT() unexpected error: %v", err)
	}
	return token
}

func TestMakeJWTValidateJWTRoundTrip(t *testing.T) {
	userID := uuid.New()

	token := makeToken(t, userID, testSecret, time.Hour)

	got, err := ValidateJWT(token, testSecret)
	if err != nil {
		t.Fatalf("ValidateJWT() unexpected error: %v", err)
	}
	if got != userID {
		t.Errorf("ValidateJWT() = %v, want %v", got, userID)
	}
}

func TestValidateJWTRejectsExpiredToken(t *testing.T) {
	token := makeToken(t, uuid.New(), testSecret, -time.Hour)

	if _, err := ValidateJWT(token, testSecret); err == nil {
		t.Error("ValidateJWT() accepted an expired token, want error")
	}
}

func TestValidateJWTRejectsWrongSecret(t *testing.T) {
	token := makeToken(t, uuid.New(), testSecret, time.Hour)

	if _, err := ValidateJWT(token, "a-completely-different-secret"); err == nil {
		t.Error("ValidateJWT() accepted a token signed with another secret, want error")
	}
}

func TestValidateJWTRejectsMalformedToken(t *testing.T) {
	valid := makeToken(t, uuid.New(), testSecret, time.Hour)

	tests := []struct {
		name  string
		token string
	}{
		{name: "empty string", token: ""},
		{name: "not a jwt at all", token: "hello-world"},
		{name: "only two segments", token: "header.payload"},
		{name: "corrupted signature", token: valid + "tampered"},
		{name: "truncated", token: valid[:len(valid)/2]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ValidateJWT(tt.token, testSecret); err == nil {
				t.Errorf("ValidateJWT(%q) = nil error, want error", tt.token)
			}
		})
	}
}

func TestValidateJWTRejectsTamperedPayload(t *testing.T) {
	token := makeToken(t, uuid.New(), testSecret, time.Hour)

	segments := strings.Split(token, ".")
	if len(segments) != 3 {
		t.Fatalf("expected a 3 segment token, got %d segments", len(segments))
	}

	forged, err := json.Marshal(jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
		Subject:   uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("marshalling forged claims: %v", err)
	}
	segments[1] = base64.RawURLEncoding.EncodeToString(forged)

	if _, err := ValidateJWT(strings.Join(segments, "."), testSecret); err == nil {
		t.Error("ValidateJWT() accepted a token whose payload was rewritten, want error")
	}
}

func TestValidateJWTRejectsWrongSigningMethod(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
		Subject:   uuid.New().String(),
	})

	signed, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("signing HS512 token: %v", err)
	}

	if _, err := ValidateJWT(signed, testSecret); err == nil {
		t.Error("ValidateJWT() accepted an HS512 token, want error")
	}
}
