package docket

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	minTitle = 8
	minLine  = 24
)

// FollowUps is how many times the authoring agent may fix-and-re-review
// after the first qualitative pass. Then remaining FAILs go to the human.
const FollowUps = 2

type Finding struct {
	Code  string `json:"code"`
	ID    string `json:"id,omitempty"`
	Field string `json:"field,omitempty"`
	Msg   string `json:"msg"`
}

var placeholderRE = regexp.MustCompile(`(?i)^(todo|tbd|tba|tk|fixme|xxx+|n/?a|n\.a\.|none|null|nil|placeholder|lorem(\s+ipsum)?|asdf+|test|temp|wip)[\s.!?]*$`)

func isPlaceholder(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if placeholderRE.MatchString(s) {
		return true
	}
	low := strings.ToLower(s)
	if strings.Contains(low, "lorem ipsum") || strings.Contains(low, "[todo]") || strings.Contains(low, "[tbd]") {
		return true
	}
	if strings.HasPrefix(low, "todo:") || strings.HasPrefix(low, "tbd:") || strings.HasPrefix(low, "fixme:") {
		return true
	}
	return false
}

func FindingsError(fs []Finding) error {
	if len(fs) == 0 {
		return nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "schema: %d findings", len(fs))
	for _, f := range fs {
		b.WriteByte('\n')
		if f.ID != "" {
			b.WriteString(f.ID)
			b.WriteString(": ")
		}
		if f.Field != "" {
			b.WriteString(f.Field)
			b.WriteString(": ")
		}
		b.WriteString(f.Msg)
	}
	return fmt.Errorf("%s", b.String())
}

// BodyFindings is the quantitative card bar: blank, too short, placeholder.
func BodyFindings(t Task) []Finding {
	var fs []Finding
	id := t.ID
	add := func(code, field, msg string) {
		fs = append(fs, Finding{Code: code, ID: id, Field: field, Msg: msg})
	}
	title := strings.TrimSpace(t.Title)
	if title == "" {
		add("title-blank", "title", "title required")
	} else if utf8.RuneCountInString(title) < minTitle {
		add("title-short", "title", fmt.Sprintf("title too short (need ≥%d characters)", minTitle))
	} else if isPlaceholder(title) {
		add("title-placeholder", "title", "title looks like a placeholder")
	}

	notes := strings.TrimSpace(t.Notes)
	if notes == "" {
		add("notes-blank", "notes", "notes required (user story or technical brief)")
	} else if strings.EqualFold(notes, title) {
		add("notes-title", "notes", "notes must be a description, not the title")
	} else if utf8.RuneCountInString(notes) < minNotes || sentences(notes) < 2 {
		add("notes-terse", "notes", fmt.Sprintf("notes too terse (need a user story or technical brief, ≥%d characters, ≥2 sentences)", minNotes))
	} else if isPlaceholder(notes) {
		add("notes-placeholder", "notes", "notes look like a placeholder")
	}

	fs = append(fs, listFindings(id, "requirements", "req", t.Requirements, 2, "requirements required (≥2)")...)
	needCases := 1
	caseMsg := "test-cases required"
	if NormType(t.Type) == "TEST" {
		needCases = 2
		caseMsg = "test-cases required (≥2 for TEST)"
	}
	fs = append(fs, listFindings(id, "test-cases", "cases", t.TestCases, needCases, caseMsg)...)
	fs = append(fs, listFindings(id, "acceptance-criteria", "accept", t.Acceptance, 2, "acceptance-criteria required (≥2)")...)
	return fs
}

func listFindings(id, field, code string, rows []string, minN int, missing string) []Finding {
	var fs []Finding
	add := func(c, f, msg string) {
		fs = append(fs, Finding{Code: c, ID: id, Field: f, Msg: msg})
	}
	got := filled(rows)
	if len(got) < minN {
		add(code+"-missing", field, missing)
		return fs
	}
	seen := map[string]int{}
	for i, line := range got {
		key := strings.ToLower(line)
		slot := fmt.Sprintf("%s[%d]", field, i)
		if utf8.RuneCountInString(line) < minLine {
			add(code+"-short", slot, fmt.Sprintf("%s too short (need ≥%d characters)", field, minLine))
		} else if isPlaceholder(line) {
			add(code+"-placeholder", slot, field+" looks like a placeholder")
		}
		if prev, ok := seen[key]; ok {
			add(code+"-dup", slot, fmt.Sprintf("duplicate %s (same as [%d])", field, prev))
		} else {
			seen[key] = i
		}
	}
	return fs
}

// SchemaFindings is the quantitative pack check. Machine, collect-all.
// Blank, too short, placeholder, types, nesting, refs, uids. Not meaning.
func SchemaFindings(p *Pack) []Finding {
	var fs []Finding
	if p == nil {
		return []Finding{{Code: "nil", Msg: "nil pack"}}
	}
	add := func(f Finding) { fs = append(fs, f) }
	face := filepath.Join(p.Dir, "index.md")
	b, err := os.ReadFile(face)
	if err != nil {
		add(Finding{Code: "face", Field: "index.md", Msg: "face: " + err.Error()})
	} else if !strings.Contains(string(b), "profile: docket") {
		add(Finding{Code: "face-profile", Field: "index.md", Msg: "face: missing profile: docket"})
	}
	if strings.TrimSpace(p.Project.Name) == "" {
		add(Finding{Code: "project-name", Field: "name", Msg: "project name required"})
	}
	if len(p.Tasks) == 0 {
		add(Finding{Code: "no-tasks", Msg: "no tasks"})
	}

	byID := map[string]Task{}
	titles := map[string]string{}
	for _, t := range p.Tasks {
		fs = append(fs, BodyFindings(t)...)
		if err := CheckType(t.Type); err != nil {
			add(Finding{Code: "type-invalid", ID: t.ID, Field: "type", Msg: err.Error()})
		}
		if err := CheckStatus(t.Status); err != nil {
			add(Finding{Code: "status-invalid", ID: t.ID, Field: "status", Msg: err.Error()})
		}
		if err := CheckPriority(t.Priority); err != nil {
			add(Finding{Code: "priority-invalid", ID: t.ID, Field: "priority", Msg: err.Error()})
		}
		if err := CheckDay(t.Start); err != nil {
			add(Finding{Code: "start-bad", ID: t.ID, Field: "start", Msg: "start: " + err.Error()})
		}
		if err := CheckDay(t.Due); err != nil {
			add(Finding{Code: "due-bad", ID: t.ID, Field: "due", Msg: "due: " + err.Error()})
		}
		if err := CheckUID(t.UID); err != nil {
			add(Finding{Code: "uid-missing", ID: t.ID, Field: "uid", Msg: err.Error()})
		}
		if _, ok := byID[t.ID]; ok {
			add(Finding{Code: "dup-id", ID: t.ID, Msg: "duplicate " + t.ID})
		}
		byID[t.ID] = t
		key := strings.ToLower(strings.TrimSpace(t.Title))
		if key != "" {
			if other, ok := titles[key]; ok {
				add(Finding{Code: "dup-title", ID: t.ID, Field: "title", Msg: "duplicate title (also " + other + ")"})
			} else {
				titles[key] = t.ID
			}
		}
	}

	seenUID := map[string]string{}
	for _, t := range p.Tasks {
		if strings.TrimSpace(t.UID) == "" {
			continue
		}
		k := strings.ToLower(t.UID)
		if other, ok := seenUID[k]; ok {
			add(Finding{Code: "dup-uid", ID: t.ID, Field: "uid", Msg: "duplicate uid (also " + other + ")"})
		} else {
			seenUID[k] = t.ID
		}
	}
	for _, t := range p.Archive {
		if strings.TrimSpace(t.UID) == "" {
			continue
		}
		if err := CheckUID(t.UID); err != nil {
			add(Finding{Code: "uid-missing", ID: t.ID, Field: "uid", Msg: "archive " + err.Error()})
		}
		k := strings.ToLower(t.UID)
		if other, ok := seenUID[k]; ok && other != t.ID {
			add(Finding{Code: "dup-uid", ID: t.ID, Field: "uid", Msg: "archive duplicate uid (also " + other + ")"})
		} else {
			seenUID[k] = t.ID
		}
	}

	ms := map[string]bool{}
	for _, m := range p.Milestones {
		if strings.TrimSpace(m.Title) == "" {
			add(Finding{Code: "mile-title", ID: m.ID, Field: "title", Msg: "title required"})
		} else if utf8.RuneCountInString(strings.TrimSpace(m.Title)) < minTitle {
			add(Finding{Code: "mile-title-short", ID: m.ID, Field: "title", Msg: fmt.Sprintf("title too short (need ≥%d characters)", minTitle)})
		} else if isPlaceholder(m.Title) {
			add(Finding{Code: "mile-placeholder", ID: m.ID, Field: "title", Msg: "title looks like a placeholder"})
		}
		if err := CheckDay(m.Due); err != nil {
			add(Finding{Code: "mile-due", ID: m.ID, Field: "due", Msg: "due: " + err.Error()})
		}
		if err := CheckUID(m.UID); err != nil {
			add(Finding{Code: "uid-missing", ID: m.ID, Field: "uid", Msg: err.Error()})
		} else {
			k := strings.ToLower(m.UID)
			if other, ok := seenUID[k]; ok {
				add(Finding{Code: "dup-uid", ID: m.ID, Field: "uid", Msg: "duplicate uid (also " + other + ")"})
			} else {
				seenUID[k] = m.ID
			}
		}
		ms[m.ID] = true
	}

	for _, t := range p.Tasks {
		if t.Parent != "" {
			if t.Parent == t.ID {
				add(Finding{Code: "parent-self", ID: t.ID, Field: "parent", Msg: "parent is self"})
			} else if !strings.HasPrefix(t.Parent, "TASK-") {
				add(Finding{Code: "parent-unresolved", ID: t.ID, Field: "parent", Msg: fmt.Sprintf("unresolved parent %q", t.Parent)})
			} else if _, ok := byID[t.Parent]; !ok {
				add(Finding{Code: "parent-dangle", ID: t.ID, Field: "parent", Msg: "dangling parent " + t.Parent})
			}
		}
		if err := checkGuardParent(t, byID); err != nil {
			add(Finding{Code: "guard-parent", ID: t.ID, Field: "parent", Msg: err.Error()})
		}
		var par *Task
		if t.Parent != "" {
			if row, ok := byID[t.Parent]; ok {
				par = &row
			}
		}
		if err := CheckNest(t, par); err != nil {
			add(Finding{Code: "nest", ID: t.ID, Field: "parent", Msg: err.Error()})
		}
		if err := checkValidationParent(t, byID); err != nil {
			add(Finding{Code: "validation-parent", ID: t.ID, Field: "parent", Msg: err.Error()})
		}
		if t.Milestone != "" && !ms[t.Milestone] {
			add(Finding{Code: "milestone-unknown", ID: t.ID, Field: "milestone", Msg: fmt.Sprintf("unknown milestone %q", t.Milestone)})
		}
		for _, d := range t.Dependencies {
			if d == "" {
				continue
			}
			if _, ok := byID[d]; !ok {
				add(Finding{Code: "dep-dangle", ID: t.ID, Field: "dependencies", Msg: "dangling dependency " + d})
			}
		}
	}
	if id := parentCycle(byID); id != "" {
		add(Finding{Code: "cycle", ID: id, Field: "parent", Msg: "parent cycle"})
	}
	for _, f := range validationFindings(byID) {
		add(f)
	}
	return fs
}
