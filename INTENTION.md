# Why docket-prim exists

`docket-md` is for people who want markdown on disk — one `.md` file per task, readable without a tool.

`docket-prim` is for people who want one pack both a human and an agent can open: a local board, a score, a CLI, jsonl on disk.

Do not convert a `.docket/` tree and call it a prim. That is still docket-md. The new format is the product.

Same execution nouns (task, milestone, status, blocked, DoD). New container. No new fields until the format needs them.

The first Prim Tool is the **docket editor**: a **surface** that **talks** to a human and **cites** the pack. `docket-prim` is the binary. A later connector would talk to a system. Neither is the file. Do not mint `prim.surface`.
