package docket

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRejectsDanglingParent(t *testing.T) {
	p := &Pack{
		Dir:     t.TempDir(),
		Project: Project{Name: "x"},
		Tasks: []Task{
			card("TASK-0001", "a", "TASK", "TASK-9999"),
		},
	}
	if err := os.WriteFile(filepath.Join(p.Dir, "index.md"), []byte("profile: docket\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "dangling parent") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateRejectsHollowCard(t *testing.T) {
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
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "notes required") {
		t.Fatalf("got %v", err)
	}
	p.Tasks[0].Notes = "Ship login"
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "not the title") {
		t.Fatalf("got %v", err)
	}
	p.Tasks[0].Notes = "Users can sign in with SSO from the login form."
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "too terse") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateRejectsMissingUID(t *testing.T) {
	p := &Pack{
		Dir:     t.TempDir(),
		Project: Project{Name: "x"},
		Tasks: []Task{
			func() Task {
				row := live("Ship login")
				row.ID = "TASK-0001"
				row.Type = "TASK"
				row.Status = "To Do"
				return row
			}(),
		},
	}
	if err := os.WriteFile(filepath.Join(p.Dir, "index.md"), []byte("profile: docket\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "uid required") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateRejectsCycle(t *testing.T) {
	p := &Pack{
		Dir:     t.TempDir(),
		Project: Project{Name: "x"},
		Tasks: []Task{
			card("TASK-0001", "a", "TASK", "TASK-0002"),
			card("TASK-0002", "b", "TASK", "TASK-0001"),
		},
	}
	if err := os.WriteFile(filepath.Join(p.Dir, "index.md"), []byte("profile: docket\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("got %v", err)
	}
}

func TestPlanValidateRejectsMissingTypes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "thin")
	_, err := Plan(PlanRequest{
		Dir:       dir,
		Name:      "thin",
		Goal:      "thin",
		DoneWhen:  []string{"x"},
		ProofPath: "p.md",
		Tasks: []PlanTask{
			pt("only a task", "TASK"),
			pt("another task", "TASK"),
		},
	})
	if err == nil || !strings.Contains(err.Error(), "missing card types") {
		t.Fatalf("got %v", err)
	}
	if IsPack(dir) {
		t.Fatal("failed explosion left a pack")
	}
}

func TestPlanValidateRejectsDanglingParent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dangle")
	_, err := Plan(PlanRequest{
		Dir:       dir,
		Name:      "dangle",
		Goal:      "dangle",
		DoneWhen:  []string{"x"},
		ProofPath: "p.md",
		Tasks: []PlanTask{
			pt("goal", "GOAL"),
			pt("plan", "PLAN"),
			pt("task", "TASK", "nope"),
			pt("guard", "GUARD", "goal"),
			pt("test", "TEST", "task"),
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unresolved parent") {
		t.Fatalf("got %v", err)
	}
	if IsPack(dir) {
		t.Fatal("failed validate left a pack")
	}
}

func TestPlanRejectsHollowCards(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "hollow")
	_, err := Plan(PlanRequest{
		Dir:       dir,
		Name:      "hollow",
		Goal:      "hollow",
		DoneWhen:  []string{"x"},
		ProofPath: "p.md",
		Tasks: []PlanTask{
			{Title: "Root outcome for hollow mint", Type: "GOAL"},
			{Title: "Ladder for hollow mint", Type: "PLAN"},
			{Title: "Do the hollow work item", Type: "TASK"},
			{Title: "Refuse the hollow failure", Type: "GUARD"},
			{Title: "Prove the hollow work shipped", Type: "TEST"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "notes required") {
		t.Fatalf("got %v", err)
	}
	if IsPack(dir) {
		t.Fatal("failed hollow mint left a pack")
	}
}

func TestValidateRejectsUnparentedGuard(t *testing.T) {
	p := &Pack{
		Dir:     t.TempDir(),
		Project: Project{Name: "x"},
		Tasks: []Task{
			card("TASK-0001", "goal", "GOAL", ""),
			card("TASK-0002", "guard", "GUARD", ""),
		},
	}
	if err := os.WriteFile(filepath.Join(p.Dir, "index.md"), []byte("profile: docket\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "GUARD requires parent") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateRejectsGuardOnTest(t *testing.T) {
	p := &Pack{
		Dir:     t.TempDir(),
		Project: Project{Name: "x"},
		Tasks: []Task{
			card("TASK-0001", "work", "TASK", ""),
			card("TASK-0002", "proof", "TEST", "TASK-0001"),
			card("TASK-0003", "nope", "GUARD", "TASK-0002"),
		},
	}
	if err := os.WriteFile(filepath.Join(p.Dir, "index.md"), []byte("profile: docket\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "GOAL, PLAN, or TASK") {
		t.Fatalf("got %v", err)
	}
}

func TestPlanRejectsUnguardedGoal(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "unguarded")
	_, err := Plan(PlanRequest{
		Dir:       dir,
		Name:      "unguarded",
		Goal:      "unguarded",
		DoneWhen:  []string{"x"},
		ProofPath: "p.md",
		Tasks: []PlanTask{
			pt("goal", "GOAL"),
			pt("plan", "PLAN"),
			pt("task", "TASK"),
			pt("guard", "GUARD", "task"),
			pt("test", "TEST", "task"),
		},
	})
	if err == nil || !strings.Contains(err.Error(), "GOAL has no GUARD") {
		t.Fatalf("got %v", err)
	}
	if IsPack(dir) {
		t.Fatal("failed unguarded mint left a pack")
	}
}

func TestValidateNesting(t *testing.T) {
	face := func(p *Pack) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(p.Dir, "index.md"), []byte("profile: docket\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("TEST under TASK ok", func(t *testing.T) {
		p := &Pack{Dir: t.TempDir(), Project: Project{Name: "x"}, Tasks: []Task{
			card("TASK-0001", "work", "TASK", ""),
			card("TASK-0002", "proof", "TEST", "TASK-0001"),
		}}
		face(p)
		if err := Validate(p); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("GOAL under GOAL ok", func(t *testing.T) {
		p := &Pack{Dir: t.TempDir(), Project: Project{Name: "x"}, Tasks: []Task{
			card("TASK-0001", "root", "GOAL", ""),
			card("TASK-0002", "child", "GOAL", "TASK-0001"),
			card("TASK-0003", "close", "VALIDATION", "TASK-0001"),
		}}
		face(p)
		if err := Validate(p); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("PLAN under PLAN ok", func(t *testing.T) {
		p := &Pack{Dir: t.TempDir(), Project: Project{Name: "x"}, Tasks: []Task{
			card("TASK-0001", "ladder", "PLAN", ""),
			card("TASK-0002", "phase", "PLAN", "TASK-0001"),
			card("TASK-0003", "close", "VALIDATION", "TASK-0001"),
		}}
		face(p)
		if err := Validate(p); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("TEST under GOAL fails", func(t *testing.T) {
		p := &Pack{Dir: t.TempDir(), Project: Project{Name: "x"}, Tasks: []Task{
			card("TASK-0001", "root", "GOAL", ""),
			card("TASK-0002", "proof", "TEST", "TASK-0001"),
		}}
		face(p)
		if err := Validate(p); err == nil || !strings.Contains(err.Error(), "cannot nest") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("TASK under TEST fails", func(t *testing.T) {
		p := &Pack{Dir: t.TempDir(), Project: Project{Name: "x"}, Tasks: []Task{
			card("TASK-0001", "work", "TASK", ""),
			card("TASK-0002", "proof", "TEST", "TASK-0001"),
			card("TASK-0003", "more", "TASK", "TASK-0002"),
		}}
		face(p)
		if err := Validate(p); err == nil || !strings.Contains(err.Error(), "cannot nest") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("PLAN under TASK fails", func(t *testing.T) {
		p := &Pack{Dir: t.TempDir(), Project: Project{Name: "x"}, Tasks: []Task{
			card("TASK-0001", "work", "TASK", ""),
			card("TASK-0002", "ladder", "PLAN", "TASK-0001"),
		}}
		face(p)
		if err := Validate(p); err == nil || !strings.Contains(err.Error(), "cannot nest") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("unparented TEST fails", func(t *testing.T) {
		p := &Pack{Dir: t.TempDir(), Project: Project{Name: "x"}, Tasks: []Task{
			card("TASK-0001", "proof", "TEST", ""),
		}}
		face(p)
		if err := Validate(p); err == nil || !strings.Contains(err.Error(), "TEST requires parent") {
			t.Fatalf("got %v", err)
		}
	})
}

func card(id, title, typ, parent string) Task {
	uid, err := NewUID()
	if err != nil {
		panic(err)
	}
	row := typed(longTitle(title), typ)
	row.ID = id
	row.UID = uid
	row.Status = "To Do"
	row.Parent = parent
	return row
}

func pt(title, typ string, parent ...string) PlanTask {
	n, r, c, a := brief(longTitle(title))
	if NormType(typ) == "TEST" {
		c = append(c, "Repeat the proof after a refuse; it still fails.")
	}
	row := PlanTask{
		Title: longTitle(title), Type: typ, Notes: n,
		Requirements: r, TestCases: c, Acceptance: a,
	}
	if len(parent) > 0 {
		row.Parent = longTitle(parent[0])
	}
	return row
}

func longTitle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if len([]rune(s)) < minTitle {
		return s + " — named card"
	}
	return s
}
