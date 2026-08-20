package docket

import (
	"fmt"
	"strings"
	_ "embed"
)

//go:embed review_prompt.md
var reviewPrompt string

func ReviewPrompt() string {
	return strings.TrimSpace(reviewPrompt)
}

func ReviewQuestionCount() int {
	n := 0
	for _, line := range strings.Split(reviewPrompt, "\n") {
		line = strings.TrimSpace(line)
		if len(line) > 2 && line[0] >= '1' && line[0] <= '9' {
			i := 0
			for i < len(line) && line[i] >= '0' && line[i] <= '9' {
				i++
			}
			if i > 0 && i < len(line) && line[i] == '.' && strings.Contains(line, "?") {
				n++
			}
		}
	}
	return n
}

func ReviewDoc(p *Pack) string {
	var b strings.Builder
	b.WriteString(ReviewPrompt())
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "Follow-up rounds allowed after the first review: %d. Then ask the human.\n", FollowUps)
	if p == nil {
		return b.String()
	}
	fmt.Fprintf(&b, "\nPack: %s\nName: %s\nCards: %d · milestones: %d\n\n", p.Dir, p.Project.Name, len(p.Tasks), len(p.Milestones))
	b.WriteString("### Milestones\n\n")
	if len(p.Milestones) == 0 {
		b.WriteString("(none)\n\n")
	}
	for _, m := range p.Milestones {
		due := m.Due
		if due == "" {
			due = "(none)"
		}
		fmt.Fprintf(&b, "%s  %s  status=%s  due=%s\n", m.ID, m.Title, m.Status, due)
	}
	b.WriteString("\n### Cards\n\n")
	for _, t := range p.Tasks {
		fmt.Fprintf(&b, "#### %s %s\n", t.ID, NormType(t.Type))
		fmt.Fprintf(&b, "title: %s\n", t.Title)
		fmt.Fprintf(&b, "status: %s  parent: %s  milestone: %s  start: %s  due: %s\n",
			t.Status, empty(t.Parent), empty(t.Milestone), empty(t.Start), empty(t.Due))
		if t.BlockedReason != "" {
			fmt.Fprintf(&b, "blocked: %s\n", t.BlockedReason)
		}
		fmt.Fprintf(&b, "notes: %s\n", clip(t.Notes, 360))
		writeList(&b, "requirements", t.Requirements)
		writeList(&b, "test-cases", t.TestCases)
		writeList(&b, "acceptance-criteria", t.Acceptance)
		b.WriteByte('\n')
	}
	return b.String()
}

func empty(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func clip(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func writeList(b *strings.Builder, name string, rows []string) {
	fmt.Fprintf(b, "%s:\n", name)
	got := filled(rows)
	if len(got) == 0 {
		b.WriteString("  (none)\n")
		return
	}
	for _, line := range got {
		fmt.Fprintf(b, "  - %s\n", line)
	}
}
