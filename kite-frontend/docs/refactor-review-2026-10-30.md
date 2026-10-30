# Frontend refactor review — 2026-10-30

## Scope and result

Reviewed the complete frontend request flow against the Go response structs and
handlers. Changes cover the Next.js frontend, its contracts, controllers, policy,
authentication display state and tests. Go implementation and Kubernetes resources
were inspected as contract references and were not changed by this refactor.

The architecture and dependency rules are documented in
[`architecture.md`](architecture.md). The work preserves existing screens and
direct Go operations while enforcing runtime parsing and compile-time boundaries.

## Findings addressed

| Finding | Result |
| --- | --- |
| Successful HTTP JSON was trusted through `json<T>()`. | Transport returns `unknown`; all 30 consumed endpoints parse shared Zod contracts. |
| Browser and SSR models/schemas drifted. | Both use `createEndpoints`; response types derive from schemas. |
| Several mutation response annotations contradicted Go. | Power `{vm}`, access `{user}`, VM deletion `{message,vm}` and settings `{message,config}` now match Go. |
| Public/admin configuration was an all-optional combined model. | Distinct required wire schemas normalize gateway status for shared policy. |
| Console ticket code was duplicated and one path was unencoded. | One cancellable endpoint operation encodes VM segments; WebSocket query encoding remains separate. |
| Stored debug fixtures only checked their containers. | Nested users, VMs, offers, settings and counters are parsed before use; corrupt fixtures reset. |
| Missing dashboard access defaulted to Level 1. | Shared capability policy fails closed. |
| Views repeated creation, lifecycle and SSH rules. | Pure domain functions supply shared decisions and typed outcomes. |
| VM views owned fetching and mutations; settings had one broad hook. | Detail/console controllers extracted; settings split into routing/contact, gateway and security hooks. |
| Nullable authentication fields allowed inconsistent profiles. | Authenticated/anonymous union changes identity and permissions together in request-local stores. |
| Optional/index access checks were incomplete. | Exact optional, checked index, named environment, complete return and switch checks enabled; lint rejects unsafe assertions. |
| Demo tests fabricated an HTMLElement with a double cast. | DOM discovery and explicit-target GSAP animation are separate, preserving real timeline tests without a browser. |

VM creation now closes its modal only after a validated success, retaining input
on failure. Existing Level 1 resource limits remain CPU 2, memory 4Gi, disk 25Gi
and two inventory entries; deleting entries still count until removed. Account
usage keeps excluding deleting VMs. Detail refresh still tolerates an optional
configuration failure; console SSR no longer requests configuration it does not
use.

## Verification

| Command | Result / coverage |
| --- | --- |
| `npm run typecheck` | Passed with all added strict flags. |
| `npm run lint` | Passed, including the assertion/type-import rules. |
| `npm test` | 210 tests across 11 files passed. |
| `npm run build:prod` | Passed; all application pages build for dynamic SSR. |
| `npm run audit:session` | Production HTTP checks: anonymous/public routes, verified/expired login, concurrent identity isolation, private caching, Levels 0–3, dashboard/detail/console/admin rendering, required admin settings, console avoiding config, malformed success and API outage failures. GET requests only. |
| `npm run audit:metadata` | Public/private titles, canonical/indexing behavior, OG/Twitter images, crawler responses and authenticated login redirect passed. |
| `git diff --check` | Passed. |

Tests exercise the actual HTTP transport against temporary local fixtures. They
cover every endpoint's valid and malformed envelope, nested field types, integer
CPU/expiry values, role values, private-field stripping, required configuration,
omitted observation fields, errors, timeout, cancellation and encoded paths.
Other suites check policy boundaries, deep debug persistence, source import
direction, authentication isolation, account popup behavior and GSAP repetition.

The local HTTP listener required sandbox escalation after an initial `EPERM`;
the successful suite ran all tests without skips. Evidence logs and the QA matrix
are stored under `.omo/evidence/frontend-refactor/` in the repository worktree.

## Manual QA boundary

No Playwright, browser E2E or visual automation was run. No live Go service,
Kubernetes cluster, VM, account or deployed configuration was mutated. The HTTP
fixtures verify frontend protocol handling and SSR, rather than proving a real
cluster deployment. Real login, VM creation/power/deletion, offer claims, SSH,
clipboard, terminal interaction and responsive layout remain user QA items.
