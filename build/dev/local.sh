#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PROFILE="kite-dev"
# kite-api currently reads this exact file, rather than the KUBECONFIG variable.
LOCAL_KUBECONFIG="${HOME}/.kube/config"
# OrbStack's older kubectl may precede Homebrew on Apple Silicon macOS.
KUBECTL_BIN="${KITE_KUBECTL_BIN:-kubectl}"
if [[ -z "${KITE_KUBECTL_BIN:-}" && -x /opt/homebrew/bin/kubectl ]]; then
  KUBECTL_BIN=/opt/homebrew/bin/kubectl
fi

# local_kubectl targets only the dedicated development context in the host config.
# Arguments are kubectl subcommands used by setup, status, and namespace preparation.
local_kubectl() {
  "$KUBECTL_BIN" --kubeconfig "$LOCAL_KUBECONFIG" --context "$PROFILE" "$@"
}

# require_local_context prevents the host Go process from opening another cluster.
# It has no arguments and exits before API startup if the current context differs.
require_local_context() {
  local current_context
  current_context="$("$KUBECTL_BIN" --kubeconfig "$LOCAL_KUBECONFIG" config current-context)"
  if [[ "$current_context" != "$PROFILE" ]]; then
    echo "kite-api reads ~/.kube/config; select kite-dev before starting it:" >&2
    echo "kubectl --kubeconfig \"$LOCAL_KUBECONFIG\" config use-context $PROFILE" >&2
    exit 1
  fi
  local_kubectl get --raw=/readyz >/dev/null
}

case "${1:-help}" in
  setup)
    KUBECONFIG="$LOCAL_KUBECONFIG" minikube start -p "$PROFILE" \
      --driver=docker --container-runtime=containerd --cpus=4 --memory=4096
    local_kubectl apply -f "$ROOT_DIR/build/kite/crds.yaml"
    local_kubectl wait --for=condition=Established --timeout=90s \
      crd/kiteusers.hy3ons.github.io \
      crd/kitevirtualmachines.hy3ons.github.io \
      crd/kitevirtualmachineoffers.hy3ons.github.io
    echo "Local cluster ready. Start Go with: bash build/dev/local.sh api"
    ;;
  api)
    require_local_context
    cd "$ROOT_DIR/kite"
    exec go run ./cmd/kite-api
    ;;
  namespace)
    # No controller is installed in this lightweight environment. Prepare the
    # namespace for an existing signup before testing VM CRD writes through Go.
    resource_name="${2:?Usage: bash build/dev/local.sh namespace KITEUSER_NAME}"
    namespace="$(local_kubectl get kiteuser "$resource_name" -o jsonpath='{.spec.namespace}')"
    if [[ -z "$namespace" ]]; then
      echo "KiteUser has no spec.namespace: $resource_name" >&2
      exit 1
    fi
    local_kubectl create namespace "$namespace" --dry-run=client -o yaml | local_kubectl apply -f -
    ;;
  status)
    KUBECONFIG="$LOCAL_KUBECONFIG" minikube status -p "$PROFILE"
    local_kubectl get nodes
    local_kubectl get kiteusers
    ;;
  stop)
    KUBECONFIG="$LOCAL_KUBECONFIG" minikube stop -p "$PROFILE"
    ;;
  *)
    echo "Usage: bash build/dev/local.sh {setup|api|namespace KITEUSER_NAME|status|stop}"
    ;;
esac
