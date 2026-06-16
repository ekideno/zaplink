package apperror

import (
	"errors"
	"net/http"
	"testing"
)

func TestWrapAndFrom(t *testing.T) {
	base := errors.New("db failed")
	err := Wrap(base, http.StatusNotFound, "link_not_found", "link not found")

	if err == nil {
		t.Fatal("expected wrapped error")
	}
	if From(err) != err {
		t.Fatal("expected From to return the same app error")
	}
	if !errors.Is(err, base) {
		t.Fatal("expected wrapped error to unwrap to base error")
	}
}

func TestStatusMessageAndCode(t *testing.T) {
	err := New(http.StatusGone, "inactive_link", "link is inactive")

	if got := Status(err); got != http.StatusGone {
		t.Fatalf("expected status 410, got %d", got)
	}
	if got := Message(err); got != "link is inactive" {
		t.Fatalf("expected message, got %q", got)
	}
	if got := Code(err); got != "inactive_link" {
		t.Fatalf("expected code inactive_link, got %q", got)
	}
}

func TestStatusMessageAndCodeFallbacks(t *testing.T) {
	plain := errors.New("plain error")

	if got := Status(plain); got != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", got)
	}
	if got := Message(plain); got != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("expected fallback message, got %q", got)
	}
	if got := Code(plain); got != "" {
		t.Fatalf("expected empty code, got %q", got)
	}
	if From(plain) != nil {
		t.Fatal("expected From to return nil for plain error")
	}
}
