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

Host account cleanup is enabled by default. Before deleting the Kite manifests,
the script removes Kite-managed Linux users and `/var/lib/kite/accounts/*.json`
metadata by using the metadata files as the ownership source. Set
`DELETE_HOST_ACCOUNTS=false` to skip this host account cleanup.

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
