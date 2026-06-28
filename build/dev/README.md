# Kite Development Install

`dev.sh` builds the Kite API, controller, gateway, and frontend images with
local Docker, then applies the shared `build/kite` manifests to the selected
cluster.

For a full development install that also prepares Longhorn, KubeVirt, CDI, and
the Ubuntu golden image before building and deploying Kite from local source, use
the root wrapper:

```sh
KITE_CLUSTER=k3s ./dev.sh
```

`./dev.sh` calls `build/dev/all-in-one.sh`, and that script calls
`build/dev/dev.sh` after the infrastructure is ready. The Kite API, controller,
gateway, and frontend are built from local source and deployed as Kubernetes
workloads.

Each phase can be skipped through environment flags:

```sh
INSTALL_LONGHORN=false \
INSTALL_KUBEVIRT=false \
INSTALL_CDI=false \
APPLY_GOLDEN_IMAGE=false \
KITE_CLUSTER=k3s \
./dev.sh
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

The frontend image does not read `.env.*` files. `dev.sh` injects Vite values
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
rolled out. They reuse the same image loading rules as `dev.sh`, but only touch
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

`./clear.sh` is the root cleanup wrapper. It removes Kite development resources
and local Kite images through `build/dev/clear.sh`.

```sh
KITE_CLUSTER=k3s ./clear.sh
```

The default cleanup removes Kite-owned application resources, Kite CRDs,
KiteUser namespaces, and VM allocations inside those namespaces. It does not
delete shared KubeVirt, CDI, or Longhorn installations, because those components
may already be used by workloads outside Kite.

Longhorn cleanup is disabled by default because it deletes VM disk data.

```sh
CLEAR_LONGHORN=true KITE_CLUSTER=k3s ./clear.sh
```

`CLEAR_LONGHORN=true` uninstalls Longhorn only when no Longhorn PV remains. If
another workload is still using Longhorn, the script skips Longhorn uninstall.

To remove Kite-owned host data under `/mnt/kite-longhorn`, use the extra
confirmation flag. This can be used without uninstalling Longhorn, but the
script skips data deletion while Longhorn PVs still exist.

```sh
CLEAR_LONGHORN_DATA=true CLEAR_LONGHORN_DATA_CONFIRM=true KITE_CLUSTER=k3s ./clear.sh
```

## Gateway

`dev.sh` also builds and deploys `kite-gateway`. The Service is a
`LoadBalancer` that exposes external SSH on port `22` and forwards it to the
pod's internal `2222` port.

```sh
ssh <sshId>@<node-ip>
```

The current implementation authenticates with `KiteVirtualMachine.spec.sshPasswordHash`
and proxies to `vps-access-<vmName>.<namespace>.svc.cluster.local:22` with the
VM SSH key Secret created by `kite-controller`.

`dev.sh` creates `kite-gateway-host-key` automatically when it does not exist.
On Linux hosts it first tries to copy the existing OpenSSH host key from
`/etc/ssh/ssh_host_ed25519_key`, `ssh_host_ecdsa_key`, or `ssh_host_rsa_key`.
That keeps the SSH fingerprint consistent when Kite takes over port `22`. If no
host key is available, the script generates a gateway key instead.

Existing Secrets are kept by default so client fingerprints do not change on
every deploy. To intentionally replace an already-created gateway key from the
host key, run:

```sh
KITE_GATEWAY_HOST_KEY_REFRESH=true KITE_GATEWAY_HOST_KEY_SOURCE=host KITE_CLUSTER=k3s build/dev/dev.sh
```

When the host is Linux with systemd OpenSSH, `dev.sh` asks before moving host
sshd away from port `22`. If confirmed, it backs up `/etc/ssh/sshd_config` under
`/etc/kite/host-sshd`, configures host sshd to listen on `2222`, and restarts
the service so the gateway can own port `22`. `./clear.sh` asks before restoring
that backup. Set `KITE_MANAGE_HOST_SSHD=true` or `KITE_RESTORE_HOST_SSHD=true`
for non-interactive opt-in, and set `MANAGE_HOST_SSHD=false` or
`RESTORE_HOST_SSHD=false` to skip these host changes.

When no Kite VM uses the SSH login username, `kite-gateway` falls back to the
host sshd at the node IP on port `2222`. This lets existing host accounts keep
using `ssh <host-user>@<node-ip>` on port `22` after the gateway is installed.
If a Kite VM `sshId` conflicts with a host user, the VM route has priority and
host administration should use `ssh <host-user>@<node-ip> -p 2222`.
