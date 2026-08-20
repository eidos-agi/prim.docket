package docket

import (
	"fmt"
	"strings"
)

type Milestone struct {
	ID      string `json:"id"`
	UID     string `json:"uid"`
	Title   string `json:"title"`
	Status  string `json:"status,omitempty"`
	Due     string `json:"due,omitempty"`
	Created string `json:"created,omitempty"`
	Updated string `json:"updated,omitempty"`
}

func (p *Pack) NextMilestoneID() string {
	n := 0
	for _, m := range p.Milestones {
		var i int
		if _, err := fmt.Sscanf(m.ID, "MS-%d", &i); err == nil && i > n {
			n = i
		}
	}
	return fmt.Sprintf("MS-%04d", n+1)
}

func (p *Pack) CreateMilestone(title, due string) (Milestone, error) {
	if strings.TrimSpace(title) == "" {
		return Milestone{}, fmt.Errorf("title required")
	}
	if err := CheckDay(due); err != nil {
		return Milestone{}, fmt.Errorf("due: %w", err)
	}
	uid, err := NewUID()
	if err != nil {
		return Milestone{}, err
	}
	m := Milestone{
		ID:      p.NextMilestoneID(),
		UID:     uid,
		Title:   title,
		Status:  "open",
		Due:     due,
		Created: Today(),
	}
	p.Milestones = append(p.Milestones, m)
	if err := p.appendMilestone(m); err != nil {
		return Milestone{}, err
	}
	if err := p.Log("created " + m.ID + " " + m.Title); err != nil {
		return Milestone{}, err
	}
	return m, nil
}

func (p *Pack) CloseMilestone(id string) (Milestone, error) {
	for i := range p.Milestones {
		if p.Milestones[i].ID == id {
			p.Milestones[i].Status = "closed"
			p.Milestones[i].Updated = Today()
			if err := fillUID(&p.Milestones[i].UID); err != nil {
				return Milestone{}, err
			}
			if err := p.appendMilestone(p.Milestones[i]); err != nil {
				return Milestone{}, err
			}
			if err := p.Log("closed " + id); err != nil {
				return Milestone{}, err
			}
			return p.Milestones[i], nil
		}
	}
	return Milestone{}, fmt.Errorf("milestone %s not found", id)
}
