---
date: 2026-06-08
status: proposed
---

# Stateless Operations Layer Shared by the CLI and MCP Adapters

## Context and Problem Statement

The MCP server runs each tool call as a CLI-shaped command through the
bridge (`tango/mcp_dodder`), reusing the package-level command singletons
registered with `command.Utility` (`utility.AddCmd("new", &New{})`). Those
singletons are mutable: flag parsing writes into the one struct instance.
In the one-shot CLI this is harmless (a fresh process per invocation), but
the long-lived MCP process reuses the same instance across tool calls, so
state from one call leaks into the next.

dodder#247 fixed the first symptom — two `new` calls concatenating their
descriptions, plus a concurrency race — with an opt-in `ResetCLIState`
interface and a bridge-wide mutex. dodder#252 proposed going further by
registering *command factories* so the bridge builds a fresh command per
call. Both are workarounds for a deeper mismatch: the MCP bridge is
driving the CLI **composition layer** (flag parsing → stateful command →
render) when it only needs the **operation** underneath it. Factories in
particular reinvent a request/response server on top of a framework that
was never meant to be one, badly.

How should the command framework be structured so the MCP bridge (and any
future adapter) can invoke dodder operations directly, without inheriting
the CLI's per-invocation statefulness?

## Decision Drivers

- The singleton-reuse hazard is open-ended: any future command with
  accumulating (`flag.Value`-bound) state that joins the bridge and forgets
  `ResetCLIState` silently regresses, and nothing enforces it (dodder#247,
  dodder#252).
- A working precedent already exists: the `edit` MCP tool bypasses the
  bridge entirely and calls `repo_actions.MakeUpdateObject(repo).Run(
  objectId, ObjectChanges{...})` — JSON → typed request → operation →
  render, with no command, no singleton, no reset, no mutex.
- The operation layer largely exists: `sierra/repo_actions` types
  (`UpdateObject`, `WriteNewZettels`, `Checkin`, `Checkout`, …) already
  embed the `local_working_copy.Repo` coordinator and have `Make*(repo)`
  constructors; the cleaner ones already take a pure-data request
  (`UpdateObject.Run(objectId, ObjectChanges)`).
- A prior attempt at an operations layer (`user_ops`) was a god-layer that
  tangled env construction, transport concerns, and rendering; work has
  been migrating *out* of it into `command_components_dodder`. Any new
  structure must avoid repeating that.

## Considered Options

1. **Command factories** (dodder#252 as filed): register
   `func() command.Cmd` and have the bridge construct a fresh command per
   call. Eliminates shared state but keeps the bridge coupled to the CLI
   command framework and grows a bespoke server inside it.
2. **Keep dodder#247's opt-in reset + mutex** and require every future
   bridge-exposed command to implement `ResetCLIState`. No structural
   change; relies on discipline a tool cannot enforce.
3. **Formalize `repo_actions` as a stateless operations layer** that both
   the CLI commands and the MCP bridge call as thin adapters. The bridge
   stops shelling CLI commands for mutating tools and calls operations
   directly, as `edit` already does.

## Decision Outcome

Chosen option: **3 — formalize `repo_actions` as a stateless operations
layer shared by the CLI and MCP adapters.**

The layering becomes:

```
primitives    oscar/store, romeo/local_working_copy, juliett/queries
operations    sierra/repo_actions: (repo, Request) -> (Result, error)   [stateless, shared]
  CLI adapter   uniform/commands_dodder + tango/command_components_dodder  (flags -> Request -> render)
  MCP adapter   tango/mcp_dodder bridge + handlers                         (JSON  -> Request -> render)
```

An **operation** is a pure function of `(repo, Request)` returning a typed
`Result`. The three rules that keep it from regressing into `user_ops`:

1. An operation does **not construct the env** — the `Repo`/`Env` is built
   by the adapter and passed in.
2. An operation does **not parse transport input** — no `flag.FlagSet`, no
   JSON tags on its request types.
3. An operation does **not render** — it returns a typed `Result`; the
   adapter formats it (CLI printers, or MCP content blocks).

`command_components_dodder` stays exactly where it is: it is the **CLI
adapter's toolkit** (flag definitions plus env builders — `blob_store.go`,
`query.go`, `genesis.go`, `local_working_copy.go`). It is legitimately
stateful because a CLI command is a per-process throwaway; it simply stops
being where operation *logic* lives.

The pure-data `Request`/`Result` types live **in `repo_actions` (sierra)**.
The NATO DAG permits it: `sierra (19)` sits below both `tango (20)` and
`uniform (21)`, so both adapters import the operation layer with no cycle.
A separate lower-tier types package is deferred until something below
sierra needs to reference a request shape (none does today).

Scope is bounded to the **mutating** path (`new`, `edit`, `checkin`),
where the state hazard lives and the typed request matters most. The
**read** tools (`show`, `query`, `status`, `diff`) stay on the bridge for
now: they are idempotent (no state leak) and route through it to reuse the
CLI box/organize formatters. A later pass can give reads their own
read-operations with injected formatters if warranted.

### Consequences

- Good: dodder#247's `CommandWithResetCLIState` and the bridge state-mutex
  are deleted once no mutating tool routes through the bridge-as-CLI —
  nothing mutable is shared because no singleton is reused.
- Good: dodder#252 is reframed from "add command factories" to "migrate
  mutating MCP tools off the CLI bridge onto `repo_actions`", finishing
  the pattern `edit` already established.
- Good: a third adapter (e.g. a future HTTP/websocket API, cf. dodder#253)
  targets the same operations layer instead of re-driving the CLI.
- Bad: normalizing every `repo_actions` Run signature to
  `Run(Request) (Result, error)` and lifting the request types out of the
  command structs is broad, touching the command packages.
- Bad: request-assembly currently entangled in the CLI command structs
  (Proto building, checkout options, query-group specs) must be extracted
  into the shared request types; some of it is genuinely CLI-shaped
  (sigils, query strings) and needs care to keep adapter-specific.

### Confirmation

A strangler migration, not a big-bang:

1. Normalize `repo_actions` to `Run(Request) (Result, error)` and lift the
   request types into the package.
2. Migrate `new` off the bridge to a direct `repo_actions` call (proving
   the `sku.Proto` handoff), then `checkin`. `edit` is already the
   template.
3. CLI commands adopt the same request types: `New.Run` becomes
   flags → `WriteNewZettelsRequest` → operation → render.
4. Delete `ResetCLIState` and the bridge state-mutex.

Each step keeps `just test` (unit + bats) green; the mutating-tool bats
coverage in `mcp.bats` is the integration gate, and dodder#247's
description-leak test is the regression guard until the singleton path is
gone.

## More Information

- dodder#247 — flag-state leak/race across bridge tool calls (the symptom;
  introduces `ResetCLIState` + the bridge mutex this ADR removes).
- dodder#252 — originally "factory-based command registration"; this ADR
  supersedes its approach and reframes the work.
- dodder#253 — websocket transfer protocol; a future API adapter would
  target the operations layer described here.
- The `edit` MCP tool handler (`tango/mcp_dodder` `makeEditHandler`) — the
  in-tree precedent for an adapter calling `repo_actions` directly.
- ADR-0002 (workspace filename resolution to the command layer) — prior
  decision separating CLI-adapter concerns from the query primitives.
