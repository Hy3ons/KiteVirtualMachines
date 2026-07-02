# Kite Manifests

`kustomization.yaml` applies Kite CRDs, namespace-scoped runtime resources,
RBAC, API/controller/frontend workloads, and the `kite-gateway` SSH entrypoint.

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

## SSH Gateway

`gateway.yaml` deploys `kite-gateway` and exposes it through the
`kite-gateway` LoadBalancer Service.

```text
external SSH :22
  -> service/kite-gateway port 22
  -> deployment/kite-gateway container port 2222
  -> vps-access-<vmName>.<namespace>.svc.cluster.local:22
```

The gateway reads `KiteVirtualMachine` CRDs, VM SSH key Secrets, and VM access
Services through Kubernetes RBAC. It does not create host Linux users.

`build/kite` expects the optional `kite-gateway-host-key` Secret when stable SSH
host fingerprints are required. `./build-install.sh` and `./ghcr-install.sh` create this Secret
automatically when it is missing. Manual `kubectl apply -k build/kite` still
starts the gateway with an ephemeral host key if the Secret does not exist.
