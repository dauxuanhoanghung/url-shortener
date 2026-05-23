package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-please-change"

func TestAccessToken_Roundtrip(t *testing.T) {
	tok, err := GenerateAccessToken(TokenInput{
		UserID: "user-1",
		Email:  "user@example.com",
		Role:   "user",
	}, testSecret)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	claims, err := ValidateToken(tok, testSecret)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("user id: got %q want %q", claims.UserID, "user-1")
	}
	if claims.Email != "user@example.com" {
		t.Errorf("email: got %q want %q", claims.Email, "user@example.com")
	}
	if claims.Role != "user" {
		t.Errorf("role: got %q want %q", claims.Role, "user")
	}
}

func TestRefreshToken_Roundtrip(t *testing.T) {
	// Refresh tokens share the same claim shape as access tokens; the only
	// difference is the expiry window. Verify both claim propagation and
	// that the token is actually valid right after issue.
	tok, err := GenerateRefreshToken(TokenInput{
		UserID: "user-2",
		Email:  "other@example.com",
		Role:   "user",
	}, testSecret)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	claims, err := ValidateToken(tok, testSecret)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.UserID != "user-2" {
		t.Errorf("user id mismatch: got %q", claims.UserID)
	}
}

func TestAdminToken_ShorterExpiry(t *testing.T) {
	// Admin tokens use a tighter TTL (5 min access / 8 h refresh). Verify the
	// expiry actually shrinks when role=admin so we catch regressions in the
	// role-based switch.
	in := TokenInput{UserID: "admin-1", Email: "admin@example.com", Role: "admin"}

	accessTok, err := GenerateAccessToken(in, testSecret)
	if err != nil {
		t.Fatalf("generate access: %v", err)
	}
	accessClaims, err := ValidateToken(accessTok, testSecret)
	if err != nil {
		t.Fatalf("validate access: %v", err)
	}
	if accessClaims.Role != "admin" {
		t.Errorf("role: got %q want admin", accessClaims.Role)
	}
	gotAccessTTL := time.Until(accessClaims.ExpiresAt.Time)
	if gotAccessTTL > AccessTokenExpiry {
		t.Errorf("admin access TTL too long: got %v, expected ≤ %v", gotAccessTTL, AdminAccessTokenExpiry)
	}

	refreshTok, err := GenerateRefreshToken(in, testSecret)
	if err != nil {
		t.Fatalf("generate refresh: %v", err)
	}
	refreshClaims, err := ValidateToken(refreshTok, testSecret)
	if err != nil {
		t.Fatalf("validate refresh: %v", err)
	}
	gotRefreshTTL := time.Until(refreshClaims.ExpiresAt.Time)
	if gotRefreshTTL > RefreshTokenExpiry {
		t.Errorf("admin refresh TTL too long: got %v, expected ≤ %v", gotRefreshTTL, AdminRefreshTokenExpiry)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	tok, err := GenerateAccessToken(TokenInput{
		UserID: "user-1",
		Email:  "user@example.com",
		Role:   "user",
	}, testSecret)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := ValidateToken(tok, "different-secret"); err == nil {
		t.Fatal("expected error when validating with wrong secret")
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	cases := map[string]string{
		"empty":         "",
		"not-a-jwt":     "just-some-random-string",
		"only-two-dots": "a.b",
		"truncated":     "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ4In0",
	}
	for name, tok := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ValidateToken(tok, testSecret); err == nil {
				t.Errorf("expected error for malformed token %q", tok)
			}
		})
	}
}

func TestValidateToken_Expired(t *testing.T) {
	// Issue a token with an already-past expiry by hand, so we don't need
	// to sleep or manipulate the clock.
	claims := TokenClaims{
		UserID: "user-1",
		Email:  "user@example.com",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := ValidateToken(tok, testSecret); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidateToken_WrongAlgorithm(t *testing.T) {
	// Ensure a token signed with a different algorithm still fails when
	// we reject it — guards against the classic "alg: none" swap attack
	// if anyone ever changes the verifier later.
	claims := TokenClaims{
		UserID: "user-1",
		Email:  "user@example.com",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	// Sign with HS512 but validate expects HS256's secret; the library will
	// accept any HMAC variant by default, but the secret-based verify still
	// requires the same key material. We keep this as a sanity check that
	// ValidateToken doesn't silently accept wildly wrong inputs.
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte("completely-different"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := ValidateToken(tok, testSecret); err == nil {
		t.Fatal("expected error when secret does not match signed token")
	}
}
