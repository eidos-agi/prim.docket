package editor

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eidos-agi/prim.docket/internal/docket"
)

func TestEditorCreatesAndEdits(t *testing.T) {
	dir := t.TempDir()
	if _, err := docket.Init(dir, "Fixture", "fix"); err != nil {
		t.Fatal(err)
	}
	s := &Server{Dir: dir}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)

	created := postJSON(t, ts.URL+"/api/tasks", cardJSON("Ship login", "high"))
	if created["id"] != "TASK-0001" {
		t.Fatalf("create %+v", created)
	}

	edited := patchJSON(t, ts.URL+"/api/tasks/TASK-0001", map[string]string{
		"title": "Ship login now", "status": "In Progress", "priority": "high",
		"start": "2026-08-20", "due": "2026-09-01",
	})
	if edited["title"] != "Ship login now" || edited["status"] != "In Progress" {
		t.Fatalf("edit %+v", edited)
	}
	if edited["start"] != "2026-08-20" || edited["due"] != "2026-09-01" {
		t.Fatalf("edit dates %+v", edited)
	}

	moved := patchJSON(t, ts.URL+"/api/tasks/TASK-0001", map[string]string{"status": "Done"})
	if moved["status"] != "Done" || moved["title"] != "Ship login now" || moved["priority"] != "high" {
		t.Fatalf("partial status %+v", moved)
	}
	if moved["start"] != "2026-08-20" || moved["due"] != "2026-09-01" {
		t.Fatalf("dates dropped on status patch %+v", moved)
	}

	res, err := http.Get(ts.URL + "/api/state")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var state map[string]any
	if err := json.NewDecoder(res.Body).Decode(&state); err != nil {
		t.Fatal(err)
	}
	tool := state["tool"].(map[string]any)
	if tool["name"] != "docket-editor" || tool["as"] != "editor" {
		t.Fatalf("tool %+v", tool)
	}
	if tool["cites"] != "docket:fix" {
		t.Fatalf("cites %v", tool["cites"])
	}
	if res, err := http.Get(ts.URL + "/"); err != nil || res.StatusCode != 200 {
		t.Fatalf("page %v %v", err, res)
	} else {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(res.Body)
		res.Body.Close()
		body := buf.String()
		if !bytes.Contains(buf.Bytes(), []byte(`id="poll-pie"`)) || !bytes.Contains(buf.Bytes(), []byte("Time until next pack check")) {
			t.Fatalf("page missing poll pie: %d bytes", len(body))
		}
		if !bytes.Contains(buf.Bytes(), []byte("function treeWalk")) {
			t.Fatal("page missing treeWalk")
		}
		if !bytes.Contains(buf.Bytes(), []byte(`id="view-dag"`)) || !bytes.Contains(buf.Bytes(), []byte("function dagLayout")) {
			t.Fatal("page missing DAG view")
		}
		if !bytes.Contains(buf.Bytes(), []byte("Governs")) || !bytes.Contains(buf.Bytes(), []byte("dag-sec-governs")) {
			t.Fatal("page missing GUARD governs section")
		}
		if !bytes.Contains(buf.Bytes(), []byte("dag-sec-tree")) || !bytes.Contains(buf.Bytes(), []byte("Hierarchy")) {
			t.Fatal("page missing DAG hierarchy section")
		}
		if !bytes.Contains(buf.Bytes(), []byte("dag-sec-loose")) || !bytes.Contains(buf.Bytes(), []byte("No parent")) {
			t.Fatal("page missing DAG no-parent section")
		}
		if !bytes.Contains(buf.Bytes(), []byte("gantt-sec-head")) {
			t.Fatal("page missing Gantt section separators")
		}
		if bytes.Contains(buf.Bytes(), []byte("aside { display: none")) {
			t.Fatal("sidebar still hidden on narrow viewports")
		}
		if !bytes.Contains(buf.Bytes(), []byte("@media (max-width: 860px)")) {
			t.Fatal("page missing responsive breakpoint")
		}
		if !bytes.Contains(buf.Bytes(), []byte(`(t.type || "TASK") !== "GUARD"`)) {
			t.Fatal("page missing GUARD skip in dagRanks")
		}
		if !bytes.Contains(buf.Bytes(), []byte(`id="view-gantt"`)) || !bytes.Contains(buf.Bytes(), []byte("function ganttHTML")) {
			t.Fatal("page missing Gantt view")
		}
		if !bytes.Contains(buf.Bytes(), []byte("function ganttLayout")) || !bytes.Contains(buf.Bytes(), []byte(`gantt-corner">Rank`)) || !bytes.Contains(buf.Bytes(), []byte("gantt-ticks")) {
			t.Fatal("page missing rank Gantt")
		}
		if bytes.Contains(buf.Bytes(), []byte("start → due")) {
			t.Fatal("Gantt still keyed on calendar dates")
		}
		if !bytes.Contains(buf.Bytes(), []byte(`id="view-json"`)) || !bytes.Contains(buf.Bytes(), []byte("function jsonHTML")) {
			t.Fatal("page missing JSON view")
		}
		if !bytes.Contains(buf.Bytes(), []byte(`data-copy="jsonl"`)) || !bytes.Contains(buf.Bytes(), []byte(`data-download="jsonl"`)) {
			t.Fatal("page missing jsonl copy/download")
		}
		if !bytes.Contains(buf.Bytes(), []byte("Copy jsonl to clipboard")) || !bytes.Contains(buf.Bytes(), []byte("Download jsonl")) {
			t.Fatal("page missing copy/download jsonl links")
		}
		if !bytes.Contains(buf.Bytes(), []byte(`id="dag-play"`)) || !bytes.Contains(buf.Bytes(), []byte("function dagRanks")) {
			t.Fatal("page missing DAG play")
		}
		if !bytes.Contains(buf.Bytes(), []byte("PLAY_MS = 2000")) {
			t.Fatal("page missing DAG play duration")
		}
		if !bytes.Contains(buf.Bytes(), []byte("Constrains")) {
			t.Fatal("page missing GUARD constrains field")
		}
		if !bytes.Contains(buf.Bytes(), []byte("VALIDATION")) || !bytes.Contains(buf.Bytes(), []byte("Closes")) {
			t.Fatal("page missing VALIDATION close-out")
		}
		if !bytes.Contains(buf.Bytes(), []byte("t.uid")) {
			t.Fatal("page missing uid")
		}
		if !bytes.Contains(buf.Bytes(), []byte(`id="stats"`)) || !bytes.Contains(buf.Bytes(), []byte("function packStats")) {
			t.Fatal("page missing docket stats")
		}
		if !bytes.Contains(buf.Bytes(), []byte("function fmtBytes")) || !bytes.Contains(buf.Bytes(), []byte("function fmtTok")) || !bytes.Contains(buf.Bytes(), []byte("stats-size")) {
			t.Fatal("page missing prim byte/token estimates")
		}
		navAt, statsAt, listAt := bytes.Index(buf.Bytes(), []byte(`id="nav"`)), bytes.Index(buf.Bytes(), []byte(`id="stats"`)), bytes.Index(buf.Bytes(), []byte(`id="list"`))
		if navAt < 0 || statsAt < navAt || statsAt > listAt {
			t.Fatal("stats should sit in the left rail under nav")
		}
		if !bytes.Contains(buf.Bytes(), []byte("function closeSheet")) || !bytes.Contains(buf.Bytes(), []byte("sheet-close")) {
			t.Fatal("page missing sheet close")
		}
		if bytes.Contains(buf.Bytes(), []byte(".empty, .blank")) || !bytes.Contains(buf.Bytes(), []byte("no-sheet")) {
			t.Fatal("list empty-state padding still collides with app.blank")
		}
		if !bytes.Contains(buf.Bytes(), []byte(`name="requirements"`)) || !bytes.Contains(buf.Bytes(), []byte(`name="cases"`)) {
			t.Fatal("page missing requirements/test-cases fields")
		}
	}
}

func TestRevMovesAfterCreate(t *testing.T) {
	dir := t.TempDir()
	if _, err := docket.Init(dir, "Fixture", "fix"); err != nil {
		t.Fatal(err)
	}
	s := &Server{Dir: dir}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)

	before := getJSON(t, ts.URL+"/api/rev")
	if before["rev"] == "" {
		t.Fatal("empty rev")
	}
	postJSON(t, ts.URL+"/api/tasks", cardJSON("Watch me", ""))
	after := getJSON(t, ts.URL+"/api/rev")
	if after["rev"] == before["rev"] {
		t.Fatal("rev did not move after create")
	}
	state := getJSON(t, ts.URL+"/api/state")
	if state["rev"] != after["rev"] {
		t.Fatalf("state rev %v vs %v", state["rev"], after["rev"])
	}
	size, _ := state["size"].(map[string]any)
	if size == nil {
		t.Fatal("state missing size")
	}
	if toFloat(size["bytes"]) <= 0 || toFloat(size["jsonl"]) <= 0 {
		t.Fatalf("pack size %+v", size)
	}
}

func cardJSON(title, priority string) map[string]any {
	notes := "As the executor of this pack I need " + title + " as a change a stranger can prove from artifacts only. Technical contract: named inputs, named outputs, and a refuse that still fails after it ships. This brief is not the title restated."
	body := map[string]any{
		"title": title,
		"notes": notes,
		"requirements": []string{
			"The proof artifact or command is named and is the only path to Done.",
			"False green is defined: what would look done and is not.",
		},
		"test-cases": []string{"Run the named proof; the expected signal is in acceptance."},
		"acceptance-criteria": []string{
			"A stranger marks complete from the proof artifact only.",
			"The named refuse still fails after the change.",
		},
	}
	if priority != "" {
		body["priority"] = priority
	}
	return body
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

func getJSON(t *testing.T, url string) map[string]any {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		t.Fatalf("GET %s -> %d", url, res.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func postJSON(t *testing.T, url string, body any) map[string]any {
	t.Helper()
	return doJSON(t, http.MethodPost, url, body)
}

func patchJSON(t *testing.T, url string, body any) map[string]any {
	t.Helper()
	return doJSON(t, http.MethodPatch, url, body)
}

func doJSON(t *testing.T, method, url string, body any) map[string]any {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		t.Fatalf("%s %s -> %d", method, url, res.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}
