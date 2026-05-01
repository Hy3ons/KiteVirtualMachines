package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	"kite/internal/render/namespace"
)

type ClientManager struct {
	KubeClient kubernetes.Interface
	DynamicClient dynamic.Interface
}

func NewClientManager (kube kubernetes.Interface, dyn dynamic.Interface) (*ClientManager) {
	return &ClientManager{
		KubeClient: kube,
		DynamicClient: dyn,
	}
}

func (m *ClientManager) ApplyUnstructured(ctx context.Context, obj *unstructured.Unstructured) error {
	// 1. GVK(GroupVersionKind)에서 GVR(GroupVersionResource) 유추하기
	// 현석님 프로젝트라면 보통 Kind의 소문자 + 's' 가 리소스 이름이 됩니다. (예: VirtualMachine -> virtualmachines)
	gvr := schema.GroupVersionResource{
		Group:    obj.GroupVersionKind().Group,
		Version:  obj.GroupVersionKind().Version,
		Resource: strings.ToLower(obj.GetKind()) + "s", 
	}

	// 2. 데이터를 JSON으로 직렬화
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal object: %w", err)
	}

	// 3. Server-Side Apply 실행
	// .Patch() 함수를 쓰되, ApplyPatchType을 지정하는 것이 핵심입니다.
	_, err = m.DynamicClient.Resource(gvr).
		Namespace(obj.GetNamespace()).
		Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: "kite-controller", // 이 필드들을 관리하는 주체 이름
			Force:        ptr.To(true), // 충돌 시 강제로 덮어쓰기 (선택 사항)
		})

	if err != nil {
		return fmt.Errorf("failed to apply resource: %w", err)
	}

	return nil
}

func (m *ClientManager) ApplyNamespace (ctx context.Context, data namespace.NamespaceData) (error) {
	renderer, err := data.Render()

	if err != nil {
		return err
	}

	err = m.ApplyUnstructured(ctx, renderer)

	if err != nil {
		return err
	}

	return nil
}


type ApplyKubeVirtParam struct {
	Name string
	Port string
	Namespace string
	DomainName string
	Memory string
	CPU string
	VmImage string
}

func (m *ClientManager) ApplyKubeVirt (ctx context.Context, params ApplyKubeVirtParam) {

}
