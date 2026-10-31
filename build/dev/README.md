# Kite Development Install

## Host Go API + Docker Minikube + Next.js

For frontend work, keep Node and Go on macOS and use a dedicated `kite-dev`
Minikube cluster as the API's Kubernetes storage. OrbStack must be running;
its Docker engine is enough. No additional OrbStack VM is needed.
Prerequisites: `minikube`, `kubectl`, Docker, Go matching `kite/go.mod`, and
Node.js 22.12 or newer. On macOS, missing cluster tools can be installed with
`brew install minikube kubernetes-cli`.

From the repository root:

```sh
# Start Docker Minikube (4 CPUs, 4 GiB) and apply only the three Kite CRDs.
bash build/dev/local.sh setup
# Terminal 1: run the real API directly with Go on :8080.
bash build/dev/local.sh api
# Terminal 2: preview UI using browser-local mock data on :5173.
cd kite-frontend
npm ci
npm run dev
```

To check real Go requests and SSR, stop the frontend dev process with Ctrl+C
and run `npm run stage` in the same directory. Both modes use
`http://127.0.0.1:5173`; run one frontend mode at a time. `stage` here is the
npm mode, independent of the Git `stage` branch or a deployed staging cluster.
`KITE_API_INTERNAL_URL` defaults to `http://127.0.0.1:8080` for SSR and the local
same-origin `/api/v1` proxy. No frontend Docker build is needed for this flow.

```text
Browser -> Next.js :5173 (dev: mock data / stage: Go API + SSR)
                   -> Go :8080 -> ~/.kube/config -> Docker Minikube kite-dev
```

The kubeconfig stays on macOS at `~/.kube/config`, with certificates under
`~/.minikube`. Setup selects the `kite-dev` context. The current API code reads
that exact kubeconfig file; `local.sh api` refuses to start if its current
context points elsewhere. All setup kubectl writes explicitly target
`kite-dev`. The API bootstraps namespace `kite`, `kite-runtime-config`, and
`kite-runtime-secret` in this local cluster.
On Apple Silicon macOS, the script prefers `/opt/homebrew/bin/kubectl` over
OrbStack's bundled copy. Override it with `KITE_KUBECTL_BIN` if needed. For manual
commands, use a kubectl version compatible with the cluster (for example,
`/opt/homebrew/bin/kubectl --context kite-dev get nodes`).

No controller, KubeVirt, CDI, Longhorn, golden image, ingress, or Kite workload
Deployment is installed by this lightweight setup. Login, settings, user
records, and VM/offer CRD requests can be developed; actual VM provisioning,
power changes, console, and SSH need the full runtime. After a real signup,
prepare its namespace before testing VM CRD creation (normally the controller
does this):

```sh
kubectl --context kite-dev get kiteusers
bash build/dev/local.sh namespace ku-REPLACE-WITH-RESOURCE-NAME
```

HTTP checks run without a browser:

```sh
# Go must be running. First run creates a local QA administrator in a fresh cluster.
node build/dev/local-smoke.mjs
# Additionally verify stage proxy, session redirects, and authenticated SSR pages:
FRONTEND_ORIGIN=http://127.0.0.1:5173 node build/dev/local-smoke.mjs
```

### Isolated signup and frontend integration checks

To verify the **first** signup without deleting accounts in `kite-dev`, use a
separate Docker profile. These commands require OrbStack's Docker engine and
keep the existing profile intact:

```sh
minikube start -p kite-local-qa --driver=docker --container-runtime=containerd --cpus=4 --memory=4096
/opt/homebrew/bin/kubectl --context kite-local-qa apply -f build/kite/crds.yaml
/opt/homebrew/bin/kubectl --context kite-local-qa wait --for=condition=Established --timeout=60s \
  crd/kiteusers.hy3ons.github.io crd/kitevirtualmachines.hy3ons.github.io crd/kitevirtualmachineoffers.hy3ons.github.io
# Select this local context BEFORE starting Go; it reads ~/.kube/config.
/opt/homebrew/bin/kubectl config use-context kite-local-qa
# Terminal 1, from kite/:
GIN_MODE=release go run ./cmd/kite-api
# Terminal 2, from kite-frontend/, with mock data and debug sessions disabled:
NEXT_PUBLIC_USE_MOCK=false NEXT_PUBLIC_DEBUG_DIRECT_ROUTES=false npm run dev
# Terminal 3, from kite-frontend/:
npm run test:local
```

`test:local` is opt-in and excluded from `npm test`. It checks the current context
is `kite-local-qa`, then calls the real loopback Go/Next servers using the same
Zod endpoint adapters as the application. The first run needs a fresh QA profile:
it creates `qa-admin@kite.local` (Level 3), followed by `qa-member@kite.local`
(Level 0). Generated passwords are stored in `.local-dev/kite-local-qa/admin.json`
and `member.json` with owner-only permissions. Reruns reuse those accounts.

Checks cover cookie sessions, incorrect passwords, permission changes including
Level 0, forbidden operations, authenticated SSR and redirects, VM CRD creation,
power intent, deletion intent, and assigned-offer creation/claim/deletion.
The suite prepares only these QA users' namespaces and removes its temporary
VM/offer CRDs. Accounts remain for manual UI QA. It does not boot VMs, connect
a console, or use a browser. Full runtime components are still required for
those checks. Stop the QA profile with `minikube stop -p kite-local-qa` when done.

The original `local-smoke.mjs` account is `local-admin@kite.local`; its generated password is saved in
`.local-dev/login.json` with owner-only permissions, ignored by Git. Repeated
checks reuse it. If you delete and recreate the cluster, remove that local
credentials file before re-running the smoke check. Other signups start with
read-only access. UI QA is manual; no Playwright or browser E2E is run.

```sh
bash build/dev/local.sh status
# Stop Node and Go with Ctrl+C in their terminals, then stop the cluster:
bash build/dev/local.sh stop
```

## Container deployment to a cluster

`build/dev/dev.sh` builds the Kite API, controller, gateway, and frontend images with
local Docker, then applies the shared `build/kite` manifests to the selected
cluster.

For a full development install that also prepares Longhorn, KubeVirt, CDI, and
the Ubuntu golden image before building and deploying Kite from local source, use
the development wrapper:

```sh
KITE_CLUSTER=k3s ./build/dev/build-install.sh
```

`./build/dev/build-install.sh` calls `build/dev/all-in-one.sh`, and that script calls
`build/dev/dev.sh` after the infrastructure is ready. The Kite API, controller,
gateway, and frontend are built from local source and deployed as Kubernetes
workloads.

`./build/dev/build-install.sh` asks all install choices near the start of the run. If a
variable is already set in the environment, that value is used without asking.
Set `KITE_ASSUME_DEFAULTS=true` for automation that should use documented
defaults.

Each phase can be skipped through environment flags:

```sh
INSTALL_LONGHORN=false \
INSTALL_KUBEVIRT=false \
INSTALL_CDI=false \
APPLY_GOLDEN_IMAGE=false \
KITE_CLUSTER=k3s \
./build/dev/build-install.sh
```

```sh
KITE_CLUSTER=k3s build/dev/dev.sh
```

For k3s, the script imports built images into the k3s containerd namespace by
default. Override the import command when sudo is not needed:

```sh
K3S_CTR_CMD="k3s ctr -n k8s.io" KITE_CLUSTER=k3s build/dev/dev.sh
```

Supported cluster modes are `minikube`, `k3s`, `k3d`, `kind`, `k8s`,
`kubernetes`, and `current`. The default image prefix is `kite-dev`, so dev
deploys build local images such as `kite-dev/kite-api:<tag>` instead of pulling
GHCR images. `k3d` uses `k3d image import`, and `kind` uses `kind load
docker-image`. Generic Kubernetes modes build local images and can push them
only when the cluster pulls from a registry:

```sh
PUSH_IMAGES=true IMAGE_REGISTRY=registry.example.com/kite KITE_CLUSTER=k8s build/dev/dev.sh
```

The frontend Docker build excludes `.env.*` files. `build/dev/dev.sh` injects Next.js public values
through Docker build args from the current shell session:

```sh
FRONTEND_NEXT_BUILD_MODE=production \
FRONTEND_NEXT_API_BASE_URL=/api/v1 \
FRONTEND_NEXT_USE_MOCK=false \
KITE_CLUSTER=k3s build/dev/dev.sh
```

Defaults are `production`, `/api/v1`, and `false`.

## Component rebuilds

Use the component scripts when only one Kite workload needs to be rebuilt and
rolled out. They reuse the same image loading rules as `build/dev/dev.sh`, but only touch
the selected Deployment and its image.

```sh
KITE_CLUSTER=k3s build/dev/frontend.dev.sh
```

The script prints a compact deploy plan before it starts, then shows numbered
steps for Docker build, local cluster image load, manifest apply, and rollout
wait. Disable the plan table for quieter logs:

```sh
KITE_DEV_SHOW_PLAN=false KITE_CLUSTER=k3s build/dev/frontend.dev.sh
```

Preview the same flow without building images or touching the cluster:

```sh
KITE_DEV_DRY_RUN=true KITE_CLUSTER=k3s build/dev/frontend.dev.sh
```

Equivalent generic form:

```sh
KITE_CLUSTER=k3s build/dev/component.dev.sh frontend
```

Available component scripts are:

```sh
build/dev/api.dev.sh
build/dev/controller.dev.sh
build/dev/gateway.dev.sh
build/dev/frontend.dev.sh
```

The frontend script accepts the same Next.js build environment as the full dev
deploy:

```sh
FRONTEND_NEXT_BUILD_MODE=production \
FRONTEND_NEXT_API_BASE_URL=/api/v1 \
FRONTEND_NEXT_USE_MOCK=false \
IMAGE_TAG=frontend-qa \
KITE_CLUSTER=k3s \
build/dev/frontend.dev.sh
```

Use the component cleanup scripts when one workload should be removed before a
fresh rebuild:

```sh
KITE_CLUSTER=k3s build/dev/clear-frontend.sh
KITE_CLUSTER=k3s build/dev/frontend.dev.sh
```

Available cleanup scripts are:

```sh
build/dev/clear-api.sh
build/dev/clear-controller.sh
build/dev/clear-gateway.sh
build/dev/clear-frontend.sh
```

Set `CLEAR_IMAGES=false` to keep local Docker and k3s images while deleting only
the Kubernetes resources.

`./build/dev/build-clear.sh` is the development cleanup wrapper. It always targets the
local checkout flow under `build/dev`, while `./uninstall.sh` is reserved for
pull-based deployment uninstall. When `./build/dev/build-clear.sh` runs in a terminal, it asks
which cleanup scope to use with numbered choices.

```sh
./build/dev/build-clear.sh
```

The default cleanup removes Kite-owned application resources, Kite CRDs,
KiteUser namespaces, and VM allocations inside those namespaces. It does not
delete shared KubeVirt, CDI, or Longhorn installations, because those components
may already be used by workloads outside Kite.

The prompt separately asks whether to delete local images, uninstall Longhorn,
or delete Kite Longhorn host data. Longhorn
cleanup is disabled by default because it can delete VM disk data. Non-terminal
automation can still set the existing environment variables directly.
`KITE_ASSUME_DEFAULTS=true` skips prompts and uses defaults/env values.

Use `./uninstall.sh` or `build/deploy/scripts/uninstall-kite.sh` when the target is
a pull-based install that should follow the production uninstall path. The
folder-level map and naming rules are documented in `build/README.md`.

## Gateway

`build/dev/dev.sh` also builds and deploys `kite-gateway`. `build/dev/build-install.sh`
keeps it internal by default and never moves host sshd. External VM SSH access
is enabled later from Admin Settings, which drives `kite-runtime-config` and the
controller-owned `kite-gateway-external` Service. Admin Settings separates the
Gateway Service port from the user-facing port shown in the UI, so router/NAT
layouts such as public `22 -> 12311` can still show users the plain SSH command.

```sh
ssh -p <user-facing-port> <sshId>@<node-ip>
```

The current implementation authenticates with `KiteVirtualMachine.spec.sshPasswordHash`
and proxies to `vps-access-<vmName>.<namespace>.svc.cluster.local:22` with the
VM SSH key Secret created by `kite-controller`.

`build/dev/dev.sh` creates `kite-gateway-host-key` automatically when it does not exist.
On Linux hosts it first tries to copy the existing OpenSSH host key from
`/etc/ssh/ssh_host_ed25519_key`, `ssh_host_ecdsa_key`, or `ssh_host_rsa_key`.
That can keep the gateway fingerprint familiar if the operator later exposes the
gateway on a public SSH port. If no host key is available, or automatic mode
cannot read it, the script generates a gateway key instead.

Existing Secrets are kept by default so client fingerprints do not change on
every deploy. To intentionally replace an already-created gateway key from the
host key, run:

```sh
KITE_GATEWAY_HOST_KEY_REFRESH=true KITE_GATEWAY_HOST_KEY_SOURCE=host KITE_CLUSTER=k3s build/dev/dev.sh
```

When no Kite VM uses the SSH login username, authentication fails. Kite does
not proxy host Linux accounts and never manages the host sshd port.

## Maintainer Entrypoints

All developer-facing wrappers live in this directory. Run them from a checkout;
the wrappers resolve implementation paths relative to their own location:

```sh
./build/dev/build-install.sh       # Build this checkout and install dependencies
./build/dev/build-clear.sh         # Remove the development deployment
./build/dev/stage-install.sh       # Validate published stage images via install.sh
sudo env KITE_ASSUME_DEFAULTS=true ./build/dev/build-update.sh
```

`build-update.sh` fetches a disposable source checkout (`KITE_UPDATE_REF=main`
by default), then runs `build/dev/update-local.sh` from that revision. For older
revisions without the helper it uses the helper beside the local wrapper.
It builds all selected images before importing them into this host's single-node
K3s containerd. The shared runtime updater patches existing Deployment images;
it does not reinstall dependencies, reset runtime settings, or recreate VM disks.
Use the root `update.sh` for normal updates using published GHCR images.

Bootstrap routing and update safety checks run without a cluster:

```sh
bash build/test/entrypoints.test.sh
bash build/test/update-ghcr.test.sh
```
