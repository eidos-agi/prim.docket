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
	if p.Archive, err = readTasks(filepath.Join(root, "archive.jsonl")); err != nil {
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
	if err := writeJSON(filepath.Join(p.Dir, "docket.json"), p.Project); err != nil {
		return err
	}
	if err := writeJSONL(filepath.Join(p.Dir, "tasks.jsonl"), p.Tasks); err != nil {
		return err
	}
	if err := writeJSONL(filepath.Join(p.Dir, "milestones.jsonl"), p.Milestones); err != nil {
		return err
	}
	return writeJSONL(filepath.Join(p.Dir, "archive.jsonl"), p.Archive)
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

func readTasks(path string) ([]Task, error) { return readJSONL[Task](path) }

func readMilestones(path string) ([]Milestone, error) {
	return readJSONL[Milestone](path)
}
