package docket

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreateEditCompleteArchive(t *testing.T) {
	dir := t.TempDir()
	p, err := Init(dir, "Fixture", "fix")
	if err != nil {
		t.Fatal(err)
	}
	if !IsPack(dir) {
		t.Fatal("expected pack marker")
	}
	if p.Project.Name != "Fixture" {
		t.Fatalf("name %q", p.Project.Name)
	}

	a, err := p.CreateTask(Task{Title: "Ship login", Priority: "high"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "TASK-0001" || a.Status != "To Do" {
		t.Fatalf("create %+v", a)
	}

	b, err := p.CreateTask(Task{Title: "Write docs"})
	if err != nil {
		t.Fatal(err)
	}
	if b.ID != "TASK-0002" {
		t.Fatalf("id %s", b.ID)
	}

	title := "Ship login now"
	status := "In Progress"
	_, err = p.EditTask("TASK-0001", TaskEdit{Title: &title, Status: &status})
	if err != nil {
		t.Fatal(err)
	}

	re, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, _, ok := re.Task("TASK-0001")
	if !ok || got.Title != title || got.Status != status {
		t.Fatalf("reload %+v", got)
	}
	if len(re.ListTasks("In Progress")) != 1 {
		t.Fatal("list status")
	}
	if hits := re.Search("login"); len(hits) != 1 {
		t.Fatalf("search %d", len(hits))
	}

	if _, err := re.CompleteTask("TASK-0002"); err != nil {
		t.Fatal(err)
	}
	re, _ = Open(dir)
	done, _, _ := re.Task("TASK-0002")
	if done.Status != "Done" {
		t.Fatal("complete")
	}

	if _, err := re.ArchiveTask("TASK-0002"); err != nil {
		t.Fatal(err)
	}
	re, _ = Open(dir)
	if _, _, ok := re.Task("TASK-0002"); ok {
		t.Fatal("archived still in tasks")
	}
	if len(re.Archive) != 1 || re.Archive[0].ID != "TASK-0002" {
		t.Fatalf("archive %+v", re.Archive)
	}

	m, err := re.CreateMilestone("Gold", "")
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "MS-0001" {
		t.Fatalf("ms %s", m.ID)
	}
	if _, err := re.CloseMilestone("MS-0001"); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(dir, "sub", "deep")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	up, err := Open(nested)
	if err != nil || up.Dir != dir {
		t.Fatalf("walk-up %v %s", err, up.Dir)
	}
}

func TestRejectsBadStatus(t *testing.T) {
	dir := t.TempDir()
	p, err := Init(dir, "X", "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.CreateTask(Task{Title: "A", Status: "Nope"}); err == nil {
		t.Fatal("expected invalid status")
	}
}

func TestToolCitesThePack(t *testing.T) {
	dir := t.TempDir()
	p, err := Init(dir, "Fixture", "fix")
	if err != nil {
		t.Fatal(err)
	}
	tool := p.Tool()
	if tool.Name != ToolName || tool.Kind != "surface" || tool.Direction != "talk" {
		t.Fatalf("tool %+v", tool)
	}
	if tool.Counterpart != "human" {
		t.Fatal("surface counterpart is human")
	}
	if tool.As != "editor" {
		t.Fatal("this surface is the docket editor")
	}
	if tool.Cites != "docket:fix" {
		t.Fatalf("cites %q", tool.Cites)
	}
}

func TestDoesNotTreatMarkdownDocketAsPack(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docket.json"), []byte(`{"id":"md"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if IsPack(dir) {
		t.Fatal("bare docket.json is docket-md shape, not a prim")
	}
}

