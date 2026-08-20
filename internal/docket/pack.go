package docket

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const Version = "0.1.0"
const Profile = "docket"

// StoreFiles are the pack files Rev and SizeOf stamp. No parse.
var StoreFiles = []string{"docket.json", "tasks.jsonl", "milestones.jsonl", "archive.jsonl", "index.md", "log.md"}

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Pack struct {
	Dir        string
	Project    Project
	Tasks      []Task
	Milestones []Milestone
	Archive    []Task
}

func Today() string {
	return time.Now().Format("2006-01-02")
}

func IsPack(dir string) bool {
	if fileExists(filepath.Join(dir, "tasks.jsonl")) {
		return true
	}
	face := filepath.Join(dir, "index.md")
	if !fileExists(face) {
		return false
	}
	b, err := os.ReadFile(face)
	if err != nil {
		return false
	}
	return bytes.Contains(b, []byte("profile: docket"))
}

func Find(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if IsPack(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no docket-prim pack here (looked up from %s)", start)
		}
		dir = parent
	}
}

func Open(dir string) (*Pack, error) {
	root, err := Find(dir)
	if err != nil {
		return nil, err
	}
	p := &Pack{Dir: root}
	if err := readJSON(filepath.Join(root, "docket.json"), &p.Project); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if p.Tasks, err = readTasks(filepath.Join(root, "tasks.jsonl")); err != nil {
		return nil, err
	}
	if p.Archive, err = readArchive(filepath.Join(root, "archive.jsonl")); err != nil {
		return nil, err
	}
	if p.Milestones, err = readMilestones(filepath.Join(root, "milestones.jsonl")); err != nil {
		return nil, err
	}
	return p, nil
}

func Init(dir, name, id string) (*Pack, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if IsPack(root) {
		return nil, fmt.Errorf("already a docket-prim pack: %s", root)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	if name == "" {
		name = filepath.Base(root)
	}
	if id == "" {
		id = slug(name)
	}
	p := &Pack{
		Dir:        root,
		Project:    Project{ID: id, Name: name},
		Tasks:      []Task{},
		Milestones: []Milestone{},
		Archive:    []Task{},
	}
	face := fmt.Sprintf("---\nprofile: %s\ndocket_version: %q\ntype: queue\ntitle: %s\nstatus: open\n---\n\n# %s\n\nA docket-prim pack. Not markdown tasks.\n",
		Profile, Version, name, name)
	if err := os.WriteFile(filepath.Join(root, "index.md"), []byte(face), 0o644); err != nil {
		return nil, err
	}
	if err := writeJSON(filepath.Join(root, "docket.json"), p.Project); err != nil {
		return nil, err
	}
	if err := writeJSONL(filepath.Join(root, "tasks.jsonl"), []Task{}); err != nil {
		return nil, err
	}
	if err := writeJSONL(filepath.Join(root, "milestones.jsonl"), []Milestone{}); err != nil {
		return nil, err
	}
	if err := writeJSONL(filepath.Join(root, "archive.jsonl"), []Task{}); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(root, "log.md"), []byte("# Log\n\n"), 0o644); err != nil {
		return nil, err
	}
	if err := p.Log("init " + name); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Pack) Save() error {
	return writeJSON(filepath.Join(p.Dir, "docket.json"), p.Project)
}

func (p *Pack) appendTask(t Task) error {
	return appendJSONL(filepath.Join(p.Dir, "tasks.jsonl"), t)
}

func (p *Pack) appendMilestone(m Milestone) error {
	return appendJSONL(filepath.Join(p.Dir, "milestones.jsonl"), m)
}

func (p *Pack) appendArchive(t Task) error {
	return appendJSONL(filepath.Join(p.Dir, "archive.jsonl"), t)
}

func (p *Pack) persistRows() error {
	if err := p.Save(); err != nil {
		return err
	}
	for _, t := range p.Tasks {
		if err := p.appendTask(t); err != nil {
			return err
		}
	}
	for _, t := range p.Archive {
		if err := p.appendArchive(t); err != nil {
			return err
		}
	}
	for _, m := range p.Milestones {
		if err := p.appendMilestone(m); err != nil {
			return err
		}
	}
	return nil
}

// Size is on-disk bytes for the pack files. jsonl is tasks.jsonl alone.
type Size struct {
	Bytes int64            `json:"bytes"`
	JSONL int64            `json:"jsonl"`
	Files map[string]int64 `json:"files"`
}

// SizeOf sums StoreFiles. Missing files count as zero.
func SizeOf(dir string) Size {
	out := Size{Files: map[string]int64{}}
	for _, name := range StoreFiles {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		out.Files[name] = st.Size()
		out.Bytes += st.Size()
		if name == "tasks.jsonl" {
			out.JSONL = st.Size()
		}
	}
	return out
}

// Rev is a cheap pack stamp: mtime+size of the store files. No parse.
func Rev(dir string) string {
	var b strings.Builder
	for _, name := range StoreFiles {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "%s:%d:%d;", name, st.ModTime().UnixNano(), st.Size())
	}
	return b.String()
}

func (p *Pack) Log(line string) error {
	path := filepath.Join(p.Dir, "log.md")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "- %s — %s\n", Today(), line)
	return err
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular()
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "docket"
	}
	return out
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return writeAtomic(path, b)
}

func writeJSONL[T any](path string, rows []T) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	return writeAtomic(path, buf.Bytes())
}

// appendJSONL is the only legal jsonl write. Pack.appendTask / appendMilestone /
// appendArchive (planner, task-*, milestone-*, convert, editor) call it.
// Agents never write these files. Encode into a buffer then one Write so a
// crash cannot leave a truncated line.
func appendJSONL(path string, row any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(row); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(buf.Bytes())
	return err
}

func writeAtomic(path string, body []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readJSON(path string, dest any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}

func readJSONL[T any](path string) ([]T, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return []T{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rows []T
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var row T
		if err := json.Unmarshal(line, &row); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		rows = append(rows, row)
	}
	return rows, sc.Err()
}

func readTasks(path string) ([]Task, error) {
	raw, err := readJSONL[Task](path)
	if err != nil {
		return nil, err
	}
	order := make([]string, 0, len(raw))
	by := make(map[string]Task, len(raw))
	for _, t := range raw {
		if t.ID == "" {
			continue
		}
		if _, ok := by[t.ID]; !ok {
			order = append(order, t.ID)
		}
		by[t.ID] = t
	}
	out := make([]Task, 0, len(order))
	for _, id := range order {
		t := by[id]
		if t.Archived {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func readMilestones(path string) ([]Milestone, error) {
	raw, err := readJSONL[Milestone](path)
	if err != nil {
		return nil, err
	}
	order := make([]string, 0, len(raw))
	by := make(map[string]Milestone, len(raw))
	for _, m := range raw {
		if m.ID == "" {
			continue
		}
		if _, ok := by[m.ID]; !ok {
			order = append(order, m.ID)
		}
		by[m.ID] = m
	}
	out := make([]Milestone, 0, len(order))
	for _, id := range order {
		out = append(out, by[id])
	}
	return out, nil
}

func readArchive(path string) ([]Task, error) {
	raw, err := readJSONL[Task](path)
	if err != nil {
		return nil, err
	}
	order := make([]string, 0, len(raw))
	by := make(map[string]Task, len(raw))
	for _, t := range raw {
		if t.ID == "" {
			continue
		}
		if _, ok := by[t.ID]; !ok {
			order = append(order, t.ID)
		}
		by[t.ID] = t
	}
	out := make([]Task, 0, len(order))
	for _, id := range order {
		out = append(out, by[id])
	}
	return out, nil
}
