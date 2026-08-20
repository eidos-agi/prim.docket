package docket

import (
	"bytes"
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

	in := live("Ship login")
	in.Priority = "high"
	a, err := p.CreateTask(in)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "TASK-0001" || a.Status != "To Do" {
		t.Fatalf("create %+v", a)
	}
	if err := CheckUID(a.UID); err != nil {
		t.Fatal(err)
	}

	b, err := p.CreateTask(live("Write docs"))
	if err != nil {
		t.Fatal(err)
	}
	if b.ID != "TASK-0002" {
		t.Fatalf("id %s", b.ID)
	}
	if err := CheckUID(b.UID); err != nil {
		t.Fatal(err)
	}
	if a.UID == b.UID {
		t.Fatal("uid collision")
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
	if got.UID != a.UID {
		t.Fatalf("uid changed %s -> %s", a.UID, got.UID)
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
	if _, err := p.CreateTask(Task{Title: "A", Status: "Nope", Notes: "x that is not A", Acceptance: []string{"y"}}); err == nil {
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

func TestRevMovesOnWrite(t *testing.T) {
	dir := t.TempDir()
	p, err := Init(dir, "Fixture", "fix")
	if err != nil {
		t.Fatal(err)
	}
	before := Rev(dir)
	if before == "" {
		t.Fatal("empty rev")
	}
	if _, err := p.CreateTask(live("Watch me")); err != nil {
		t.Fatal(err)
	}
	after := Rev(dir)
	if after == before {
		t.Fatal("rev did not move after write")
	}
}

func TestCreateTaskRejectsTitleOnly(t *testing.T) {
	dir := t.TempDir()
	p, err := Init(dir, "Fixture", "fix")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.CreateTask(Task{Title: "Ship login"}); err == nil || !bytes.Contains([]byte(err.Error()), []byte("notes required")) {
		t.Fatalf("got %v", err)
	}
}

func TestJSONLAppendOnlyLastLineWins(t *testing.T) {
	dir := t.TempDir()
	p, err := Init(dir, "Fixture", "fix")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.CreateTask(live("Ship login")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "tasks.jsonl")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	n1 := bytes.Count(before, []byte("\n"))
	if n1 < 1 {
		t.Fatalf("expected create line, got %d", n1)
	}
	title := "Ship login now"
	if _, err := p.EditTask("TASK-0001", TaskEdit{Title: &title}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	n2 := bytes.Count(after, []byte("\n"))
	if n2 <= n1 {
		t.Fatalf("edit rewrote instead of append: %d -> %d", n1, n2)
	}
	if !bytes.Contains(after, []byte("Ship login")) || !bytes.Contains(after, []byte("Ship login now")) {
		t.Fatalf("expected both titles in file:\n%s", after)
	}
	re, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, _, ok := re.Task("TASK-0001")
	if !ok || got.Title != title {
		t.Fatalf("fold %+v", got)
	}
	if len(re.Tasks) != 1 {
		t.Fatalf("live count %d", len(re.Tasks))
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

func TestStartDueRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p, err := Init(dir, "Fixture", "fix")
	if err != nil {
		t.Fatal(err)
	}
	in := live("Ship login")
	in.Start = "2026-08-20"
	in.Due = "2026-09-01"
	a, err := p.CreateTask(in)
	if err != nil {
		t.Fatal(err)
	}
	if a.Start != "2026-08-20" || a.Due != "2026-09-01" {
		t.Fatalf("create %+v", a)
	}
	due := "2026-09-15"
	if _, err := p.EditTask(a.ID, TaskEdit{Due: &due}); err != nil {
		t.Fatal(err)
	}
	re, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, _, ok := re.Task(a.ID)
	if !ok || got.Start != "2026-08-20" || got.Due != "2026-09-15" {
		t.Fatalf("reload %+v", got)
	}
	bad := live("Bad date")
	bad.Due = "08/20/2026"
	if _, err := p.CreateTask(bad); err == nil {
		t.Fatal("expected bad due")
	}
}

func TestAppendJSONLCanonicalLine(t *testing.T) {
	row := live("Ship login")
	row.ID = "TASK-0001"
	uid, err := NewUID()
	if err != nil {
		t.Fatal(err)
	}
	row.UID = uid
	row.Status = "To Do"
	path := filepath.Join(t.TempDir(), "tasks.jsonl")
	if err := appendJSONL(path, row); err != nil {
		t.Fatal(err)
	}
	if err := appendJSONL(path, row); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace(b), []byte("\n"))
	if len(lines) != 2 || !bytes.Equal(lines[0], lines[1]) {
		t.Fatalf("encoder not stable:\n%s\n%s", lines[0], lines[1])
	}
}

func TestSizeOfCountsPackFiles(t *testing.T) {
	dir := t.TempDir()
	p, err := Init(dir, "Fixture", "fix")
	if err != nil {
		t.Fatal(err)
	}
	before := SizeOf(p.Dir)
	if before.Bytes <= 0 {
		t.Fatalf("empty pack size %+v", before)
	}
	if _, err := p.CreateTask(live("Ship login")); err != nil {
		t.Fatal(err)
	}
	after := SizeOf(p.Dir)
	if after.JSONL <= before.JSONL || after.Bytes <= before.Bytes {
		t.Fatalf("size did not grow after create: before %+v after %+v", before, after)
	}
	if after.JSONL != after.Files["tasks.jsonl"] {
		t.Fatalf("jsonl mismatch %+v", after)
	}
}

func TestSearchHitsEveryCardField(t *testing.T) {
	dir := t.TempDir()
	p, err := Init(dir, "Fixture", "fix")
	if err != nil {
		t.Fatal(err)
	}
	parent, err := p.CreateTask(live("Parent outcome named here"))
	if err != nil {
		t.Fatal(err)
	}
	child := live("Child work named here")
	child.Parent = parent.ID
	child.Tags = []string{"zxqv-unique-tag"}
	child.BlockedReason = "waiting-on-zxqv-token"
	child.Start = "2026-03-14"
	child.Assignees = []string{"zxqv-assignee"}
	got, err := p.CreateTask(child)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"zxqv-unique-tag", "waiting-on-zxqv-token", "2026-03-14", "zxqv-assignee"} {
		hits := p.Search(q)
		if len(hits) != 1 || hits[0].ID != got.ID {
			t.Fatalf("search %q got %+v", q, hits)
		}
	}
	hits := p.Search(parent.ID)
	saw := false
	for _, h := range hits {
		if h.ID == got.ID {
			saw = true
		}
	}
	if !saw {
		t.Fatalf("parent id search missed child: %+v", hits)
	}
	if hits := p.Search("Child work"); len(hits) != 1 {
		t.Fatalf("title search %d", len(hits))
	}
}

