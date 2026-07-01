#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: dev.sh
# Description: 루트에서 개발용 all-in-one 설치를 실행하기 위한 wrapper다.
#
# Usage:
#   ./dev.sh
#
# Environment Variables:
#   없음: 이 wrapper는 인자와 하위 스크립트의 환경변수를 그대로 전달한다.
#
# Side Effects:
#   Kubernetes 리소스 적용, 컨테이너 이미지 빌드/주입, rollout 대기를 수행할 수 있다.
# ==============================================================================

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${ROOT_DIR}/build/dev/all-in-one.sh" "$@"
