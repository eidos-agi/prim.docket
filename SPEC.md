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
  tasks.jsonl        # one task record per line
  milestones.jsonl   # one milestone record per line
  archive.jsonl      # archived tasks (same fields, removed from tasks.jsonl)
  log.md             # append-only (recommended)
```

Task records reuse the **docket-md field names**. No new model in v0.1:

`id`, `title`, `status`, `priority`, `milestone`, `parent`, `subtasks`, `assignees`, `tags`, `dependencies`, `acceptance-criteria`, `definition-of-done`, `blocked_reason`, `created`, `updated`, `notes`.

Status: `To Do` | `In Progress` | `Done` | `Draft`.  
Priority: `high` | `medium` | `low` | empty.  
Blocked is `blocked_reason`, not a status.

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

It is not the Prim. It does not own the tasks. `docket-prim editor` opens the surface. `docket-prim tool` prints the pairing. The category registry lists it: `prim registry tool docket-editor`.

A later **connector** may cite the same prim. If it is not surface or connector, it is a script. Do not mint `prim.surface`.

`ui` opens. The tool operates. The pack remains the file.

---

## 5. Status

v0.1.0-draft. `docket-prim` is the first Prim Tool (surface / talk). A board, if added, is another surface — not a new pack type.
