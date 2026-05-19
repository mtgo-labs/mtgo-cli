package main

import "testing"

func TestParseMessageID(t *testing.T) {
	got, err := parseMessageID("42")
	if err != nil {
		t.Fatalf("parseMessageID() error: %v", err)
	}
	if got != 42 {
		t.Fatalf("parseMessageID() = %d, want 42", got)
	}
}

func TestParseMessageIDRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"", "abc", "0", "-1", "2147483648"} {
		if _, err := parseMessageID(input); err == nil {
			t.Fatalf("parseMessageID(%q) succeeded, want error", input)
		}
	}
}
