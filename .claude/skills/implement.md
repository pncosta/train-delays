---
name: implement
description: Implement a task, bug fix, or feature in pt-train-delays. Delegates the whole change to a single implementer agent (write access) that follows the repo's existing conventions, then independently gates on build/vet/test and loops fixes until green. Does NOT run /review — that stays a separate, human-gated step. Use when the user asks to "implement", "build", "add", "fix", or "make the change" (from a BACKLOG.md item or an ad-hoc request).
---

# /implement — build a change coherently, then gate on the build

Orchestrate implementation by handing the **whole change to one `implementer` agent**, then
independently verifying it builds/tests cleanly. One agent, not a fan-out, keeps the change coherent.

```
Ideate ─▶ Implement ─▶ Review
(/ideate)  (this skill)  (/review)
```

**Your role is orchestrator, not author.** Don't write or edit code yourself here — scope the task,
delegate all of it to the `implementer` agent, and gate on the build. If you catch yourself about to
open an editor, stop and spawn the agent instead.

## 1. Pin down the task

- Take it from the user's request, or point at a specific `BACKLOG.md` bullet (grep/read it so you can
  pass its **exact text** to the agent — that bullet is the spec).
- If it's ambiguous in a way that changes *what* gets built (not just how), ask the user now — don't
  delegate a guess.

## 2. Branch

Pet-project scale doesn't need worktree isolation (that's for running several implementations in
parallel — not a concern here), but each change should still land on its own branch, not directly on
`main`:

- `git status` / `git branch --show-current` first.
- If you're on `main` (or another already-landed branch) with no relevant in-progress work, create and
  check out a new branch named for the task (e.g. `fix/<short-desc>` or `feat/<short-desc>`) from the
  current `main`.
- If you're already on a feature branch that matches this task, keep using it — don't create a new one.
- If it's unclear which applies (mid-work on something else entirely), ask.

## 3. Delegate to the single implementer agent

Spawn **one** agent (Agent tool, `subagent_type: implementer`). This step is not optional — you don't
write the code yourself. Give it:

- the task, stated clearly — or the exact `BACKLOG.md` bullet text if that's the source;
- any assumptions you've already resolved;
- the reminder that it should build/test/fix until green, and update `BACKLOG.md` itself if the task
  came from a bullet (move a fixed bug to `## Fixed`, delete a built idea/feature).

Let it do the implementation and its own build/fix loop; don't micro-manage individual edits.

## 4. Independently gate on the build

When the agent returns, **re-verify yourself** — don't just trust its report. From the repo:

```bash
cd backend && go build ./... && go vet ./...
go test ./...                      # for packages with tests

cd app/train_dashboard && flutter analyze && flutter test   # only if Flutter files changed
```

If anything fails, feed the exact failure output back to the **same** agent (`SendMessage`, not a new
spawn — it keeps context) and have it fix the cause. Loop until clean.

## 5. Report — and hand off to review

```
## Implemented
<1–3 sentence description of the change>

## Files changed
path/to/file.go — what/why
...

## Verification
build: ✓   vet: ✓   test: ✓ (N tests)   [flutter analyze/test: ✓]
<what was fixed to get to green, if anything>

## BACKLOG.md
<moved to Fixed / removed as built / unchanged — one line>

## Design notes / left for review
<convention calls, assumptions, anything review should look at closely>
```

Then **recommend `/review`** — don't run it yourself. The build gate here covers what a machine can
check; the review pass is the human go/no-go before commit. Also remind the user that committing/
pushing is their call, never automatic.

## Notes

- **Coherence over parallelism.** Exactly one `implementer` agent per `/implement` run; revise it via
  `SendMessage` rather than spawning a duplicate.
- **No commit/push here, ever.** Neither this skill nor the agent commits — that's always an explicit,
  separate ask from the user.
- **Deliberately not the company-scale version.** No story IDs, no task-breakdown/challenge handoff, no
  worktree isolation, no telemetry/cost ledger — `BACKLOG.md` plus a branch is the whole workflow at
  this scale.
- **First-run caveat.** A newly-added skill isn't dispatchable in the session that created it — reload
  (`/agents` or restart) before the first `/implement`.
