package docket

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var Statuses = map[string]bool{
	"To Do":       true,
	"In Progress": true,
	"Done":        true,
	"Draft":       true,
}

var Priorities = map[string]bool{
	"":       true,
	"high":   true,
	"medium": true,
	"low":    true,
}

var Types = map[string]bool{
	"":      true,
	"TASK":  true,
	"GOAL":  true,
	"PLAN":  true,
	"GUARD":      true,
	"TEST":       true,
	"VALIDATION": true,
}

type Task struct {
	ID            string   `json:"id"`
	UID           string   `json:"uid"`
	Title         string   `json:"title"`
	Type          string   `json:"type,omitempty"`
	Status        string   `json:"status"`
	Priority      string   `json:"priority,omitempty"`
	Milestone     string   `json:"milestone,omitempty"`
	Parent        string   `json:"parent,omitempty"`
	Subtasks      []string `json:"subtasks,omitempty"`
	Assignees     []string `json:"assignees,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Dependencies  []string `json:"dependencies,omitempty"`
	Requirements  []string `json:"requirements"`
	TestCases     []string `json:"test-cases"`
	Acceptance    []string `json:"acceptance-criteria"`
	DOD           []string `json:"definition-of-done,omitempty"`
	BlockedReason string   `json:"blocked_reason,omitempty"`
	Created       string   `json:"created,omitempty"`
	Updated       string   `json:"updated,omitempty"`
	Start         string   `json:"start,omitempty"`
	Due           string   `json:"due,omitempty"`
	Notes         string   `json:"notes"`
	Archived      bool     `json:"archived,omitempty"`
}

func (t Task) Line() string {
	badge := "○"
	if t.BlockedReason != "" {
		badge = "⛔"
	} else if t.Status == "Done" {
		badge = "✓"
	} else if t.Status == "In Progress" {
		badge = "▶"
	} else if t.Status == "Draft" {
		badge = "·"
	}
	pri := ""
	if t.Priority != "" {
		pri = " [" + t.Priority + "]"
	}
	blocked := ""
	if t.BlockedReason != "" {
		blocked = " — BLOCKED: " + t.BlockedReason
	}
	return fmt.Sprintf("%s %s%s — %s%s", badge, t.ID, pri, t.Title, blocked)
}

func CheckStatus(s string) error {
	if !Statuses[s] {
		return fmt.Errorf("invalid status %q (To Do | In Progress | Done | Draft)", s)
	}
	return nil
}

func CheckPriority(s string) error {
	if !Priorities[s] {
		return fmt.Errorf("invalid priority %q (high | medium | low)", s)
	}
	return nil
}

func NormType(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return "TASK"
	}
	return s
}

func CheckDay(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return fmt.Errorf("date %q must be YYYY-MM-DD", s)
	}
	return nil
}

func CheckType(s string) error {
	s = NormType(s)
	if !Types[s] {
		return fmt.Errorf("invalid type %q (TASK | GOAL | PLAN | GUARD | TEST | VALIDATION)", s)
	}
	return nil
}

const minNotes = 160

// CheckCard is the body bar: a title is not a card. A card is a user story
// or technical brief with requirements, test cases, and acceptance.
func CheckCard(t Task) error {
	fs := BodyFindings(t)
	if len(fs) == 0 {
		return nil
	}
	return fmt.Errorf("%s", fs[0].Msg)
}

func filled(rows []string) []string {
	var out []string
	for _, s := range rows {
		if strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

func sentences(s string) int {
	n := 0
	for _, r := range s {
		if r == '.' || r == '!' || r == '?' {
			n++
		}
	}
	return n
}

func (p *Pack) CheckGuard(t Task) error {
	if NormType(t.Type) != "GUARD" {
		return nil
	}
	if strings.TrimSpace(t.Parent) == "" {
		return fmt.Errorf("GUARD requires parent (the GOAL, PLAN, or TASK it constrains)")
	}
	target, _, ok := p.Task(t.Parent)
	if !ok {
		return fmt.Errorf("GUARD parent %s not found", t.Parent)
	}
	if t.ID != "" && target.ID == t.ID {
		return fmt.Errorf("GUARD parent is self")
	}
	switch NormType(target.Type) {
	case "GOAL", "PLAN", "TASK":
		return nil
	default:
		return fmt.Errorf("GUARD parent must be GOAL, PLAN, or TASK (not %s)", NormType(target.Type))
	}
}

func (p *Pack) NextTaskID() string {
	n := 0
	for _, t := range append(append([]Task{}, p.Tasks...), p.Archive...) {
		var i int
		if _, err := fmt.Sscanf(t.ID, "TASK-%d", &i); err == nil && i > n {
			n = i
		}
	}
	return fmt.Sprintf("TASK-%04d", n+1)
}

func (p *Pack) Task(id string) (*Task, int, bool) {
	for i := range p.Tasks {
		if p.Tasks[i].ID == id {
			return &p.Tasks[i], i, true
		}
	}
	return nil, -1, false
}

func (p *Pack) CreateTask(in Task) (Task, error) {
	if in.Status == "" {
		in.Status = "To Do"
	}
	if err := CheckStatus(in.Status); err != nil {
		return Task{}, err
	}
	if err := CheckPriority(in.Priority); err != nil {
		return Task{}, err
	}
	in.Type = NormType(in.Type)
	if err := CheckType(in.Type); err != nil {
		return Task{}, err
	}
	in.Start = strings.TrimSpace(in.Start)
	in.Due = strings.TrimSpace(in.Due)
	if err := CheckDay(in.Start); err != nil {
		return Task{}, fmt.Errorf("start: %w", err)
	}
	if err := CheckDay(in.Due); err != nil {
		return Task{}, fmt.Errorf("due: %w", err)
	}
	if err := CheckCard(in); err != nil {
		return Task{}, err
	}
	if err := p.CheckGuard(in); err != nil {
		return Task{}, err
	}
	if err := p.CheckValidation(in); err != nil {
		return Task{}, err
	}
	if err := p.checkNest(in); err != nil {
		return Task{}, err
	}
	if err := p.CheckClose(in); err != nil {
		return Task{}, err
	}
	uid, err := NewUID()
	if err != nil {
		return Task{}, err
	}
	in.ID = p.NextTaskID()
	in.UID = uid
	in.Created = Today()
	p.Tasks = append(p.Tasks, in)
	if err := p.appendTask(in); err != nil {
		return Task{}, err
	}
	if err := p.Log("created " + in.ID + " " + in.Title); err != nil {
		return Task{}, err
	}
	return in, nil
}

type TaskEdit struct {
	Title         *string
	Type          *string
	Status        *string
	Priority      *string
	Milestone     *string
	Parent        *string
	Notes         *string
	BlockedReason *string
	Assignees     *[]string
	Tags          *[]string
	Requirements  *[]string
	TestCases     *[]string
	Acceptance    *[]string
	DOD           *[]string
	Start         *string
	Due           *string
}

func (p *Pack) EditTask(id string, edit TaskEdit) (Task, error) {
	t, _, ok := p.Task(id)
	if !ok {
		return Task{}, fmt.Errorf("task %s not found", id)
	}
	if edit.Title != nil {
		t.Title = *edit.Title
	}
	if edit.Type != nil {
		typ := NormType(*edit.Type)
		if err := CheckType(typ); err != nil {
			return Task{}, err
		}
		t.Type = typ
	}
	if edit.Status != nil {
		if err := CheckStatus(*edit.Status); err != nil {
			return Task{}, err
		}
		t.Status = *edit.Status
	}
	if edit.Priority != nil {
		if err := CheckPriority(*edit.Priority); err != nil {
			return Task{}, err
		}
		t.Priority = *edit.Priority
	}
	if edit.Milestone != nil {
		t.Milestone = *edit.Milestone
	}
	if edit.Parent != nil {
		t.Parent = *edit.Parent
	}
	if edit.Notes != nil {
		t.Notes = *edit.Notes
	}
	if edit.BlockedReason != nil {
		t.BlockedReason = *edit.BlockedReason
	}
	if edit.Assignees != nil {
		t.Assignees = *edit.Assignees
	}
	if edit.Tags != nil {
		t.Tags = *edit.Tags
	}
	if edit.Requirements != nil {
		t.Requirements = *edit.Requirements
	}
	if edit.TestCases != nil {
		t.TestCases = *edit.TestCases
	}
	if edit.Acceptance != nil {
		t.Acceptance = *edit.Acceptance
	}
	if edit.DOD != nil {
		t.DOD = *edit.DOD
	}
	if edit.Start != nil {
		if err := CheckDay(*edit.Start); err != nil {
			return Task{}, fmt.Errorf("start: %w", err)
		}
		t.Start = strings.TrimSpace(*edit.Start)
	}
	if edit.Due != nil {
		if err := CheckDay(*edit.Due); err != nil {
			return Task{}, fmt.Errorf("due: %w", err)
		}
		t.Due = strings.TrimSpace(*edit.Due)
	}
	if err := CheckCard(*t); err != nil {
		return Task{}, err
	}
	if err := p.CheckGuard(*t); err != nil {
		return Task{}, err
	}
	if err := p.CheckValidation(*t); err != nil {
		return Task{}, err
	}
	if err := p.checkNest(*t); err != nil {
		return Task{}, err
	}
	if err := p.CheckClose(*t); err != nil {
		return Task{}, err
	}
	if err := fillUID(&t.UID); err != nil {
		return Task{}, err
	}
	t.Updated = Today()
	if err := p.appendTask(*t); err != nil {
		return Task{}, err
	}
	if err := p.Log("edited " + t.ID); err != nil {
		return Task{}, err
	}
	return *t, nil
}

func (p *Pack) CompleteTask(id string) (Task, error) {
	t, _, ok := p.Task(id)
	if !ok {
		return Task{}, fmt.Errorf("task %s not found", id)
	}
	t.Status = "Done"
	t.Updated = Today()
	if err := p.CheckClose(*t); err != nil {
		return Task{}, err
	}
	if err := fillUID(&t.UID); err != nil {
		return Task{}, err
	}
	if err := p.appendTask(*t); err != nil {
		return Task{}, err
	}
	if err := p.Log("completed " + t.ID); err != nil {
		return Task{}, err
	}
	return *t, nil
}

func (p *Pack) ArchiveTask(id string) (Task, error) {
	t, i, ok := p.Task(id)
	if !ok {
		return Task{}, fmt.Errorf("task %s not found", id)
	}
	row := *t
	row.Updated = Today()
	if err := fillUID(&row.UID); err != nil {
		return Task{}, err
	}
	tomb := row
	tomb.Archived = true
	p.Archive = append(p.Archive, row)
	p.Tasks = append(p.Tasks[:i], p.Tasks[i+1:]...)
	if err := p.appendTask(tomb); err != nil {
		return Task{}, err
	}
	if err := p.appendArchive(row); err != nil {
		return Task{}, err
	}
	if err := p.Log("archived " + row.ID); err != nil {
		return Task{}, err
	}
	return row, nil
}

func (p *Pack) Search(q string) []Task {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return nil
	}
	var out []Task
	for _, t := range p.Tasks {
		b, err := json.Marshal(t)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(string(b)), q) {
			out = append(out, t)
		}
	}
	return out
}

func (p *Pack) ListTasks(status string) []Task {
	if status == "" {
		return append([]Task{}, p.Tasks...)
	}
	var out []Task
	for _, t := range p.Tasks {
		if t.Status == status {
			out = append(out, t)
		}
	}
	return out
}
