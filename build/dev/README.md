# Kite Development Install

`build/dev/dev.sh` builds the Kite API, controller, gateway, and frontend images with
local Docker, then applies the shared `build/kite` manifests to the selected
cluster.

For a full development install that also prepares Longhorn, KubeVirt, CDI, and
the Ubuntu golden image before building and deploying Kite from local source, use
the root wrapper:

```sh
KITE_CLUSTER=k3s ./build-install.sh
```

`./build-install.sh` calls `build/dev/all-in-one.sh`, and that script calls
`build/dev/dev.sh` after the infrastructure is ready. The Kite API, controller,
gateway, and frontend are built from local source and deployed as Kubernetes
workloads.

`./build-install.sh` asks all install choices near the start of the run. If a
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
./build-install.sh
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

The frontend image does not read `.env.*` files. `build/dev/dev.sh` injects Vite values
through Docker build args from the current shell session:

```sh
FRONTEND_VITE_BUILD_MODE=production \
FRONTEND_VITE_API_BASE_URL=/api/v1 \
FRONTEND_VITE_USE_MOCK=false \
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

The frontend script accepts the same Vite build environment as the full dev
deploy:

```sh
FRONTEND_VITE_BUILD_MODE=production \
FRONTEND_VITE_API_BASE_URL=/api/v1 \
FRONTEND_VITE_USE_MOCK=false \
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

`./build-clear.sh` is the root development cleanup wrapper. It always targets the
local checkout flow under `build/dev`, while `./uninstall.sh` is reserved for
pull-based deployment uninstall. When `./build-clear.sh` runs in a terminal, it asks
which cleanup scope to use with numbered choices.

```sh
./build-clear.sh
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

`build/dev/dev.sh` also builds and deploys `kite-gateway`. `build-install.sh`
keeps it internal by default and never moves host sshd. External VM SSH access
is enabled later from Admin Settings, which drives `kite-runtime-config` and the
controller-owned `kite-gateway-external` Service.

```sh
ssh -p <admin-selected-port> <sshId>@<node-ip>
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

When no Kite VM uses the SSH login username, `kite-gateway` can optionally fall
back to host sshd. This is disabled by default. A Level 3 admin must explicitly
enable host fallback and enter the host sshd port in Admin Settings. Kite then
patches the gateway Deployment with `$(KITE_NODE_IP):<host-sshd-port>`.
