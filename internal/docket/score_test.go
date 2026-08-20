package docket

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestScoreEmptyIsZero(t *testing.T) {
	s := ScoreOf(&Pack{Project: Project{Name: "x"}})
	if s.Score != 0 || s.Kind != "plan" || s.OK {
		t.Fatalf("%+v", s)
	}
	if len(s.Deductions) == 0 || s.Deductions[0].Code != "empty" {
		t.Fatalf("deductions %+v", s.Deductions)
	}
}

func TestScoreFlatTasksIsZero(t *testing.T) {
	p := &Pack{Project: Project{Name: "x"}}
	for i := 1; i <= 20; i++ {
		p.Tasks = append(p.Tasks, Task{
			ID: fmt.Sprintf("TASK-%04d", i), Title: "flat work named", Type: "TASK", Status: "To Do",
		})
	}
	s := ScoreOf(p)
	if s.Score != 0 {
		t.Fatalf("got %d deductions %+v", s.Score, s.Deductions)
	}
	need := map[string]bool{"no-goal": true, "no-guard": true, "no-milestone": true, "flat": true, "tesless": true}
	got := map[string]bool{}
	for _, d := range s.Deductions {
		got[d.Code] = true
	}
	for code := range need {
		if !got[code] {
			t.Fatalf("missing %s in %+v", code, s.Deductions)
		}
	}
}

func shapedPack() *Pack {
	rows := []Task{
		card("TASK-0001", "root outcome named", "GOAL", ""),
		card("TASK-0002", "ladder phase named", "PLAN", "TASK-0001"),
		card("TASK-0003", "do the work named", "TASK", "TASK-0002"),
		card("TASK-0004", "refuse the fail named", "GUARD", "TASK-0001"),
		card("TASK-0005", "prove the work named", "TEST", "TASK-0003"),
		card("TASK-0006", "close the root named", "VALIDATION", "TASK-0001"),
		card("TASK-0007", "close the ladder named", "VALIDATION", "TASK-0002"),
	}
	for i := range rows {
		if WorkType(rows[i].Type) {
			rows[i].Milestone = "MS-0001"
		}
	}
	return &Pack{
		Project:    Project{Name: "shaped"},
		Milestones: []Milestone{{ID: "MS-0001", Title: "First wave named"}},
		Tasks:      rows,
	}
}

func TestScoreShapedIsHundred(t *testing.T) {
	s := ScoreOf(shapedPack())
	if s.Score != 100 || !s.OK {
		t.Fatalf("got %d ok=%v deductions %+v", s.Score, s.OK, s.Deductions)
	}
}

func TestScoreDropsWithoutValidation(t *testing.T) {
	p := shapedPack()
	keep := p.Tasks[:0]
	for _, t := range p.Tasks {
		if t.ID != "TASK-0007" {
			keep = append(keep, t)
		}
	}
	p.Tasks = keep
	s := ScoreOf(p)
	if s.Score != 92 {
		t.Fatalf("got %d deductions %+v", s.Score, s.Deductions)
	}
	found := false
	for _, d := range s.Deductions {
		if d.Code == "no-validation" && d.Points == 8 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected no-validation −8: %+v", s.Deductions)
	}
}

func TestScoreIgnoresHollowNotes(t *testing.T) {
	p := shapedPack()
	p.Tasks[0].Notes = "short"
	p.Tasks[0].Requirements = nil
	shape := ScoreOf(p)
	if shape.Score != 100 {
		t.Fatalf("plan score used notes: %d %+v", shape.Score, shape.Deductions)
	}
	if err := Validate(p); err == nil {
		t.Fatal("validate should still fail hollow notes")
	}
}

func TestScoreDeterministic(t *testing.T) {
	p := shapedPack()
	a, err := json.Marshal(ScoreOf(p))
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(ScoreOf(p))
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatalf("json drifted\n%s\n%s", a, b)
	}
}

func TestScoreDepCycle(t *testing.T) {
	p := shapedPack()
	p.Tasks[2].Dependencies = []string{"TASK-0003"}
	s := ScoreOf(p)
	found := false
	for _, d := range s.Deductions {
		if d.Code == "dep-cycle" {
			found = true
		}
	}
	if !found || s.Score > 80 {
		t.Fatalf("dep-cycle %+v score %d", s.Deductions, s.Score)
	}
}
