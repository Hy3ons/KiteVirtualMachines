package gateway

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	"kite/internal/auth"
)

const (
	defaultSSHServicePrefix = "vps-access-"
	defaultSSHKeySuffix     = "-ssh-key"
	vmSSHPrivateKeyName     = "id_rsa"
)

var linuxUsernamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
var dnsLabelPattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

var (
	ErrRouteNotFound  = errors.New("route not found")
	ErrRouteDuplicate = errors.New("duplicate sshId route")
)

// Route describes one SSH login target derived from a live KiteVirtualMachine.
// Username is spec.sshId and is matched against the external SSH login username.
// VMNamespace and VMName identify the owning KiteVirtualMachine.
// ServiceName and SecretName point to resources created by kite-controller for VM SSH access.
// PasswordHash is spec.sshPasswordHash and is verified with the runtime password salt.
type Route struct {
	Username     string
	PasswordHash string
	VMNamespace  string
	VMName       string
	ServiceName  string
	SecretName   string
}

// TargetAddress returns the Kubernetes Service DNS address used by kite-gateway.
// The returned address targets port 22 of the VM access Service.
func (r Route) TargetAddress() string {
	return fmt.Sprintf("%s.%s.svc.cluster.local:22", r.ServiceName, r.VMNamespace)
}

// RouteTable stores KiteVirtualMachine SSH routes keyed by spec.sshId.
// It is updated by the informer handlers and read by SSH authentication and session proxy code.
type RouteTable struct {
	mu           sync.RWMutex
	routes       map[string]Route
	duplicates   map[string]bool
	passwordSalt string
}

// NewRouteTable creates an empty SSH route table.
// passwordSalt verifies external SSH passwords against spec.sshPasswordHash.
// The returned table is expected to be populated by RegisterRouteInformer.
func NewRouteTable(passwordSalt string) *RouteTable {
	return &RouteTable{
		routes:       make(map[string]Route),
		duplicates:   make(map[string]bool),
		passwordSalt: passwordSalt,
	}
}

// Get returns a route for one SSH login username.
// username is the external SSH login user.
// The returned error is ErrRouteNotFound or ErrRouteDuplicate when authentication should fail.
func (t *RouteTable) Get(username string) (Route, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	username = strings.TrimSpace(username)
	if t.duplicates[username] {
		return Route{}, ErrRouteDuplicate
	}
	route, ok := t.routes[username]
	if !ok {
		return Route{}, ErrRouteNotFound
	}
	return route, nil
}

// ReplaceAll atomically replaces the route table from a snapshot.
// routes are live KiteVirtualMachine routes collected from an informer list.
// This function is used after informer sync and on resync-driven rebuilds.
func (t *RouteTable) ReplaceAll(routes []Route) {
	nextRoutes := make(map[string]Route, len(routes))
	nextDuplicates := make(map[string]bool)
	for _, route := range routes {
		if _, exists := nextRoutes[route.Username]; exists {
			nextDuplicates[route.Username] = true
			delete(nextRoutes, route.Username)
			continue
		}
		if nextDuplicates[route.Username] {
			continue
		}
		nextRoutes[route.Username] = route
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.routes = nextRoutes
	t.duplicates = nextDuplicates
}

// AuthenticatePassword checks whether one SSH username/password pair matches a route.
// username is the external SSH login user.
// password is the client-provided SSH password.
// The returned Route is used later to dial the VM Service after SSH handshake completes.
func (t *RouteTable) AuthenticatePassword(username string, password []byte) (Route, error) {
	route, err := t.Get(username)
	if err != nil {
		return Route{}, err
	}
	if !auth.VerifyPassword(string(password), t.passwordSalt, route.PasswordHash) {
		return Route{}, errors.New("invalid SSH password")
	}
	return route, nil
}

// RegisterRouteInformer attaches route-table rebuild behavior to a KiteVirtualMachine informer.
// informer watches hy3ons.github.io/v1 KiteVirtualMachine resources across all namespaces.
// table receives rebuilt live routes whenever add, update, delete, or resync events happen.
func RegisterRouteInformer(informer cache.SharedIndexInformer, table *RouteTable) {
	rebuild := func() {
		routes := routesFromInformerStore(informer.GetStore())
		table.ReplaceAll(routes)
	}
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			rebuild()
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			rebuild()
		},
		DeleteFunc: func(obj interface{}) {
			rebuild()
		},
	})
}

// RunRouteInformer starts the KiteVirtualMachine informer and blocks until ctx is cancelled.
// ctx controls informer lifetime.
// dynamicClient reads KiteVirtualMachine resources from the Kubernetes API.
// table stores sshId routes used by the SSH server.
// A nil error means the informer started and later stopped because ctx was cancelled.
func RunRouteInformer(ctx context.Context, dynamicClient dynamic.Interface, table *RouteTable) error {
	if dynamicClient == nil {
		return errors.New("dynamic client is required")
	}
	if table == nil {
		return errors.New("route table is required")
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, 30*time.Second)
	informer := factory.ForResource(kiteVirtualMachineGVR).Informer()
	RegisterRouteInformer(informer, table)

	stopCh := make(chan struct{})
	defer close(stopCh)
	factory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
		return errors.New("failed to sync KiteVirtualMachine route informer")
	}
	table.ReplaceAll(routesFromInformerStore(informer.GetStore()))

	<-ctx.Done()
	return nil
}

// routesFromInformerStore converts the current informer cache into gateway routes.
// store is the shared informer store for KiteVirtualMachine objects.
// The returned slice excludes deleting, incomplete, duplicate-unsafe, or malformed VMs later in ReplaceAll.
// This helper is used by informer event handlers and initial cache sync.
func routesFromInformerStore(store cache.Store) []Route {
	items := store.List()
	routes := make([]Route, 0, len(items))
	for _, item := range items {
		obj, ok := item.(*unstructured.Unstructured)
		if !ok {
			continue
		}
		route, ok := RouteFromKiteVirtualMachine(obj)
		if ok {
			routes = append(routes, route)
		}
	}
	return routes
}

// RouteFromKiteVirtualMachine extracts one SSH route from a KiteVirtualMachine object.
// obj is a hy3ons.github.io/v1 KiteVirtualMachine read from the informer.
// The boolean return is false when the VM is deleting, delete-intended, missing unsafe route fields, or missing password hash.
func RouteFromKiteVirtualMachine(obj *unstructured.Unstructured) (Route, bool) {
	if obj == nil || obj.GetNamespace() == "" || obj.GetName() == "" || obj.GetDeletionTimestamp() != nil {
		return Route{}, false
	}

	deleteIntent, _, _ := unstructured.NestedBool(obj.Object, "spec", "delete")
	if deleteIntent {
		return Route{}, false
	}

	username, _, _ := unstructured.NestedString(obj.Object, "spec", "sshId")
	passwordHash, _, _ := unstructured.NestedString(obj.Object, "spec", "sshPasswordHash")
	username = strings.TrimSpace(username)
	passwordHash = strings.TrimSpace(passwordHash)
	if !isSafeLinuxUsername(username) || passwordHash == "" {
		return Route{}, false
	}

	serviceName, _, _ := unstructured.NestedString(obj.Object, "status", "serviceName")
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		serviceName = defaultSSHServicePrefix + obj.GetName()
	}
	if !isSafeDNSLabel(serviceName) {
		return Route{}, false
	}

	secretName, _, _ := unstructured.NestedString(obj.Object, "status", "sshKeySecretName")
	secretName = strings.TrimSpace(secretName)
	if secretName == "" {
		secretName = obj.GetName() + defaultSSHKeySuffix
	}
	if !isSafeDNSLabel(secretName) {
		return Route{}, false
	}

	return Route{
		Username:     username,
		PasswordHash: passwordHash,
		VMNamespace:  obj.GetNamespace(),
		VMName:       obj.GetName(),
		ServiceName:  serviceName,
		SecretName:   secretName,
	}, true
}

// isSafeLinuxUsername checks whether value is safe to use as an SSH and VM Linux username.
// value comes from KiteVirtualMachine.spec.sshId.
// The result is used to keep malformed or injection-like sshId values out of gateway routing.
func isSafeLinuxUsername(value string) bool {
	return linuxUsernamePattern.MatchString(value)
}

// isSafeDNSLabel checks whether value is safe as a Service or Secret metadata.name.
// value comes from KiteVirtualMachine status fields or stable controller naming.
// The result prevents invalid DNS names from becoming backend SSH targets.
func isSafeDNSLabel(value string) bool {
	return len(value) <= 63 && dnsLabelPattern.MatchString(value)
}

// ReadPrivateKey reads the VM SSH private key Secret for one route.
// ctx controls the Kubernetes API request.
// dynamicClient reads core/v1 secrets in the VM namespace.
// route identifies the Secret and namespace created by kite-controller.
func ReadPrivateKey(ctx context.Context, dynamicClient dynamic.Interface, route Route) (string, error) {
	secret, err := dynamicClient.Resource(secretGVR).Namespace(route.VMNamespace).Get(ctx, route.SecretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to read SSH key Secret %s/%s: %w", route.VMNamespace, route.SecretName, err)
	}

	data, _, _ := unstructured.NestedStringMap(secret.Object, "data")
	privateKeyData := strings.TrimSpace(data[vmSSHPrivateKeyName])
	if privateKeyData == "" {
		return "", fmt.Errorf("SSH key Secret %s/%s is missing %s", route.VMNamespace, route.SecretName, vmSSHPrivateKeyName)
	}
	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyData)
	if err != nil {
		return "", fmt.Errorf("failed to decode %s from SSH key Secret %s/%s: %w", vmSSHPrivateKeyName, route.VMNamespace, route.SecretName, err)
	}
	privateKey := strings.TrimSpace(string(privateKeyBytes))
	if privateKey == "" {
		return "", fmt.Errorf("SSH key Secret %s/%s is missing %s", route.VMNamespace, route.SecretName, vmSSHPrivateKeyName)
	}
	return privateKey, nil
}

// EnsureServiceExists checks that the VM access Service exists before trying to SSH.
// ctx controls the Kubernetes API request.
// dynamicClient reads core/v1 services in the VM namespace.
// route identifies the Service expected to forward port 22 to the VM.
func EnsureServiceExists(ctx context.Context, dynamicClient dynamic.Interface, route Route) error {
	if _, err := dynamicClient.Resource(serviceGVR).Namespace(route.VMNamespace).Get(ctx, route.ServiceName, metav1.GetOptions{}); err != nil {
		return fmt.Errorf("failed to read VM SSH Service %s/%s: %w", route.VMNamespace, route.ServiceName, err)
	}
	return nil
}
