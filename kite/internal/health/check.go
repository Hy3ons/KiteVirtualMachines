package health

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const defaultTimeout = 3 * time.Second

var kiteUserGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kiteusers",
}

var kiteVirtualMachineGVR = schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kitevirtualmachines",
}

// Report describes the API server health state.
// Status is "ok" when every check succeeds and "degraded" otherwise.
// Checks contains per-dependency results for the frontend or operators.
// This report is returned by kite-api's /health endpoint.
type Report struct {
	Status string  `json:"status"`
	Checks []Check `json:"checks"`
}

// Check describes one dependency health check result.
// Name identifies the dependency or API path checked.
// Status is "ok" or "failed".
// Message gives a short human-readable summary.
type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Run checks whether kite-api can reach Kubernetes CRD-backed state.
// ctx controls the overall health check lifetime.
// dynamicClient is used to list Kite custom resources through Kubernetes API server.
// The returned report treats successful Kite CRD list calls as an inferred Kubernetes API and etcd read-path check.
func Run(ctx context.Context, dynamicClient dynamic.Interface) Report {
	report := Report{
		Status: "ok",
		Checks: []Check{
			{
				Name:    "api",
				Status:  "ok",
				Message: "kite-api process is running",
			},
		},
	}

	if dynamicClient == nil {
		return withFailedCheck(report, Check{
			Name:    "kubernetes.dynamicClient",
			Status:  "failed",
			Message: "dynamic Kubernetes client is not configured",
		})
	}

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	report = withCheck(report, listCheck(ctx, dynamicClient, "kiteusers.crd.etcdReadPath", kiteUserGVR, "KiteUser CRD list succeeded"))
	report = withCheck(report, listCheck(ctx, dynamicClient, "kitevirtualmachines.crd.etcdReadPath", kiteVirtualMachineGVR, "KiteVirtualMachine CRD list succeeded"))

	return report
}

// listCheck verifies that one Kite CRD can be listed through the Kubernetes API.
// ctx controls the Kubernetes list call timeout.
// dynamicClient performs the CRD list request.
// name and successMessage are copied into the returned health check.
func listCheck(ctx context.Context, dynamicClient dynamic.Interface, name string, gvr schema.GroupVersionResource, successMessage string) Check {
	_, err := dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{
		Limit: 1,
	})
	if err != nil {
		return Check{
			Name:    name,
			Status:  "failed",
			Message: err.Error(),
		}
	}

	return Check{
		Name:    name,
		Status:  "ok",
		Message: successMessage,
	}
}

// withCheck appends one check and degrades the report when it failed.
// report is the current health report.
// check is the dependency check to append.
// The returned report is ready to return from the health endpoint.
func withCheck(report Report, check Check) Report {
	if check.Status != "ok" {
		report.Status = "degraded"
	}

	report.Checks = append(report.Checks, check)
	return report
}

// withFailedCheck appends a failed check and marks the report degraded.
// report is the current health report.
// check is the failed dependency check to append.
// The returned report is ready to return from the health endpoint.
func withFailedCheck(report Report, check Check) Report {
	report.Status = "degraded"
	report.Checks = append(report.Checks, check)
	return report
}
