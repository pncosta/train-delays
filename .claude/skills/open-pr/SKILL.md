---
name: open-pr
description: Open a GitHub PR for the current, already-implemented change. Resolves the interactive bits (branch sanity, review status) then hands the git/gh mechanics — commit, push, draft title/body, gh pr create — to an isolated open-pr executor agent, so that churn stays out of this session. Use after /review, or whenever the user says "open a PR", "open an MR", "create the PR", or "ship this".
---

# /open-pr — open a GitHub PR

```
Ideate ─▶ Implement ─▶ Review ─▶ Open PR
(/ideate)  (/implement)  (/review)  (this skill)
```

**This skill is a thin launcher.** The mechanical work — git status/diff inspection, committing,
pushing, drafting the PR body, `gh pr create` — runs in the **`open-pr` executor agent**, in its own
context. Your job here is only: check the few things that need a human call, spawn the agent, report
its result.

## 1. Preconditions — resolve here

1. **Not on `main`.** `git branch --show-current`. If it's `main`, stop and ask the user to point at a
   branch — don't create one yourself (`/implement` already does that when it's the one making the
   change).
2. **Review status — soft gate.** If `/review` ran on this change earlier in this session, you have its
   one-line verdict to pass along. If it didn't run, **push back once**: explain that opening a PR
   without a review ships unreviewed work, and offer to run `/review` now. If the user explicitly says
   to skip it anyway, honor that — don't re-run `/review` yourself unasked, and don't silently pretend
   it ran. Pass the agent the tag `review: skipped at user's request` in that case, so the PR body says
   so plainly instead of presenting an empty ledger as a clean review.
3. **`gh` works.** A quick `gh auth status` is enough; if it's not authenticated, point at
   `gh auth login --with-token < .claude/.gh_token` and stop rather than letting the agent hit the same
   wall.

## 2. Spawn the executor

Spawn **one** agent (`subagent_type: open-pr`). Give it:
- the branch (confirmed not `main`);
- the review status from step 1 — either the `/review` verdict or the explicit skip tag;
- the relevant `BACKLOG.md` entry, if you know which one this change closes/builds.

It commits (if needed), pushes, drafts the PR, and creates it — unattended. Don't micromanage its git
commands.

## 3. Report

On `outcome: opened`: report the PR URL, the branch, whether it created a new commit, and the ledger
headline it folded into the body. One or two lines.

On `outcome: blocked`: surface its `reason` and raw error, and stop — don't retry blindly or guess at a
fix.

## Notes

- **Never merges.** Merging is a separate, explicitly-requested action — never implied by opening a PR.
- **No story IDs / external ledger.** `BACKLOG.md` plus the PR description is the whole record at this
  scale.
- **No live review-comment threads.** Unlike a heavier pipeline that posts per-finding PR comments, the
  `/review` ledger just goes in the PR body — proportionate for a solo project.
- **First-run caveat.** A newly-added agent/skill isn't dispatchable in the session that created it —
  reload (`/agents` or restart) before the first `/open-pr`.
