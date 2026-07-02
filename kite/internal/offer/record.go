package offer

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/resource"

	"kite/internal/store"
)

func createRecord(req CreateRequest) (store.KiteVirtualMachineOfferRecord, error) {
	req.TargetNamespace = strings.TrimSpace(req.TargetNamespace)
	req.Name = strings.TrimSpace(req.Name)
	req.Memory = strings.TrimSpace(req.Memory)
	req.Image = strings.TrimSpace(req.Image)
	req.Disk = strings.TrimSpace(req.Disk)
	req.ExpiresAt = strings.TrimSpace(req.ExpiresAt)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)

	if req.Name == "" {
		req.Name = "offer-" + uuid.NewString()
	}
	if req.TargetNamespace == "" {
		return store.KiteVirtualMachineOfferRecord{}, invalid("target namespace is required")
	}
	if req.CPU < 1 {
		return store.KiteVirtualMachineOfferRecord{}, invalid("cpu must be greater than zero")
	}
	if req.Memory == "" {
		return store.KiteVirtualMachineOfferRecord{}, invalid("memory is required")
	}
	if req.Image == "" {
		req.Image = defaultImage
	}
	if req.Disk == "" {
		return store.KiteVirtualMachineOfferRecord{}, invalid("disk is required")
	}
	if req.CreatedBy == "" {
		return store.KiteVirtualMachineOfferRecord{}, invalid("createdBy is required")
	}
	if err := validateDisk(req.Disk); err != nil {
		return store.KiteVirtualMachineOfferRecord{}, err
	}
	expiresAt, err := normalizeExpiresAt(req.ExpiresAt)
	if err != nil {
		return store.KiteVirtualMachineOfferRecord{}, err
	}

	return store.KiteVirtualMachineOfferRecord{
		Name:      req.Name,
		Namespace: req.TargetNamespace,
		Spec: store.KiteVirtualMachineOfferSpec{
			CPU:       req.CPU,
			Memory:    req.Memory,
			Image:     req.Image,
			Disk:      req.Disk,
			ExpiresAt: expiresAt,
			CreatedBy: req.CreatedBy,
		},
	}, nil
}

func normalizeExpiresAt(value string) (string, error) {
	if value == "" {
		return time.Now().UTC().Add(defaultTTL).Format(time.RFC3339), nil
	}
	expiresAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return "", invalid("expiresAt must be RFC3339")
	}
	if !expiresAt.After(time.Now().UTC()) {
		return "", invalid("expiresAt must be in the future")
	}
	return expiresAt.UTC().Format(time.RFC3339), nil
}

func validateDisk(disk string) error {
	quantity, err := resource.ParseQuantity(strings.TrimSpace(disk))
	if err != nil {
		return invalid("disk must be a valid Kubernetes quantity")
	}
	if quantity.Cmp(minimumDiskQuantity) < 0 {
		return invalid(fmt.Sprintf("disk must be at least %s", minimumDiskString))
	}
	return nil
}

func invalid(message string) error {
	return RequestError{Kind: ErrorKindInvalid, Message: message}
}

func conflict(message string) error {
	return RequestError{Kind: ErrorKindConflict, Message: message}
}
