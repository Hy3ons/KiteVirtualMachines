# Kite Frontend

React + TypeScript + Next.js App Router frontend. Interactive screens use Ant
Design and the existing Go API endpoints. See
[`docs/architecture.md`](docs/architecture.md) for rendering, controller, policy,
transport and Zod contract boundaries.
Use Node.js 22.12 or newer and the checked-in npm lockfile.

The visual system is documented in [`DESIGN.md`](DESIGN.md). Public pages use a
white/ink identity and pale sky product framing; workspace screens use a shared
Cloud-style shell, current resource totals, filters and dense tables. Continuous
squircle corners use `corner-shape` with a `border-radius` fallback. Brand assets
are original SVGs with matching ICO/PNG favicons and an Apple touch icon.

`src/styles/` separates tokens, common components, shell, marketing, dashboard,
admin and machine surfaces. `src/screens/dashboard/` and `src/screens/admin/`
hold existing request/state logic, keeping presentation components small. Go API
routes and SSR session authorization remain the same. Offer forms retain input
when creation or claim fails.

## Request flow

```text
Browser -> Ingress /          -> kite-frontend:80 -> Next.js:3000
Browser -> Ingress /api/*     -> kite-api:8080
Next.js SSR -> internal HTTP  -> kite-api:8080/api/v1
```

VM, user, offer, settings and console operations stay in Go. Next.js checks
sessions and page permissions, and reads initial data for SSR. Production has
no Next.js business API or API proxy. Local `next dev` alone rewrites `/api/v1`
to Go to preserve same-origin cookies while developing.

`src/app/` owns routes, metadata and server entry points. `src/screens/` preserves
existing interactive screens; it replaces `src/pages/` so Next.js does not treat
those components as legacy Pages Router routes. `src/server/` is server-only.

## JWT and SSR

Go issues and clears the `HttpOnly`, `SameSite=Lax` `accessToken` cookie, with
`Secure` according to the existing HTTPS policy. Next.js forwards that cookie
to `/me`; Go verifies the JWT and reads the user's current access level.
Next.js does not copy the token to JavaScript, decode claims as authorization,
or require the Go JWT signing secret.

Every protected page checks the session before loading private data, including
client navigation requests. SSR user data uses `cache: no-store`; request-scoped
React memoization only avoids duplicate reads in one render. The Zustand store
is created per page/provider, never shared across server users. Authentication
and permissions are no longer restored from `localStorage`.

Expired/revoked sessions return to login. API outages render a retryable error
instead of silently logging out. Login failures stay on the form; logout only
clears local display state after Go confirms cookie removal. The existing API
has no refresh-token endpoint, so expired JWTs require login again.

Dashboard, VM detail, console metadata, admin users/VMs and settings load initial
data on the server. Buttons and refresh operations continue to call Go from the
browser. xterm runs only in the browser; its ticket and WebSocket endpoints stay
unchanged.

## Development and environment

For the macOS host-Go + Docker Minikube workflow, start with
[`build/dev/README.md`](../build/dev/README.md#host-go-api--docker-minikube--nextjs).
From the repository root, `bash build/dev/local.sh setup` prepares the cluster
and `bash build/dev/local.sh api` starts Go directly on port 8080.

```sh
npm ci
# Real Go API, listening locally on port 8080:
npm run stage
# If Go runs elsewhere, set its origin (without /api/v1):
KITE_API_INTERNAL_URL=http://127.0.0.1:8080 npm run stage
# UI-only debug mode, with browser-local mock data and synthetic admin session:
npm run dev
```

Both development commands serve `http://127.0.0.1:5173`. Use `stage` for real
login/SSR/API QA; stop `dev` before starting `stage` (one mode at a time).
`dev` intentionally defaults to mock data and is not evidence
that the real backend works. Next.js can load local `.env*` files; these files
are ignored by Git and excluded from Docker build context.

| Variable | Meaning | Default |
| --- | --- | --- |
| `KITE_API_INTERNAL_URL` | Server-only Go origin for SSR and local development proxy | `http://127.0.0.1:8080`; container: `http://kite-api:8080` |
| `NEXT_PUBLIC_API_BASE_URL` | Browser API base, embedded at build time | `/api/v1` |
| `NEXT_PUBLIC_KITE_MODE` | `production`, `staging`, or explicit `debug` | Build scripts set it |
| `NEXT_PUBLIC_USE_MOCK` | Enable mock API only in debug mode | Release builds: `false` |
| `NEXT_PUBLIC_DEBUG_DIRECT_ROUTES` | Synthetic debug admin session; also requires debug/mock | `dev`: `true` |
| `KITE_PUBLIC_URL` | Optional fixed public HTTP(S) origin for canonical/social URLs | Request origin (trusted Ingress forwarding headers, otherwise Host) |

Keep `/api/v1` on the same public origin in production so Go's host-only cookie
also reaches Next.js SSR requests. Server-only variables must never use the
`NEXT_PUBLIC_` prefix. Public variables require a rebuild to change.

Social previews use generated 1200×630 JPEG cards at
`/images/social/kite-social.jpg` and `/images/social/kite-guide-social.jpg`.
Image URLs, canonical URLs and crawler discovery use the current public origin.
For a fixed deployment, set `KITE_PUBLIC_URL=https://your-public-host` (origin
only, no path/query/credentials), consistently in build and runtime environments.
Public pages render a request-scoped verified account header, so they remain
dynamic even with a fixed metadata origin and must not be shared-cached with
personalized account data. Anonymous requests skip Go's /me lookup. Without a fixed origin,
metadata resolves per request so a single self-hosted image works across domains.
Ingress must overwrite `X-Forwarded-Host` and `X-Forwarded-Proto` with trusted
values. Forwarded headers are used only to format public metadata, never as an
authentication source or a server-side fetch destination.

Login/signup, console screens and policy pages are `noindex, nofollow`;
private and missing pages do not inherit the home canonical URL. `sitemap.xml`
lists the homepage and guide, while `robots.txt` excludes API/console crawling.
Policy wording and its repository data-flow audit are maintained in
`docs/privacy-data-audit.md`. Actual operator contacts, retention and Google OAuth
settings still require installation-specific verification; removing draft wording
does not supply those facts. Public accessibility does not depend on search indexing.

With the dev server running, `npm run audit:metadata` checks actual HTTP metadata,
social image responses, crawler head rendering, canonical URLs and indexing.
An authenticated `/login` correctly passes by redirecting to the landing page.
After `npm run build:prod`, run `npm run audit:session` to start an isolated
production standalone server with synthetic Go-shaped HTTP fixtures. It checks
anonymous/authenticated pages, login redirects, role guards, concurrent identity
isolation, private cache headers and the absence of hidden-panel VM reads. It uses
ephemeral local ports, stops its own servers, performs GET requests only and does
not connect to a live Go backend, Kubernetes cluster or browser.
Pass a different request origin and optional expected public origin as arguments:
`npm run audit:metadata -- http://127.0.0.1:5174 https://your-public-host`.
Private routes are checked in mock mode or validated as login redirects when
anonymous; this script does not sign in, mutate data or run browser automation.

## Build and deployment

```sh
npm run build:prod
npm start

docker build \
  --build-arg NEXT_PUBLIC_KITE_MODE=production \
  --build-arg NEXT_PUBLIC_API_BASE_URL=/api/v1 \
  --build-arg NEXT_PUBLIC_USE_MOCK=false \
  -t ghcr.io/hy3ons/kite-frontend:latest .
```

Docker runs Next.js standalone as a non-root Node.js process on port 3000.
The Kubernetes Service stays on port 80 and targets container port 3000.
Ingress `/api` continues to route directly to Go. When testing `npm start` or
the container without Ingress, provide an external reverse proxy for `/api`, or
use `npm run stage` for the built-in development proxy. A frontend-only port
forward can show pages but does not provide the production API routing.

Dev deployment scripts now use `FRONTEND_NEXT_BUILD_MODE`,
`FRONTEND_NEXT_API_BASE_URL` and `FRONTEND_NEXT_USE_MOCK`. Legacy Vite variable
names are retired. Deployment/publishing remains a separate operation.

## Automated checks and manual QA

```sh
npm run typecheck
npm test
npm run lint
```

TypeScript also checks exact optional properties and unchecked indexed access.
Response types come from shared Zod schemas used by browser and SSR adapters;
every consumed API success response is parsed, including mutation results.

Tests use a local HTTP fixture and the actual fetch transport. They verify
payloads, cookies, concurrent-user isolation, timeouts, authentication errors,
SSR response contracts and UI API endpoint mapping. They do not execute the
Go service or prove cluster integration. Playwright and browser E2E are not run;
the previous Playwright files remain as historical references and are excluded
from the test command.

For manual QA with the real Go API, check:

1. Sign up, incorrect login, successful login, reload, logout and browser Back.
2. Open a protected URL without a cookie, with an expired cookie and after a
   permission downgrade; localStorage edits must not grant access.
3. Verify levels 0/1/2/3: personal VMs, create limits, manager users, admin VMs
   and system settings. Confirm account data does not cross browser sessions.
4. Create/start/stop/delete a VM, claim an offer, open VM details and refresh.
5. Open the serial console, reconnect, type, resize and close it; verify SSH help.
6. Save domain, HTTPS, SSH gateway, contact and certificate settings. Only rotate
   runtime secrets in an environment intended for that manual test.
7. Check direct URLs, back/forward navigation, desktop/mobile layout and initial
   paint. Restart/unavailable API should show an error with a retry action.
