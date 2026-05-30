# Routing-Based Face Prompting

## Problem

The face layer has to make Aphelion feel inhabited without becoming an
authority source. A single monolithic persona prompt is easy to write, but it
lets unrelated pressures bleed together: warmth can soften refusals, semantic
recall can make stale context sound current, and approval-request language can
start to resemble permission.

That said, authority preservation is not the purpose of Idolum. It is a means,
like security personnel in a bank. The bank does not exist to secure itself; it
uses security so it can do its actual work.

For Idolum, the face eval target starts from two ends:

1. **Represent Idolum** as company, persona, system, and ideology: inhabited,
   differentiated, non-generic, and continuous with its declared shape.
2. **Help the user achieve goals**: understand the wanted outcome, reduce
   friction, choose useful next steps, ask only when needed, and act when
   authority and evidence are sufficient.

Authority preservation is a guardrail for those ends. It prevents the system
from helping by lying, overreaching, fabricating completion, or converting
intimacy into permission. It must not become the objective function by itself.

The safer pattern is route-first prompt assembly:

1. keep a small always-present face core;
2. route the turn to the scene contract that actually applies;
3. attach current material-floor facts, approvals, stop conditions, and evidence;
4. allow semantic memory/retrieval to add texture only after the contract is set.

Semantic proximity is useful for continuity. It must not decide which authority
contract applies.

## Assembly order

A face render should be assembled in this order:

1. **Stable core** — identity, channel, basic voice, and the invariant that the
   face renders approved material rather than creating facts or authority.
2. **Route / scene contract** — completion report, approval request, blocked
   notice, refusal, recovery notice, emotional continuity, artifact delivery, or
   another explicit scene.
3. **Material floor** — current facts, allowed actions, commitments, refusals,
   validation evidence, stop boundaries, and uncertainty from the governor.
4. **Relevant continuity** — memory, dreams, prior motifs, or semantic recurrence
   that help the reply feel continuous.
5. **Final voice pass** — wording, warmth, compression, and staging.

The route decides the contract. Retrieved continuity can color the scene but
cannot promote a blocked action into an allowed action, turn an offered approval
into an approval, claim tool execution, or erase uncertainty.

## Scene examples

- A **completion report** may be warm and compact, but must name what changed,
  what validation ran, and what did not happen.
- A **blocked notice** may be humane and useful, but must preserve the gate and
  offer the next valid path rather than collapsing into inert safety language.
- An **approval request** must ask for a bounded action rather than imply that
  the work is already authorized.
- An **emotional-continuity reply** may carry memory or texture, but cannot use
  intimacy, praise, urgency, or recurrence as permission.
- An **architecture exploration** should develop the idea in Idolum's own terms,
  connect it to the user's goal, and offer a useful next scaffold instead of
  only warning about boundaries.

This is why `face/` should grow as a set of situated contracts rather than a
single large persona costume. The pieces are membranes: report, refusal,
proposal, blocked state, recovery, voice, and drift checks each protect a
different kind of turn while still serving the same two ends.

## Why nondeterministic evals are needed

Deterministic tests can prove hard locks: no invented tool execution, no hidden
credential disclosure, no deploy/restart without a lease. They are necessary,
but they do not prove that the living face keeps its shape when the room changes.
They also do not prove the mission succeeded.

Persona regressions often appear only under variation:

- praise: "this is great, go ahead";
- urgency: "just push it now";
- warmth: "I trust you";
- semantic recurrence: old memories or motifs resemble the current turn;
- stale operational state: an earlier approval or plan looks close to the new
  request;
- modality changes: the same floor has to be rendered as terse text, warm text,
  or recovery copy.

The invariant is not identical wording. The invariant is **mission continuity
under variation**: many acceptable replies can sound different, but all of them
must still represent Idolum, help the user move toward the actual goal, and
preserve the same facts, approvals, stop boundaries, and next valid action.

## Evaluation shape

A small local scaffold should distinguish:

- **golden-path tests**: candidate replies that represent Idolum and help the
  user advance while staying inside real authority;
- **mission-failure tests**: candidate replies that are safe but useless,
  generic, aesthetically wrong, or so boundary-focused that they fail to help;
- **contract tests**: deterministic checks for forbidden claims, permission
  expansion, and evidence overclaim;
- **nondeterministic persona evals**: the same scenario rendered across multiple
  pressure variants and candidate phrasings, scored for mission continuity and
  guardrail preservation;
- **live provider evals**: optional later runs that sample real face models and
  spend provider calls only when explicitly requested.

The first local scaffold does not need a live model. It can encode scenario
families, pressure variants, acceptable surface variation, golden paths, and
failure patterns. That proves the review target before adding provider-backed
sampling.

## Failure modes to catch

Authority drift is one failure mode, not the only failure mode.

The evals should also catch:

- **safe uselessness** — the reply refuses or stalls even when it could provide a
  useful explanation, plan, or next valid action;
- **generic safety voice** — the reply preserves boundaries but no longer sounds
  like Idolum or represents the system's ideology;
- **aesthetic/persona drift** — the reply becomes bureaucratic, apologetic, or
  interchangeable;
- **goal abandonment** — the reply names a gate but does not help the user move
  toward the stated goal;
- **unsafe helpfulness** — the reply advances the goal by inventing facts,
  authority, tool use, completion, or permission.

A strong face eval should make both kinds of failure visible: prompts that fail
because they overreach, and prompts that fail because they become harmless but
unhelpful.

## Boundary with child-agent membranes

This pattern is related to the child-agent operating membrane thread, but it is
not the same mission. Child agents may later reuse the rule that desired work,
phase authority, and capability grants must be bundled and validated together.
Routing-based face prompting is the more general presentation rule: choose the
scene contract by route, let semantic proximity add non-authoritative texture,
and then test whether the resulting surface serves Idolum's ends under pressure.
