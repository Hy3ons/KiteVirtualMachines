# Frontend architecture

Kite uses Next.js App Router with separate rendering, interaction, policy and
transport responsibilities. The MVC boundary is implemented through React
components and feature hooks rather than a second routing or controller framework.
Go remains responsible for authorization, VM operations and Kubernetes state.

## Responsibilities and dependency direction

| Layer | Location | Responsibility |
| --- | --- | --- |
| Server entry | `src/app/`, `src/server/` | Authorize each request, compose initial data, provide metadata and request-local cookies. |
| View | `src/screens/*.tsx`, `src/components/` | Render data and bind events. Shared presentational components receive data instead of fetching server sessions. |
| Interaction controller | `src/screens/dashboard/`, `src/screens/admin/use*`, `src/screens/machine/use*`, `src/hooks/` | Coordinate API operations, forms, pending state, feedback and navigation. |
| Domain policy | `src/domain/` | Decide access, creation limits, VM action availability and SSH command formatting using explicit inputs. No React, store or HTTP runtime dependencies. |
| Endpoint adapter | `src/api/endpoints.ts`, `src/api/index.ts` | Select the Go route, method, payload and response schema. The same endpoint implementation serves browser and SSR callers. |
| Contract/model | `src/api/contracts.ts`, `src/api/configContracts.ts`, `src/api/types.ts` | Parse external JSON and derive response types from Zod schemas. Outbound PATCH fields remain optional when omission has API meaning. |
| Transport | `src/api/client.ts`, `src/api/http.ts` | Apply timeout, cookie, error, cache and retry policy. Successful untrusted JSON is `unknown` until an endpoint parses it. |

```text
Next server page ── authorization ── request-local endpoint adapter
       │                                      │
       └── validated initial data             │
                      │                       │
React view ── feature controller ── endpoint adapter ── Go API
                      │                       │
                 pure domain             Zod contract
```

`src/api/pageData.ts` composes validated reads for SSR. It does not own schemas,
and browser adapters do not import it. `src/api/types.ts` remains the compatibility
surface for model imports. Type-only imports allow policy functions to consume
these models without loading transport or schema implementations.

## External data and explicit absence

Every consumed Go JSON response, including mutation acknowledgements, is parsed
before controllers receive it. Unknown response keys are stripped, so password or
token fields cannot accidentally enter a session profile. Missing required data
and wrong field types fail at the endpoint boundary rather than becoming empty
lists, successful notifications or asserted TypeScript models.

Public and administrator configuration have distinct wire contracts. Public
gateway status arrives as flat fields; administrator status is nested. The public
adapter normalizes both into a required `sshGateway.status` structure for shared
connection policy. Only the gateway observation fields that Go omits remain
optional. Administrator secret-presence flags are required booleans.

`null` means an unavailable session, unavailable connection settings, or an initial
screen that must load its debug data on the client. It is not a substitute for an
invalid response. Optional React props and omitted Go request fields keep their
normal semantics. `exactOptionalPropertyTypes` prevents explicitly assigning
`undefined` to them; `noUncheckedIndexedAccess` requires checking indexed values.

The authentication store uses an authenticated/anonymous union: identity and
access level change together. Each provider owns a separate store. Debug fixture
storage validates nested records with the same model schemas and resets corrupt
fixture data. It never supplies production authorization.

## Policies and lifecycle

- Missing permissions deny VM mutation. Levels 2 and 3 share user-management
  navigation; only level 3 receives system administration and cluster VM reads.
- Level 1 creation applies the existing fixed CPU, memory and disk limits and
  inventory quota. Deleting VMs still consume that quota until removed. Account
  usage deliberately excludes deleting VMs.
- Shared VM lifecycle policy governs row actions, details and console access.
  Deletion blocks mutation and connection; a running VM can connect. SSH guidance
  stays available before startup, and copying a ready command does not depend on
  the VM power state because it only prepares instructions.
- Shared TCP-port validation and SSH formatting serve settings and connection help.
  A ready command and an unavailable command are different typed outcomes.
- Detail refresh may retain VM data if the optional configuration read fails.
  Console SSR only loads VM metadata. Ticket creation keeps cancellation and path
  encoding separate from the raw terminal WebSocket stream.

## Request and session guarantees

The browser calls Go directly through `/api/v1`; Next.js has no business mutation
API. Local development uses the existing same-origin rewrite. Go owns the
HttpOnly JWT cookie. SSR forwards only that cookie using a new request-local
client, with `no-store` caching and no retry. React memoization deduplicates only
within a request. Authentication failures and service outages remain distinct.

The landing animation uses isolated example state and explicit GSAP targets. Its
DOM adapter discovers targets; its animation engine controls timing. It never
submits real forms or invokes VM/authentication APIs.

## Enforced checks

`npm run typecheck` includes strict null checks, exact optional fields, checked
indexed access, declared environment keys, complete returns and switch fallthrough
checks. ESLint rejects explicit `any`, type assertions and non-null assertions.
Architecture tests check the source import graph so contracts and policy cannot
depend on UI/server infrastructure and browser API adapters cannot import SSR
composition.

`npm test` covers real local HTTP transport, response rejection, payloads,
cancellation, fixtures, policy decisions, isolated authentication and the GSAP
clock. `npm run build:prod` plus `npm run audit:session` exercises production SSR
against isolated Go-shaped fixtures, including concurrent identities and role
guards. These checks do not execute Go/Kubernetes or replace user QA against a
running backend. Browser E2E and visual QA remain manual.
