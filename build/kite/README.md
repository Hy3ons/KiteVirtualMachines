# Kite Manifests

`kustomization.yaml` applies Kite CRDs, namespace-scoped runtime resources,
RBAC, API/controller/frontend workloads, the default HTTP platform Ingress, and
the `kite-gateway` SSH entrypoint.

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

`platform-ingress.yaml` opens the default HTTP web entrypoint after install:

```text
http://<node-or-load-balancer>/
  /api -> service/kite-api:8080
  /    -> service/kite-frontend:80
```

This Ingress is hostless by default so a fresh k3s/Traefik install is reachable
on port 80 without first setting a domain. When an admin later sets a base
domain or TLS policy, `kite-controller` reconciles the same `kite-platform`
Ingress to the configured host/TLS shape.

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
Install scripts keep this Service internal. The controller creates a separate
`kite-gateway-external` `LoadBalancer` Service only after a Level 3 admin enables
SSH Gateway exposure in Admin Settings.

```text
user-facing SSH port
  -> service/kite-gateway-external when enabled
  -> service/kite-gateway port 22 inside the cluster
  -> deployment/kite-gateway container port 2222
  -> vps-access-<vmName>.<namespace>.svc.cluster.local:22
```

Admin Settings stores the Kubernetes Service port separately from the port shown
to users. This supports router/NAT layouts such as public `22 -> 12311`, where
`12311` is the LoadBalancer/Service port and `22` is the command shown in the
Dashboard.

The gateway reads `KiteVirtualMachine` CRDs, VM SSH key Secrets, and VM access
Services through Kubernetes RBAC. It does not create host Linux users.

`build/kite` expects the optional `kite-gateway-host-key` Secret when stable SSH
host fingerprints are required. `./build-install.sh` and `./ghcr-install.sh`
create this Secret automatically when it is missing. Manual
`kubectl apply -k build/kite` still starts the gateway with an ephemeral host key
if the Secret does not exist.
