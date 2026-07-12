package auth

import (
	"errors"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"abc":         "abc",
		" oauth:xyz ": "xyz",
		"OAuth:Tok":   "Tok",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestErrorInvalid(t *testing.T) {
	err := &Error{Cause: ErrInvalidToken, Status: 401}
	if !err.Invalid() {
		t.Fatal("expected Invalid")
	}
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatal("errors.Is should match ErrInvalidToken")
	}
	if !IsInvalid(err) {
		t.Fatal("IsInvalid")
	}
}
