#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build-install.sh
# Purpose:
#   개발자가 현재 checkout의 소스로 API/controller/gateway/frontend 이미지를 빌드하고
#   선택한 클러스터에 설치하는 공개 진입점이다.
#
# Usage:
#   ./build-install.sh
#
# Required Commands:
#   kubectl, docker. 대상 cluster에 따라 k3s, minikube, k3d, kind, registry push 권한이 필요할 수 있다.
#
# Environment Variables:
#   KITE_CLUSTER: default auto; 대상 cluster 종류다. 초반 질문 없이 env/default로 결정한다.
#   INSTALL_LONGHORN: default true; Longhorn 설치 여부를 초반에 묻는다.
#   CONFIGURE_LONGHORN: default true; Longhorn disk/tag 설정 여부를 초반에 묻는다.
#   KITE_LONGHORN_USE_DEDICATED_DISK: default false; 전용 Longhorn host path disk 생성 여부를 초반에 묻는다.
#   APPLY_STORAGECLASS: default true; Kite VM StorageClass 적용 여부를 초반에 묻는다.
#   INSTALL_KUBEVIRT: default true; KubeVirt 설치 여부를 초반에 묻는다.
#   INSTALL_CDI: default true; CDI 설치 여부를 초반에 묻는다.
#   APPLY_GOLDEN_IMAGE: default true; Ubuntu golden image 적용 여부를 초반에 묻는다.
#   DEPLOY_KITE: default true; Kite workload build/deploy 여부를 초반에 묻는다.
#   FRONTEND_VITE_USE_MOCK: default false; frontend mock API build 여부를 초반에 묻는다.
#   K3S_IMPORT_IMAGES, K3D_LOAD_IMAGES, KIND_LOAD_IMAGES, MINIKUBE_START, PUSH_IMAGES: cluster별 image 전달 여부를 초반에 묻는다.
#   KITE_GATEWAY_HOST_KEY_REFRESH: default false; 기존 gateway host key Secret 갱신 여부를 초반에 묻는다.
#   MANAGE_HOST_SSHD: default false; gateway 22번 사용을 위한 host sshd handoff 여부를 초반에 묻는다.
#   KITE_HOST_SSHD_PORT: default 2222; host sshd handoff 대상 포트다.
#   RUN_VERIFY: default true; 설치 후 verify 실행 여부를 초반에 묻는다.
#   KITE_ASSUME_DEFAULTS: default false; true면 모든 interactive 질문을 건너뛰고 env/default 값으로 실행한다.
#
# Interactive Behavior:
#   TTY에서 실행하고 env가 없는 항목은 설치 초반에 모두 질문한다. 하위 build/dev/dev.sh
#   실행 중에는 image load/push/mock 관련 질문을 다시 하지 않도록 env를 export한다.
#
# Noninteractive Behavior:
#   env가 있으면 그 값을 그대로 쓰고 질문하지 않는다. env가 없으면 위 기본값으로 진행한다.
#
# Dangerous Options:
#   MANAGE_HOST_SSHD=true는 host sshd 포트를 바꿀 수 있다. Longhorn 관련 옵션은
#   cluster storage 상태를 바꿀 수 있다.
#
# Side Effects:
#   Kubernetes 리소스 적용, 컨테이너 이미지 빌드/주입, rollout 대기를 수행할 수 있다.
# ==============================================================================

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${ROOT_DIR}/build/dev/all-in-one.sh" "$@"
