package offer

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/dynamic"

	"kite/internal/store"
	vmservice "kite/internal/vm"
)

type ErrorKind string

const (
	ErrorKindInvalid  ErrorKind = "invalid"
	ErrorKindConflict ErrorKind = "conflict"
)

const (
	defaultTTL        = 24 * time.Hour
	defaultImage      = "ubuntu-22.04"
	phaseAvailable    = "Available"
	phaseClaimed      = "Claimed"
	minimumDiskString = "20Gi"
)

var minimumDiskQuantity = resource.MustParse(minimumDiskString)

// RequestError describes an offer request error that is safe to return through HTTP.
// Kind separates validation errors from claim conflicts.
// Message explains what the API client should fix.
type RequestError struct {
	Kind    ErrorKind
	Message string
}

func (e RequestError) Error() string {
	return e.Message
}

// CreateRequest contains admin-provided VM capacity fields for a target user namespace.
// TargetNamespace selects where the offer CRD is stored.
// ExpiresAt may be empty, in which case the service uses the default 24 hour TTL.
type CreateRequest struct {
	TargetNamespace string
	Name            string
	CPU             int
	Memory          string
	Disk            string
	Image           string
	ExpiresAt       string
	CreatedBy       string
}

// ClaimRequest contains the user-provided fields needed to turn an offer into a VM.
// VMName becomes KiteVirtualMachine metadata.name.
// InitialLoginPassword is hashed by the VM service and is never stored as plaintext.
type ClaimRequest struct {
	VMName               string
	DomainPrefix         string
	SSHID                string
	InitialLoginPassword string
	PowerState           string
}

// VirtualMachineOffer is the frontend-safe offer response returned by kite-api.
// It combines metadata, spec, and status fields from KiteVirtualMachineOffer CRDs.
// The target namespace is included so admins can audit where an offer is assigned.
type VirtualMachineOffer struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	CPU       int64  `json:"cpu"`
	Memory    string `json:"memory"`
	Disk      string `json:"disk"`
	Image     string `json:"image"`
	ExpiresAt string `json:"expiresAt"`
	CreatedBy string `json:"createdBy"`
	Phase     string `json:"phase"`
	ClaimedBy string `json:"claimedBy"`
	Message   string `json:"message"`
}

// Service provides KiteVirtualMachineOffer operations backed by CRDs.
// offerStore reads and writes namespaced offer resources.
// vmService creates the final KiteVirtualMachine during claim.
type Service struct {
	offerStore *store.VirtualMachineOfferStore
	vmService  *vmservice.Service
}

// NewService creates an offer service backed by a dynamic Kubernetes client.
// dynamicClient is used by both offer and VM stores.
// passwordSalt is passed to the VM service for gateway password hashing during claim.
func NewService(dynamicClient dynamic.Interface, passwordSalt string) *Service {
	return &Service{
		offerStore: store.NewVirtualMachineOfferStore(dynamicClient),
		vmService:  vmservice.NewService(dynamicClient, passwordSalt),
	}
}

// Create creates one KiteVirtualMachineOffer CRD in a target user's namespace.
// ctx controls Kubernetes API calls.
// req contains normalized desired capacity fields from the admin API.
// The returned offer is converted from the created CRD object.
func (s *Service) Create(ctx context.Context, req CreateRequest) (VirtualMachineOffer, error) {
	record, err := createRecord(req)
	if err != nil {
		return VirtualMachineOffer{}, err
	}

	created, err := s.offerStore.Create(ctx, record)
	if err != nil {
		return VirtualMachineOffer{}, err
	}
	return offerFromObject(created), nil
}

// List returns unexpired KiteVirtualMachineOffer CRDs from one namespace.
// ctx controls Kubernetes API calls.
// namespace is the authenticated user's namespace or an admin-selected namespace.
// The returned offers are safe for frontend display.
func (s *Service) List(ctx context.Context, namespace string) ([]VirtualMachineOffer, error) {
	list, err := s.offerStore.List(ctx, namespace)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	offers := make([]VirtualMachineOffer, 0, len(list.Items))
	for i := range list.Items {
		offer := offerFromObject(&list.Items[i])
		if offer.Phase == phaseClaimed || offerExpired(offer.ExpiresAt, now) {
			continue
		}
		offers = append(offers, offer)
	}
	return offers, nil
}

// Claim claims one offer and creates a KiteVirtualMachine in the same namespace.
// ctx controls Kubernetes API calls.
// namespace and name identify the offer visible to the current user.
// username records who claimed the offer in status before the offer is removed.
func (s *Service) Claim(ctx context.Context, namespace string, name string, username string, req ClaimRequest) (vmservice.VirtualMachine, error) {
	current, err := s.offerStore.Get(ctx, namespace, name)
	if err != nil {
		return vmservice.VirtualMachine{}, err
	}
	offer := offerFromObject(current)
	if offer.Phase == phaseClaimed {
		return vmservice.VirtualMachine{}, conflict("offer is already claimed")
	}
	if offerExpired(offer.ExpiresAt, time.Now().UTC()) {
		return vmservice.VirtualMachine{}, invalid("offer is expired")
	}
	if _, err := s.offerStore.UpdateStatus(ctx, current, claimedStatus(current.GetGeneration(), username)); err != nil {
		if apierrors.IsConflict(err) {
			return vmservice.VirtualMachine{}, conflict("offer was claimed by another request")
		}
		return vmservice.VirtualMachine{}, err
	}

	vm, err := s.vmService.Create(ctx, vmservice.CreateRequest{
		Name:         req.VMName,
		Namespace:    namespace,
		CPU:          int(offer.CPU),
		Memory:       offer.Memory,
		Image:        offer.Image,
		Disk:         offer.Disk,
		DomainPrefix: req.DomainPrefix,
		SSHID:        req.SSHID,
		SSHPassword:  req.InitialLoginPassword,
		PowerState:   req.PowerState,
	})
	if err != nil {
		return vmservice.VirtualMachine{}, err
	}
	if err := s.offerStore.Delete(ctx, namespace, name); err != nil && !apierrors.IsNotFound(err) {
		return vmservice.VirtualMachine{}, err
	}

	return vm, nil
}

// ctx controls Kubernetes API calls.
// namespace and name identify the offer selected by a level 3 admin.
// A nil error means Kubernetes accepted the delete request.
func (s *Service) Delete(ctx context.Context, namespace string, name string) error {
	return s.offerStore.Delete(ctx, namespace, name)
}

// RequestErrorKind returns the classified request error kind when err is safe for clients.
// err is usually returned by Create or Claim.
// The boolean return is false for infrastructure or unexpected Kubernetes failures.
func RequestErrorKind(err error) (ErrorKind, bool) {
	requestErr, ok := err.(RequestError)
	if !ok {
		return "", false
	}
	return requestErr.Kind, true
}
