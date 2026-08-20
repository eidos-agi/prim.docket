package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/eidos-agi/prim.docket/internal/docket"
)

func runPlanner(args []string) error {
	req, asJSON, err := parsePlannerArgs(args, os.Stdin)
	if err != nil {
		return err
	}
	res, err := docket.Plan(req)
	if err != nil {
		return err
	}
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(res)
	}
	fmt.Printf("dir: %s\ntask: %s\n%s\n", res.Dir, res.TaskID, res.Handoff)
	return nil
}

func parsePlannerArgs(args []string, stdin io.Reader) (docket.PlanRequest, bool, error) {
	asJSON := false
	for _, a := range args {
		if a == "--json" {
			asJSON = true
		}
	}
	if asJSON && !isTTY(stdin) {
		var req docket.PlanRequest
		if err := json.NewDecoder(stdin).Decode(&req); err != nil {
			return req, true, fmt.Errorf("planner json: %w", err)
		}
		return req, true, nil
	}

	var req docket.PlanRequest
	req.Priority = "high"
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
		case "--json":
			asJSON = true
		case "--dir":
			req.Dir = take()
		case "--id":
			req.ID = take()
		case "--name":
			req.Name = take()
		case "--title":
			req.Title = take()
		case "--goal":
			req.Goal = take()
		case "--done":
			req.DoneWhen = append(req.DoneWhen, take())
		case "--negative":
			req.Negative = take()
		case "--out":
			req.OutOfScope = append(req.OutOfScope, splitKV(take()))
		case "--shipped":
			req.Shipped = append(req.Shipped, splitKV(take()))
		case "--proof":
			req.ProofPath = take()
		case "--linear":
			req.Linear = take()
		case "--stop":
			req.Stop = take()
		case "--priority":
			req.Priority = take()
		case "--notes":
			req.Notes = take()
		case "--handoff-rel":
			req.HandoffRel = take()
		case "-h", "--help":
			fmt.Fprint(os.Stderr, plannerHelp)
			os.Exit(0)
		default:
			if strings.HasPrefix(a, "-") {
				return req, asJSON, fmt.Errorf("planner: unknown flag %s", a)
			}
		}
	}
	return req, asJSON, nil
}

func splitKV(s string) docket.KV {
	item, why, ok := strings.Cut(s, "|")
	if !ok {
		return docket.KV{Item: s}
	}
	return docket.KV{Item: strings.TrimSpace(item), Why: strings.TrimSpace(why)}
}

func isTTY(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

const plannerHelp = `docket-prim planner — mint a /goal contract as a docket-prim pack

The pack is the Prim. This verb writes it. /plandocket gathers and calls this.

  docket-prim planner --dir PATH --name "outcome" --goal "outcome" \
    --done "..." --done "..." --negative "..." \
    --out "thing|why" --shipped "piece|status" \
    --proof PATH [--linear none] [--json]

  docket-prim planner --json < request.json

JSON shape is docket.PlanRequest (upgrade the planner by adding fields there).
`
