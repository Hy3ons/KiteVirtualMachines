#!/bin/bash
set -e

echo "========================================="
echo "  KubeVirt CDI 설치를 시작합니다. "
echo "========================================="

# 1. 최신 안정화 버전 가져오기
echo "[1/3] 최신 CDI 버전 확인 중..."
export CDI_VERSION=$(curl -s https://api.github.com/repos/kubevirt/containerized-data-importer/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$CDI_VERSION" ]; then
    echo "❌ CDI 버전을 가져오는 데 실패했습니다."
    exit 1
fi
echo "▶ 배포할 CDI 버전: $CDI_VERSION"

# 2. CDI Operator 배포
echo "[2/3] CDI Operator 배포 중..."
kubectl apply -f "https://github.com/kubevirt/containerized-data-importer/releases/download/${CDI_VERSION}/cdi-operator.yaml"

# 3. CDI Custom Resource(CR) 배포 (실제 컴포넌트 생성)
echo "[3/3] CDI Custom Resource 배포 중..."
kubectl apply -f "https://github.com/kubevirt/containerized-data-importer/releases/download/${CDI_VERSION}/cdi-cr.yaml"

echo "-----------------------------------------"
echo "💡 설치 완료 대기 중 (인프라 팟이 뜰 때까지 잠시 기다려주세요)"
echo "   확인 명령어: kubectl get pods -n cdi"
echo "========================================="