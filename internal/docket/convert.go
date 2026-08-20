package docket

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type mdConfig struct {
	ID      string `json:"id"`
	Project string `json:"project"`
}

func FindMarkdownDocket(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	if isMarkdownDocket(dir) {
		return dir, nil
	}
	nested := filepath.Join(dir, ".docket")
	if isMarkdownDocket(nested) {
		return nested, nil
	}
	for {
		candidate := filepath.Join(dir, ".docket")
		if isMarkdownDocket(candidate) {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no docket-md .docket here (looked up from %s)", start)
		}
		dir = parent
	}
}

func isMarkdownDocket(dir string) bool {
	return fileExists(filepath.Join(dir, "docket.json")) &&
		(dirExists(filepath.Join(dir, "tasks")) || dirExists(filepath.Join(dir, "completed")))
}

func ConvertMarkdown(from, dest string) (*Pack, error) {
	src, err := FindMarkdownDocket(from)
	if err != nil {
		return nil, err
	}
	if dest == "" {
		dest = filepath.Join(filepath.Dir(src), "docket.prim")
	}
	dest, err = filepath.Abs(dest)
	if err != nil {
		return nil, err
	}
	if IsPack(dest) {
		return nil, fmt.Errorf("already a docket-prim pack: %s", dest)
	}

	cfg, err := readMarkdownConfig(src)
	if err != nil {
		return nil, err
	}
	name := cfg.Project
	if name == "" {
		name = filepath.Base(filepath.Dir(src))
	}
	p, err := Init(dest, name, cfg.ID)
	if err != nil {
		return nil, err
	}

	tasks, err := readMarkdownTasks(filepath.Join(src, "tasks"))
	if err != nil {
		return nil, err
	}
	completed, err := readMarkdownTasks(filepath.Join(src, "completed"))
	if err != nil {
		return nil, err
	}
	for i := range completed {
		if completed[i].Status == "" {
			completed[i].Status = "Done"
		}
	}
	archived, err := readMarkdownTasks(filepath.Join(src, "archive"))
	if err != nil {
		return nil, err
	}
	miles, err := readMarkdownMilestones(filepath.Join(src, "milestones"))
	if err != nil {
		return nil, err
	}

	p.Tasks = append(tasks, completed...)
	p.Archive = archived
	p.Milestones = miles
	for i := range p.Tasks {
		if err := fillUID(&p.Tasks[i].UID); err != nil {
			return nil, fmt.Errorf("convert %s: %w", p.Tasks[i].ID, err)
		}
	}
	for i := range p.Archive {
		if err := fillUID(&p.Archive[i].UID); err != nil {
			return nil, fmt.Errorf("convert archive %s: %w", p.Archive[i].ID, err)
		}
	}
	for i := range p.Milestones {
		if err := fillUID(&p.Milestones[i].UID); err != nil {
			return nil, fmt.Errorf("convert %s: %w", p.Milestones[i].ID, err)
		}
	}
	for _, t := range p.Tasks {
		if err := CheckCard(t); err != nil {
			return nil, fmt.Errorf("convert %s: %w", t.ID, err)
		}
		if err := p.CheckGuard(t); err != nil {
			return nil, fmt.Errorf("convert %s: %w", t.ID, err)
		}
	}
	if err := p.persistRows(); err != nil {
		return nil, err
	}
	if err := p.Log(fmt.Sprintf("converted from %s (%d tasks, %d milestones)", src, len(p.Tasks), len(p.Milestones))); err != nil {
		return nil, err
	}
	return p, nil
}

func readMarkdownConfig(src string) (mdConfig, error) {
	var cfg mdConfig
	b, err := os.ReadFile(filepath.Join(src, "docket.json"))
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func readMarkdownTasks(dir string) ([]Task, error) {
	names, err := listMarkdown(dir)
	if err != nil {
		return nil, err
	}
	var out []Task
	for _, name := range names {
		fm, notes, err := readFrontmatter(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		t := taskFromMap(fm)
		t.Notes = notes
		if t.ID == "" {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func readMarkdownMilestones(dir string) ([]Milestone, error) {
	names, err := listMarkdown(dir)
	if err != nil {
		return nil, err
	}
	var out []Milestone
	for _, name := range names {
		fm, _, err := readFrontmatter(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		m := Milestone{
			ID:      asString(fm["id"]),
			Title:   asString(fm["title"]),
			Status:  asString(fm["status"]),
			Due:     asString(fm["due"]),
			Created: asString(fm["created"]),
			Updated: asString(fm["updated"]),
		}
		if m.ID == "" {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func taskFromMap(fm map[string]any) Task {
	pri := strings.ToLower(asString(fm["priority"]))
	if pri != "" && !Priorities[pri] {
		pri = ""
	}
	status := asString(fm["status"])
	if status == "" {
		status = "To Do"
	}
	return Task{
		ID:            asString(fm["id"]),
		Title:         asString(fm["title"]),
		Type:          asString(fm["type"]),
		Status:        status,
		Priority:      pri,
		Milestone:     asString(fm["milestone"]),
		Parent:        asString(fm["parent"]),
		Subtasks:      asStrings(fm["subtasks"]),
		Assignees:     asStrings(fm["assignees"]),
		Tags:          asStrings(fm["tags"]),
		Dependencies:  asStrings(fm["dependencies"]),
		Requirements:  firstStrings(fm, "requirements"),
		TestCases:     firstStrings(fm, "test-cases", "test_cases"),
		Acceptance:    firstStrings(fm, "acceptance-criteria", "acceptance_criteria"),
		DOD:           firstStrings(fm, "definition-of-done", "definition_of_done"),
		BlockedReason: asString(fm["blocked_reason"]),
		Created:       asString(fm["created"]),
		Updated:       asString(fm["updated"]),
		Start:         asString(fm["start"]),
		Due:           asString(fm["due"]),
	}
}

func readFrontmatter(path string) (map[string]any, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	raw := string(b)
	if !strings.HasPrefix(raw, "---\n") {
		return map[string]any{}, strings.TrimSpace(raw), nil
	}
	rest := raw[4:]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return nil, "", fmt.Errorf("unclosed frontmatter")
	}
	var fm map[string]any
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return nil, "", err
	}
	if fm == nil {
		fm = map[string]any{}
	}
	notes := strings.TrimSpace(rest[end+5:])
	return fm, notes, nil
}

func listMarkdown(dir string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, ent := range ents {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".md") {
			continue
		}
		names = append(names, ent.Name())
	}
	return names, nil
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func asString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

func asStrings(v any) []string {
	switch x := v.(type) {
	case nil:
		return nil
	case string:
		if strings.TrimSpace(x) == "" {
			return nil
		}
		return []string{x}
	case []any:
		var out []string
		for _, item := range x {
			s := asString(item)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return x
	default:
		return nil
	}
}

func firstStrings(fm map[string]any, keys ...string) []string {
	for _, k := range keys {
		if s := asStrings(fm[k]); len(s) > 0 {
			return s
		}
	}
	return nil
}
