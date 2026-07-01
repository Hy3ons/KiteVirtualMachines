#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: clear.sh
# Description: 루트에서 개발용 정리 스크립트를 실행하기 위한 wrapper다.
#
# Usage:
#   ./clear.sh
#
# Environment Variables:
#   없음: 이 wrapper는 인자와 하위 스크립트의 환경변수를 그대로 전달한다.
#
# Side Effects:
#   Kubernetes 리소스, 이미지 캐시, 선택적 Longhorn/host sshd 상태를 변경하거나 삭제할 수 있다.
# ==============================================================================

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

exec "${ROOT_DIR}/build/dev/clear.sh" "$@"
