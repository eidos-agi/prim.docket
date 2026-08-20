# Blind qualitative review — docket-prim

You did **not** write this pack. You do not see the interview, the planner prompt, or the authoring agent's rationale. You see only the digest below and these questions.

This is **not** the schema bar. Schema (`docket-prim validate`) already rejected blanks, too-short lines, placeholders, bad types, nesting, dangling refs, and missing uids. If validate is dirty, stop — tell the authoring agent to run validate first. Do not paper over a hollow card with a kind comment.

Answer **every** question. One line each:

`Q<n> PASS|FAIL <TASK-id or MS-id or pack> — one sentence why`

FAIL when unsure. Do not invent cards. Do not give the author a pass because the title sounds serious.

## Protocol (authoring agent, after this returns)

- Quantitative `docket-prim validate --dir <pack>` must be clean before this review counts.
- Fix every FAIL (`docket-prim task-edit` / add cards). Re-run validate, then `docket-prim review`.
- At most **two follow-up rounds** after the first review (initial + 2 fix cycles = 3 reviews max).
- After two follow-ups, remaining FAILs go to the **human**. Do not start a third rewrite. Ask.

---

## Nesting and overlay

1. Is milestone used only as overlay (`milestone: MS-*` on cards), never as a `parent`?
2. Does every TEST parent a TASK (never a GOAL, PLAN, GUARD, or milestone)?
3. Does every GUARD parent a GOAL, PLAN, or TASK it actually constrains — not a leftover pile?
4. Is anything nested under a GUARD, TEST, or VALIDATION? (Forbidden.)
5. Is any PLAN parented to a TASK? (Forbidden.)
6. Are GOALs only under nothing or another GOAL?
7. Are PLANs only under GOAL or PLAN?
8. Are TASKs only under GOAL, PLAN, or TASK?
9. If there are multiple GOALs, is that a real split of outcomes, not one outcome chopped to look exploded?
10. Are `dependencies` a DAG of order, not a second parent tree pretending to be nesting?

## Explosion shape

11. Is there at least one GOAL that names an outcome that will exist after the run?
12. Is there at least one PLAN that is a sequence/ladder, not a renamed TASK?
13. Is there at least one TASK that is work, not a restated GOAL?
14. Is there at least one TEST that a stranger can run without the authoring agent in the room?
15. Is there at least one GUARD that still fails after the change ships?
16. Does every GOAL have a GUARD child?
17. Is a single GOAL with checkboxes pretending to be the whole plan? (The muzzle. FAIL.)
18. Are TESTs missing on TASKs that claim to be done-when / proofable work?

## Close-out

67. Does every GOAL, PLAN, or TASK that has work children (GOAL/PLAN/TASK) have a VALIDATION child?
68. Is any VALIDATION just a renamed TEST (a command on one TASK) rather than closing the parent after its children?
69. Would a 4-deep tree have a VALIDATION at each depth, or did they skip levels to go faster?
70. Is a parent marked Done while its VALIDATION is still open?
71. Is a VALIDATION marked Done while a work sibling is still open?
72. Does any VALIDATION parent a GUARD, TEST, VALIDATION, or milestone?

## Titles

19. Is any title vague (fix, handle, improve, update, cleanup, various, misc, stuff)?
20. Is any title a restatement of its parent's title with one adjective swapped?
21. Is any title an implementation how ("add a mutex") when the card is a GOAL or TEST?
22. Is any title longer than a headline (≈12 words) without earning it?
23. Do two cards share a title that is not an obvious clone the digest explains?
24. Does a GUARD title name the refuse (what must still fail), not the happy path?

## Notes (user story / technical brief)

25. Does every notes field add facts the title does not already say?
26. Is any notes a prompt-residue ("as an AI", "the user wants", "we should consider")?
27. Is any GOAL notes a task list instead of the outcome?
28. Is any TASK notes a design essay with no named inputs/outputs/refuse?
29. Is any TEST notes missing the command or artifact a stranger would look at?
30. Is any GUARD notes describing work to do, rather than a constraint that must keep failing?
31. Do notes disagree with `status` (says done / already shipped vs To Do)?
32. Do notes disagree with `blocked_reason` (blocked in prose, empty reason, or the reverse)?

## Requirements

33. Does each requirement say what must be true of the change, not how to code it?
34. Is any requirement a restatement of the title?
35. Are two requirements on the same card the same claim in different words?
36. Is any requirement non-binary ("better", "more robust", "clean", "nice UX")?
37. Is any requirement an open research question instead of a lock?
38. Do requirements on a TEST belong on the TASK it proves instead?

## Test cases

39. Is every test-case falsifiable (a command, screenshot path, URL, or Linear state — not "it works")?
40. Does any TEST lack a refuse case (what still fails after ship)?
41. Are test-cases implementation steps ("write the function") instead of observations?
42. Would a stranger know pass vs fail from the test-case line alone?
43. Are two TESTs proving the same TASK with the same observation?
44. Is a test-case a requirement in disguise (no observation)?

## Acceptance

45. Can a stranger mark the card done from acceptance-criteria only, without chatting the author?
46. Is any acceptance line "LGTM" / "looks good" / "human is happy"?
47. Is any acceptance an implementation leftover ("PR merged") with no product proof?
48. Do acceptance lines duplicate the test-cases verbatim with no extra bar?
49. Is false-green named somewhere (what would look done and is not)?
50. For a GOAL, do acceptance lines describe the world after, not the first TASK?

## Guards and refuses

51. Does each GUARD name something that must still fail, deny, or never happen?
52. Is any GUARD a duplicate of another GUARD with a different parent?
53. Is a pack-level refuse (don't do X anywhere) incorrectly parented as if it were a TASK bug?
54. Would shipping the parent while ignoring the GUARD still look green?

## Milestones and dates

55. Does every milestone have at least one live card tagging it? (Or FAIL the unused overlay.)
56. Are cards that clearly belong to a wave/gate missing `milestone`?
57. Is any milestone acting as a parent in prose ("under Wave 2") while `parent` points somewhere else — confirm overlay is consistent?
58. If `start`/`due` are set, does the span make sense vs the milestone due and vs created/updated?
59. Is a closed milestone still carrying open TASKs without a note?
60. Is a Done TASK carrying an open TEST child?

## Safety and portability

61. Do notes or lists contain secrets, tokens, passwords, or private PII that should not be in the pack?
62. Are paths absolute-to-one-laptop in a way that a second machine cannot run the TEST?
63. Are there encoding glitches, replacement characters, or truncated JSON leftovers in titles/notes?
64. Do Linear ids or URLs look invented (`EID-0000`, example.com) when the interview said a real id?
65. Is out-of-scope work smuggled in as a TASK anyway?
66. Does any live card lack a uid or use an illegal type (FEATURE, EPIC) — evidence the line was printf'd into jsonl instead of minted by docket-prim?

---

## Pack digest

(appended by `docket-prim review`)
