package docket

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBodyFindingsRejectsShortRequirement(t *testing.T) {
	row := live("Ship login")
	row.Requirements = []string{"too short to count", "also too short here"}
	fs := BodyFindings(row)
	if len(fs) == 0 {
		t.Fatal("expected short-line findings")
	}
	got := FindingsError(fs).Error()
	if !strings.Contains(got, "too short") {
		t.Fatalf("got %s", got)
	}
}

func TestBodyFindingsRejectsPlaceholder(t *testing.T) {
	row := live("Ship login")
	row.Requirements = []string{
		"TODO: fill this in later with a real requirement that is long enough",
		"The proof artifact or command is named and is the only path to Done.",
	}
	fs := BodyFindings(row)
	if FindingsError(fs) == nil || !strings.Contains(FindingsError(fs).Error(), "placeholder") {
		t.Fatalf("got %v", fs)
	}
}

func TestSchemaCollectsAll(t *testing.T) {
	p := &Pack{
		Dir:     t.TempDir(),
		Project: Project{Name: "x"},
		Tasks: []Task{
			{ID: "TASK-0001", Title: "Ship login", Type: "TASK", Status: "To Do"},
		},
	}
	if err := os.WriteFile(filepath.Join(p.Dir, "index.md"), []byte("profile: docket\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fs := SchemaFindings(p)
	codes := map[string]bool{}
	for _, f := range fs {
		codes[f.Code] = true
	}
	for _, need := range []string{"notes-blank", "req-missing", "cases-missing", "accept-missing", "uid-missing"} {
		if !codes[need] {
			t.Fatalf("missing %s in %+v", need, fs)
		}
	}
}

func TestCheckCardRejectsShortLine(t *testing.T) {
	row := live("Ship login")
	row.Requirements = []string{"too short to count", "also too short here"}
	if err := CheckCard(row); err == nil || !strings.Contains(err.Error(), "too short") {
		t.Fatalf("got %v", err)
	}
}
