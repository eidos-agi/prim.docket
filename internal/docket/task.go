package docket

import (
	"fmt"
	"strings"
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

type Task struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Status        string   `json:"status"`
	Priority      string   `json:"priority,omitempty"`
	Milestone     string   `json:"milestone,omitempty"`
	Parent        string   `json:"parent,omitempty"`
	Subtasks      []string `json:"subtasks,omitempty"`
	Assignees     []string `json:"assignees,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Dependencies  []string `json:"dependencies,omitempty"`
	Acceptance    []string `json:"acceptance-criteria,omitempty"`
	DOD           []string `json:"definition-of-done,omitempty"`
	BlockedReason string   `json:"blocked_reason,omitempty"`
	Created       string   `json:"created,omitempty"`
	Updated       string   `json:"updated,omitempty"`
	Notes         string   `json:"notes,omitempty"`
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
	if strings.TrimSpace(in.Title) == "" {
		return Task{}, fmt.Errorf("title required")
	}
	if in.Status == "" {
		in.Status = "To Do"
	}
	if err := CheckStatus(in.Status); err != nil {
		return Task{}, err
	}
	if err := CheckPriority(in.Priority); err != nil {
		return Task{}, err
	}
	in.ID = p.NextTaskID()
	in.Created = Today()
	p.Tasks = append(p.Tasks, in)
	if err := p.Save(); err != nil {
		return Task{}, err
	}
	if err := p.Log("created " + in.ID + " " + in.Title); err != nil {
		return Task{}, err
	}
	return in, nil
}

type TaskEdit struct {
	Title         *string
	Status        *string
	Priority      *string
	Milestone     *string
	Parent        *string
	Notes         *string
	BlockedReason *string
	Assignees     *[]string
	Tags          *[]string
	Acceptance    *[]string
	DOD           *[]string
}

func (p *Pack) EditTask(id string, edit TaskEdit) (Task, error) {
	t, _, ok := p.Task(id)
	if !ok {
		return Task{}, fmt.Errorf("task %s not found", id)
	}
	if edit.Title != nil {
		t.Title = *edit.Title
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
	if edit.Acceptance != nil {
		t.Acceptance = *edit.Acceptance
	}
	if edit.DOD != nil {
		t.DOD = *edit.DOD
	}
	t.Updated = Today()
	if err := p.Save(); err != nil {
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
	if err := p.Save(); err != nil {
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
	p.Archive = append(p.Archive, row)
	p.Tasks = append(p.Tasks[:i], p.Tasks[i+1:]...)
	if err := p.Save(); err != nil {
		return Task{}, err
	}
	if err := p.Log("archived " + row.ID); err != nil {
		return Task{}, err
	}
	return row, nil
}

func (p *Pack) Search(q string) []Task {
	q = strings.ToLower(q)
	var out []Task
	for _, t := range p.Tasks {
		blob := strings.ToLower(t.ID + " " + t.Title + " " + t.Notes + " " + t.BlockedReason)
		if strings.Contains(blob, q) {
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
