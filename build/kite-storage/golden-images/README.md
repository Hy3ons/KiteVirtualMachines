# Golden Images

Golden image DataVolumes seed VM disks through CDI.

`ubuntu-22.04.yaml` imports the Ubuntu cloud image into the `kite` namespace
using the Longhorn-backed `kite-vm-storage` StorageClass. VM-owned DataVolumes
clone from this PVC and should use the same StorageClass.

```sh
kubectl apply -f build/kite-storage/golden-images
build/deploy/scripts/wait-golden-image.sh ubuntu-22.04
```
