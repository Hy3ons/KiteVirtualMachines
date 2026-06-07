# Longhorn

Longhorn is the storage backend for Kite VM disks.

Install Longhorn before applying `kite-vm-storage`. Single-node k3s clusters
should keep `numberOfReplicas: "1"` in `storageclass.yaml`. Multi-node clusters
can raise the replica count after confirming node disk capacity.

Kite uses a dedicated Longhorn disk tag. The install flow creates
`/mnt/kite-longhorn` on every node, registers that path as a Longhorn disk named
`kite-longhorn`, and tags it with `kite`. The `kite-vm-storage` StorageClass then
uses `diskSelector: "kite"` so Kite VM disks stay on Kite-owned Longhorn disks.

```sh
build/deploy/scripts/install-longhorn.sh
build/deploy/scripts/wait-longhorn.sh
build/deploy/scripts/configure-longhorn-kite-disk.sh
kubectl apply -f build/kite-storage/longhorn/storageclass.yaml
```

Set `LONGHORN_VERSION` or `LONGHORN_MANIFEST_URL` to pin an exact production
install artifact.
