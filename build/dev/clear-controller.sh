#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/dev/clear-controller.sh
# Description: controller 리소스와 이미지 캐시만 정리하도록 clear-component.sh에 위임한다.
#
# Usage:
#   build/dev/clear-controller.sh
#
# Environment Variables:
#   없음: 이 wrapper는 인자와 하위 스크립트의 환경변수를 그대로 전달한다.
#
# Side Effects:
#   Kubernetes 리소스, 이미지 캐시, 선택적 Longhorn/host sshd 상태를 변경하거나 삭제할 수 있다.
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# controller 리소스와 이미지만 지우도록 공용 clear-component.sh에 컴포넌트 이름을 넘긴다.
exec "${SCRIPT_DIR}/clear-component.sh" controller "$@"
