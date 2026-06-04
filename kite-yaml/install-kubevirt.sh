#!/bin/bash

# 에러 발생 시 스크립트 중단
set -e

echo "========================================="
echo "  KubeVirt 설치를 시작합니다. "
echo "========================================="

# 1. 최신 안정화 버전 가져오기
echo "[1/4] 최신 KubeVirt 버전 확인 중..."
export KUBEVIRT_VERSION=$(curl -s https://api.github.com/repos/kubevirt/kubevirt/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$KUBEVIRT_VERSION" ]; then
    echo "❌ KubeVirt 버전을 가져오는 데 실패했습니다. 네트워크 연결을 확인하세요."
    exit 1
fi
echo "▶ 배포할 KubeVirt 버전: $KUBEVIRT_VERSION"

# 2. KubeVirt Operator 배포
echo "[2/4] KubeVirt Operator 배포 중..."
kubectl create -f "https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-operator.yaml"

# 3. KubeVirt Custom Resource(CR) 배포 (실제 컴포넌트 생성)
echo "[3/4] KubeVirt Custom Resource 배포 중..."
kubectl create -f "https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-cr.yaml"

# 4. 설치 완료 및 상태 확인 안내
echo "[4/4] 배포 명령 완료! 컴포넌트가 활성화될 때까지 기다립니다..."
echo "-----------------------------------------"
echo "💡 설치 상태를 확인하려면 다음 명령어를 실행하세요:"
echo "   kubectl get kubevirt -n kubevirt"
echo "   kubectl get pods -n kubevirt"
echo "========================================="