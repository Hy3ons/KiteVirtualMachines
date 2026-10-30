# Local runtime review — 2026-10-30

## Environment

Ran the actual Next.js development server on `127.0.0.1:5173` with both
`NEXT_PUBLIC_USE_MOCK=false` and `NEXT_PUBLIC_DEBUG_DIRECT_ROUTES=false`.
Go 1.26.4 runs `kite/cmd/kite-api` directly on port 8080. It uses the macOS
`~/.kube/config` context `kite-local-qa`, an isolated Docker/containerd Minikube
profile on OrbStack (4 CPUs, 4 GiB, Kubernetes 1.37.0).

The pre-existing `kite-dev` profile and its administrator were preserved.
Installed the three Kite CRDs only; Go bootstrapped its own runtime config and
secret in namespace `kite`. The current kubeconfig context remains
`kite-local-qa` for the running API. No additional OrbStack VM was created.

## Observed results

| Scenario | Result |
| --- | --- |
| First signup against the initially empty QA cluster | HTTP 201; `qa-admin` persisted with access level 3. |
| Second signup | HTTP 201; `qa-member` persisted with access level 0. |
| Login and `/me` | Name, email and persisted access level agree; cookie is HttpOnly, SameSite=Lax, without Secure on local HTTP. |
| Anonymous session / incorrect password | HTTP 401. |
| Level 0 admin settings and direct VM creation | HTTP 403. |
| Admin/user VM lists, offers, settings and configuration | Real JSON passes the application's shared Zod endpoint adapters. |
| Next proxy login and logout | Session cookie forwarded; logout clears it with Max-Age=0. |
| Anonymous `/dashboard` | HTTP 307 to login. |
| Authenticated `/login` | HTTP 307 to `/`. |
| Authenticated dashboard, admin dashboard and settings | HTTP 200, actual account name in SSR HTML, no streamed render failure. |
| Level 0 `/admin/settings` | HTTP 307 to dashboard. |
| Level 1 -> 0 change | HTTP 200 and CRD read-back; existing session sees the new role without re-login. |
| VM create/read/start/stop/delete | Accepted and reflected in VM CRD desired state; temporary CRD removed after checks. |
| Offer create/list/claim/delete | Level 0 can claim its assigned capacity; resulting VM has assigned specs; successful claim consumes the offer. Unused offer deletion also succeeds. |
| Console ticket for an unprovisioned VM | HTTP 409; unavailable console is rejected. |

## Bug fixed

`access_level: 0` was rejected with HTTP 400 because Gin's `required` validation
treated an integer zero as absent. Reproduced this using the live Go API and the
existing manager access-level handler test before changing the implementation.
The request now uses a required `*int`, distinguishing explicit zero from omitted
or null input. The owner test verifies missing/null rejection, successful downgrade,
persisted Level 0, and existing manager restrictions. The same live request now
succeeds.

## Verification

- `npm run test:local`: 5 integration scenarios passed (44.54 seconds).
- `npm test`: 217 tests across 12 files passed.
- `npm run typecheck` and `npm run lint`: passed.
- `go test -race -shuffle=on -count=1 ./cmd/kite-api/apis`: passed.

The integration suite is opt-in: `vitest.local.config.ts` includes only
`tests/local-runtime.integration.ts`, while default `npm test` excludes it.
Generated account credentials are kept in ignored `.local-dev/kite-local-qa/`
with file mode 0600, separate from existing local development credentials.
See `build/dev/README.md` for setup and rerun commands.

## Limits and manual QA

No browser automation or Playwright was run. Click behavior, layout and interactive
terminal behavior remain manual QA. The Next development log emitted an Ant Design
Portal SSR warning while rendering settings; the response rendered successfully,
but this warning is not evidence that client interactions have been verified.

This lightweight cluster has no controller, KubeVirt, CDI, Longhorn or VM image.
VM tests prove API contracts and CRD desired state, not guest boot, actual power
changes, serial console sessions or SSH forwarding. Those require the full runtime
and were not reported as passing. QA namespaces stand in for the namespace creation
normally performed by the controller. Test VM/offer CRDs were removed; the two
QA accounts and their namespaces remain available for manual login testing.
