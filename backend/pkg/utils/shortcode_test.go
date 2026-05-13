package utils

import (
	"strings"
	"testing"
)

func TestGenerateShortCode_Length(t *testing.T) {
	cases := []int{1, 6, 12, 32}
	for _, n := range cases {
		code, err := GenerateShortCode(n)
		if err != nil {
			t.Fatalf("length %d: unexpected error: %v", n, err)
		}
		if len(code) != n {
			t.Errorf("length %d: got code of length %d: %q", n, len(code), code)
		}
	}
}

func TestGenerateShortCode_ZeroLength(t *testing.T) {
	// Zero length is degenerate but should not error — mirrors `make([]byte, 0)`.
	code, err := GenerateShortCode(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != "" {
		t.Errorf("expected empty string, got %q", code)
	}
}

func TestGenerateShortCode_AlphabetCompliance(t *testing.T) {
	// Draw a lot of codes and verify every char is in the declared alphabet.
	// This also gives basic confidence that no random byte smuggles in.
	for i := 0; i < 200; i++ {
		code, err := GenerateShortCode(8)
		if err != nil {
			t.Fatalf("iter %d: unexpected error: %v", i, err)
		}
		for _, ch := range code {
			if !strings.ContainsRune(shortCodeAlphabet, ch) {
				t.Fatalf("char %q not in alphabet (code: %q)", ch, code)
			}
		}
	}
}

func TestGenerateShortCode_ProbabilisticUniqueness(t *testing.T) {
	// At length 6, the keyspace is 62^6 ≈ 5.7e10. Drawing 1000 codes should
	// produce ~1000 unique values. We allow a tiny collision budget to keep
	// this test deterministically stable, but in practice it should be 0.
	const draws = 1000
	const length = 6
	seen := make(map[string]struct{}, draws)
	collisions := 0
	for i := 0; i < draws; i++ {
		code, err := GenerateShortCode(length)
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		if _, dup := seen[code]; dup {
			collisions++
		}
		seen[code] = struct{}{}
	}
	if collisions > 1 {
		t.Errorf("unexpectedly high collision count: %d (expected 0, budget 1)", collisions)
	}
}
