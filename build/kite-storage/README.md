# Kite Storage

This directory contains storage resources owned by Kite.

- `longhorn/storageclass.yaml` defines the Longhorn-backed `kite-vm-storage` StorageClass.
- `longhorn/disk-directory-daemonset.yaml` temporarily creates `/mnt/kite-longhorn` on each node when `KITE_LONGHORN_USE_DEDICATED_DISK=true`.
- `longhorn-cleanup/disk-cleanup-daemonset.yaml` temporarily removes `/mnt/kite-longhorn` during explicit cleanup.
- `golden-images/ubuntu-22.04.yaml` imports the Ubuntu golden image through CDI.

Development cleanup and production uninstall scripts delete these manifests
before removing broader Longhorn infrastructure.

By default, install scripts do not register `/mnt/kite-longhorn` as a second
Longhorn disk. Local single-disk nodes often expose `/mnt/kite-longhorn` and
`/var/lib/longhorn` from the same filesystem, and Longhorn rejects duplicate
filesystem IDs. The default install therefore adds only the `kite` tag to
existing Ready Longhorn disks so `kite-vm-storage` can satisfy
`diskSelector: kite`.

Use `KITE_LONGHORN_USE_DEDICATED_DISK=true` only when
`KITE_LONGHORN_DISK_PATH` is a real separate mount or disk.

Cleanup and uninstall scripts remove the Kite-owned dedicated disk entry and
remove only the `kite` tag from existing Longhorn disks. They do not remove
Longhorn itself unless the explicit Longhorn cleanup flags are set.
