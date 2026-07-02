# Kite Build Directory

This directory is the operational surface for installing, developing, testing,
and removing Kite. The layout is intentionally split by purpose so `clear`,
`clean`, development scripts, and deployment scripts do not point at surprising
places.

## Naming Contract

| Name | Meaning | Primary entrypoint |
| --- | --- | --- |
| `dev` | Build Kite images from this checkout and deploy them to a selected cluster. | `./dev.sh` -> `build/dev/all-in-one.sh` |
| `clear` | Clear a local development deployment from this checkout. | `./clear.sh` -> `build/dev/clear.sh` |
| `install` | Install a pull-based deployment using GHCR images. | `./install.sh` -> `build/deploy/scripts/install-all.sh` |
| `clean` | Remove a pull-based deployment, including the no-git `curl` cleanup path. | `./clean.sh` -> `build/deploy/scripts/clean.sh` |
| `uninstall` | The actual deploy cleanup implementation. | `build/deploy/scripts/uninstall-kite.sh` |

`clear` and `clean` are deliberately different:

- Use `clear` when you are working from a repository checkout and want to remove
  the development deployment created by `./dev.sh`.
- Use `clean` when the target was installed through the deployment path or when
  you want the remote `curl .../clean.sh | bash` cleanup flow.

## Directory Map

| Path | Role |
| --- | --- |
| `build/dev` | Development build, rollout, component rebuild, and development cleanup scripts. These scripts may build local images and load them into minikube, k3s, k3d, kind, or another selected cluster. |
| `build/deploy` | Pull-based deployment documentation and scripts. All install, verify, uninstall, Longhorn, KubeVirt, CDI, host sshd, and remote cleanup scripts live under `build/deploy/scripts`. |
| `build/kite` | Kite-owned Kubernetes manifests: CRDs, namespace, RBAC, API, controller, frontend, gateway, service account, and runtime ConfigMap. |
| `build/kite-storage` | Kite storage manifests: Longhorn StorageClass, optional Longhorn disk directory setup/cleanup, and CDI golden image DataVolumes. |
| `build/examples` | Example Kite CRs for manual testing after the CRDs are installed. |
| `build/test` | Manual integration test helpers. |
| `build/lib` | Shared shell helpers used by both development and deployment scripts. This is not an executable entrypoint directory. |

## Root Wrappers

The repository root keeps small wrappers for human ergonomics and remote
bootstrap compatibility:

```text
./dev.sh
  -> build/dev/all-in-one.sh

./clear.sh
  -> build/dev/clear.sh

./install.sh
  -> build/deploy/scripts/install-all.sh
  -> when piped from curl, downloads the selected archive first

./clean.sh
  -> build/deploy/scripts/clean.sh
  -> build/deploy/scripts/uninstall-kite.sh
  -> when piped from curl, downloads the selected archive first
```

The root wrappers should stay thin. New deployment behavior belongs in
`build/deploy/scripts`, new development behavior belongs in `build/dev`, and
shared shell helpers belong in `build/lib`.

## Development Flow

Use the development path when you want to build local images from this checkout:

```sh
KITE_CLUSTER=k3s ./dev.sh
```

Use `./clear.sh` to remove the development deployment:

```sh
KITE_CLUSTER=k3s ./clear.sh
```

`./clear.sh` asks for numbered choices in an interactive terminal. Longhorn
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

Use `./clean.sh` for the matching deployment cleanup:

```sh
./clean.sh
```

Without a git checkout:

```sh
curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/main/clean.sh | bash
```

The deployment cleanup path removes Kite resources by default. Golden image,
Longhorn uninstall, Longhorn host data removal, and host sshd restoration are
controlled by explicit prompts or environment variables documented in
`build/deploy/README.md`.

## Manifests

`build/kite` is the source of Kite runtime manifests. Development and deployment
flows both apply these manifests, but they patch image references differently:

- Development flow points workloads at locally built images.
- Deployment flow points workloads at `ghcr.io/hy3ons` images.

`build/kite-storage` contains storage resources that are not Kite application
workloads but are required for VM disks and golden image cloning.

## Maintenance Rules

- Do not add deployment entrypoints under `build/dev`.
- Do not add development build or component scripts under `build/deploy`.
- Do not add executable entrypoints under `build/lib`; keep it for shared
  sourced helpers only.
- Keep root wrappers small and make them delegate into the correct directory.
- Update this file when a new build/deploy directory or root wrapper is added.
