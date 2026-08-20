package docket

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLiftRaisesScore(t *testing.T) {
	dir := t.TempDir()
	p, err := Init(dir, "Lift", "lift")
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
	before := ScoreOf(p)
	got, err := Lift(p, LiftOpts{Guards: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.After.Score <= before.Score {
		t.Fatalf("before %d after %d ops %+v", before.Score, got.After.Score, got.Ops)
	}
	if got.After.Counts.Tests == 0 || got.After.Counts.Validations == 0 || got.After.Counts.Guards == 0 {
		t.Fatalf("counts %+v", got.After.Counts)
	}
	re, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range re.Tasks {
		if NormType(row.Type) == "VALIDATION" || NormType(row.Type) == "TEST" || NormType(row.Type) == "GUARD" {
			if row.UID == "" {
				t.Fatalf("lifted %s missing uid", row.ID)
			}
		}
	}
}

func TestLiftDryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	p, err := Init(dir, "Lift", "lift")
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
	path := filepath.Join(dir, "tasks.jsonl")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Lift(p, LiftOpts{Guards: true}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.After.Score <= got.Before.Score || !got.DryRun || len(got.Ops) == 0 {
		t.Fatalf("%+v", got)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("dry-run wrote tasks.jsonl")
	}
}

func TestLiftRestoresShapedValidation(t *testing.T) {
	p := shapedPack()
	keep := p.Tasks[:0]
	for _, row := range p.Tasks {
		if row.ID != "TASK-0007" {
			keep = append(keep, row)
		}
	}
	p.Tasks = keep
	if ScoreOf(p).Score != 92 {
		t.Fatalf("setup %d", ScoreOf(p).Score)
	}
	got, err := Lift(p, LiftOpts{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.After.Score != 100 {
		t.Fatalf("after %d ops %+v", got.After.Score, got.Ops)
	}
}
