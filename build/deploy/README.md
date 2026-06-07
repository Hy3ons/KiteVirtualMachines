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

Install everything. Longhorn installation is opt-in because production nodes
must satisfy Longhorn prerequisites such as usable disks and required host
packages.

```sh
INSTALL_LONGHORN=true ./install.sh
```

If Longhorn is already installed:

```sh
./install.sh
```

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

Set `DELETE_GOLDEN_IMAGE=true` to explicitly delete the imported golden image
DataVolume and PVC before removing the namespace.

Longhorn removal is opt-in because it deletes VM disk infrastructure:

```sh
DELETE_LONGHORN=true build/deploy/scripts/uninstall-kite.sh
```

`DELETE_LONGHORN=true` uninstalls Longhorn only when no Longhorn PV remains. If
another workload is still using Longhorn, the script skips Longhorn uninstall.

To remove Kite-owned host data under `/mnt/kite-longhorn`, use the extra
confirmation flag. This can be used without uninstalling Longhorn, but the
script skips data deletion while Longhorn PVs still exist.

```sh
DELETE_LONGHORN_DATA=true DELETE_LONGHORN_DATA_CONFIRM=true build/deploy/scripts/uninstall-kite.sh
```

## Gateway

Kite installs `kite-gateway` as a Kubernetes Deployment and exposes it with
the `kite-gateway` LoadBalancer Service. External SSH uses port `22` and is
forwarded to the pod's internal `2222` port.

```sh
ssh <sshId>@<node-ip>
```

The host OS does not need Kite-managed Linux users for this path. The gateway
authenticates from Kite VM state, reads the VM SSH key Secret, and proxies the
SSH session to the VM access Service inside the cluster.

`install.sh` creates `kite-gateway-host-key` automatically when it does not
exist. That Secret stores the SSH server host key used by external clients, so
gateway pod restarts do not change the host fingerprint.

When the host is Linux with systemd OpenSSH, `install.sh` asks before moving
host sshd away from port `22`. If confirmed, it backs up
`/etc/ssh/sshd_config` under `/etc/kite/host-sshd`, configures host sshd to
listen on `2222`, and restarts the service so the gateway can own port `22`.
`build/deploy/scripts/uninstall-kite.sh` asks before restoring that backup. Set
`KITE_MANAGE_HOST_SSHD=true` or `KITE_RESTORE_HOST_SSHD=true` for
non-interactive opt-in, and set `MANAGE_HOST_SSHD=false` or
`RESTORE_HOST_SSHD=false` to skip these host changes.
