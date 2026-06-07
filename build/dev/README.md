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
`kubernetes`, and `current`. `k3d` uses `k3d image import`, and `kind` uses
`kind load docker-image`. Generic Kubernetes modes build local images and can
push them when the cluster pulls from a registry:

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

`./clear.sh` is the root cleanup wrapper. It removes Kite development resources
and local Kite images through `build/dev/clear.sh`.

```sh
KITE_CLUSTER=k3s ./clear.sh
```

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
That Secret stores the SSH server host key used by external clients, so gateway
pod restarts do not change the host fingerprint.

When the host is Linux with systemd OpenSSH, `dev.sh` asks before moving host
sshd away from port `22`. If confirmed, it backs up `/etc/ssh/sshd_config` under
`/etc/kite/host-sshd`, configures host sshd to listen on `2222`, and restarts
the service so the gateway can own port `22`. `./clear.sh` asks before restoring
that backup. Set `KITE_MANAGE_HOST_SSHD=true` or `KITE_RESTORE_HOST_SSHD=true`
for non-interactive opt-in, and set `MANAGE_HOST_SSHD=false` or
`RESTORE_HOST_SSHD=false` to skip these host changes.
