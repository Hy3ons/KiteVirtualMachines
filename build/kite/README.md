# Kite Manifests

`kustomization.yaml` applies Kite CRDs, namespace-scoped runtime resources,
RBAC, and workloads.

```sh
kubectl apply -k build/kite
```

The runtime ConfigMap sets VM disk storage defaults:

```text
vmStorageClassName=kite-vm-storage
goldenImageNamespace=kite
defaultVmImage=ubuntu-22.04
```

The controller uses `vmStorageClassName` when rendering VM-owned DataVolumes.
