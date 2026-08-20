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

	created := postJSON(t, ts.URL+"/api/tasks", map[string]string{"title": "Ship login", "priority": "high"})
	if created["id"] != "TASK-0001" {
		t.Fatalf("create %+v", created)
	}

	edited := patchJSON(t, ts.URL+"/api/tasks/TASK-0001", map[string]string{
		"title": "Ship login now", "status": "In Progress", "priority": "high",
	})
	if edited["title"] != "Ship login now" || edited["status"] != "In Progress" {
		t.Fatalf("edit %+v", edited)
	}

	moved := patchJSON(t, ts.URL+"/api/tasks/TASK-0001", map[string]string{"status": "Done"})
	if moved["status"] != "Done" || moved["title"] != "Ship login now" || moved["priority"] != "high" {
		t.Fatalf("partial status %+v", moved)
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
		res.Body.Close()
	}
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
