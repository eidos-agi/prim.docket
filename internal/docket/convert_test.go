package docket

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConvertMarkdown(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, ".docket")
	if err := os.MkdirAll(filepath.Join(src, "tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "completed"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "milestones"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "docket.json"), []byte(`{"id":"guid-1","project":"Cerebro"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	task := "---\nid: TASK-0001\ntitle: Ship login\nstatus: To Do\npriority: High\nblocked_reason: waiting\nacceptance-criteria:\n  - users can log in\n---\nNotes here.\n"
	if err := os.WriteFile(filepath.Join(src, "tasks", "TASK-0001.md"), []byte(task), 0o644); err != nil {
		t.Fatal(err)
	}
	done := "---\nid: TASK-0002\ntitle: Wrote ADR\nstatus: Done\n---\nClosed.\n"
	if err := os.WriteFile(filepath.Join(src, "completed", "TASK-0002.md"), []byte(done), 0o644); err != nil {
		t.Fatal(err)
	}
	ms := "---\nid: MS-0001\ntitle: Architecture\nstatus: closed\n---\n"
	if err := os.WriteFile(filepath.Join(src, "milestones", "MS-0001.md"), []byte(ms), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(root, "docket.prim")
	p, err := ConvertMarkdown(root, dest)
	if err != nil {
		t.Fatal(err)
	}
	if p.Project.Name != "Cerebro" || p.Project.ID != "guid-1" {
		t.Fatalf("project %+v", p.Project)
	}
	if len(p.Tasks) != 2 {
		t.Fatalf("tasks %d", len(p.Tasks))
	}
	var ship *Task
	for i := range p.Tasks {
		if p.Tasks[i].ID == "TASK-0001" {
			ship = &p.Tasks[i]
		}
	}
	if ship == nil || ship.Priority != "high" || ship.Notes != "Notes here." || len(ship.Acceptance) != 1 {
		t.Fatalf("TASK-0001 %+v", ship)
	}
	if len(p.Milestones) != 1 || p.Milestones[0].ID != "MS-0001" {
		t.Fatalf("milestones %+v", p.Milestones)
	}
	re, err := Open(dest)
	if err != nil {
		t.Fatal(err)
	}
	if re.Tool().Cites != "docket:guid-1" {
		t.Fatalf("cites %s", re.Tool().Cites)
	}
}
