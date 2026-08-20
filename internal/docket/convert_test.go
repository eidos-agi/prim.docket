package docket

import (
	"fmt"
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
	n, r, c, a := brief("Ship login")
	task := fmt.Sprintf("---\nid: TASK-0001\ntitle: Ship login\nstatus: To Do\npriority: High\nblocked_reason: waiting\nrequirements:\n  - %q\n  - %q\ntest-cases:\n  - %q\nacceptance-criteria:\n  - %q\n  - %q\n---\n%s\n", r[0], r[1], c[0], a[0], a[1], n)
	if err := os.WriteFile(filepath.Join(src, "tasks", "TASK-0001.md"), []byte(task), 0o644); err != nil {
		t.Fatal(err)
	}
	dn, dr, dc, da := brief("Wrote ADR")
	done := fmt.Sprintf("---\nid: TASK-0002\ntitle: Wrote ADR\nstatus: Done\nrequirements:\n  - %q\n  - %q\ntest-cases:\n  - %q\nacceptance-criteria:\n  - %q\n  - %q\n---\n%s\n", dr[0], dr[1], dc[0], da[0], da[1], dn)
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
	if ship == nil || ship.Priority != "high" || ship.Notes != n || len(ship.Acceptance) != 2 || len(ship.Requirements) != 2 || len(ship.TestCases) != 1 {
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
	if err := CheckUID(re.Tasks[0].UID); err != nil {
		t.Fatal(err)
	}
	if err := CheckUID(re.Milestones[0].UID); err != nil {
		t.Fatal(err)
	}
}
