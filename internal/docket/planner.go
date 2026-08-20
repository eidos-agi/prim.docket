package docket

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PlanRequest is the upgrade seam for `docket-prim planner`.
// /plandocket (and any other agent) fills this; the verb writes the pack.
// Add fields here when the planner grows. Do not grow the skill instead.
type PlanRequest struct {
	Dir        string          `json:"dir"`
	ID         string          `json:"id,omitempty"`
	Name       string          `json:"name"`
	Title      string          `json:"title,omitempty"`
	Goal       string          `json:"goal"`
	DoneWhen   []string        `json:"done_when"`
	Negative   string          `json:"negative,omitempty"`
	OutOfScope []KV            `json:"out_of_scope,omitempty"`
	Shipped    []KV            `json:"already_shipped,omitempty"`
	ProofPath  string          `json:"proof_path"`
	Linear     string          `json:"linear,omitempty"`
	Stop       string          `json:"stop,omitempty"`
	Priority   string          `json:"priority,omitempty"`
	Notes      string          `json:"notes,omitempty"`
	HandoffRel string          `json:"handoff_rel,omitempty"`
	Milestones []PlanMilestone `json:"milestones,omitempty"`
	Tasks      []PlanTask      `json:"tasks,omitempty"`
}

type PlanMilestone struct {
	Title string `json:"title"`
	Due   string `json:"due,omitempty"`
}

type PlanTask struct {
	Title        string   `json:"title"`
	Type         string   `json:"type,omitempty"`
	Status       string   `json:"status,omitempty"`
	Priority     string   `json:"priority,omitempty"`
	Milestone    string   `json:"milestone,omitempty"`
	Parent       string   `json:"parent,omitempty"`
	Notes        string   `json:"notes,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Requirements []string `json:"requirements,omitempty"`
	TestCases    []string `json:"test-cases,omitempty"`
	Acceptance   []string `json:"acceptance-criteria,omitempty"`
	DOD          []string `json:"definition-of-done,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	Blocked      string   `json:"blocked_reason,omitempty"`
	Start        string   `json:"start,omitempty"`
	Due          string   `json:"due,omitempty"`
}

type KV struct {
	Item string `json:"item"`
	Why  string `json:"why,omitempty"`
}

type PlanResult struct {
	Dir        string   `json:"dir"`
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	TaskID     string   `json:"task_id"`
	TaskIDs    []string `json:"task_ids,omitempty"`
	TaskCount  int      `json:"task_count"`
	Milestones int      `json:"milestone_count"`
	Face       string   `json:"face"`
	Handoff    string   `json:"handoff"`
	Review     string   `json:"review"`
}

// Plan mints a new docket-prim pack from a PlanRequest: face + milestones + typed tasks.
func Plan(req PlanRequest) (PlanResult, error) {
	if strings.TrimSpace(req.Dir) == "" {
		return PlanResult{}, fmt.Errorf("planner: dir required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return PlanResult{}, fmt.Errorf("planner: name required (one-sentence outcome)")
	}
	if strings.TrimSpace(req.Goal) == "" {
		req.Goal = req.Name
	}
	if len(req.DoneWhen) == 0 {
		return PlanResult{}, fmt.Errorf("planner: done_when required")
	}
	if strings.TrimSpace(req.ProofPath) == "" {
		return PlanResult{}, fmt.Errorf("planner: proof_path required")
	}
	if len(req.Tasks) == 0 {
		return PlanResult{}, fmt.Errorf("planner: tasks required (explode GOAL PLAN TASK GUARD TEST VALIDATION; a single GOAL is the muzzle)")
	}
	if req.Priority == "" {
		req.Priority = "high"
	}
	if req.Linear == "" {
		req.Linear = "none"
	}
	if req.Stop == "" {
		req.Stop = "Acceptance met, or blocked with evidence (credentials, human decision, hard external gate)."
	}
	if req.Title == "" {
		req.Title = req.Name
	}

	root, err := filepath.Abs(req.Dir)
	if err != nil {
		return PlanResult{}, err
	}
	if IsPack(root) {
		return PlanResult{}, fmt.Errorf("planner: already a pack: %s", root)
	}

	p, err := Init(root, req.Name, req.ID)
	if err != nil {
		return PlanResult{}, err
	}
	fail := func(err error) (PlanResult, error) {
		_ = os.RemoveAll(root)
		return PlanResult{}, err
	}

	msBy := map[string]string{}
	for _, m := range req.Milestones {
		got, err := p.CreateMilestone(m.Title, m.Due)
		if err != nil {
			return fail(err)
		}
		msBy[m.Title] = got.ID
		msBy[got.ID] = got.ID
	}

	idBy := map[string]string{}
	var first Task
	var ids []string
	for _, pt := range req.Tasks {
		mile := pt.Milestone
		if id, ok := msBy[mile]; ok {
			mile = id
		}
		parent := pt.Parent
		if id, ok := idBy[parent]; ok {
			parent = id
		}
		deps := append([]string{}, pt.Dependencies...)
		for i, d := range deps {
			if id, ok := idBy[d]; ok {
				deps[i] = id
			}
		}
		tags := pt.Tags
		if len(tags) == 0 {
			tags = []string{"plandocket"}
		}
		pri := pt.Priority
		if pri == "" {
			pri = req.Priority
		}
		t, err := p.CreateTask(Task{
			Title:         pt.Title,
			Type:          pt.Type,
			Status:        pt.Status,
			Priority:      pri,
			Milestone:     mile,
			Parent:        parent,
			Notes:         pt.Notes,
			Tags:          tags,
			Requirements:  pt.Requirements,
			TestCases:     pt.TestCases,
			Acceptance:    pt.Acceptance,
			DOD:           pt.DOD,
			Dependencies:  deps,
			BlockedReason: pt.Blocked,
			Start:         pt.Start,
			Due:           pt.Due,
		})
		if err != nil {
			return fail(err)
		}
		idBy[pt.Title] = t.ID
		idBy[t.ID] = t.ID
		ids = append(ids, t.ID)
		if first.ID == "" {
			first = t
		}
	}

	face := renderPlanFace(req, first.ID, len(ids), len(req.Milestones))
	if err := os.WriteFile(filepath.Join(root, "index.md"), []byte(face), 0o644); err != nil {
		return fail(err)
	}
	if err := p.Log(fmt.Sprintf("planner minted %d tasks %d milestones", len(ids), len(req.Milestones))); err != nil {
		return fail(err)
	}

	re, err := Open(p.Dir)
	if err != nil {
		return fail(err)
	}
	if err := Validate(re); err != nil {
		return fail(fmt.Errorf("planner: validate: %w", err))
	}
	if err := ValidateExplosion(re); err != nil {
		return fail(fmt.Errorf("planner: validate: %w", err))
	}

	facePath := filepath.Join(root, "index.md")
	handoffPath := facePath
	if rel := strings.TrimSpace(req.HandoffRel); rel != "" {
		handoffPath = filepath.Join(rel, "index.md")
	} else if cwd, err := os.Getwd(); err == nil {
		if r, err := filepath.Rel(cwd, facePath); err == nil && !strings.HasPrefix(r, "..") {
			handoffPath = r
		}
	}

	return PlanResult{
		Dir:        p.Dir,
		ID:         p.Project.ID,
		Name:       p.Project.Name,
		TaskID:     first.ID,
		TaskIDs:    ids,
		TaskCount:  len(ids),
		Milestones: len(req.Milestones),
		Face:       facePath,
		Handoff:    "/goal use this file " + handoffPath,
		Review:     "docket-prim review --dir " + p.Dir,
	}, nil
}

func renderPlanFace(req PlanRequest, taskID string, nTasks, nMiles int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\nprofile: %s\ndocket_version: %q\ntype: queue\ntitle: %s\nstatus: open\n---\n\n",
		Profile, Version, req.Title)
	fmt.Fprintf(&b, "# %s\n\n", req.Title)
	fmt.Fprintf(&b, "**Goal:** %s\n\n", req.Goal)
	fmt.Fprintf(&b, "Queue: `tasks.jsonl` (%d cards, first `%s`) · milestones: %d. This face is the `/goal` contract.\n\n", nTasks, taskID, nMiles)
	fmt.Fprintf(&b, "Card types: **GOAL · PLAN · MILESTONE · TASK · GUARD · TEST · VALIDATION**.\n\n")
	b.WriteString("## Done when\n\n")
	for _, d := range req.DoneWhen {
		fmt.Fprintf(&b, "- [ ] %s\n", d)
	}
	if n := strings.TrimSpace(req.Negative); n != "" {
		fmt.Fprintf(&b, "- [ ] NOT: %s\n", n)
	}
	fmt.Fprintf(&b, "- [ ] Proof artifact at: `%s`\n\n", req.ProofPath)
	if len(req.OutOfScope) > 0 {
		b.WriteString("## Out of scope\n\n| Out | Why |\n|-----|-----|\n")
		for _, r := range req.OutOfScope {
			fmt.Fprintf(&b, "| %s | %s |\n", r.Item, r.Why)
		}
		b.WriteString("\n")
	}
	if len(req.Shipped) > 0 {
		b.WriteString("## Already shipped / do not re-litigate\n\n| Piece | Status |\n|-------|--------|\n")
		for _, r := range req.Shipped {
			fmt.Fprintf(&b, "| %s | %s |\n", r.Item, r.Why)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "## Stop\n\n%s\n\nLinear: %s\n", req.Stop, req.Linear)
	return b.String()
}
