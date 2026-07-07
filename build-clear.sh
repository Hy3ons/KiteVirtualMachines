#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build-clear.sh
# Purpose:
#   개발자가 build-install.sh 또는 component build로 만든 개발 배포 산출물과
#   선택적 image/storage 상태를 정리하는 공개 진입점이다.
#
# Usage:
#   ./build-clear.sh
#
# Required Commands:
#   kubectl. 대상 cluster와 image cleanup에 따라 docker, k3s, minikube, k3d, kind가 필요할 수 있다.
#
# Environment Variables:
#   KITE_CLUSTER: default auto; 대상 cluster 종류다.
#   KITE_NAMESPACE: default kite; 삭제할 Kite namespace다.
#   CLEAR_IMAGES: default true; local/k3s 개발 image 삭제 여부를 초반에 묻는다.
#   MINIKUBE_PURGE: default false; minikube 전체 purge 여부를 초반에 묻는다.
#   CLEAR_LONGHORN: default false; Longhorn 설치 자체 제거 여부를 초반에 묻는다.
#   CLEAR_LONGHORN_DATA: default false; Kite Longhorn host data 삭제 여부를 초반에 묻는다.
#   CLEAR_LONGHORN_FORCE: default false; Longhorn PV가 남아도 강제 삭제할지 초반에 묻는다.
#   KITE_ASSUME_DEFAULTS: default false; true면 모든 interactive 질문을 건너뛰고 env/default 값으로 실행한다.
#
# Interactive Behavior:
#   TTY에서 실행하고 env가 없는 항목은 삭제 초반에 모두 질문한다. 삭제 중에는 같은
#   항목을 다시 묻지 않는다.
#
# Noninteractive Behavior:
#   env가 있으면 그 값을 그대로 쓰고 질문하지 않는다. env가 없으면 위 기본값으로 진행한다.
#
# Dangerous Options:
#   CLEAR_LONGHORN_DATA, CLEAR_LONGHORN, CLEAR_LONGHORN_FORCE는 VM disk infrastructure를
#   삭제할 수 있다. CLEAR_LONGHORN_FORCE 기본값은 false다.
#
# Side Effects:
#   Kubernetes 리소스, 이미지 캐시, 선택적 Longhorn 상태를 변경하거나 삭제할 수 있다.
# ==============================================================================

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

exec "${ROOT_DIR}/build/dev/clear.sh" "$@"
