package vm

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"

	"kite/internal/store"
)

type ErrorKind string

const (
	ErrorKindInvalid  ErrorKind = "invalid"
	ErrorKindConflict ErrorKind = "conflict"
)

const (
	defaultCPU        = 2
	defaultMemory     = "4Gi"
	defaultImage      = "ubuntu-22.04"
	defaultPowerState = "Off"
	minDiskSize       = "20Gi"
	maxSSHPasswordLen = 128
)

var minimumDiskQuantity = resource.MustParse(minDiskSize)

var sshIDPattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)

// RequestError describes a VM request error that is safe to return through HTTP.
// Kind separates validation errors from unexpected Kubernetes failures.
// Message explains what the API client should fix.
type RequestError struct {
	Kind    ErrorKind
	Message string
}

func (e RequestError) Error() string {
	return e.Message
}

// CreateRequest contains frontend-provided VM creation fields.
// Name is optional; when empty the service generates a DNS-safe VM name.
// Namespace is selected by the API handler from the authenticated user, not from public user input.
// Disk accepts values like "25Gi" after the HTTP layer normalizes form input.
type CreateRequest struct {
	Name         string
	Namespace    string
	CPU          int
	Memory       string
	Image        string
	Disk         string
	DomainPrefix string
	SSHID        string
	SSHPassword  string
	PowerState   string
}

// UpdateRequest contains mutable VM fields.
// Nil fields are left unchanged.
// Delete marks the VM for controller-managed cleanup when true.
type UpdateRequest struct {
	CPU          *int
	Memory       *string
	Image        *string
	Disk         *string
	DomainPrefix *string
	SSHID        *string
	SSHPassword  *string
	PowerState   *string
	Delete       *bool
}

// VirtualMachine is the frontend-safe VM response model returned by kite-api.
// It combines metadata, spec, and status fields from KiteVirtualMachine CRDs.
// Password-like fields such as spec.sshPassword are intentionally excluded.
type VirtualMachine struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Namespace          string `json:"namespace"`
	Owner              string `json:"owner"`
	Domain             string `json:"domain"`
	Phase              string `json:"phase"`
	PowerState         string `json:"powerState"`
	CurrentPowerState  string `json:"currentPowerState"`
	CPU                int64  `json:"cpu"`
	Memory             string `json:"memory"`
	Image              string `json:"image"`
	Disk               string `json:"disk"`
	DomainPrefix       string `json:"domainPrefix"`
	SSHID              string `json:"sshId"`
	Delete             bool   `json:"delete"`
	DataVolumePhase    string `json:"dataVolumePhase"`
	DataVolumeProgress string `json:"dataVolumeProgress"`
	DataVolumeMessage  string `json:"dataVolumeMessage"`
}

// Service provides KiteVirtualMachine operations backed by CRDs.
// vmStore reads and writes namespaced KiteVirtualMachine resources.
// This service is used by kite-api VM handlers before kite-controller reconciles real resources.
type Service struct {
	vmStore *store.VirtualMachineStore
}

// NewService creates a VM service backed by a dynamic Kubernetes client.
// dynamicClient is used by the underlying store for KiteVirtualMachine CRD operations.
// The returned service is request-scoped by API handlers.
func NewService(dynamicClient dynamic.Interface) *Service {
	return &Service{
		vmStore: store.NewVirtualMachineStore(dynamicClient),
	}
}

// Create creates one KiteVirtualMachine CRD in a user namespace.
// ctx controls Kubernetes API calls.
// req contains normalized desired VM fields from the HTTP handler.
// The returned VM is converted from the created CRD object.
func (s *Service) Create(ctx context.Context, req CreateRequest) (VirtualMachine, error) {
	record, err := createRecord(req)
	if err != nil {
		return VirtualMachine{}, err
	}
	if err := s.ensureUniqueSSHID(ctx, record.Spec.SSHID, "", ""); err != nil {
		return VirtualMachine{}, err
	}

	created, err := s.vmStore.Create(ctx, record)
	if err != nil {
		return VirtualMachine{}, err
	}

	return vmFromObject(created), nil
}

// List returns KiteVirtualMachine CRDs from one namespace.
// ctx controls Kubernetes API calls.
// namespace is the authenticated user's namespace.
// The returned VMs are safe for frontend display.
func (s *Service) List(ctx context.Context, namespace string) ([]VirtualMachine, error) {
	list, err := s.vmStore.List(ctx, namespace)
	if err != nil {
		return nil, err
	}

	return vmsFromList(list), nil
}

// ListAll returns KiteVirtualMachine CRDs across every namespace.
// ctx controls Kubernetes API calls.
// The returned VMs are intended for manager and admin dashboards.
func (s *Service) ListAll(ctx context.Context) ([]VirtualMachine, error) {
	list, err := s.vmStore.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	return vmsFromList(list), nil
}

// Get reads one VM by namespace and name.
// ctx controls Kubernetes API calls.
// namespace and name identify the KiteVirtualMachine CRD.
// The returned VM is safe for frontend display.
func (s *Service) Get(ctx context.Context, namespace string, name string) (VirtualMachine, error) {
	obj, err := s.vmStore.Get(ctx, namespace, name)
	if err != nil {
		return VirtualMachine{}, err
	}

	return vmFromObject(obj), nil
}

// Update changes mutable spec fields on one VM.
// ctx controls Kubernetes API calls.
// namespace and name identify the KiteVirtualMachine CRD.
// req contains optional spec fields; nil fields preserve the current CRD value.
func (s *Service) Update(ctx context.Context, namespace string, name string, req UpdateRequest) (VirtualMachine, error) {
	current, err := s.vmStore.Get(ctx, namespace, name)
	if err != nil {
		return VirtualMachine{}, err
	}

	record, err := recordFromObject(current)
	if err != nil {
		return VirtualMachine{}, err
	}

	if err := applyUpdate(&record, req); err != nil {
		return VirtualMachine{}, err
	}
	if req.SSHID != nil {
		if err := s.ensureUniqueSSHID(ctx, record.Spec.SSHID, namespace, name); err != nil {
			return VirtualMachine{}, err
		}
	}

	updated, err := s.vmStore.Update(ctx, record)
	if err != nil {
		return VirtualMachine{}, err
	}

	return vmFromObject(updated), nil
}

// ensureUniqueSSHID checks that no other KiteVirtualMachine uses the same host login id.
// ctx controls the cluster-wide KiteVirtualMachine list request.
// sshID is the desired spec.sshId value that maps to one Linux account on the single-node host.
// currentNamespace and currentName identify the VM being updated and are empty during create.
// This function is used by create and sshId update flows before writing the KiteVirtualMachine CRD.
func (s *Service) ensureUniqueSSHID(ctx context.Context, sshID string, currentNamespace string, currentName string) error {
	sshID = strings.TrimSpace(sshID)
	if sshID == "" {
		return invalid("sshId is required")
	}

	list, err := s.vmStore.ListAll(ctx)
	if err != nil {
		return err
	}

	for i := range list.Items {
		item := &list.Items[i]
		if item.GetNamespace() == currentNamespace && item.GetName() == currentName {
			continue
		}

		spec, _ := item.Object["spec"].(map[string]any)
		if strings.TrimSpace(stringValue(spec, "sshId")) == sshID {
			return conflict("sshId is already used by another virtual machine")
		}
	}

	return nil
}

// MarkDelete sets spec.delete=true so kite-controller performs cleanup.
// ctx controls Kubernetes API calls.
// namespace and name identify the KiteVirtualMachine CRD.
// The returned VM shows the updated desired delete state.
func (s *Service) MarkDelete(ctx context.Context, namespace string, name string) (VirtualMachine, error) {
	deleteIntent := true
	return s.Update(ctx, namespace, name, UpdateRequest{Delete: &deleteIntent})
}

// createRecord validates and converts a create request into a store record.
// req contains HTTP-normalized VM fields.
// The returned record can be written directly to the KiteVirtualMachine store.
func createRecord(req CreateRequest) (store.KiteVirtualMachineRecord, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Namespace = strings.TrimSpace(req.Namespace)
	req.Memory = strings.TrimSpace(req.Memory)
	req.Image = strings.TrimSpace(req.Image)
	req.Disk = strings.TrimSpace(req.Disk)
	req.DomainPrefix = strings.TrimSpace(req.DomainPrefix)
	req.SSHID = strings.TrimSpace(req.SSHID)
	req.PowerState = strings.TrimSpace(req.PowerState)

	if req.Name == "" {
		req.Name = "vm-" + uuid.NewString()
	}
	if req.Namespace == "" {
		return store.KiteVirtualMachineRecord{}, invalid("namespace is required")
	}
	if req.CPU == 0 {
		req.CPU = defaultCPU
	}
	if req.Memory == "" {
		req.Memory = defaultMemory
	}
	if req.Image == "" {
		req.Image = defaultImage
	}
	if req.PowerState == "" {
		req.PowerState = defaultPowerState
	}
	if req.Disk == "" || req.SSHID == "" || req.SSHPassword == "" {
		return store.KiteVirtualMachineRecord{}, invalid("disk, sshId, and sshPassword are required")
	}
	if err := validateSSHPassword(req.SSHPassword); err != nil {
		return store.KiteVirtualMachineRecord{}, err
	}
	if err := validateDisk(req.Disk); err != nil {
		return store.KiteVirtualMachineRecord{}, err
	}
	if !sshIDPattern.MatchString(req.SSHID) {
		return store.KiteVirtualMachineRecord{}, invalid("sshId must be a Linux username using lowercase letters, numbers, underscore, or hyphen")
	}
	if req.CPU < 1 {
		return store.KiteVirtualMachineRecord{}, invalid("cpu must be greater than zero")
	}
	if req.PowerState != "On" && req.PowerState != "Off" {
		return store.KiteVirtualMachineRecord{}, invalid("powerState must be On or Off")
	}

	return store.KiteVirtualMachineRecord{
		Name:      req.Name,
		Namespace: req.Namespace,
		Spec: store.KiteVirtualMachineSpec{
			CPU:          req.CPU,
			Memory:       req.Memory,
			Image:        req.Image,
			Disk:         req.Disk,
			PowerState:   req.PowerState,
			DomainPrefix: req.DomainPrefix,
			SSHID:        req.SSHID,
			SSHPassword:  req.SSHPassword,
		},
	}, nil
}

// recordFromObject converts a KiteVirtualMachine CRD into a store update record.
// obj is the current unstructured CRD object.
// The returned record preserves current spec values before applying API updates.
func recordFromObject(obj *unstructured.Unstructured) (store.KiteVirtualMachineRecord, error) {
	spec, ok := obj.Object["spec"].(map[string]any)
	if !ok {
		return store.KiteVirtualMachineRecord{}, invalid("invalid kite virtual machine spec")
	}

	return store.KiteVirtualMachineRecord{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		Spec: store.KiteVirtualMachineSpec{
			CPU:          int(intValue(spec, "cpu")),
			Memory:       stringValue(spec, "memory"),
			Image:        stringValue(spec, "image"),
			Disk:         stringValue(spec, "disk"),
			PowerState:   stringValue(spec, "powerState"),
			DomainPrefix: stringValue(spec, "domainPrefix"),
			SSHID:        stringValue(spec, "sshId"),
			SSHPassword:  stringValue(spec, "sshPassword"),
			Delete:       boolValue(spec, "delete"),
		},
	}, nil
}

// applyUpdate applies optional request fields to a store record.
// record is modified in place.
// req contains nil-able fields from PATCH handlers.
// A nil error means the updated record is valid.
func applyUpdate(record *store.KiteVirtualMachineRecord, req UpdateRequest) error {
	if req.CPU != nil {
		if *req.CPU < 1 {
			return invalid("cpu must be greater than zero")
		}
		record.Spec.CPU = *req.CPU
	}
	if req.Memory != nil {
		record.Spec.Memory = strings.TrimSpace(*req.Memory)
	}
	if req.Image != nil {
		record.Spec.Image = strings.TrimSpace(*req.Image)
	}
	if req.Disk != nil {
		record.Spec.Disk = strings.TrimSpace(*req.Disk)
		if err := validateDisk(record.Spec.Disk); err != nil {
			return err
		}
	}
	if req.DomainPrefix != nil {
		record.Spec.DomainPrefix = strings.TrimSpace(*req.DomainPrefix)
	}
	if req.SSHID != nil {
		record.Spec.SSHID = strings.TrimSpace(*req.SSHID)
		if !sshIDPattern.MatchString(record.Spec.SSHID) {
			return invalid("sshId must be a Linux username using lowercase letters, numbers, underscore, or hyphen")
		}
	}
	if req.SSHPassword != nil {
		if err := validateSSHPassword(*req.SSHPassword); err != nil {
			return err
		}
		record.Spec.SSHPassword = *req.SSHPassword
	}
	if req.PowerState != nil {
		powerState := strings.TrimSpace(*req.PowerState)
		if powerState != "On" && powerState != "Off" {
			return invalid("powerState must be On or Off")
		}
		record.Spec.PowerState = powerState
	}
	if req.Delete != nil {
		record.Spec.Delete = *req.Delete
	}

	return nil
}

// validateDisk checks that a VM disk value is a valid Kubernetes quantity and at least 20Gi.
// disk is the normalized spec.disk value that will be stored in KiteVirtualMachine.
// A nil error means the disk can be accepted by create and update flows.
// This function is used by VM service validation before writing CRDs.
func validateDisk(disk string) error {
	quantity, err := resource.ParseQuantity(strings.TrimSpace(disk))
	if err != nil {
		return invalid("disk must be a valid Kubernetes quantity")
	}
	if quantity.Cmp(minimumDiskQuantity) < 0 {
		return invalid("disk must be at least 20Gi")
	}

	return nil
}

// validateSSHPassword rejects values that break chpasswd or hide user mistakes.
// password is stored as KiteVirtualMachine spec.sshPassword and later passed to VM and host chpasswd.
// A nil error means the value can be safely carried through YAML/base64 rendering and chpasswd stdin.
// This function is used by create and update request validation.
func validateSSHPassword(password string) error {
	if password == "" {
		return invalid("sshPassword is required")
	}
	if password != strings.TrimSpace(password) {
		return invalid("sshPassword must not start or end with whitespace")
	}
	if len(password) > maxSSHPasswordLen {
		return invalid("sshPassword must be at most 128 bytes")
	}
	if strings.Contains(password, ":") {
		return invalid("sshPassword must not contain colon")
	}
	for _, r := range password {
		if r < 0x20 || r == 0x7f {
			return invalid("sshPassword must not contain control characters")
		}
	}

	return nil
}

// vmFromObject converts one KiteVirtualMachine CRD object into an API response model.
// obj is the unstructured CRD object returned by Kubernetes.
// The returned VM includes spec and status fields needed by frontend pages.
func vmFromObject(obj *unstructured.Unstructured) VirtualMachine {
	spec, _ := obj.Object["spec"].(map[string]any)
	status, _ := obj.Object["status"].(map[string]any)

	phase := stringValue(status, "phase")
	if phase == "" {
		phase = "Creating"
	}

	return VirtualMachine{
		ID:                 obj.GetNamespace() + "/" + obj.GetName(),
		Name:               obj.GetName(),
		Namespace:          obj.GetNamespace(),
		Owner:              obj.GetNamespace(),
		Domain:             stringValue(status, "domain"),
		Phase:              phase,
		PowerState:         stringValue(spec, "powerState"),
		CurrentPowerState:  stringValue(status, "currentPowerState"),
		CPU:                intValue(spec, "cpu"),
		Memory:             stringValue(spec, "memory"),
		Image:              stringValue(spec, "image"),
		Disk:               stringValue(spec, "disk"),
		DomainPrefix:       stringValue(spec, "domainPrefix"),
		SSHID:              stringValue(spec, "sshId"),
		Delete:             boolValue(spec, "delete"),
		DataVolumePhase:    stringValue(status, "dataVolumePhase"),
		DataVolumeProgress: stringValue(status, "dataVolumeProgress"),
		DataVolumeMessage:  stringValue(status, "dataVolumeMessage"),
	}
}

// vmsFromList converts an unstructured list into frontend-safe VM responses.
// list is returned by the KiteVirtualMachine store.
// The returned slice preserves Kubernetes list order.
func vmsFromList(list *unstructured.UnstructuredList) []VirtualMachine {
	vms := make([]VirtualMachine, 0, len(list.Items))
	for i := range list.Items {
		vms = append(vms, vmFromObject(&list.Items[i]))
	}
	return vms
}

// NormalizeDisk converts frontend disk input into a CRD disk string.
// value may be a JSON number, string number, or Kubernetes quantity string.
// The returned string uses Gi when the input is numeric.
func NormalizeDisk(value any) string {
	switch typed := value.(type) {
	case float64:
		return strconv.FormatInt(int64(typed), 10) + "Gi"
	case int:
		return strconv.Itoa(typed) + "Gi"
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return ""
		}
		if _, err := strconv.Atoi(trimmed); err == nil {
			return trimmed + "Gi"
		}
		return trimmed
	default:
		return ""
	}
}

// RequestErrorKind returns the kind from a VM RequestError.
// err is any service error.
// The returned boolean is false when err is not a RequestError.
func RequestErrorKind(err error) (ErrorKind, bool) {
	requestErr, ok := err.(RequestError)
	if !ok {
		return "", false
	}
	return requestErr.Kind, true
}

func invalid(message string) error {
	return RequestError{Kind: ErrorKindInvalid, Message: message}
}

func conflict(message string) error {
	return RequestError{Kind: ErrorKindConflict, Message: message}
}

func stringValue(data map[string]any, key string) string {
	value, _ := data[key].(string)
	return value
}

func intValue(data map[string]any, key string) int64 {
	switch value := data[key].(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case float64:
		return int64(value)
	default:
		return 0
	}
}

func boolValue(data map[string]any, key string) bool {
	value, _ := data[key].(bool)
	return value
}

// IsNotFound reports whether a VM service error is a Kubernetes NotFound error.
// err is returned by the store or Kubernetes API.
// The boolean return is used by HTTP handlers to choose 404 responses.
func IsNotFound(err error) bool {
	return apierrors.IsNotFound(err)
}

// ConflictMessage returns a safe message for AlreadyExists errors.
// err is returned by Kubernetes on create collisions.
// The returned string is empty for non-conflict errors.
func ConflictMessage(err error) string {
	if apierrors.IsAlreadyExists(err) {
		return "kite virtual machine already exists"
	}
	if requestErr, ok := err.(RequestError); ok && requestErr.Kind == ErrorKindConflict {
		return requestErr.Message
	}
	return ""
}
