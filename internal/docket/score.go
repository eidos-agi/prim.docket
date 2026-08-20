package docket

import (
	"fmt"
	"sort"
	"strings"
)

// Plan score is the quantitative graph bar. Same pack, same number.
// It does not read note meaning. Hollow bodies stay on validate.

const planMax = 100

var planWeight = map[string]int{
	"empty":               100,
	"no-goal":             20,
	"no-plan":             5,
	"no-task":             10,
	"no-guard":            15,
	"no-test":             10,
	"no-validation-cards": 10,
	"no-milestone":        10,
	"unguarded-goal":      8,
	"no-validation":       8,
	"tesless":             4,
	"loose-work":          4,
	"extra-root":          6,
	"orphan":              6,
	"flat":                15,
	"cycle":               25,
	"dep-cycle":           20,
	"parent-dangle":       5,
	"dep-dangle":          5,
	"untagged":            1,
	"parent-unvalidated":  4,
	"validation-early":    4,
}

var planCap = map[string]int{
	"unguarded-goal":     24,
	"no-validation":      32,
	"tesless":            20,
	"loose-work":         20,
	"extra-root":         18,
	"orphan":             24,
	"parent-dangle":      15,
	"dep-dangle":         15,
	"untagged":           10,
	"parent-unvalidated": 12,
	"validation-early":   12,
}

type Deduction struct {
	Code   string   `json:"code"`
	Points int      `json:"points"`
	Count  int      `json:"count"`
	IDs    []string `json:"ids,omitempty"`
	Msg    string   `json:"msg"`
}

type ScoreCounts struct {
	Cards       int `json:"cards"`
	Goals       int `json:"goals"`
	Plans       int `json:"plans"`
	Tasks       int `json:"tasks"`
	Guards      int `json:"guards"`
	Tests       int `json:"tests"`
	Validations int `json:"validations"`
	Milestones  int `json:"milestones"`
}

type Score struct {
	Score      int         `json:"score"`
	Max        int         `json:"max"`
	Kind       string      `json:"kind"`
	OK         bool        `json:"ok"`
	Deductions []Deduction `json:"deductions"`
	Counts     ScoreCounts `json:"counts"`
}

func scoreCounts(p *Pack) ScoreCounts {
	c := ScoreCounts{}
	if p == nil {
		return c
	}
	c.Cards = len(p.Tasks)
	c.Milestones = len(p.Milestones)
	for _, t := range p.Tasks {
		switch NormType(t.Type) {
		case "GOAL":
			c.Goals++
		case "PLAN":
			c.Plans++
		case "TASK":
			c.Tasks++
		case "GUARD":
			c.Guards++
		case "TEST":
			c.Tests++
		case "VALIDATION":
			c.Validations++
		}
	}
	return c
}

func capPoints(code string, n int) int {
	w := planWeight[code]
	if w == 0 {
		w = 3
	}
	pts := n * w
	if c, ok := planCap[code]; ok && pts > c {
		return c
	}
	return pts
}

// PlanFindings is the quantitative plan-shape check. Graph only.
func PlanFindings(p *Pack) []Finding {
	if p == nil || len(p.Tasks) == 0 {
		return []Finding{{Code: "empty", Msg: "no cards"}}
	}
	var fs []Finding
	have := map[string]int{}
	byID := map[string]Task{}
	for _, t := range p.Tasks {
		have[NormType(t.Type)]++
		byID[t.ID] = t
	}
	add0 := func(code, msg string) {
		fs = append(fs, Finding{Code: code, Msg: msg})
	}
	if have["GOAL"] == 0 {
		add0("no-goal", "0 GOAL cards")
	}
	if have["PLAN"] == 0 {
		add0("no-plan", "0 PLAN cards")
	}
	if have["TASK"] == 0 {
		add0("no-task", "0 TASK cards")
	}
	if have["GUARD"] == 0 {
		add0("no-guard", "0 GUARD cards")
	}
	if have["TEST"] == 0 {
		add0("no-test", "0 TEST cards")
	}
	if have["VALIDATION"] == 0 {
		add0("no-validation-cards", "0 VALIDATION cards")
	}
	if len(p.Milestones) == 0 {
		add0("no-milestone", "0 milestones")
	}

	kids := childrenOf(byID)
	workN, workEdges := 0, 0
	var rootGoals []string
	for _, t := range p.Tasks {
		typ := NormType(t.Type)
		if WorkType(typ) {
			workN++
			if t.Parent != "" {
				if par, ok := byID[t.Parent]; ok && WorkType(par.Type) {
					workEdges++
				}
			}
		}
		if typ == "GOAL" && strings.TrimSpace(t.Parent) == "" {
			rootGoals = append(rootGoals, t.ID)
		}
		if (typ == "GUARD" || typ == "TEST" || typ == "VALIDATION") && strings.TrimSpace(t.Parent) == "" {
			fs = append(fs, Finding{Code: "orphan", ID: t.ID, Field: "parent", Msg: typ + " has no parent"})
		}
		if typ == "GOAL" {
			ok := false
			for _, c := range kids[t.ID] {
				if NormType(c.Type) == "GUARD" {
					ok = true
					break
				}
			}
			if !ok {
				fs = append(fs, Finding{Code: "unguarded-goal", ID: t.ID, Msg: "GOAL has no GUARD"})
			}
		}
		if typ == "TASK" {
			ok := false
			for _, c := range kids[t.ID] {
				if NormType(c.Type) == "TEST" {
					ok = true
					break
				}
			}
			if !ok {
				fs = append(fs, Finding{Code: "tesless", ID: t.ID, Msg: "TASK has no TEST child"})
			}
		}
		if WorkType(typ) && have["GOAL"] > 0 && strings.TrimSpace(t.Parent) == "" && typ != "GOAL" {
			fs = append(fs, Finding{Code: "loose-work", ID: t.ID, Field: "parent", Msg: typ + " is unparented while a GOAL exists"})
		}
		if t.Parent != "" && t.Parent != t.ID {
			if _, ok := byID[t.Parent]; !ok {
				fs = append(fs, Finding{Code: "parent-dangle", ID: t.ID, Field: "parent", Msg: "dangling parent " + t.Parent})
			}
		}
		for _, d := range t.Dependencies {
			if d == "" {
				continue
			}
			if _, ok := byID[d]; !ok {
				fs = append(fs, Finding{Code: "dep-dangle", ID: t.ID, Field: "dependencies", Msg: "dangling dependency " + d})
			}
		}
		if WorkType(typ) && len(p.Milestones) > 0 && strings.TrimSpace(t.Milestone) == "" {
			fs = append(fs, Finding{Code: "untagged", ID: t.ID, Field: "milestone", Msg: "work card has no milestone overlay"})
		}
	}
	if len(rootGoals) > 1 {
		for _, id := range rootGoals[1:] {
			fs = append(fs, Finding{Code: "extra-root", ID: id, Msg: "extra unparented GOAL"})
		}
	}
	if workN >= 2 && workEdges == 0 {
		fs = append(fs, Finding{Code: "flat", Msg: "work cards have no parent edges"})
	}
	if id := parentCycle(byID); id != "" {
		fs = append(fs, Finding{Code: "cycle", ID: id, Field: "parent", Msg: "parent cycle"})
	}
	if id := depCycle(byID); id != "" {
		fs = append(fs, Finding{Code: "dep-cycle", ID: id, Field: "dependencies", Msg: "dependency cycle"})
	}
	fs = append(fs, validationFindings(byID)...)
	return fs
}

func depCycle(byID map[string]Task) string {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var walk func(id string) string
	walk = func(id string) string {
		color[id] = gray
		for _, d := range byID[id].Dependencies {
			if _, ok := byID[d]; !ok {
				continue
			}
			switch color[d] {
			case gray:
				return id
			case white:
				if hit := walk(d); hit != "" {
					return hit
				}
			}
		}
		color[id] = black
		return ""
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if color[id] == white {
			if hit := walk(id); hit != "" {
				return hit
			}
		}
	}
	return ""
}

// ScoreOf is the 0–100 plan score. Deterministic. Graph only.
func ScoreOf(p *Pack) Score {
	counts := scoreCounts(p)
	fs := PlanFindings(p)
	type bucket struct {
		n   int
		ids []string
		msg string
	}
	order := []string{}
	seen := map[string]*bucket{}
	for _, f := range fs {
		b, ok := seen[f.Code]
		if !ok {
			b = &bucket{msg: f.Msg}
			seen[f.Code] = b
			order = append(order, f.Code)
		}
		b.n++
		if f.ID != "" && len(b.ids) < 8 {
			b.ids = append(b.ids, f.ID)
		}
	}
	lost := 0
	ds := make([]Deduction, 0, len(order))
	for _, code := range order {
		b := seen[code]
		pts := capPoints(code, b.n)
		ds = append(ds, Deduction{Code: code, Points: pts, Count: b.n, IDs: b.ids, Msg: b.msg})
		lost += pts
	}
	n := planMax - lost
	if n < 0 {
		n = 0
	}
	return Score{
		Score:      n,
		Max:        planMax,
		Kind:       "plan",
		OK:         n == planMax,
		Deductions: ds,
		Counts:     counts,
	}
}

func (s Score) Line() string {
	return fmt.Sprintf("%d/%d plan", s.Score, s.Max)
}
