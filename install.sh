#!/usr/bin/env bash
set -euo pipefail

# install.sh is the pull-based installer.
# It installs the virtualization/storage stack and applies Kite manifests that pull
# prebuilt images from GHCR, without building local Docker images.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${ROOT_DIR}/build/deploy/scripts/install-all.sh" "$@"
