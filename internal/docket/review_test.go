package docket

import (
	"strings"
	"testing"
)

func TestReviewQuestionCount(t *testing.T) {
	n := ReviewQuestionCount()
	if n < 50 {
		t.Fatalf("qualitative review needs ≥50 questions, got %d", n)
	}
}

func TestReviewPromptProtocol(t *testing.T) {
	p := ReviewPrompt()
	for _, need := range []string{
		"Blind qualitative",
		"two follow-up rounds",
		"human",
		"docket-prim validate",
		"milestone used only as overlay",
		"The muzzle",
		"VALIDATION",
	} {
		if !strings.Contains(p, need) {
			t.Fatalf("prompt missing %q", need)
		}
	}
	if FollowUps != 2 {
		t.Fatalf("FollowUps %d", FollowUps)
	}
}

func TestReviewDocIncludesDigest(t *testing.T) {
	dir := t.TempDir()
	pack, err := Init(dir, "Fixture", "fix")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pack.CreateTask(live("Ship login")); err != nil {
		t.Fatal(err)
	}
	re, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	doc := ReviewDoc(re)
	if !strings.Contains(doc, "Ship login") || !strings.Contains(doc, "TASK-0001") {
		t.Fatalf("digest missing card")
	}
	if !strings.Contains(doc, dir) {
		t.Fatalf("digest missing pack path")
	}
}
