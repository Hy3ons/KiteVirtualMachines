# Kite Development Install

`dev.sh` builds the Kite API, controller, host-agent, and frontend images with
local Docker, then applies the shared `build/kite` manifests to the selected
cluster.

```sh
KITE_CLUSTER=k3s build/dev/dev.sh
```

For k3s, the script imports built images into the k3s containerd namespace by
default. Override the import command when sudo is not needed:

```sh
K3S_CTR_CMD="k3s ctr -n k8s.io" KITE_CLUSTER=k3s build/dev/dev.sh
```

Supported cluster modes are `minikube`, `k3s`, `k3d`, `kind`, `k8s`,
`kubernetes`, and `current`. `k3d` uses `k3d image import`, and `kind` uses
`kind load docker-image`. Generic Kubernetes modes build local images and can
push them when the cluster pulls from a registry:

```sh
PUSH_IMAGES=true IMAGE_REGISTRY=registry.example.com/kite KITE_CLUSTER=k8s build/dev/dev.sh
```

`clear.sh` removes Kite development resources and local Kite images.

```sh
KITE_CLUSTER=k3s build/dev/clear.sh
```

Longhorn cleanup is disabled by default because it deletes VM disk data.

```sh
CLEAR_LONGHORN=true KITE_CLUSTER=k3s build/dev/clear.sh
```

`CLEAR_LONGHORN=true` uninstalls Longhorn only when no Longhorn PV remains. If
another workload is still using Longhorn, the script skips Longhorn uninstall.

To remove Kite-owned host data under `/mnt/kite-longhorn`, use the extra
confirmation flag. This can be used without uninstalling Longhorn, but the
script skips data deletion while Longhorn PVs still exist.

```sh
CLEAR_LONGHORN_DATA=true CLEAR_LONGHORN_DATA_CONFIRM=true KITE_CLUSTER=k3s build/dev/clear.sh
```
