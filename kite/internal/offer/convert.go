package offer

import (
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func offerFromObject(obj *unstructured.Unstructured) VirtualMachineOffer {
	spec, _ := obj.Object["spec"].(map[string]any)
	status, _ := obj.Object["status"].(map[string]any)
	phase := stringValue(status, "phase")
	if phase == "" {
		phase = phaseAvailable
	}

	return VirtualMachineOffer{
		ID:        string(obj.GetUID()),
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		CPU:       intValue(spec, "cpu"),
		Memory:    stringValue(spec, "memory"),
		Disk:      stringValue(spec, "disk"),
		Image:     stringValue(spec, "image"),
		ExpiresAt: stringValue(spec, "expiresAt"),
		CreatedBy: stringValue(spec, "createdBy"),
		Phase:     phase,
		ClaimedBy: stringValue(status, "claimedBy"),
		Message:   stringValue(status, "message"),
	}
}

func claimedStatus(observedGeneration int64, username string) map[string]any {
	return map[string]any{
		"phase":              phaseClaimed,
		"claimedBy":          username,
		"message":            "offer claimed and VM creation started",
		"observedGeneration": observedGeneration,
	}
}

func offerExpired(expiresAt string, now time.Time) bool {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(expiresAt))
	return err != nil || !parsed.After(now)
}

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	if value, ok := values[key].(string); ok {
		return value
	}
	return ""
}

func intValue(values map[string]any, key string) int64 {
	if values == nil {
		return 0
	}
	switch value := values[key].(type) {
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
