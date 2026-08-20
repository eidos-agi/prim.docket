# prim.docket — docket-prim

**The first Prim Tool is the docket editor.** Kind: **surface**. Direction: **talk**. Counterpart: a human.

The pack is the Prim. This binary is the editor that cites it. It is not the file and not a `prim.surface` pack.

Same job as [docket.md](https://github.com/eidos-agi/docket.md): schedule, dispatch, close. Different store.

| Product | For | Store |
|---------|-----|--------|
| **docket-md** | People who want markdown | `.docket/**/*.md` |
| **docket-prim** | People using the Prim format | this pack (`tasks.jsonl`) |

The pack is the file. Fable plans it, Luna executes nodes cheap, Sol closes each subtree expensive (VALIDATION). The LLM is the engine.

This is not a wrap of the markdown tree. `docket-prim editor` opens the surface. `docket-prim tool` prints the pairing.

```bash
go install github.com/eidos-agi/prim.docket/cmd/docket-prim@latest
# or from this clone:
go install ./cmd/docket-prim
```

```bash
docket-prim editor --dir /path/to/docket.prim
docket-prim convert --from /path/to/project          # .docket/ → docket.prim
docket-prim init --name Cerebro
docket-prim planner --dir ./notes/dockets/slice --name "outcome" --goal "outcome" \
  --done "proofable thing" --proof notes/goals/proof/x.md --json
docket-prim task-create --title "Ship login" --priority high
docket-prim task-list
docket-prim task-edit TASK-0001 --status "In Progress"
docket-prim task-complete TASK-0001
docket-prim info --json
```

`--dir` walks up to the pack. `--json` for agents.

Read [INTENTION.md](./INTENTION.md) and [SPEC.md](./SPEC.md). Category: [prim](https://github.com/eidos-agi/prim). Prim is not OKF.
