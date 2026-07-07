# Kite
<div align="center">
  <img width="250" height="250" alt="Kite" src="https://github.com/user-attachments/assets/98cc9bf6-5876-40cf-8c49-014421cdf7ee" />
</div>

Kite는 Kubernetes 클러스터 위에서 사용자별 KubeVirt 가상 머신을 생성하고 운영하기 위한 컨트롤 플레인입니다.

사용자는 웹 UI 또는 HTTP API로 계정과 VM을 요청합니다. `kite-api`는 요청을 검증하고 Kite CRD를 Kubernetes API server에 기록합니다. `kite-controller`는 CRD를 watch하면서 Namespace, KubeVirt VirtualMachine, CDI DataVolume, Service, Ingress, Secret 같은 실제 클러스터 리소스를 원하는 상태로 맞춥니다. VM 디스크는 Longhorn StorageClass와 CDI DataVolume을 사용합니다.

## Architecture

```mermaid
flowchart TD
    user[User / Admin]

    subgraph ui["Client Layer"]
        frontend["kite-frontend<br/>React web UI"]
    end

    subgraph control["Kite Control Plane<br/>namespace: kite"]
        api["kite-api<br/>Gin HTTP API"]
        controller["kite-controller<br/>CRD reconciler"]
        gateway["kite-gateway<br/>SSH gateway"]
    end

    subgraph apiServer["Kubernetes API Server"]
        kiteUser["KiteUser CRD<br/>cluster-scoped"]
        kiteVM["KiteVirtualMachine CRD<br/>namespaced"]
        status["CRD status<br/>observed state"]
    end

    subgraph workload["Reconciled Cluster Resources"]
        namespace["User Namespace"]
        vm["KubeVirt VirtualMachine"]
        dv["CDI DataVolume"]
        service["Service / Ingress"]
        secret["Secret / Cloud-init"]
        storage["Longhorn PVC<br/>kite-vm-storage"]
    end

    user --> frontend
    user -->|"ssh -p user-facing-port sshId@node-ip"| gateway
    frontend --> api
    api -->|"create/update desired state"| kiteUser
    api -->|"create/update desired state"| kiteVM

    kiteUser --> controller
    kiteVM --> controller
    controller --> namespace
    controller --> vm
    controller --> dv
    controller --> service
    controller --> secret
    dv --> storage

    vm -->|"runtime phase / nodeName"| controller
    dv -->|"import / clone progress"| controller
    controller --> status
    status --> frontend

    kiteVM --> gateway
    secret --> gateway
    gateway -->|"proxy SSH session"| service
    service --> vm
```

Kite는 명령형 RPC로 controller를 호출하지 않습니다. API 서버는 CRD의 desired state를 쓰고, controller는 Kubernetes controller 방식으로 reconcile합니다.

## Design Decisions

### CRD 중심 reconcile

Kite는 API 서버가 controller에 직접 명령을 보내는 구조를 쓰지 않습니다.
`kite-api`는 사용자 요청을 검증한 뒤 `KiteUser`와 `KiteVirtualMachine` CRD의
`spec`만 갱신합니다. `kite-controller`는 이 desired state를 watch하고 실제
Namespace, KubeVirt VM, DataVolume, Service, Secret, Ingress 상태를 맞춥니다.

이 방식을 선택한 이유는 다음과 같습니다.

- Kubernetes API server와 etcd를 단일 상태 저장소로 사용해 API와 controller를 느슨하게 분리합니다.
- API process가 재시작되어도 이미 기록된 CRD spec을 기준으로 controller가 계속 수렴할 수 있습니다.
- 실제 KubeVirt/CDI 리소스 상태 변화는 controller가 `status`에 다시 기록하므로 프론트엔드는 CRD 상태만 보면 됩니다.
- gRPC 같은 별도 API 계약을 유지하지 않아도 Kubernetes validation, RBAC, watch cache를 그대로 활용할 수 있습니다.

### SSH gateway

Kite는 VM별 NodePort를 만들지 않습니다. VM마다 외부 포트를 하나씩 소모하면
포트 관리가 어려워지고, VM 삭제 실패 시 고아 포트가 남을 수 있기 때문입니다.
대신 `kite-gateway`가 하나의 SSH 진입점을 받고, SSH login username을
`KiteVirtualMachine.spec.sshId`와 매칭해 내부 `vps-access-<vmName>` ClusterIP
Service로 프록시합니다. 설치 직후에는 외부 SSH LoadBalancer를 만들지 않으며,
Level 3 admin이 Admin Settings에서 “SSH Gateway”를 활성화하고 Kubernetes
Service 포트와 사용자 안내 포트를 지정해야 `kite-gateway-external` Service가
생성됩니다. 두 포트는 다를 수 있습니다. 예를 들어 공유기나 외부 라우터가
`22 -> 12311`로 포워딩한다면 Service 포트는 `12311`, 사용자 안내 포트는
`22`로 저장합니다.

```text
ssh -p <user-facing-port> <sshId>@<node-ip>
  -> kite-gateway
  -> KiteVirtualMachine(spec.sshId) lookup
  -> vps-access-<vmName>.<namespace>.svc.cluster.local:22
  -> VM sshd
```

외부 사용자의 password는 `spec.sshPasswordHash`로 검증하고, VM 내부 접속은
controller가 만든 SSH key Secret을 사용합니다. VM cloud-init에는 이 public key만
들어가며 VM 내부 password login은 꺼져 있습니다.

Kite는 host sshd 설정을 이동, 수정, 복원하지 않습니다. 또한 gateway는 VM
`sshId`와 매칭되는 Kite VM만 프록시하며, VM route가 없는 username을 host
sshd로 우회하지 않습니다. host 운영 접속은 운영자가 관리하는 기존 SSH 경로를
그대로 사용해야 합니다.

## Components

### `kite/cmd`

- `kite/cmd/kite-api`: Gin 기반 HTTP API 서버입니다. 로그인, 회원가입, 사용자 관리, VM 관리, 전역 설정 API를 제공하고 `KiteUser`, `KiteVirtualMachine` CRD를 Kubernetes API server에 기록합니다.
- `kite/cmd/kite-controller`: Kite CRD와 KubeVirt/CDI 리소스를 watch하는 controller입니다. CRD spec을 원하는 상태로 보고 실제 Kubernetes 리소스를 생성, 갱신, 삭제한 뒤 CRD status에 관측 상태를 씁니다.
- `kite/cmd/kite-gateway`: Kubernetes 내부에서 실행되는 SSH gateway입니다. 외부 SSH 연결을 받아 Kite VM 상태와 Secret을 기준으로 인증하고, 사용자 VM의 `vps-access-<vmName>` Service로 SSH session을 프록시합니다.

### `kite/api`

- `kite/api/v1`: Kite CRD spec/status를 Go 코드에서 다루기 위한 타입입니다.
- `kite/api/proto`: 이전 gRPC 설계 초안입니다. 현재 설계에서는 사용하지 않으며, API 서버와 controller 사이의 기본 흐름은 CRD 기반 reconcile입니다.

### `kite/internal`

- `kite/internal/kube`: in-cluster config와 local kubeconfig fallback을 포함한 Kubernetes client 생성 코드입니다.
- `kite/internal/store`: API 서버가 `KiteUser`, `KiteVirtualMachine` CRD를 읽고 쓰기 위한 dynamic client 기반 store입니다.
- `kite/internal/render`: controller가 Namespace, DataVolume, KubeVirt VM, Service, Ingress, Secret, NetworkPolicy, QuotaPolicy 등을 만들 때 사용하는 YAML renderer입니다.
- `kite/internal/account`, `kite/internal/auth`, `kite/internal/vm`: API 요청을 CRD spec으로 변환하고 인증, 권한, VM 요청 처리를 담당하는 service layer입니다.
- `kite/internal/platform`: base domain, TLS Secret, runtime config 같은 platform 설정을 관리합니다.
- `kite/internal/gateway`: `kite-gateway`의 route table, SSH server, Kubernetes Secret/Service 조회, VM SSH 프록시 로직입니다.

### Frontend

- `kite-frontend`: Vite/React 기반 웹 UI입니다. 사용자 로그인, VM 목록/상세, 관리자 대시보드, 전역 설정 화면을 제공합니다.

## Custom Resources

Kite가 관리하는 Kubernetes API는 `build/kite/crds.yaml`에 정의되어 있습니다.

| Kind | Scope | Resource | Purpose |
| --- | --- | --- | --- |
| `KiteUser` | Cluster | `kiteusers.hy3ons.github.io` | Kite 사용자, 권한, 사용자 namespace desired state |
| `KiteVirtualMachine` | Namespaced | `kitevirtualmachines.hy3ons.github.io` | 사용자별 VM spec, 전원 의도, 디스크/접속 정보, VM status |

`KiteUser`는 cluster-scoped 리소스이므로 namespace 없이 생성됩니다. `KiteVirtualMachine`은 사용자 namespace에 생성되고, controller가 같은 namespace에 VM 관련 리소스를 만듭니다.

## Repository Layout

```text
.
├── build/
│   ├── kite/              # Kite application 공통 Kubernetes manifests
│   ├── kite-storage/      # Longhorn StorageClass, cleanup, golden image manifests
│   ├── dev/               # local Docker build + current cluster deploy scripts
│   ├── deploy/            # k3s production-oriented install scripts
│   └── examples/          # KiteUser, KiteVirtualMachine example resources
├── docs/                  # project conventions
├── kite/
│   ├── cmd/               # kite-api, kite-controller, kite-gateway entrypoints
│   ├── api/               # CRD Go types and retired proto draft
│   └── internal/          # Kubernetes clients, stores, renderers, services
├── kite-frontend/         # web frontend
├── test/                  # release E2E gates for k3s, minikube, and generic k8s
├── ghcr-install.sh        # GHCR image pull based install entrypoint
├── ghcr-stage-install.sh  # maintainer QA wrapper for stage GHCR images
├── ghcr-update.sh         # GHCR image pull based update entrypoint
├── build-install.sh       # local build based development install entrypoint
├── uninstall.sh           # operator-facing Kite uninstall entrypoint
└── build-clear.sh         # developer-facing build/deploy cleanup entrypoint
```

## Kubernetes Manifests

- `build/kite`: Kite runtime manifests shared by development and production installs. It includes namespace, CRDs, component ServiceAccounts, RBAC, API deployment, controller deployment, gateway deployment, frontend deployment, and Services.
- `build/kite-storage/longhorn/storageclass.yaml`: Kite VM disks use `kite-vm-storage`, backed by Longhorn with `diskSelector: "kite"`.
- `build/kite-storage/golden-images/ubuntu-22.04.yaml`: Ubuntu golden image DataVolume imported by CDI.
- `build/kite-storage/longhorn-cleanup`: optional cleanup DaemonSet for Kite-owned Longhorn host data.
- `build/examples`: example custom resources for manual CRD testing.

## Install And Update Modes

Kite has two top-level install entrypoints and one GHCR update entrypoint:

- `./ghcr-install.sh`: pull-based install. It installs or waits for Longhorn,
  KubeVirt, CDI, applies the Ubuntu golden image, and deploys Kite manifests
  that pull prebuilt production images from `ghcr.io/hy3ons`.
- `./ghcr-stage-install.sh`: maintainer QA install. It calls the same
  pull-based installer as `ghcr-install.sh`, but pins the repository ref and
  image tag to `stage` so maintainers do not have to remember environment
  variables.
- `./build-install.sh`: local-build install. It prepares the same infrastructure, builds
  the API, controller, gateway, and frontend images from this checkout, then
  imports or loads those images into the selected local cluster before deploying
  Kite.
- `./ghcr-update.sh`: pull-based update. It preserves existing Kite users, VMs,
  PVCs, runtime config, and shared infrastructure while reapplying Kite manifests
  and rolling Deployments to a selected GHCR image tag.

Both install modes deploy `kite-gateway` with only an internal `ClusterIP`
Service. They do not move host `sshd`, do not expose external SSH, and do not
claim port `22`. A Level 3 admin enables VM SSH exposure later from Admin
Settings by setting the SSH Gateway Service port. The controller then creates
or updates `service/kite-gateway-external` as a `LoadBalancer`. Admin Settings
also stores the user-facing SSH port shown in Dashboard/VM Detail. This may be
different from the Service port when an external router maps public `22` to a
custom node/LB port. The gateway routes only VM `sshId` logins and never proxies
host Linux accounts.

The gateway host key is also part of the install contract. On Linux hosts the
installer tries to reuse the existing OpenSSH host key from `/etc/ssh` so users
do not see a different SSH fingerprint just because port `22` is now served by
`kite-gateway`. If the key exists but cannot be read in automatic mode, the
installer generates a gateway key instead of stopping halfway. Existing gateway
host key Secrets are kept by default; set
`KITE_GATEWAY_HOST_KEY_REFRESH=true KITE_GATEWAY_HOST_KEY_SOURCE=host` when you
intentionally want to replace the gateway Secret with the current host key.

All public root scripts collect their interactive choices at the beginning of
the run. If an environment variable is already set, the script treats it as an
operator decision and does not ask again. Set `KITE_ASSUME_DEFAULTS=true` to
skip prompts and use defaults/env values for automation.

## Quick Install and Uninstall

Install Kite on a k3s or Kubernetes host without cloning this repository:

```sh
curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/main/ghcr-install.sh | bash
```

Maintainer stage QA uses the same installer path but pulls `stage` images:

```sh
curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/stage/ghcr-stage-install.sh | bash
```

The remote installer downloads the selected GitHub archive into a temporary
directory and runs `build/deploy/scripts/install-all.sh`. It uses GHCR images
and does not build containers locally.

Update an existing Kite install without cloning this repository:

```sh
curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/main/ghcr-update.sh | bash
```

Update from a specific repository ref, or pin runtime images to a specific
published image tag:

```sh
curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/main/ghcr-update.sh \
  | KITE_GHCR_UPDATE_REF=main KITE_UPDATE_TAG=sha-<commit> bash
```

The remote updater downloads the selected GitHub archive into a temporary
directory and runs `build/deploy/scripts/update-ghcr.sh`. It does not remove
Kite users, VMs, PVCs, Longhorn, KubeVirt, CDI, or host sshd settings. It
applies CRDs/RBAC/runtime manifests, preserves an existing
`kite-runtime-config`, changes Deployment images, waits for rollout, and rolls
back to the previous images if rollout or smoke checks fail.

Uninstall Kite resources without cloning this repository:

```sh
curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/main/uninstall.sh | bash
```

Uninstall from a specific branch or tag:

```sh
curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/main/uninstall.sh \
  | KITE_UNINSTALL_REF=stage bash
```

The remote cleanup flow downloads the selected GitHub archive into a temporary
directory and runs `build/deploy/scripts/clean.sh`, which delegates to
`build/deploy/scripts/uninstall-kite.sh`. By default it removes Kite CRDs,
namespace resources, Deployments, Services, and Kite-owned runtime state.
Longhorn storage cleanup stays opt-in because it can delete VM disk
infrastructure.

## Development Install

`./build-install.sh` builds local Docker images and deploys them to the selected Kubernetes cluster.

```sh
KITE_CLUSTER=k3s ./build-install.sh
```

Supported `KITE_CLUSTER` values are `minikube`, `k3s`, `k3d`, `kind`, `k8s`, `kubernetes`, and `current`.

For local clusters, the script builds these images and loads or imports them into the cluster runtime when needed:

- `ghcr.io/hy3ons/kite-api:<tag>`
- `ghcr.io/hy3ons/kite-controller:<tag>`
- `ghcr.io/hy3ons/kite-gateway:<tag>`
- `ghcr.io/hy3ons/kite-frontend:<tag>`

Generic Kubernetes clusters usually need a registry push:

```sh
PUSH_IMAGES=true IMAGE_REGISTRY=registry.example.com/kite KITE_CLUSTER=k8s ./build-install.sh
```

Development cleanup:

```sh
KITE_CLUSTER=k3s ./build-clear.sh
```

Longhorn cleanup is opt-in because it can remove VM disk infrastructure:

```sh
CLEAR_LONGHORN=true KITE_CLUSTER=k3s ./build-clear.sh
CLEAR_LONGHORN_DATA=true CLEAR_LONGHORN_DATA_CONFIRM=true KITE_CLUSTER=k3s ./build-clear.sh
```

More details are in `build/dev/README.md`. The full `build` directory layout
and root command naming contract are documented in `build/README.md`.

## Production-Oriented k3s Install

`./ghcr-install.sh` contains a pull-based install flow for k3s clusters. Longhorn,
KubeVirt, and CDI are required for VM disk provisioning and VM runtime, so the
default install path applies them when they are missing. Kite images are pulled
from GHCR instead of being built locally.

The download install flow is intentionally split into two steps:

1. `ghcr-install.sh` is the small bootstrap script fetched by `curl`.
2. The bootstrap script downloads the selected repository ref as a tar archive,
   extracts it under a temporary directory, and executes
   `build/deploy/scripts/install-all.sh` from that extracted copy.

That inner installer installs or waits for Longhorn, KubeVirt, and CDI, prepares
the Ubuntu golden image DataVolume, creates the Kite gateway host key Secret,
and deploys the Kite manifests using GHCR images.

```sh
./ghcr-install.sh
```

If Longhorn is managed outside Kite and you do not want Kite to apply the
Longhorn manifest:

```sh
INSTALL_LONGHORN=false ./ghcr-install.sh
```

On apt-based Linux hosts, `INSTALL_LONGHORN=true` also installs the Longhorn host
packages `open-iscsi` and `nfs-common` when they are missing. If the current
user cannot use `sudo`, install those packages first or run the installer as
root.

Expected storage flow:

```text
kite/ubuntu-22.04 DataVolume
  -> PVC using StorageClass kite-vm-storage

user namespace VM DataVolume
  -> clone source pvc kite/ubuntu-22.04
  -> PVC using StorageClass kite-vm-storage
```

Uninstall Kite resources:

```sh
build/deploy/scripts/uninstall-kite.sh
```

The remote cleanup commands are listed in
[Quick Install and Uninstall](#quick-install-and-uninstall). The cleanup removes
Kite application resources and Kite CRDs first, then removes Kite Longhorn disk
data only when the explicit cleanup environment variables are set.

More details are in `build/deploy/README.md`.

## GHCR Update

Use `./ghcr-update.sh` after Kite is already installed and the cluster should
move to a newer GHCR image tag.

```sh
./ghcr-update.sh
```

For unattended updates:

```sh
KITE_ASSUME_DEFAULTS=true KITE_UPDATE_TAG=latest ./ghcr-update.sh
```

For a pinned production rollout or rollback:

```sh
KITE_ASSUME_DEFAULTS=true KITE_UPDATE_TAG=sha-<commit> ./ghcr-update.sh
```

Useful update variables:

| Variable | Default | Meaning |
| --- | --- | --- |
| `KITE_UPDATE_TAG` | `latest` | GHCR tag applied to all selected Kite images. |
| `KITE_UPDATE_COMPONENTS` | `api,controller,gateway,frontend` | Comma-separated component list. |
| `KITE_UPDATE_APPLY_CRDS` | `true` | Reapply Kite CRD definitions before rollout. |
| `KITE_UPDATE_APPLY_RBAC` | `true` | Reapply ServiceAccount and RBAC manifests. |
| `KITE_UPDATE_HEALTH_CHECK` | `true` | Port-forward API/frontend services and smoke check them. |
| `KITE_UPDATE_RUN_VERIFY` | `false` | Also run the heavier deploy verify script. |
| `KITE_UPDATE_ROLLBACK_ON_FAIL` | `true` | Restore previous Deployment images when rollout/smoke fails. |
| `KITE_UPDATE_DRY_RUN` | `false` | Print the plan and commands without changing the cluster. |

The update script is intentionally not an infrastructure upgrader. Longhorn,
KubeVirt, CDI, golden image data, user namespaces, VM resources, and PVCs stay
outside the default update scope.

## Container Images

Production images are published to GHCR when commits land on `main`.

| Component | Image |
| --- | --- |
| `kite-api` | `ghcr.io/hy3ons/kite-api` |
| `kite-controller` | `ghcr.io/hy3ons/kite-controller` |
| `kite-gateway` | `ghcr.io/hy3ons/kite-gateway` |
| `kite-frontend` | `ghcr.io/hy3ons/kite-frontend` |

The workflow publishes these tags:

- `latest`
- `main`
- `production`
- `sha-<commit>`

The workflow logs in with the `GHCR_TOKEN` GitHub secret. `./ghcr-install.sh`
uses the `production` tag by default through a temporary kustomize overlay, and
`./ghcr-update.sh` uses GHCR images by default, while
`./build-install.sh` builds images locally and imports or loads them into the
selected development cluster.

Before production publishing, `Stage GHCR Image Builds` runs on `stage` pushes
and pull requests. Pull requests build the same image matrix without pushing.
`stage` branch pushes publish `stage` and `stage-sha-<commit>` tags, which
`./ghcr-stage-install.sh` uses for maintainer QA before `stage` is promoted to
`main`.

GHCR cleanup is intentionally conservative. Moving tags and unknown tags are
protected. Only managed SHA tags are removed after their retention window:

- `sha-<commit>` from `main`: keep for 30 days.
- `stage-sha-<commit>` from `stage`: keep for 10 days.
- `latest`, `main`, `production`, and `stage`: always keep.

On slow or flaky registry networks, set `KITE_ROLLOUT_TIMEOUT=15m` or higher
when running `./ghcr-install.sh`; this controls how long the installer waits for
Kite workloads to pull GHCR images and become ready.

## Release E2E Tests

Release validation lives under `test/`. These scripts build the current
checkout, deploy it to a real cluster, exercise API/signup/login/CRD/controller
flows, create a real VM, wait for KubeVirt `Running`, check the frontend, and
verify the SSH gateway.

```sh
./test/all-test-k3s.sh
```

```sh
./test/all-test-minikube.sh
```

```sh
TEST_IMAGE_REGISTRY=registry.example.com/kite ./test/all-test-k8s.sh
```

k3s E2E additionally verifies that default install keeps `kite-gateway` internal
until Admin Settings enables external exposure. Kite no longer ships a host sshd
mutation test because installers do not move or restore the host SSH daemon.

More details are in `test/Readme.md` and `test/Test-Specification.md`.

## Runtime Notes

- Kite runtime resources run in the `kite` namespace.
- CRDs are cluster-wide API extensions and do not have a namespace.
- `KiteUser` instances are cluster-scoped.
- `KiteVirtualMachine` instances are namespaced.
- In-cluster execution uses the mounted service account. Local kubeconfig fallback is for development.
- VM disks use CDI DataVolume and Longhorn.
- The controller writes observed state to CRD `status`; user intent belongs in CRD `spec`.

## Related Docs

- `CONTRIBUTE-KO.md`
- `CONTRIBUTE-EN.md`
- `docs/branch-policy.md`
- `docs/commit-convention.md`
- `test/Readme.md`
- `test/Test-Specification.md`
- `kite/cmd/kite-api/Readme.md`
- `kite/cmd/kite-controller/Readme.md`
- `kite/cmd/kite-gateway/Readme.md`
- `build/dev/README.md`
- `build/deploy/README.md`
- `build/kite/README.md`
- `build/kite-storage/README.md`
- `build/examples/README.md`
