package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/eidos-agi/prim.docket/internal/docket"
	"github.com/eidos-agi/prim.docket/internal/editor"
)

func usage() {
	fmt.Fprintf(os.Stderr, `docket-prim %s — docket editor (surface / talk)

The pack is the Prim. This binary is the docket editor that cites it.
Not the file. Not a prim.surface pack. docket-md is markdown.

Usage:
  docket-prim editor [--dir PATH] [--port N]
  docket-prim tool [--dir PATH] [--json]
  docket-prim convert [--from PATH] [--dir PATH] [--json]
  docket-prim init [--dir PATH] [--name NAME] [--id ID]
  docket-prim info [--dir PATH] [--json]
  docket-prim task-create --title TEXT [--status S] [--priority P] [--milestone ID]
                          [--notes TEXT] [--blocked TEXT] [--parent ID]
                          [--tags a,b] [--assignees a,b] [--dir PATH] [--json]
  docket-prim task-list [--status S] [--dir PATH] [--json]
  docket-prim task-view ID [--dir PATH] [--json]
  docket-prim task-edit ID [--title T] [--status S] [--priority P] [--milestone ID]
                           [--notes TEXT] [--blocked TEXT] [--parent ID] [--dir PATH] [--json]
  docket-prim task-complete ID [--dir PATH] [--json]
  docket-prim task-archive ID [--dir PATH] [--json]
  docket-prim task-search QUERY [--dir PATH] [--json]
  docket-prim milestone-create --title TEXT [--due DATE] [--dir PATH] [--json]
  docket-prim milestone-list [--dir PATH] [--json]
  docket-prim milestone-close ID [--dir PATH] [--json]

  --dir defaults to cwd and walks up to the pack.
`, docket.Version)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "-h", "--help", "help":
		usage()
		return
	case "--version", "version":
		fmt.Println("docket-prim", docket.Version)
		return
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd string, args []string) error {
	f := parseFlags(args)
	switch cmd {
	case "editor", "edit":
		return editor.Serve(f.dir, f.port, !f.json)
	case "tool":
		p, err := docket.Open(f.dir)
		if err != nil {
			return err
		}
		t := p.Tool()
		if f.json {
			return emit(t, true)
		}
		fmt.Println(t.Line())
		return nil
	case "convert":
		from := f.from
		if from == "" {
			from = "."
		}
		dest := f.dir
		if dest == "." {
			dest = ""
		}
		p, err := docket.ConvertMarkdown(from, dest)
		if err != nil {
			return err
		}
		return emit(map[string]any{
			"dir": p.Dir, "id": p.Project.ID, "name": p.Project.Name,
			"tasks": len(p.Tasks), "milestones": len(p.Milestones), "archived": len(p.Archive),
			"cites": p.Tool().Cites,
		}, f.json)
	case "init":
		p, err := docket.Init(f.dir, f.name, f.id)
		if err != nil {
			return err
		}
		return emit(map[string]string{"dir": p.Dir, "id": p.Project.ID, "name": p.Project.Name}, f.json)
	case "info":
		p, err := docket.Open(f.dir)
		if err != nil {
			return err
		}
		return emit(map[string]any{
			"dir": p.Dir, "id": p.Project.ID, "name": p.Project.Name,
			"tasks": len(p.Tasks), "milestones": len(p.Milestones), "archived": len(p.Archive),
		}, f.json)
	case "task-create":
		p, err := docket.Open(f.dir)
		if err != nil {
			return err
		}
		t, err := p.CreateTask(docket.Task{
			Title: f.title, Status: f.status, Priority: f.priority, Milestone: f.milestone,
			Notes: f.notes, BlockedReason: f.blocked, Parent: f.parent,
			Tags: splitCSV(f.tags), Assignees: splitCSV(f.assignees),
		})
		if err != nil {
			return err
		}
		return emitTask(t, f.json)
	case "task-list":
		p, err := docket.Open(f.dir)
		if err != nil {
			return err
		}
		rows := p.ListTasks(f.status)
		if f.json {
			return emit(rows, true)
		}
		if len(rows) == 0 {
			fmt.Println("no tasks")
			return nil
		}
		for _, t := range rows {
			fmt.Println(t.Line())
		}
		return nil
	case "task-view":
		id, err := f.argID()
		if err != nil {
			return err
		}
		p, err := docket.Open(f.dir)
		if err != nil {
			return err
		}
		t, _, ok := p.Task(id)
		if !ok {
			return fmt.Errorf("task %s not found", id)
		}
		return emitTask(*t, f.json)
	case "task-edit":
		id, err := f.argID()
		if err != nil {
			return err
		}
		p, err := docket.Open(f.dir)
		if err != nil {
			return err
		}
		t, err := p.EditTask(id, f.edit())
		if err != nil {
			return err
		}
		return emitTask(t, f.json)
	case "task-complete":
		id, err := f.argID()
		if err != nil {
			return err
		}
		p, err := docket.Open(f.dir)
		if err != nil {
			return err
		}
		t, err := p.CompleteTask(id)
		if err != nil {
			return err
		}
		return emitTask(t, f.json)
	case "task-archive":
		id, err := f.argID()
		if err != nil {
			return err
		}
		p, err := docket.Open(f.dir)
		if err != nil {
			return err
		}
		t, err := p.ArchiveTask(id)
		if err != nil {
			return err
		}
		return emitTask(t, f.json)
	case "task-search":
		if f.query == "" && len(f.rest) > 0 {
			f.query = strings.Join(f.rest, " ")
		}
		if f.query == "" {
			return fmt.Errorf("task-search needs a query")
		}
		p, err := docket.Open(f.dir)
		if err != nil {
			return err
		}
		rows := p.Search(f.query)
		if f.json {
			return emit(rows, true)
		}
		for _, t := range rows {
			fmt.Println(t.Line())
		}
		return nil
	case "milestone-create":
		p, err := docket.Open(f.dir)
		if err != nil {
			return err
		}
		m, err := p.CreateMilestone(f.title, f.due)
		if err != nil {
			return err
		}
		return emit(m, f.json)
	case "milestone-list":
		p, err := docket.Open(f.dir)
		if err != nil {
			return err
		}
		if f.json {
			return emit(p.Milestones, true)
		}
		if len(p.Milestones) == 0 {
			fmt.Println("no milestones")
			return nil
		}
		for _, m := range p.Milestones {
			fmt.Printf("%s — %s (%s)\n", m.ID, m.Title, m.Status)
		}
		return nil
	case "milestone-close":
		id, err := f.argID()
		if err != nil {
			return err
		}
		p, err := docket.Open(f.dir)
		if err != nil {
			return err
		}
		m, err := p.CloseMilestone(id)
		if err != nil {
			return err
		}
		return emit(m, f.json)
	default:
		usage()
		os.Exit(2)
		return nil
	}
}

type flags struct {
	dir, from, name, id, title, status, priority, milestone, notes, blocked, parent string
	tags, assignees, due, query                                                     string
	port                                                                            int
	json                                                                            bool
	setTitle, setStatus, setPriority, setMilestone, setNotes, setBlocked, setParent bool
	rest                                                                            []string
}

func parseFlags(args []string) *flags {
	f := &flags{dir: ".", port: 7420}
	for i := 0; i < len(args); i++ {
		a := args[i]
		take := func() string {
			i++
			if i < len(args) {
				return args[i]
			}
			return ""
		}
		switch a {
		case "--dir":
			f.dir = take()
		case "--from":
			f.from = take()
		case "--name":
			f.name = take()
		case "--id":
			f.id = take()
		case "--title":
			f.title = take()
			f.setTitle = true
		case "--status":
			f.status = take()
			f.setStatus = true
		case "--priority":
			f.priority = take()
			f.setPriority = true
		case "--milestone":
			f.milestone = take()
			f.setMilestone = true
		case "--notes":
			f.notes = take()
			f.setNotes = true
		case "--blocked":
			f.blocked = take()
			f.setBlocked = true
		case "--parent":
			f.parent = take()
			f.setParent = true
		case "--tags":
			f.tags = take()
		case "--assignees":
			f.assignees = take()
		case "--due":
			f.due = take()
		case "--port":
			n, err := strconv.Atoi(take())
			if err != nil || n < 0 {
				fmt.Fprintln(os.Stderr, "invalid --port")
				os.Exit(2)
			}
			f.port = n
		case "--json":
			f.json = true
		case "-h", "--help":
			usage()
			os.Exit(0)
		default:
			if strings.HasPrefix(a, "-") {
				fmt.Fprintln(os.Stderr, "unknown flag", a)
				os.Exit(2)
			}
			f.rest = append(f.rest, a)
		}
	}
	return f
}

func (f *flags) argID() (string, error) {
	if len(f.rest) == 0 {
		return "", fmt.Errorf("task/milestone id required")
	}
	return f.rest[0], nil
}

func (f *flags) edit() docket.TaskEdit {
	var e docket.TaskEdit
	if f.setTitle {
		e.Title = &f.title
	}
	if f.setStatus {
		e.Status = &f.status
	}
	if f.setPriority {
		e.Priority = &f.priority
	}
	if f.setMilestone {
		e.Milestone = &f.milestone
	}
	if f.setNotes {
		e.Notes = &f.notes
	}
	if f.setBlocked {
		e.BlockedReason = &f.blocked
	}
	if f.setParent {
		e.Parent = &f.parent
	}
	return e
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func emit(v any, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(v)
	}
	switch x := v.(type) {
	case docket.Task:
		fmt.Println(x.Line())
	case docket.Milestone:
		fmt.Printf("%s — %s (%s)\n", x.ID, x.Title, x.Status)
	case map[string]string:
		for k, val := range x {
			fmt.Printf("%s: %s\n", k, val)
		}
	case map[string]any:
		b, err := json.MarshalIndent(x, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	default:
		b, err := json.MarshalIndent(x, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	}
	return nil
}

func emitTask(t docket.Task, asJSON bool) error {
	return emit(t, asJSON)
}
