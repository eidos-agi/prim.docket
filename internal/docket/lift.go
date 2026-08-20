package docket

import (
	"fmt"
	"strings"
)

type LiftOpts struct {
	Guards       bool
	AttachGuards string
	AttachLoose  string
}

type LiftOp struct {
	Op        string `json:"op"`
	ID        string `json:"id,omitempty"`
	Type      string `json:"type,omitempty"`
	Title     string `json:"title,omitempty"`
	Parent    string `json:"parent,omitempty"`
	Milestone string `json:"milestone,omitempty"`
	Reason    string `json:"reason"`
}

type LiftResult struct {
	Before Score    `json:"before"`
	After  Score    `json:"after"`
	Ops    []LiftOp `json:"ops"`
	DryRun bool     `json:"dry_run"`
}

func clonePack(p *Pack) *Pack {
	if p == nil {
		return &Pack{}
	}
	q := *p
	q.Tasks = append([]Task{}, p.Tasks...)
	q.Milestones = append([]Milestone{}, p.Milestones...)
	q.Archive = append([]Task{}, p.Archive...)
	return &q
}

func applySim(p *Pack, ops []LiftOp) {
	for _, op := range ops {
		switch op.Op {
		case "create":
			t := typed(op.Title, op.Type)
			t.ID = p.NextTaskID()
			t.Status = "To Do"
			t.Parent = op.Parent
			t.Milestone = op.Milestone
			p.Tasks = append(p.Tasks, t)
		case "edit":
			for i := range p.Tasks {
				if p.Tasks[i].ID != op.ID {
					continue
				}
				if op.Parent != "" {
					p.Tasks[i].Parent = op.Parent
				}
				if op.Milestone != "" {
					p.Tasks[i].Milestone = op.Milestone
				}
			}
		}
	}
}

func liftEdits(p *Pack, opt LiftOpts) ([]LiftOp, error) {
	var ops []LiftOp
	if id := strings.TrimSpace(opt.AttachGuards); id != "" {
		par, _, ok := p.Task(id)
		if !ok {
			return nil, fmt.Errorf("attach-guards %s not found", id)
		}
		if !WorkType(par.Type) {
			return nil, fmt.Errorf("attach-guards parent must be GOAL, PLAN, or TASK")
		}
		for _, t := range p.Tasks {
			if NormType(t.Type) == "GUARD" && strings.TrimSpace(t.Parent) == "" {
				ops = append(ops, LiftOp{Op: "edit", ID: t.ID, Parent: id, Reason: "orphan"})
			}
		}
	}
	if id := strings.TrimSpace(opt.AttachLoose); id != "" {
		par, _, ok := p.Task(id)
		if !ok {
			return nil, fmt.Errorf("attach-loose %s not found", id)
		}
		for _, t := range p.Tasks {
			if !WorkType(t.Type) || strings.TrimSpace(t.Parent) != "" || t.ID == id {
				continue
			}
			child := t
			child.Parent = id
			if err := CheckNest(child, par); err != nil {
				continue
			}
			reason := "loose-work"
			if NormType(t.Type) == "GOAL" {
				reason = "extra-root"
			}
			ops = append(ops, LiftOp{Op: "edit", ID: t.ID, Parent: id, Reason: reason})
		}
	}
	if len(p.Milestones) == 1 {
		ms := p.Milestones[0].ID
		for _, t := range p.Tasks {
			if WorkType(t.Type) && strings.TrimSpace(t.Milestone) == "" {
				ops = append(ops, LiftOp{Op: "edit", ID: t.ID, Milestone: ms, Reason: "untagged"})
			}
		}
	}
	return ops, nil
}

func liftCreates(p *Pack, opt LiftOpts) []LiftOp {
	byID := map[string]Task{}
	for _, t := range p.Tasks {
		byID[t.ID] = t
	}
	kids := childrenOf(byID)
	var ops []LiftOp
	for _, t := range p.Tasks {
		if !WorkType(t.Type) {
			continue
		}
		work, vals := 0, 0
		for _, c := range kids[t.ID] {
			if WorkType(c.Type) {
				work++
			}
			if NormType(c.Type) == "VALIDATION" {
				vals++
			}
		}
		if work > 0 && vals == 0 {
			ops = append(ops, LiftOp{
				Op: "create", Type: "VALIDATION", Title: "Close " + t.Title,
				Parent: t.ID, Milestone: t.Milestone, Reason: "no-validation",
			})
		}
	}
	for _, t := range p.Tasks {
		if NormType(t.Type) != "TASK" {
			continue
		}
		ok := false
		for _, c := range kids[t.ID] {
			if NormType(c.Type) == "TEST" {
				ok = true
				break
			}
		}
		if !ok {
			ops = append(ops, LiftOp{
				Op: "create", Type: "TEST", Title: "Prove " + t.Title,
				Parent: t.ID, Milestone: t.Milestone, Reason: "tesless",
			})
		}
	}
	if opt.Guards {
		for _, t := range p.Tasks {
			if NormType(t.Type) != "GOAL" {
				continue
			}
			ok := false
			for _, c := range kids[t.ID] {
				if NormType(c.Type) == "GUARD" {
					ok = true
					break
				}
			}
			if !ok {
				ops = append(ops, LiftOp{
					Op: "create", Type: "GUARD", Title: "Refuse false-green on " + t.Title,
					Parent: t.ID, Milestone: t.Milestone, Reason: "unguarded-goal",
				})
			}
		}
	}
	return ops
}

// PlanLift is the nest-legal patch list. Edits first, then creates on the edited graph.
func PlanLift(p *Pack, opt LiftOpts) ([]LiftOp, error) {
	edits, err := liftEdits(p, opt)
	if err != nil {
		return nil, err
	}
	sim := clonePack(p)
	applySim(sim, edits)
	creates := liftCreates(sim, opt)
	return append(edits, creates...), nil
}

func ApplyLift(p *Pack, ops []LiftOp) error {
	for _, op := range ops {
		switch op.Op {
		case "create":
			t := typed(op.Title, op.Type)
			t.Parent = op.Parent
			t.Milestone = op.Milestone
			if _, err := p.CreateTask(t); err != nil {
				return fmt.Errorf("%s %s: %w", op.Type, op.Title, err)
			}
		case "edit":
			e := TaskEdit{}
			if op.Parent != "" {
				parent := op.Parent
				e.Parent = &parent
			}
			if op.Milestone != "" {
				ms := op.Milestone
				e.Milestone = &ms
			}
			if _, err := p.EditTask(op.ID, e); err != nil {
				return fmt.Errorf("edit %s: %w", op.ID, err)
			}
		default:
			return fmt.Errorf("unknown lift op %q", op.Op)
		}
	}
	return nil
}

// Lift applies PlanLift, or dry-runs it. jsonl is only written on apply.
func Lift(p *Pack, opt LiftOpts, dry bool) (LiftResult, error) {
	ops, err := PlanLift(p, opt)
	if err != nil {
		return LiftResult{}, err
	}
	out := LiftResult{Before: ScoreOf(p), Ops: ops, DryRun: dry}
	if dry {
		sim := clonePack(p)
		applySim(sim, ops)
		out.After = ScoreOf(sim)
		return out, nil
	}
	if err := ApplyLift(p, ops); err != nil {
		return LiftResult{}, err
	}
	out.After = ScoreOf(p)
	return out, nil
}

func (r LiftResult) Line() string {
	kind := "lift"
	if r.DryRun {
		kind = "lift dry-run"
	}
	return fmt.Sprintf("%s  %s → %s  %d ops", kind, r.Before.Line(), r.After.Line(), len(r.Ops))
}
