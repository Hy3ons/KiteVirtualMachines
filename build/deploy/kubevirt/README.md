# KubeVirt

KubeVirt provides the VM runtime used by KiteVirtualMachine reconciliation.

```sh
build/deploy/scripts/install-kubevirt.sh
build/deploy/scripts/wait-kubevirt.sh
```

Override the installed version with:

```sh
KUBEVIRT_VERSION=<version> build/deploy/scripts/install-kubevirt.sh
```
