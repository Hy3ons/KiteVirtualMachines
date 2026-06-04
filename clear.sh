#!/usr/bin/env bash
set -euo pipefail

echo "[kite] deleting every minikube cluster and purging local minikube state"
minikube delete --all --purge
echo "[kite] minikube state cleared"
