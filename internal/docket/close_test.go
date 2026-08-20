package docket

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRequiresValidationOnDecompose(t *testing.T) {
	p := &Pack{
		Dir:     t.TempDir(),
		Project: Project{Name: "x"},
		Tasks: []Task{
			card("TASK-0001", "root outcome named", "GOAL", ""),
			card("TASK-0002", "child work named", "TASK", "TASK-0001"),
		},
	}
	if err := os.WriteFile(filepath.Join(p.Dir, "index.md"), []byte("profile: docket\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "no VALIDATION close-out") {
		t.Fatalf("got %v", err)
	}
	p.Tasks = append(p.Tasks, card("TASK-0003", "close the root outcome", "VALIDATION", "TASK-0001"))
	if err := Validate(p); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsParentDoneBeforeValidation(t *testing.T) {
	p := &Pack{
		Dir:     t.TempDir(),
		Project: Project{Name: "x"},
		Tasks: []Task{
			card("TASK-0001", "root outcome named", "GOAL", ""),
			card("TASK-0002", "child work named", "TASK", "TASK-0001"),
			card("TASK-0003", "close the root outcome", "VALIDATION", "TASK-0001"),
		},
	}
	p.Tasks[0].Status = "Done"
	p.Tasks[1].Status = "Done"
	if err := os.WriteFile(filepath.Join(p.Dir, "index.md"), []byte("profile: docket\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "Done while VALIDATION") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateRejectsValidationBeforeSiblings(t *testing.T) {
	p := &Pack{
		Dir:     t.TempDir(),
		Project: Project{Name: "x"},
		Tasks: []Task{
			card("TASK-0001", "root outcome named", "GOAL", ""),
			card("TASK-0002", "child work named", "TASK", "TASK-0001"),
			card("TASK-0003", "close the root outcome", "VALIDATION", "TASK-0001"),
		},
	}
	p.Tasks[2].Status = "Done"
	if err := os.WriteFile(filepath.Join(p.Dir, "index.md"), []byte("profile: docket\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "VALIDATION Done while") {
		t.Fatalf("got %v", err)
	}
}

func TestCompleteWaitsForValidation(t *testing.T) {
	dir := t.TempDir()
	p, err := Init(dir, "Fixture", "fix")
	if err != nil {
		t.Fatal(err)
	}
	goal, err := p.CreateTask(typed("Root outcome named here", "GOAL"))
	if err != nil {
		t.Fatal(err)
	}
	child := typed("Child work named here", "TASK")
	child.Parent = goal.ID
	if _, err := p.CreateTask(child); err != nil {
		t.Fatal(err)
	}
	val := typed("Close the root after children", "VALIDATION")
	val.Parent = goal.ID
	got, err := p.CreateTask(val)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.CompleteTask(goal.ID); err == nil || !strings.Contains(err.Error(), "VALIDATION") {
		t.Fatalf("parent complete: %v", err)
	}
	if _, err := p.CompleteTask(got.ID); err == nil || !strings.Contains(err.Error(), "still open") {
		t.Fatalf("early validation: %v", err)
	}
	if _, err := p.CompleteTask(p.Tasks[1].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := p.CompleteTask(got.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := p.CompleteTask(goal.ID); err != nil {
		t.Fatal(err)
	}
}

func TestValidateNestingValidation(t *testing.T) {
	face := func(p *Pack) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(p.Dir, "index.md"), []byte("profile: docket\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("VALIDATION under TASK ok", func(t *testing.T) {
		p := &Pack{Dir: t.TempDir(), Project: Project{Name: "x"}, Tasks: []Task{
			card("TASK-0001", "work", "TASK", ""),
			card("TASK-0002", "child", "TASK", "TASK-0001"),
			card("TASK-0003", "close", "VALIDATION", "TASK-0001"),
		}}
		face(p)
		if err := Validate(p); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("VALIDATION under TEST fails", func(t *testing.T) {
		p := &Pack{Dir: t.TempDir(), Project: Project{Name: "x"}, Tasks: []Task{
			card("TASK-0001", "work", "TASK", ""),
			card("TASK-0002", "proof", "TEST", "TASK-0001"),
			card("TASK-0003", "close", "VALIDATION", "TASK-0002"),
		}}
		face(p)
		if err := Validate(p); err == nil || !strings.Contains(err.Error(), "cannot nest under TEST") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("unparented VALIDATION fails", func(t *testing.T) {
		p := &Pack{Dir: t.TempDir(), Project: Project{Name: "x"}, Tasks: []Task{
			card("TASK-0001", "close", "VALIDATION", ""),
		}}
		face(p)
		if err := Validate(p); err == nil || !strings.Contains(err.Error(), "VALIDATION requires parent") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("leaf TASK needs no VALIDATION", func(t *testing.T) {
		p := &Pack{Dir: t.TempDir(), Project: Project{Name: "x"}, Tasks: []Task{
			card("TASK-0001", "work", "TASK", ""),
			card("TASK-0002", "proof", "TEST", "TASK-0001"),
		}}
		face(p)
		if err := Validate(p); err != nil {
			t.Fatal(err)
		}
	})
}
