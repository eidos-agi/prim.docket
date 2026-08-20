package docket

import (
	"fmt"
	"strings"
)

// WorkType is a decomposable card. GUARD, TEST, and VALIDATION are not work.
func WorkType(typ string) bool {
	switch NormType(typ) {
	case "GOAL", "PLAN", "TASK":
		return true
	}
	return false
}

func (p *Pack) CheckValidation(t Task) error {
	if NormType(t.Type) != "VALIDATION" {
		return nil
	}
	if strings.TrimSpace(t.Parent) == "" {
		return fmt.Errorf("VALIDATION requires parent (the GOAL, PLAN, or TASK it closes)")
	}
	target, _, ok := p.Task(t.Parent)
	if !ok {
		return fmt.Errorf("VALIDATION parent %s not found", t.Parent)
	}
	if t.ID != "" && target.ID == t.ID {
		return fmt.Errorf("VALIDATION parent is self")
	}
	if !WorkType(target.Type) {
		return fmt.Errorf("VALIDATION parent must be GOAL, PLAN, or TASK (not %s)", NormType(target.Type))
	}
	return nil
}

func childrenOf(byID map[string]Task) map[string][]Task {
	kids := map[string][]Task{}
	for _, t := range byID {
		if t.Parent != "" {
			kids[t.Parent] = append(kids[t.Parent], t)
		}
	}
	return kids
}

func validationFindings(byID map[string]Task) []Finding {
	kids := childrenOf(byID)
	var fs []Finding
	add := func(code, id, msg string) {
		fs = append(fs, Finding{Code: code, ID: id, Field: "parent", Msg: msg})
	}
	for _, t := range byID {
		if WorkType(t.Type) {
			var work, vals []Task
			for _, c := range kids[t.ID] {
				if WorkType(c.Type) {
					work = append(work, c)
				}
				if NormType(c.Type) == "VALIDATION" {
					vals = append(vals, c)
				}
			}
			if len(work) > 0 && len(vals) == 0 {
				add("no-validation", t.ID, "has children and no VALIDATION close-out")
			}
			if t.Status == "Done" {
				for _, v := range vals {
					if v.Status != "Done" {
						add("parent-unvalidated", t.ID, "Done while VALIDATION "+v.ID+" is still open")
					}
				}
			}
		}
		if NormType(t.Type) == "VALIDATION" && t.Status == "Done" {
			for _, c := range kids[t.Parent] {
				if WorkType(c.Type) && c.Status != "Done" {
					add("validation-early", t.ID, "VALIDATION Done while "+c.ID+" is still open")
				}
			}
		}
	}
	return fs
}

// CheckClose is the write-path gate: a parent with children closes only through
// a Done VALIDATION, and that VALIDATION waits for work siblings.
func (p *Pack) CheckClose(t Task) error {
	byID := map[string]Task{}
	for _, row := range p.Tasks {
		byID[row.ID] = row
	}
	if t.ID != "" {
		byID[t.ID] = t
	}
	for _, f := range validationFindings(byID) {
		if f.ID == t.ID {
			return fmt.Errorf("%s", f.Msg)
		}
	}
	return nil
}
