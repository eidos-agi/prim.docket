package docket

import "testing"

func TestNewUIDFormatAndUnique(t *testing.T) {
	a, err := NewUID()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewUID()
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckUID(a); err != nil {
		t.Fatal(err)
	}
	if err := CheckUID(b); err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("collision")
	}
	if err := CheckUID(""); err == nil {
		t.Fatal("empty uid")
	}
	if err := CheckUID("TASK-0001"); err == nil {
		t.Fatal("face id is not uid")
	}
}
