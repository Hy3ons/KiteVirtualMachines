package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kite/internal/kube"
)

// TestStoreCRUDWithRealClient verifies Kite CRD CRUD through a real Kubernetes client.
// The test uses kube.GetClientManager to connect to the current cluster.
// KITE_STORE_INTEGRATION_TEST must be set to 1 because this test creates and deletes real resources.
// The test expects KiteUser and KiteVirtualMachine CRDs to already be installed in the cluster.
func TestStoreCRUDWithRealClient(t *testing.T) {
	// 이 테스트는 실제 클러스터에 임시 리소스를 생성/삭제합니다.
	// 터미널에서 KITE_STORE_INTEGRATION_TEST=1 환경변수를 넣어 실행할 때만 동작합니다.
	// 예: KITE_STORE_INTEGRATION_TEST=1 go test ./internal/store -run TestStoreCRUDWithRealClient -v
	if os.Getenv("KITE_STORE_INTEGRATION_TEST") != "1" {
		t.Skip("set KITE_STORE_INTEGRATION_TEST=1 to run real Kubernetes store integration tests")
	}

	ctx := context.Background()
	clientManager, err := kube.GetClientManager()
	if err != nil {
		t.Fatalf("failed to create Kubernetes client manager: %v", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	namespace := "kite-store-test-" + suffix
	userName := "kite-user-test-" + suffix
	vmName := "kite-vm-test-" + suffix

	if _, err := clientManager.KubeClient.CoreV1().Namespaces().Create(ctx, newNamespace(namespace), metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create test namespace %q: %v", namespace, err)
	}
	t.Cleanup(func() {
		if err := clientManager.KubeClient.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Logf("failed to delete test namespace %q: %v", namespace, err)
		}
	})

	userStore := NewUserStore(clientManager.DynamicClient)
	vmStore := NewVirtualMachineStore(clientManager.DynamicClient)

	testRealUserStoreCRUD(t, ctx, userStore, userName, namespace)
	testRealVirtualMachineStoreCRUD(t, ctx, vmStore, namespace, vmName)
}

// testRealUserStoreCRUD runs create, get, list, update, and delete against a real KiteUser CRD.
// t reports failures for the parent integration test.
// ctx controls Kubernetes API request lifetime.
// userStore is the real dynamic-client backed KiteUser store under test.
// userName is the unique metadata.name used by this test run.
// namespace is stored in spec.namespace to connect the test user with its namespace.
func testRealUserStoreCRUD(t *testing.T, ctx context.Context, userStore *UserStore, userName string, namespace string) {
	t.Helper()

	record := KiteUserRecord{
		Name: userName,
		Spec: KiteUserSpec{
			Username:     userName,
			Email:        userName + "@example.com",
			Password:     "hashed-password",
			Namespace:    namespace,
			ProfileImage: "base64",
			AccessLevel:  1,
		},
	}
	t.Cleanup(func() {
		if err := userStore.Delete(context.Background(), userName); err != nil && !apierrors.IsNotFound(err) {
			t.Logf("failed to delete KiteUser %q: %v", userName, err)
		}
	})

	created, err := userStore.Create(ctx, record)
	if err != nil {
		t.Fatalf("failed to create KiteUser: %v", err)
	}
	if created.GetNamespace() != "" {
		t.Fatalf("expected KiteUser to be cluster-scoped, got namespace %q", created.GetNamespace())
	}

	got, err := userStore.Get(ctx, userName)
	if err != nil {
		t.Fatalf("failed to get KiteUser: %v", err)
	}
	assertNestedString(t, got, record.Spec.Email, "spec", "email")

	list, err := userStore.List(ctx)
	if err != nil {
		t.Fatalf("failed to list KiteUsers: %v", err)
	}
	if len(list.Items) == 0 {
		t.Fatalf("expected KiteUser list to include at least the created test user")
	}

	record.Spec.AccessLevel = 2
	updated, err := userStore.Update(ctx, record)
	if err != nil {
		t.Fatalf("failed to update KiteUser: %v", err)
	}
	assertNestedInt64(t, updated, 2, "spec", "access_level")

	if err := userStore.Delete(ctx, userName); err != nil {
		t.Fatalf("failed to delete KiteUser: %v", err)
	}
	if _, err := userStore.Get(ctx, userName); !apierrors.IsNotFound(err) {
		t.Fatalf("expected KiteUser to be deleted, got error %v", err)
	}
}

// testRealVirtualMachineStoreCRUD runs create, get, list, update, and delete against a real KiteVirtualMachine CRD.
// t reports failures for the parent integration test.
// ctx controls Kubernetes API request lifetime.
// vmStore is the real dynamic-client backed KiteVirtualMachine store under test.
// namespace is the Kubernetes namespace created for this test run.
// vmName is the unique metadata.name used by this test run.
func testRealVirtualMachineStoreCRUD(t *testing.T, ctx context.Context, vmStore *VirtualMachineStore, namespace string, vmName string) {
	t.Helper()

	record := KiteVirtualMachineRecord{
		Name:      vmName,
		Namespace: namespace,
		Spec: KiteVirtualMachineSpec{
			CPU:        1,
			Memory:     "1Gi",
			Image:      "ubuntu-22.04",
			Disk:       "5Gi",
			PowerState: "Off",
		},
	}
	t.Cleanup(func() {
		if err := vmStore.Delete(context.Background(), namespace, vmName); err != nil && !apierrors.IsNotFound(err) {
			t.Logf("failed to delete KiteVirtualMachine %q/%q: %v", namespace, vmName, err)
		}
	})

	created, err := vmStore.Create(ctx, record)
	if err != nil {
		t.Fatalf("failed to create KiteVirtualMachine: %v", err)
	}
	if created.GetNamespace() != namespace {
		t.Fatalf("expected namespace %q, got %q", namespace, created.GetNamespace())
	}

	got, err := vmStore.Get(ctx, namespace, vmName)
	if err != nil {
		t.Fatalf("failed to get KiteVirtualMachine: %v", err)
	}
	assertNestedString(t, got, "Off", "spec", "powerState")

	list, err := vmStore.List(ctx, namespace)
	if err != nil {
		t.Fatalf("failed to list KiteVirtualMachines: %v", err)
	}
	if len(list.Items) == 0 {
		t.Fatalf("expected KiteVirtualMachine list to include at least the created test VM")
	}

	record.Spec.CPU = 2
	record.Spec.PowerState = "On"
	updated, err := vmStore.Update(ctx, record)
	if err != nil {
		t.Fatalf("failed to update KiteVirtualMachine: %v", err)
	}
	assertNestedInt64(t, updated, 2, "spec", "cpu")
	assertNestedString(t, updated, "On", "spec", "powerState")

	if err := vmStore.Delete(ctx, namespace, vmName); err != nil {
		t.Fatalf("failed to delete KiteVirtualMachine: %v", err)
	}
	if _, err := vmStore.Get(ctx, namespace, vmName); !apierrors.IsNotFound(err) {
		t.Fatalf("expected KiteVirtualMachine to be deleted, got error %v", err)
	}
}

// newNamespace creates the temporary namespace object used by the real client integration test.
// name is the metadata.name of the namespace to create.
// The returned object is passed to the typed Kubernetes client before testing namespaced CRDs.
func newNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
