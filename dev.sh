#!/usr/bin/env bash
set -euo pipefail

# dev.sh is the local-build all-in-one installer.
# It prepares the virtualization/storage stack, builds Kite images from this checkout,
# imports or loads those images into the selected cluster, and deploys Kite workloads.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${ROOT_DIR}/build/dev/all-in-one.sh" "$@"
