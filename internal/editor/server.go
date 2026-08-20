package editor

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/eidos-agi/prim.docket/internal/docket"
)

//go:embed page.html
var page []byte

type Server struct {
	Dir  string
	Addr string
	mu   sync.Mutex
}

func Listen(dir string, port int) (*Server, net.Listener, error) {
	p, err := docket.Open(dir)
	if err != nil {
		return nil, nil, err
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, nil, err
	}
	return &Server{Dir: p.Dir, Addr: "http://" + ln.Addr().String()}, ln, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.page)
	mux.HandleFunc("/api/state", s.state)
	mux.HandleFunc("/api/rev", s.rev)
	mux.HandleFunc("/api/tasks", s.tasks)
	mux.HandleFunc("/api/tasks/", s.task)
	return mux
}

func Serve(dir string, port int, openBrowser bool) error {
	s, ln, err := Listen(dir, port)
	if err != nil {
		return err
	}
	p, err := docket.Open(s.Dir)
	if err != nil {
		ln.Close()
		return err
	}
	fmt.Println(p.Tool().Line())
	fmt.Println(s.Addr)
	if openBrowser {
		_ = OpenURL(s.Addr)
	}
	return http.Serve(ln, s.Handler())
}

func OpenURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func (s *Server) page(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(page)
}

func (s *Server) rev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]string{"rev": docket.Rev(s.Dir)})
}

func (s *Server) state(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := docket.Open(s.Dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tool := p.Tool()
	writeJSON(w, map[string]any{
		"dir":        p.Dir,
		"rev":        docket.Rev(p.Dir),
		"size":       docket.SizeOf(p.Dir),
		"score":      docket.ScoreOf(p),
		"project":    p.Project,
		"tasks":      p.Tasks,
		"milestones": p.Milestones,
		"tool": map[string]string{
			"name":        tool.Name,
			"kind":        tool.Kind,
			"direction":   tool.Direction,
			"counterpart": tool.Counterpart,
			"as":          tool.As,
			"cites":       tool.Cites,
			"line":        tool.Line(),
		},
	})
}

type taskIn struct {
	Title         *string  `json:"title"`
	Type          *string  `json:"type"`
	Status        *string  `json:"status"`
	Priority      *string  `json:"priority"`
	Milestone     *string  `json:"milestone"`
	Parent        *string  `json:"parent"`
	Notes         *string  `json:"notes"`
	BlockedReason *string  `json:"blocked_reason"`
	Requirements  []string `json:"requirements"`
	TestCases     []string `json:"test-cases"`
	Acceptance    []string `json:"acceptance-criteria"`
	Start         *string  `json:"start"`
	Due           *string  `json:"due"`
}

func (s *Server) tasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var in taskIn
	if err := readJSON(r, &in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := docket.Open(s.Dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t, err := p.CreateTask(docket.Task{
		Title: deref(in.Title), Type: deref(in.Type), Status: deref(in.Status), Priority: deref(in.Priority),
		Milestone: deref(in.Milestone), Notes: deref(in.Notes), BlockedReason: deref(in.BlockedReason),
		Parent: deref(in.Parent), Requirements: in.Requirements, TestCases: in.TestCases, Acceptance: in.Acceptance,
		Start: deref(in.Start), Due: deref(in.Due),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, t)
}

func (s *Server) task(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	id, act, _ := strings.Cut(rest, "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := docket.Open(s.Dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var t docket.Task
	switch {
	case act == "" && r.Method == http.MethodPatch:
		var in taskIn
		if err := readJSON(r, &in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		t, err = p.EditTask(id, docket.TaskEdit{
			Title:         in.Title,
			Type:          in.Type,
			Status:        in.Status,
			Priority:      in.Priority,
			Milestone:     in.Milestone,
			Parent:        in.Parent,
			Notes:         in.Notes,
			BlockedReason: in.BlockedReason,
			Requirements:  acceptEdit(in.Requirements),
			TestCases:     acceptEdit(in.TestCases),
			Acceptance:    acceptEdit(in.Acceptance),
			Start:         in.Start,
			Due:           in.Due,
		})
	case act == "complete" && r.Method == http.MethodPost:
		t, err = p.CompleteTask(id)
	case act == "archive" && r.Method == http.MethodPost:
		t, err = p.ArchiveTask(id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, t)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func acceptEdit(rows []string) *[]string {
	if rows == nil {
		return nil
	}
	return &rows
}

func readJSON(r *http.Request, dest any) error {
	b, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return fmt.Errorf("empty body")
	}
	return json.Unmarshal(b, dest)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}
