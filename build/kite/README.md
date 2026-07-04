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

Runtime secrets are not stored in this ConfigMap. `kite-api` bootstraps
`kite/kite-runtime-secret` for `jwtSecret` and `passwordSalt`, and migrates
legacy ConfigMap secret values into that Secret on startup.

## RBAC

`build/kite` uses separate service accounts for each runtime component:

- `kite-api`: creates and updates Kite CRDs, runtime config/secret, guest login
  Secrets, and KubeVirt console subresources.
- `kite-controller`: reconciles namespaces, quotas, network policies,
  DataVolumes, KubeVirt VMs, Services, Ingresses, and VM-owned Secrets.
- `kite-gateway`: reads runtime config, KiteVirtualMachine routes, VM SSH key
  Secrets, and VM access Services. It does not create or delete cluster
  resources.

## SSH Gateway

`gateway.yaml` deploys `kite-gateway` behind an internal `ClusterIP` Service.
Install scripts promote that Service to `LoadBalancer` port `22` after host sshd
handoff succeeds. This keeps raw manifest apply from taking host SSH by itself.

```text
handoff-enabled external SSH :22
  -> service/kite-gateway port 22 when exposed
  -> deployment/kite-gateway container port 2222
  -> vps-access-<vmName>.<namespace>.svc.cluster.local:22
```

The gateway reads `KiteVirtualMachine` CRDs, VM SSH key Secrets, and VM access
Services through Kubernetes RBAC. It does not create host Linux users.

`build/kite` expects the optional `kite-gateway-host-key` Secret when stable SSH
host fingerprints are required. `./build-install.sh` and `./ghcr-install.sh` create this Secret
automatically when it is missing. Manual `kubectl apply -k build/kite` still
starts the gateway with an ephemeral host key if the Secret does not exist.
