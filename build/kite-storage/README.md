# Kite Storage

This directory contains storage resources owned by Kite.

- `longhorn/storageclass.yaml` defines the Longhorn-backed `kite-vm-storage` StorageClass.
- `longhorn/disk-directory-daemonset.yaml` creates `/mnt/kite-longhorn` on each node.
- `longhorn-cleanup/disk-cleanup-daemonset.yaml` removes `/mnt/kite-longhorn` during explicit cleanup.
- `golden-images/ubuntu-22.04.yaml` imports the Ubuntu golden image through CDI.

Development cleanup and production uninstall scripts delete these manifests
before removing broader Longhorn infrastructure.
