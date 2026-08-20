package docket

import (
	"strings"
	"testing"
)

func TestCheckCardRejectsTerseNotes(t *testing.T) {
	if err := CheckCard(Task{Title: "Ship login"}); err == nil || !strings.Contains(err.Error(), "notes required") {
		t.Fatalf("got %v", err)
	}
	if err := CheckCard(Task{Title: "Ship login", Notes: "Ship login"}); err == nil || !strings.Contains(err.Error(), "not the title") {
		t.Fatalf("got %v", err)
	}
	short := live("Ship login")
	short.Notes = "Users can sign in with SSO from the login form."
	if err := CheckCard(short); err == nil || !strings.Contains(err.Error(), "too terse") {
		t.Fatalf("got %v", err)
	}
}

func TestCheckCardRejectsThinLists(t *testing.T) {
	t.Run("requirements", func(t *testing.T) {
		row := live("Ship login")
		row.Requirements = []string{"only one"}
		if err := CheckCard(row); err == nil || !strings.Contains(err.Error(), "requirements") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("test-cases", func(t *testing.T) {
		row := live("Ship login")
		row.TestCases = nil
		if err := CheckCard(row); err == nil || !strings.Contains(err.Error(), "test-cases") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("acceptance", func(t *testing.T) {
		row := live("Ship login")
		row.Acceptance = []string{"only one"}
		if err := CheckCard(row); err == nil || !strings.Contains(err.Error(), "acceptance-criteria") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("TEST needs two cases", func(t *testing.T) {
		row := typed("Prove login", "TEST")
		row.TestCases = []string{"one case only"}
		if err := CheckCard(row); err == nil || !strings.Contains(err.Error(), "≥2 for TEST") {
			t.Fatalf("got %v", err)
		}
	})
}

func TestCheckCardAcceptsStory(t *testing.T) {
	if err := CheckCard(live("Ship login")); err != nil {
		t.Fatal(err)
	}
	if err := CheckCard(typed("Prove login", "TEST")); err != nil {
		t.Fatal(err)
	}
}

func TestCheckDay(t *testing.T) {
	if err := CheckDay(""); err != nil {
		t.Fatal(err)
	}
	if err := CheckDay("2026-08-20"); err != nil {
		t.Fatal(err)
	}
	if err := CheckDay("08/20/2026"); err == nil {
		t.Fatal("expected reject")
	}
}
