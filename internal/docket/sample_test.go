package docket

func brief(title string) (notes string, req, cases, acc []string) {
	notes = "As the executor of this pack I need " + title + " as a change a stranger can prove from artifacts only. Technical contract: named inputs, named outputs, and a refuse that still fails after it ships. This brief is not the title restated."
	req = []string{
		"The proof artifact or command is named and is the only path to Done.",
		"False green is defined: what would look done and is not.",
	}
	cases = []string{
		"Run the named proof; the expected signal is in acceptance.",
	}
	acc = []string{
		"A stranger marks complete from the proof artifact only.",
		"The named refuse still fails after the change.",
	}
	return
}

func live(title string) Task {
	n, r, c, a := brief(title)
	return Task{Title: title, Notes: n, Requirements: r, TestCases: c, Acceptance: a}
}

func typed(title, typ string) Task {
	t := live(title)
	t.Type = typ
	if NormType(typ) == "TEST" {
		t.TestCases = append(append([]string{}, t.TestCases...), "Repeat the proof after a refuse; it still fails.")
	}
	return t
}
