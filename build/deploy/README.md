# Kite Production Install

This directory contains the production-oriented k3s install flow for Kite.

Kite VM disks use Longhorn through the `kite-vm-storage` StorageClass. CDI is
still required because Kite uses DataVolumes to import the Ubuntu golden image
and clone VM-owned disks from that source PVC. Kite application manifests are
shared with development installs through `build/kite`; Kite-owned storage
manifests live in `build/kite-storage`.

## Install

Prepare a k3s cluster and confirm `kubectl` can reach it.

```sh
kubectl get nodes
```

Install without git or a repository clone:

```sh
curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/main/ghcr-install.sh | bash
```

Maintainer stage QA uses the same install flow with the stage wrapper:

```sh
curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/stage/ghcr-stage-install.sh | bash
```

`ghcr-stage-install.sh` sets the repository ref and image tag to `stage` before
calling `ghcr-install.sh`. This keeps the installation process identical to
production while avoiding maintainer mistakes from memorized environment
variables.

By default the installer applies Longhorn, KubeVirt, CDI, Kite storage, golden
image, and Kite runtime manifests. On apt-based Linux hosts it also installs
missing Longhorn host packages such as `open-iscsi` and `nfs-common`.

If Longhorn is managed outside Kite and should not be applied by this installer:

```sh
INSTALL_LONGHORN=false ./ghcr-install.sh
```

`./ghcr-install.sh` asks all install choices near the start of the run. If a
variable is already set in the environment, that value is used without asking.
Set `KITE_ASSUME_DEFAULTS=true` for non-interactive automation that should use
the documented defaults.

## Verify

```sh
build/deploy/scripts/verify.sh
```

Expected storage flow:

```text
kite/ubuntu-22.04 DataVolume
  -> PVC using StorageClass kite-vm-storage

user namespace VM DataVolume
  -> clone source pvc kite/ubuntu-22.04
  -> PVC using StorageClass kite-vm-storage
```

## Uninstall Kite

This removes Kite resources only. It does not uninstall Longhorn, KubeVirt, or
CDI.

```sh
build/deploy/scripts/uninstall-kite.sh
```

`./uninstall.sh` is the public root wrapper for the same deploy cleanup
path. Inside this directory, `build/deploy/scripts/clean.sh` is the bootstrap
entrypoint and `build/deploy/scripts/uninstall-kite.sh` is the implementation.
`KITE_UNINSTALL_PRESET=safe` keeps dangerous deletion off by default.
`KITE_UNINSTALL_PRESET=full` enables golden image, Kite Longhorn host data, and
Longhorn uninstall choices, while shared infrastructure protection still applies.
`DELETE_LONGHORN_FORCE` never overrides non-Kite Longhorn PVC/PV protection.

Run the same cleanup without git or a repository clone:

```sh
curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/main/uninstall.sh | bash
```

Use a specific branch or tag:

```sh
curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/main/uninstall.sh \
  | KITE_UNINSTALL_REF=stage bash
```

Set `DELETE_GOLDEN_IMAGE=true` to explicitly delete the imported golden image
DataVolume and PVC before removing the namespace.

Longhorn removal is opt-in because it deletes VM disk infrastructure:

```sh
DELETE_LONGHORN=true build/deploy/scripts/uninstall-kite.sh
```

`DELETE_LONGHORN=true` uninstalls Longhorn only when `longhorn-system` is marked
as Kite-installed and no non-Kite Longhorn PV remains. If Longhorn existed before
Kite, or another workload is still using Longhorn, the script skips Longhorn
uninstall.

To remove Kite-owned host data under `/mnt/kite-longhorn`, use the extra
confirmation flag. This can be used without uninstalling Longhorn, but the
script skips data deletion while non-Kite Longhorn PVs still exist.

```sh
DELETE_LONGHORN_DATA=true DELETE_LONGHORN_DATA_CONFIRM=true build/deploy/scripts/uninstall-kite.sh
```

## Gateway

Kite installs `kite-gateway` as a Kubernetes Deployment and keeps the default
Service internal. The installer does not move, restore, or rewrite host sshd.
External VM SSH access is enabled later from Admin Settings, which updates
`kite-runtime-config` and lets the controller create `service/kite-gateway-external`.
Admin Settings stores the Gateway Service port separately from the user-facing
port shown in Dashboard/VM Detail, so an external router can map public `22` to
a custom Service port without confusing users.

```sh
ssh -p <user-facing-port> <sshId>@<node-ip>
```

The host OS does not need Kite-managed Linux users for this path. The gateway
authenticates from Kite VM state, reads the VM SSH key Secret, and proxies the
SSH session to the VM access Service inside the cluster.

`./ghcr-install.sh` creates `kite-gateway-host-key` automatically when it does not
exist. On Linux hosts it first tries to copy the existing OpenSSH host key from
`/etc/ssh/ssh_host_ed25519_key`, `ssh_host_ecdsa_key`, or `ssh_host_rsa_key`.
That can keep the gateway fingerprint familiar if the operator later exposes the
gateway on a public SSH port. If no host key is available, or automatic mode
cannot read it, the script generates a gateway key instead.

Existing Secrets are kept by default so client fingerprints do not change on
every deploy. To intentionally replace an already-created gateway key from the
host key, run:

```sh
KITE_GATEWAY_HOST_KEY_REFRESH=true KITE_GATEWAY_HOST_KEY_SOURCE=host ./ghcr-install.sh
```

When no Kite VM uses the SSH login username, authentication fails. Kite does
not proxy host Linux accounts and never manages the host sshd port.
