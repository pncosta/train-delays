---
name: ideate
description: Ideate/refine a bug, idea, or feature with the user before it's tracked in BACKLOG.md — the leftmost step, before handing work to the implementer agent. Works interactively in the conversation to draw out the problem/purpose, context, and acceptance criteria, then delegates drafting to the read-only `ideator` agent, which researches the real code and the current BACKLOG.md. Creates a NEW backlog bullet, or REFINES an existing one in place. Persists to BACKLOG.md only after the user approves. Use when the user wants to "ideate", "refine an idea", "write up a bug", "flesh out this backlog item", or "scope something before building it".
---

# /ideate — shape or refine a BACKLOG.md entry

A deliberately small version of a bigger "Ideate" pipeline stage, sized for a solo pet project:

```
Ideate ─▶ Implement ─▶ Review
(this skill) (implementer agent) (reviewer agent)
```

There's no Task-Breakdown/Challenge/Codify stage here — there's no Revolution/Jira, no epics or
story points, just `BACKLOG.md` at the repo root. A backlog item here is small enough to hand
straight to the `implementer` agent once it's well-shaped.

**The interaction is yours; the drafting is the agent's.** A subagent can't hold a back-and-forth
with the user — so *you* do the interactive elicitation and the approval loop in this conversation,
and delegate the research + drafting to the `ideator` agent. Don't draft the bullet yourself.

## 1. Establish the mode

Is the user creating a **new** backlog item, or **refining an existing** one? Usually obvious from
the request ("add a bug for X" vs "the DB-connection-reuse item is too vague, flesh it out").

If refining, find the exact entry together (ask the user, or grep `BACKLOG.md` yourself) and note
its **exact current bullet text** — the agent needs this verbatim to refine in place rather than
re-drafting from scratch.

## 2. Elicit the intent (interactive, conversational — not a rigid form)

Draw out, as far as the user actually knows it:
- **Purpose** — what's the problem or outcome, and why does it matter now?
- **Context** — where does it fit (which part of the code, related existing bugs/ideas already in
  BACKLOG.md)?
- **Acceptance criteria / done-when** — how would they know it's fixed/built?
- **Constraints / non-goals** — anything explicitly out of scope, deadlines, things already ruled out.

Don't force every dimension — if the user already gave a crisp one-liner, take it and move to step
3. The agent will research the rest and surface gaps as open questions for you to relay back.

Don't fabricate the "why now" for the user — if it's not obviously stated, ask, or let the agent
flag it as an open question. Don't read `BACKLOG.md` or the code yourself here — that's the agent's
job in step 3.

## 3. Delegate drafting to one `ideator` agent

Spawn exactly **one** agent (Agent tool, `subagent_type: ideator`). Give it:
- the **mode** (`draft` or `refine`);
- for `refine`, the **exact existing bullet text** plus what the user wants improved;
- the elicited intent, verbatim where the user was specific;
- a reminder to read the current `BACKLOG.md` and the relevant code before drafting.

If the user has follow-up answers or change requests after seeing the draft (step 4), send them
back to the **same** agent via `SendMessage` to revise — don't spawn a fresh one, it loses context.

## 4. Present the draft, resolve open questions

Relay the agent's output **verbatim** — the full bullet text and which `BACKLOG.md` section it
belongs under (or, for `refine`, the **Changes** delta plus the revised bullet). This is the
approval gate: nothing is written to `BACKLOG.md` until the user says yes.

If the agent raised open questions that change *what* gets built, put them to the user now, then
feed the answers back to the same agent (`SendMessage`) and re-present. Loop until approved.

## 5. Persist — only after approval

Once approved, **you** (not the agent — it's read-only) edit `BACKLOG.md`:
- **New entry** → append the bullet under the right `##` section, matching existing formatting.
- **Refine** → replace the existing bullet's text in place with the revised version.

Don't touch any other part of the file.

## 6. Report and hand off

Summarize briefly: what was written, under which section, and recommend the next step —
run the `implementer` agent against the new/refined bullet when the user is ready to build it.

## Notes

- **Deliberately not the company-scale version.** No epics/stories, no estimates or complexity
  labels, no handoff ledger, no cost tracking — this is one well-written `BACKLOG.md` bullet,
  nothing more. If an idea is clearly too big for one bullet, say so and suggest splitting it into
  two or more rather than writing a sprawling entry.
- **Human-gated write.** Same discipline as everywhere else in this repo: the file write happens
  only after the user's explicit approval of the draft.
- **One agent, revised not replaced.** Exactly one `ideator` run per `/ideate` invocation; loop on
  it via `SendMessage` rather than spawning duplicates.
