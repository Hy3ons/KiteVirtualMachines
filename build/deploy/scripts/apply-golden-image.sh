#!/usr/bin/env bash
set -euo pipefail

# Apply the Ubuntu golden DataVolume matching the cluster's CPU architecture.
# KITE_VM_ARCH can select amd64 or arm64 explicitly; automatic detection requires
# a homogeneous cluster. Both source-build and remote installers use this helper.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
KITE_VM_ARCH="${KITE_VM_ARCH:-$(kubectl get nodes -o jsonpath='{range .items[*]}{.status.nodeInfo.architecture}{"\n"}{end}' | sort -u)}"

case "${KITE_VM_ARCH}" in
  amd64|arm64) ;;
  *)
    echo "[kite] golden image requires an amd64 or arm64 cluster; set KITE_VM_ARCH explicitly for mixed clusters" >&2
    exit 1
    ;;
esac

overlay="$(mktemp -d "${TMPDIR:-/tmp}/kite-golden-image.XXXXXX")"
trap 'rm -rf "${overlay}"' EXIT
cp "${ROOT_DIR}/build/kite-storage/golden-images/ubuntu-22.04.yaml" "${overlay}/ubuntu.yaml"
cat > "${overlay}/kustomization.yaml" <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: ${KITE_NAMESPACE}
resources:
  - ubuntu.yaml
patches:
  - target:
      kind: DataVolume
      name: ubuntu-22.04
    patch: |-
      - op: replace
        path: /spec/source/http/url
        value: https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-${KITE_VM_ARCH}.img
EOF
echo "[kite] applying Ubuntu golden image for ${KITE_VM_ARCH}"
kubectl apply -k "${overlay}"
