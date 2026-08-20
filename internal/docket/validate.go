package docket

import (
	"fmt"
	"strings"
)

var explosionTypes = []string{"GOAL", "PLAN", "TASK", "GUARD", "TEST"}

// Validate is the quantitative pack bar: schema findings (blank, too short,
// placeholder, types, nesting, refs, uids). Collect-all.
func Validate(p *Pack) error {
	return FindingsError(SchemaFindings(p))
}

// ValidateExplosion is the /plandocket bar: a typed bucket, not one stub.
func ValidateExplosion(p *Pack) error {
	if err := Validate(p); err != nil {
		return err
	}
	have := map[string]int{}
	for _, t := range p.Tasks {
		have[NormType(t.Type)]++
	}
	var missing []string
	for _, k := range explosionTypes {
		if have[k] == 0 {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing card types %s (need GOAL PLAN TASK GUARD TEST)", strings.Join(missing, ","))
	}
	guarded := map[string]bool{}
	for _, t := range p.Tasks {
		if NormType(t.Type) == "GUARD" {
			guarded[t.Parent] = true
		}
	}
	for _, t := range p.Tasks {
		if NormType(t.Type) == "GOAL" && !guarded[t.ID] {
			return fmt.Errorf("%s: GOAL has no GUARD", t.ID)
		}
	}
	return nil
}

func checkGuardParent(t Task, byID map[string]Task) error {
	if NormType(t.Type) != "GUARD" {
		return nil
	}
	if strings.TrimSpace(t.Parent) == "" {
		return fmt.Errorf("GUARD requires parent (the GOAL, PLAN, or TASK it constrains)")
	}
	p, ok := byID[t.Parent]
	if !ok {
		return nil
	}
	switch NormType(p.Type) {
	case "GOAL", "PLAN", "TASK":
		return nil
	default:
		return fmt.Errorf("GUARD parent must be GOAL, PLAN, or TASK (not %s)", NormType(p.Type))
	}
}

func checkValidationParent(t Task, byID map[string]Task) error {
	if NormType(t.Type) != "VALIDATION" {
		return nil
	}
	if strings.TrimSpace(t.Parent) == "" {
		return fmt.Errorf("VALIDATION requires parent (the GOAL, PLAN, or TASK it closes)")
	}
	p, ok := byID[t.Parent]
	if !ok {
		return nil
	}
	if !WorkType(p.Type) {
		return fmt.Errorf("VALIDATION parent must be GOAL, PLAN, or TASK (not %s)", NormType(p.Type))
	}
	return nil
}

func parentCycle(byID map[string]Task) string {
	for id := range byID {
		seen := map[string]bool{}
		cur := id
		for cur != "" {
			if seen[cur] {
				return id
			}
			seen[cur] = true
			t, ok := byID[cur]
			if !ok || t.Parent == "" {
				break
			}
			cur = t.Parent
		}
	}
	return ""
}
