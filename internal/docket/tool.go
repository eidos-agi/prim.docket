package docket

import "fmt"

// SPEC §10 — a Prim Tool cites a Prim. It is not the file.
const (
	ToolName        = "docket-editor"
	ToolKind        = "surface"
	ToolDirection   = "talk"
	ToolCounterpart = "human"
	ToolAs          = "editor"
)

type Tool struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Direction   string `json:"direction"`
	Counterpart string `json:"counterpart"`
	As          string `json:"as"`
	Cites       string `json:"cites"`
}

func (p *Pack) Tool() Tool {
	cites := p.Dir
	if p.Project.ID != "" {
		cites = "docket:" + p.Project.ID
	}
	return Tool{
		Name:        ToolName,
		Kind:        ToolKind,
		Direction:   ToolDirection,
		Counterpart: ToolCounterpart,
		As:          ToolAs,
		Cites:       cites,
	}
}

func (t Tool) Line() string {
	return fmt.Sprintf("%s (%s) %s/%s cites %s", t.Name, t.As, t.Kind, t.Direction, t.Cites)
}
