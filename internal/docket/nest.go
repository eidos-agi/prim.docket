package docket

import (
	"fmt"
	"strings"
)

// Nesting is the parent grammar. Milestone is not in this table — it is an overlay
// field (milestone: MS-*), never a parent. GUARD/TEST/VALIDATION are leaves.
//
//	GOAL        → none | GOAL
//	PLAN        → GOAL | PLAN
//	TASK        → GOAL | PLAN | TASK
//	GUARD       → GOAL | PLAN | TASK   (required)
//	TEST        → TASK                 (required)
//	VALIDATION  → GOAL | PLAN | TASK   (required; close-out)
var nestUnder = map[string][]string{
	"GOAL":       {"GOAL"},
	"PLAN":       {"GOAL", "PLAN"},
	"TASK":       {"GOAL", "PLAN", "TASK"},
	"GUARD":      {"GOAL", "PLAN", "TASK"},
	"TEST":       {"TASK"},
	"VALIDATION": {"GOAL", "PLAN", "TASK"},
}

func nestOK(child, parent string) bool {
	for _, p := range nestUnder[child] {
		if p == parent {
			return true
		}
	}
	return false
}

func CheckNest(child Task, parent *Task) error {
	ct := NormType(child.Type)
	if parent == nil || strings.TrimSpace(child.Parent) == "" {
		if ct == "GUARD" || ct == "TEST" || ct == "VALIDATION" {
			return fmt.Errorf("%s requires parent", ct)
		}
		return nil
	}
	pt := NormType(parent.Type)
	if !nestOK(ct, pt) {
		return fmt.Errorf("%s cannot nest under %s", ct, pt)
	}
	return nil
}

func (p *Pack) checkNest(t Task) error {
	if strings.TrimSpace(t.Parent) == "" {
		return CheckNest(t, nil)
	}
	par, _, ok := p.Task(t.Parent)
	if !ok {
		return nil
	}
	return CheckNest(t, par)
}
