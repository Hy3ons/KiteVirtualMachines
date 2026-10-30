# Privacy and service-terms implementation audit

Reviewed on 2026-10-30. This is a repository audit, not a claim that a deployed
instance, its node configuration or Google Cloud console has been inspected.
Public copy is in `src/app/privacy/page.tsx` and `src/app/terms/page.tsx`.
Evidence paths below are relative to `kite-frontend/`.

## Confirmed data flows

| Flow | Evidence | Policy consequence |
| --- | --- | --- |
| Email/password signup and login | `../kite/cmd/kite-api/apis/auth.go`, `../kite/internal/account/service.go` | Username, email, internal account/workspace IDs, permissions and password hash; first account is admin, later accounts start read-only. No email delivery integration. |
| Account storage | `../kite/internal/store/user.go` | KiteUser resources in the installation's Kubernetes cluster; not a developer-hosted account database. |
| Profile image | `../kite/internal/account/service.go` | Image input ignored and image output always empty. Do not claim photos are collected. |
| Session | `../kite/cmd/kite-api/apis/auth.go`, `src/server/api.ts`, `src/api/session.ts` | HttpOnly, SameSite=Lax accessToken cookie; HTTPS Secure/max-age 1 hour, HTTP max-age 10 minutes. JWT lifetime separately configured. Go validates session; no browser localStorage identity. |
| VM credentials | `../kite/internal/vm/service.go`, `../kite/internal/guestlogin/password.go` | Verification hash in VM spec, Linux guest password hash in Secret; initial plaintext is not retained. |
| VM access keys | `../kite/cmd/kite-controller/apps/machine-reconcile.go`, `../kite/internal/sshkey/keypair.go` | Managed SSH keys and cloud-init data are part of the installation's VM access credentials. |
| VM and allocation records | `../kite/internal/store/virtual_machine.go`, `../kite/internal/store/vm_offer.go` | VM specs, owner, domain, SSH ID, state; offer creator/claimant and expiry. Not a complete immutable audit history. |
| API logs | `../kite/cmd/kite-api/main.go` | gin.Default request logging; request timing, IP, path, method, status/latency/errors. No custom User-Agent logging or request-body recorder identified. |
| Gateway logs | `../kite/internal/gateway/server.go` | Rejected authentication and connection errors can contain remote address/port, SSH username and VM identifiers. |
| Console and SSH transport | `../kite/cmd/kite-api/apis/console_proxy.go`, `../kite/internal/gateway/server.go` | Input/output is proxied; no persistent full-session recorder identified. Guest OS or operator-added tooling can record separately. |
| Account deletion | `../kite/internal/account/service.go` | Administrator removal also deletes the account's VM custom resources. No self-service deletion page or automatic retention scheduler. |
| VM deletion | `../kite/cmd/kite-controller/apps/machine-reconcile.go`, `../build/kite-storage/longhorn/storageclass.yaml` | Controller cleans managed resources; default storage reclaim policy Delete, one replica. No secure media-erase guarantee or default backup service. |
| External font delivery | `src/app/layout.tsx` | Browser loads Pretendard CSS/fonts from cdn.jsdelivr.net, disclosing normal network request information to the CDN. |
| GitHub | Public header/footer links | User-initiated external navigation, not an account-data integration. Installation/image downloads are operator network activity. |
| Analytics, advertising, email, error SaaS | Source/deployment search and frontend dependencies | No GA, ad tracking, mail provider, Sentry or similar runtime integration found. Next build/developer telemetry is not a user behavior analytics integration. |

Kubernetes, KubeVirt, CDI, Traefik and the configured local/Longhorn storage are
self-hosted infrastructure components, not evidence of disclosure to their
upstream developers. A specific operator may add hosting, backup, ingress or
monitoring providers; those cannot be inferred from this repository.

## Google authentication boundary

The user requested a concrete Google-login privacy policy in place of future
tense. The policy limits its use to identity, email and verification, excludes
unrelated Google APIs/ads/training, and describes access withdrawal. This is an
authorized target processing policy, **not a finding that OAuth is implemented**.

The current repository has no Google login route, OAuth callback/client config,
provider subject mapping or consent scopes. Existing login is email/password.
The indirect `golang.org/x/oauth2` dependency does not establish Google login.
Do not advertise functional Google login or mark rollout complete until actual
authorization, received/stored claims, token lifetime and deletion are checked.
Use the actual Google scopes to reconcile the policy; do not add `profile` or
claim name/photo collection solely because Google supports them.

## Facts still required for an actual production publication

- The installation's legal operator name, privacy contact and responsible person.
  `adminContact` exists in runtime config but has no published default; the debug
  `ops@example.com` address is fictional and must not appear as a real contact.
- Actual request/gateway/ingress/node log retention and deletion rotation. There
  is no fixed retention period enforced by Kite. Do not invent a 30/90-day period.
- Whether backups or storage snapshots exist, their location, retention and
  deletion handling, including underlying media. Default manifests alone cannot
  prove no operator backups exist.
- Actual Google OAuth settings and any applicable cross-border disclosures;
  jsDelivr CDN processing countries/recipients/retention and the installation's
  applicable transfer requirements also need verification. Linking a provider's
  privacy policy does not replace a required transfer notice or consent.
- The signup screen has no recorded terms-acceptance field. Policy publication
  alone is not proof of a user's contractual consent.
- Concrete idle-VM criteria and notification procedure; there is no new automatic
  activity-monitoring or stop scheduler in this content change.

The policy pages remain public without login and retain their existing noindex
metadata. Removing draft wording does not establish these operational facts.
The unknown details were requested from the user; none were fabricated.

## Terms changes

The terms distinguish the software developer, the self-hosting installation
operator and a VM user. Node disk/power/network/storage/backup capacity belongs
to the operator; guest disk/program/data management belongs to the VM user.
Software defects and operator mistakes are assessed by cause, not automatically
assigned to a hosted VM user. Mandatory liability remains intact.

Idle VM stopping/resource recovery requires announced criteria, reasons and
notice. Stopping is separate from disk deletion; planned deletion allows data
export. Emergency intervention for immediate security/illegal-use harm can occur
first, with subsequent explanation and an objection/resumption channel.

## Reference materials

- [Google OpenID Connect claims and scopes](https://developers.google.com/identity/openid-connect/openid-connect)
- [Google API Services User Data Policy](https://developers.google.com/terms/api-services-user-data-policy)
- [jsDelivr privacy policy](https://www.jsdelivr.com/terms/privacy-policy)
- [개인정보보호법 제30조](https://www.law.go.kr/법령/개인정보보호법/제30조)
- [약관규제법 제7조](https://law.go.kr/LSW/lsLinkCommonInfo.do?chrClsCd=010202&lsJoLnkSeq=1025032399)

Validation passed: `npm run typecheck`, `npm run lint`, `npm run build:prod`, and
HTTP 200 anonymous rendering of both policies with correct titles, section
structure and removal of draft/conditional-introduction wording.
No browser or Kubernetes inspection, live account mutation or Go test was run.
