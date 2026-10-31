# Kite Build Directory

This directory is the operational surface for installing, developing, testing,
and removing Kite. The layout is intentionally split by purpose so the root
commands say what they do before the user has to read the implementation.

## Naming Contract

| Name | Meaning | Primary entrypoint |
| --- | --- | --- |
| `install.sh` | 일반 사용자/운영자가 GHCR 이미지를 pull해서 설치한다. | `./install.sh` -> `build/deploy/scripts/install-all.sh` |
| `build/dev/stage-install.sh` | 메인테이너가 stage GHCR 이미지를 production 설치 흐름으로 검증한다. | `./build/dev/stage-install.sh` -> `./install.sh` -> `build/deploy/scripts/install-all.sh` |
| `build/dev/build-install.sh` | 개발자가 현재 checkout 이미지를 빌드해서 설치한다. | `./build/dev/build-install.sh` -> `build/dev/all-in-one.sh` |
| `update.sh` | 일반 사용자가 배포된 이미지를 받아 기존 설치를 갱신한다. | `./update.sh` -> `build/deploy/scripts/update-ghcr.sh` |
| `build/dev/build-update.sh` | 서버에서 최신 소스를 받아 빌드하고 기존 컨테이너만 교체한다. | `./build/dev/build-update.sh` -> `build/dev/update-local.sh` |
| `uninstall.sh` | 일반 사용자/운영자가 Kite 배포를 제거한다. | `./uninstall.sh` -> `build/deploy/scripts/clean.sh` |
| `build/dev/build-clear.sh` | 개발자가 local build/deploy 산출물을 제거한다. | `./build/dev/build-clear.sh` -> `build/dev/clear.sh` |
| `build/deploy/scripts/uninstall-kite.sh` | 배포 제거의 실제 구현이다. | `build/deploy/scripts/uninstall-kite.sh` |

`uninstall.sh` and `build/dev/build-clear.sh` are deliberately different:

- Use `build/dev/build-clear.sh` when you are working from a repository checkout and want
  to remove the development deployment created by `./build/dev/build-install.sh`.
- Use `uninstall.sh` when the target was installed through the deployment path or when
  you want the remote `curl .../uninstall.sh | bash` cleanup flow.

The public root scripts ask all interactive questions at the beginning of the
run. If an environment variable is already set, it is treated as a fixed answer.
Set `KITE_ASSUME_DEFAULTS=true` to skip prompts and use defaults/env values in
automation.

## Directory Map

| Path | Role |
| --- | --- |
| `build/dev` | Development build, rollout, component rebuild, and development cleanup scripts. These scripts may build local images and load them into minikube, k3s, k3d, kind, or another selected cluster. |
| `build/deploy` | Pull-based deployment documentation and scripts. Install, verify, uninstall, Longhorn, KubeVirt, CDI, gateway host key, and remote cleanup scripts live under `build/deploy/scripts`. |
| `build/kite` | Kite-owned Kubernetes manifests: CRDs, namespace, RBAC, API, controller, frontend, gateway, component service accounts, runtime ConfigMap, and runtime Secret bootstrap path. |
| `build/kite-storage` | Kite storage manifests: Longhorn StorageClass, optional Longhorn disk directory setup/cleanup, and CDI golden image DataVolumes. |
| `build/examples` | Example Kite CRs for manual testing after the CRDs are installed. |
| `build/test` | Manual integration test helpers. |
| `build/lib` | Shared shell helpers used by both development and deployment scripts. This is not an executable entrypoint directory. |

## Public Entrypoints

Only three executable scripts live at the repository root. They work from a
checkout or when streamed from the corresponding raw GitHub URL:

```text
./install.sh
  -> build/deploy/scripts/install-all.sh

./update.sh
  -> build/deploy/scripts/update-ghcr.sh

./uninstall.sh
  -> build/deploy/scripts/clean.sh
  -> build/deploy/scripts/uninstall-kite.sh
```

Without a checkout, each public entrypoint downloads the selected source archive
before delegating to its implementation. Existing `KITE_GHCR_INSTALL_*` and
`KITE_GHCR_UPDATE_*` environment variables remain supported.

## Developer Entrypoints

Build tools and the maintainer's stage installer live together under `build/dev`:

```text
./build/dev/build-install.sh
  -> build/dev/all-in-one.sh

./build/dev/build-update.sh
  -> fresh source checkout
  -> build/dev/update-local.sh
  -> build/deploy/scripts/update-ghcr.sh (shared runtime update engine)

./build/dev/build-clear.sh
  -> build/dev/clear.sh

./build/dev/stage-install.sh
  -> sets stage ref and stage image tag
  -> ../../install.sh (or the selected ref's remote install.sh)
```

New deployment behavior belongs in `build/deploy/scripts`, development behavior
belongs in `build/dev`, and shared shell helpers belong in `build/lib`.
The stage installer is a maintainer QA wrapper around the public installer.

## Development Flow

Use the development path when you want to build local images from this checkout:

```sh
KITE_CLUSTER=k3s ./build/dev/build-install.sh
```

Use `./build/dev/build-clear.sh` to remove the development deployment:

```sh
KITE_CLUSTER=k3s ./build/dev/build-clear.sh
```

`./build/dev/build-clear.sh` asks for numbered choices in an interactive terminal. Longhorn
uninstall and Longhorn host data removal stay disabled by default because they
can remove VM disk infrastructure. Automation can still provide the explicit
environment variables documented in `build/dev/README.md`.

## Deployment Flow

Use the deployment path when the target should pull already-published GHCR
images:

```sh
./install.sh
```

Without a git checkout, pipe the root bootstrapper:

```sh
curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/main/install.sh | bash
```

Use `./uninstall.sh` for the matching deployment cleanup:

```sh
./uninstall.sh
```

Without a git checkout:

```sh
curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/main/uninstall.sh | bash
```

The deployment cleanup path removes Kite resources by default. Golden image,
Longhorn uninstall, and Longhorn host data removal are controlled by explicit
prompts or environment variables documented in `build/deploy/README.md`.

## Manifests

`build/kite` is the source of Kite runtime manifests. Development and deployment
flows both apply these manifests, but they patch image references differently:

- Development flow points workloads at locally built images.
- Deployment flow points workloads at `ghcr.io/hy3ons` images.

`build/kite-storage` contains storage resources that are not Kite application
workloads but are required for VM disks and golden image cloning.

## Maintenance Rules

- Keep public deployment entrypoints at the root; keep maintainer-only wrappers,
  including the stage installer, under `build/dev`.
- Do not add development build or component scripts under `build/deploy`.
- Do not add executable entrypoints under `build/lib`; keep it for shared
  sourced helpers only.
- Keep root wrappers small and make them delegate into the correct directory.
- Update this file when a new build/deploy directory or root wrapper is added.
