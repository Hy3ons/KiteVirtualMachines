#!/usr/bin/env bash
set -euo pipefail

# Exercise installer rendering without a cluster. A narrow kubectl fake supplies
# node architectures and delegates only local Kustomize rendering to real kubectl.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
REAL_KUBECTL="$(command -v kubectl)"
fixture="$(mktemp -d "${TMPDIR:-/tmp}/kite-golden-test.XXXXXX")"
trap 'rm -rf "${fixture}"' EXIT
export REAL_KUBECTL
export GOLDEN_RENDER="${fixture}/rendered.yaml"
mkdir "${fixture}/bin"
cat > "${fixture}/bin/kubectl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "$1" in
  get) printf '%s\n' "${FAKE_NODE_ARCH}" ;;
  apply) "${REAL_KUBECTL}" kustomize "$3" > "${GOLDEN_RENDER}" ;;
  *) exit 1 ;;
esac
EOF
chmod +x "${fixture}/bin/kubectl"
export PATH="${fixture}/bin:${PATH}"
unset KITE_VM_ARCH

for arch in amd64 arm64; do
  FAKE_NODE_ARCH="${arch}" KITE_NAMESPACE=kite-qa "${ROOT_DIR}/build/deploy/scripts/apply-golden-image.sh"
  rg -q "jammy-server-cloudimg-${arch}.img" "${GOLDEN_RENDER}"
  rg -q 'namespace: kite-qa' "${GOLDEN_RENDER}"
  rg -q 'storageClassName: kite-vm-storage' "${GOLDEN_RENDER}"
done

for arch in ppc64le "$(printf 'amd64\narm64')" ''; do
  rm -f "${GOLDEN_RENDER}"
  if FAKE_NODE_ARCH="${arch}" "${ROOT_DIR}/build/deploy/scripts/apply-golden-image.sh"; then
    echo 'unexpected acceptance of unsupported or ambiguous architecture' >&2
    exit 1
  fi
  test ! -e "${GOLDEN_RENDER}"
done

FAKE_NODE_ARCH=ppc64le KITE_VM_ARCH=arm64 "${ROOT_DIR}/build/deploy/scripts/apply-golden-image.sh"
rg -q 'jammy-server-cloudimg-arm64.img' "${GOLDEN_RENDER}"
echo 'PASS: golden image architecture, namespace, storage and rejection cases'
