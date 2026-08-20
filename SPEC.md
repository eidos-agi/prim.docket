# prim.docket — docket-prim SPEC (v0.1.0-draft)

Profile for an **execution prim**. Product name: **docket-prim**. Family name: `prim.docket`.

Not an OKF profile. Not a markdown tree. [docket-md](https://github.com/eidos-agi/docket.md) remains the markdown product.

---

## 1. Split

| | docket-md | docket-prim |
|---|-----------|-------------|
| Audience | Want `.md` | Want the Prim format |
| Store | `.docket/tasks/*.md` + YAML frontmatter | This pack |
| Open | Any editor | Profile tools + `ui` |
| Board | Optional markdown viewer | Surface tool that **cites** this prim |

Do not mint `prim.surface`. The board is a tool.

---

## 2. Face

```yaml
---
profile: docket
docket_version: "0.1.0"
type: queue
title: Cerebro
status: open
---
```

`profile: docket`. No `okf_version` required.

---

## 3. Store

Directory (canonical) or a `.prim.zip` / `.prim` interchange whose root is that directory.

```
<pack>/
  index.md           # face
  docket.json        # project id + name
  tasks.jsonl        # append-only; last line per id wins
  milestones.jsonl   # append-only; last line per id wins
  archive.jsonl      # append-only; last line per id wins
  log.md             # append-only
```

`index.md` and `docket.json` rewrite. The jsonl files never compact on the write path: each mutation appends one record. `Open` folds last-line-wins. A live-task tombstone is `archived: true` on `tasks.jsonl` (that id drops out of the live set); the full row also appends to `archive.jsonl`.

**`docket-prim` is the only appender.** `tasks.jsonl`, `milestones.jsonl`, `archive.jsonl`, and `log.md` are written only by this binary: `planner`, `task-*`, `milestone-*`, `convert`, and the editor API — all of them call `Pack.append*` / `Pack.Log`. Agents never `printf >>`, never `WriteFile` those paths, never invent `uid`. One encoder, one id allocator, `CheckCard` before the line hits disk. That is what makes a change deterministic. A hand-rolled line is not a card; `validate` will say so (missing `uid`, hollow body, illegal `type`).

Task records reuse the **docket-md field names**, plus `type`:

`id`, `uid`, `title`, `type`, `status`, `priority`, `milestone`, `parent`, `subtasks`, `assignees`, `tags`, `dependencies`, `requirements`, `test-cases`, `acceptance-criteria`, `definition-of-done`, `blocked_reason`, `created`, `updated`, `start`, `due`, `notes`.

`id` is the face number (`TASK-0001`, `MS-0001`). `uid` is the unique record id: UUID v4 `+` 16-byte hex salt (`xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx+` 32 hex). The verb generates it at mint. Never reuse. Unique across live tasks, archive, and milestones. Parent/dep edges still use face `id`. Agents never invent `uid` and never append a line that needs one.

`notes` is the **user story or technical brief**. A title is not a card. `notes` must not be the title, must be ≥160 characters, and must contain ≥2 sentences. Every live card (`GOAL`, `PLAN`, `TASK`, `GUARD`, `TEST`, `VALIDATION`) also requires:

- `requirements` — ≥2 non-empty lines (what must be true of the change)
- `test-cases` — ≥1 non-empty line; **≥2 if `type` is `TEST`**
- `acceptance-criteria` — ≥2 non-empty lines (how a stranger marks the card done)

`definition-of-done` stays optional extra proof.

`type`: `TASK` | `GOAL` | `PLAN` | `GUARD` | `TEST` | `VALIDATION` (default `TASK`). Milestones stay in `milestones.jsonl`, not as a task type. A milestone is a **date/checkpoint overlay** (`milestone: MS-0001` on a card). It is never a `parent`. Sprint/release, if they appear later, are the same kind of overlay. Optional `start` and `due` (`YYYY-MM-DD`) on a card are schedule overlay on the sheet, not a nesting parent, not required by the body bar, and **not** the Gantt axis. Empty is unscheduled. Gantt (`?view=gantt`) is rank: same parent-plus-dependencies order as DAG play. GUARD sits in a Gov column; a parent bar spans its descendants so the tree does not collapse into one column. DAG play walks topological ranks from `parent` plus `dependencies`; same-rank cards glow together for two seconds, then the next rank, so parallel work is visible as a wave. GUARD cards sit in a **Governs** band above that tree — they constrain the work below, they are not a rank in it. Play skips them. A **Hierarchy** band holds the parent tree. A **No parent** band holds unlinked work. Those three sections are separated. The editor surface is responsive: the rail stays visible, the sheet stacks, charts reflow. JSON view (`?view=json`) dumps the visible cards as pretty records with the jsonl field names; click a record to open the sheet. **Copy jsonl** is one record per line — the blob a cheap builder gets. **Download jsonl** saves that same blob as `{project-id}.jsonl`.

`GUARD` is a constraint on another card — not a leftover type. `parent` is required and must be a `GOAL`, `PLAN`, or `TASK` (the work it constrains). Pack-level refuses parent the root GOAL. A pile of unparented GUARDs is invalid.

`TEST` proves a TASK (command, screenshot, artifact a stranger can run). `VALIDATION` is not a TEST. It is how the system **closes a parent that has children**. Any `GOAL` / `PLAN` / `TASK` that decomposes into work children (`GOAL` / `PLAN` / `TASK`) needs its own VALIDATION child. A 4-deep tree has four VALIDATION cards — that is the point: slower, more automated. Leaves with only GUARD/TEST children do not get one. VALIDATION is a leaf. Work siblings must be Done before VALIDATION can be Done. The parent cannot be Done until its VALIDATION is Done. Completing the children is not closing the parent.

Nesting (`parent`) is the only hierarchy. Allowed child → parent:

| Child | May nest under |
|---|---|
| **GOAL** | none, or another **GOAL** |
| **PLAN** | **GOAL**, **PLAN** |
| **TASK** | **GOAL**, **PLAN**, **TASK** |
| **GUARD** | **GOAL**, **PLAN**, **TASK** (required; leaf) |
| **TEST** | **TASK** (required; leaf) |
| **VALIDATION** | **GOAL**, **PLAN**, **TASK** (required; leaf; close-out) |

Forbidden: anything under **GUARD**, **TEST**, or **VALIDATION**; **TEST** under **GOAL**/**PLAN**; **PLAN** under **TASK**; `parent` pointing at `MS-*`. `dependencies` are a separate DAG (not parent).

Status: `To Do` | `In Progress` | `Done` | `Draft`.  
Priority: `high` | `medium` | `low` | empty.  
Blocked is `blocked_reason`, not a status.

The pack is the file. The LLM is the engine. A docket-prim pack is the plan plus the data: jsonl nodes, not a workflow app sitting next to a spreadsheet.

| Role | Cost | Job |
|---|---|---|
| **Fable** | plan | `docket-prim planner` / `/plandocket` mints the tree. |
| **Luna** | cheap | Executes a node from that jsonl. |
| **Sol** | expensive | Closes a finished subtree via the **VALIDATION** card. |

You send jsonl to Luna. As each node completes, Sol validates. Completing children is not closing the parent. JSON view **Download jsonl** and **Copy jsonl to clipboard** are the same blob Luna gets (visible cards, one record per line). TEST is still the leaf proof; VALIDATION is Sol's close-out.

---

## 4. The first Prim Tool

The first Prim Tool is the **docket editor**. The binary that hosts it is `docket-prim`.

| | |
|---|---|
| **name** | `docket-editor` |
| **as** | editor |
| **kind** | surface |
| **direction** | talk |
| **counterpart** | human |
| **cites** | `docket:<project id>` or the pack path |
| **bin** | `docket-prim` |

It is not the Prim. It does not own the tasks. `docket-prim editor` opens the surface. `docket-prim tool` prints the pairing. The category registry lists it: `prim registry tool docket-editor`. The left rail estimates **prim** bytes (pack files on disk) and tokens (`bytes / 4`), and calls out **jsonl** separately — that is the blob the engine eats.

A later **connector** may cite the same prim. If it is not surface or connector, it is a script. Do not mint `prim.surface`.

`ui` opens. The tool operates. The pack remains the file.

---

## 5. Planner verb

`docket-prim planner` mints a **new** pack from a `PlanRequest`: face + milestones + typed tasks (`GOAL`, `PLAN`, `TASK`, `GUARD`, `TEST`, `VALIDATION`). Empty `tasks` fails (a single GOAL is the muzzle). After write it **validates** the pack (types, nesting, parents, milestones as overlay, deps, no cycles, every card has a story/brief + `requirements` + `test-cases` + `acceptance-criteria` + `uid`, every `GUARD` parents the card it constrains, every decomposing `GOAL`/`PLAN`/`TASK` has a `VALIDATION` close-out). The `tasks` list must include GOAL PLAN TASK GUARD TEST **and** a `GUARD` child on every `GOAL`, or the verb deletes the pack and fails. Same binary, not a second Prim Tool.

`/plandocket` explodes a massive context window into that bucket, then **only** calls this verb (JSON stdin = `PlanRequest`). Upgrade the planner by adding fields to `PlanRequest` in this repo — do not grow the skill.

---

## 6. Two bars after mint

**Quantitative (hollow)** — `docket-prim validate`. Machine. Collect-all schema findings: blank fields, too-short titles/notes/list lines, placeholders (`TODO`/`TBD`/`n/a`/…), bad types, nesting, dangling refs, duplicate titles/ids/uids, missing uids. This is the authoring-agent hollow-card check. A title is not a card. A 12-character requirement is not a requirement. JSON: `{ok, kind: schema, findings[], score}`.

**Quantitative (graph)** — `docket-prim score`. Machine. Deterministic 0–100 plan score. Same pack, same number. No model. It counts what does not make sense as a plan: missing GOAL / PLAN / TASK / GUARD / TEST / VALIDATION, no milestones, unguarded GOALs, decomposing nodes with no VALIDATION close-out, TASKs with no TEST child, unparented leaves, loose work beside a GOAL, extra root GOALs, a flat tree (no parent edges), parent/dependency cycles, dangling refs, work untagged when milestones exist. Deduction weights are fixed in `internal/docket/score.go`. Hollow notes do **not** move this number — that is `validate`. JSON: `{score, max: 100, kind: plan, ok, deductions[], counts}`. The editor left rail shows `N/100 plan`. The **Plan QA** view (`?view=qa`) sits after Gantt and before JSON and lists the same score, counts, and clickable deduction ids.

**Lift** — `docket-prim lift`. Machine. Raises the graph score with nest-legal patches written only by this binary: mint `VALIDATION` on decomposing parents, mint `TEST` under TASKs that have none, optionally mint `GUARD` on unguarded GOALs (`--guards`), optionally parent orphan GUARDs (`--attach-guards ID`) and loose PLAN/TASK/extra-root GOAL (`--attach-loose ID`). One milestone tags untagged work; many milestones stay untagged. `--dry-run` prints before → after and does not append jsonl. It does not invent a GOAL. Remaining deductions print after the ops.

**Qualitative** — `docket-prim review`. Prints a ≥50-question prompt plus a pack digest for a **blind** agent (no planner context, no interview). Meaning, overlay vs parent, falsifiable tests, refuses, secrets — not length counts. The authoring agent must not answer this prompt itself.

Protocol:

1. `validate` clean, or the review is invalid. `score` is the graph bar; it does not replace validate.
2. Blind agent answers every question `Q<n> PASS|FAIL <id> — why`.
3. Authoring agent fixes FAILs, re-runs validate then review.
4. At most **two follow-up rounds** after the first review.
5. Remaining FAILs go to the **human**. Do not start a third rewrite.

The planner stdout includes `review: docket-prim review --dir <pack>`.

---

## 7. Status

v0.1.0-draft. `docket-prim` is the first Prim Tool (surface / talk). A board, if added, is another surface — not a new pack type. `planner` is a mint verb on that binary.
