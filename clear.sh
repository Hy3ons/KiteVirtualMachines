#!/usr/bin/env bash
set -euo pipefail

# clear.sh is the root cleanup entrypoint for local Kite development installs.
# It delegates to build/dev/clear.sh so the same cleanup logic is shared with
# lower-level development workflows.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

exec "${ROOT_DIR}/build/dev/clear.sh" "$@"
