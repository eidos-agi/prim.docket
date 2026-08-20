package docket

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanRejectsEmptyTasks(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "slice")
	_, err := Plan(PlanRequest{
		Dir:        dir,
		ID:         "glass-reads-segments",
		Name:       "Checkup cannot GREEN-wash compressor segments",
		Goal:       "Checkup cannot GREEN-wash compressor segments",
		DoneWhen:   []string{"sample --once matches sysctl segment_limit", "Memory Pressure names segment percent"},
		Negative:   "do not CRITICAL at 27 percent segments",
		OutOfScope: []KV{{Item: "Rust rewrite", Why: "D4"}},
		Shipped:    []KV{{Item: "aad sample --once", Why: "live"}},
		ProofPath:  "notes/goals/proof/x.md",
		Linear:     "none",
	})
	if err == nil || !strings.Contains(err.Error(), "tasks required") {
		t.Fatalf("got %v", err)
	}
	if IsPack(dir) {
		t.Fatal("empty-tasks mint left a pack")
	}
}

func TestPlanExplodesTypedCards(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bucket")
	got, err := Plan(PlanRequest{
		Dir:        dir,
		Name:       "Empty the boot graph",
		Goal:       "One Applets runtime, cold until fetch",
		DoneWhen:   []string{"two workers one listener"},
		OutOfScope: []KV{{Item: "Rust rewrite", Why: "D4"}},
		ProofPath:  "notes/dockets/proof/x.md",
		Milestones: []PlanMilestone{
			{Title: "Wave 0 — prove the runtime"},
		},
		Tasks: []PlanTask{
			pt("One runtime, catalog rows, cold until fetch", "GOAL"),
			pt("Rewrite ladder then delete the old process the same day", "PLAN", "One runtime, catalog rows, cold until fetch"),
			func() PlanTask {
				row := pt("Pavo sleeps in the manager", "TASK", "Rewrite ladder then delete the old process the same day")
				row.Milestone = "Wave 0 — prove the runtime"
				return row
			}(),
			pt("Do not vendor workerd", "GUARD", "One runtime, catalog rows, cold until fetch"),
			pt("Sibling store hostile read fails", "TEST", "Pavo sleeps in the manager"),
			pt("Close the runtime after the ladder is proven", "VALIDATION", "One runtime, catalog rows, cold until fetch"),
			pt("Close the rewrite ladder after Pavo sleeps", "VALIDATION", "Rewrite ladder then delete the old process the same day"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.TaskCount != 7 || got.Milestones != 1 {
		t.Fatalf("got %+v", got)
	}
	if !strings.Contains(got.Review, "docket-prim review --dir") {
		t.Fatalf("review %q", got.Review)
	}
	p, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	byType := map[string]int{}
	var test, guard Task
	for _, tk := range p.Tasks {
		byType[tk.Type]++
		if tk.Type == "TEST" {
			test = tk
		}
		if tk.Type == "GUARD" {
			guard = tk
		}
	}
	if byType["GOAL"] != 1 || byType["PLAN"] != 1 || byType["TASK"] != 1 || byType["GUARD"] != 1 || byType["TEST"] != 1 || byType["VALIDATION"] != 2 {
		t.Fatalf("types %+v", byType)
	}
	if test.Parent == "" || !strings.HasPrefix(test.Parent, "TASK-") {
		t.Fatalf("test parent %q", test.Parent)
	}
	if guard.Parent != p.Tasks[0].ID {
		t.Fatalf("guard parent %q want %q", guard.Parent, p.Tasks[0].ID)
	}
	if p.Tasks[2].Milestone != "MS-0001" {
		t.Fatalf("milestone %q", p.Tasks[2].Milestone)
	}
	face, err := os.ReadFile(filepath.Join(dir, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(face)
	for _, need := range []string{"profile: docket", "## Done when", "## Out of scope", "Rust rewrite", "## Stop"} {
		if !strings.Contains(s, need) {
			t.Fatalf("face missing %q", need)
		}
	}
}

func TestPlanRejectsEmptyDoneWhen(t *testing.T) {
	_, err := Plan(PlanRequest{Dir: t.TempDir(), Name: "x", ProofPath: "p"})
	if err == nil {
		t.Fatal("expected error")
	}
}
