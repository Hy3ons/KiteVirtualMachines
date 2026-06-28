package apis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"kite/internal/auth"
	"kite/internal/config"
)

func TestConsoleTicketIssuedWhenVMRunning(t *testing.T) {
	tokenService := newTestTokenService(t)
	token, _, err := tokenService.IssueAccessToken("alice", auth.AccessLevelUser)
	if err != nil {
		t.Fatalf("failed to issue access token: %v", err)
	}

	tickets := NewConsoleTicketService(time.Minute, "test-secret")
	router := newConsoleTestRouter(t, tokenService, tickets, "Running", false)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vms/vm-a/console-ticket", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var res vmConsoleTicketResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if strings.TrimSpace(res.Ticket) == "" {
		t.Fatal("expected console ticket")
	}

	otherReplicaTickets := NewConsoleTicketService(time.Minute, "test-secret")
	ticket, err := otherReplicaTickets.Consume(res.Ticket, "alice-ns", "vm-a", time.Now().UTC())
	if err != nil {
		t.Fatalf("expected issued ticket to verify on another replica: %v", err)
	}
	if ticket.Subject != "alice" || ticket.Namespace != "alice-ns" || ticket.VMName != "vm-a" {
		t.Fatalf("unexpected ticket binding: %+v", ticket)
	}
}

func TestConsoleTicketRejectedWhenVMStopped(t *testing.T) {
	tokenService := newTestTokenService(t)
	token, _, err := tokenService.IssueAccessToken("alice", auth.AccessLevelUser)
	if err != nil {
		t.Fatalf("failed to issue access token: %v", err)
	}

	router := newConsoleTestRouter(t, tokenService, NewConsoleTicketService(time.Minute, "test-secret"), "Stopped", false)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vms/vm-a/console-ticket", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, rec.Code, rec.Body.String())
	}
}

func TestConsoleTicketRejectedWhenVMDeleting(t *testing.T) {
	tokenService := newTestTokenService(t)
	token, _, err := tokenService.IssueAccessToken("alice", auth.AccessLevelUser)
	if err != nil {
		t.Fatalf("failed to issue access token: %v", err)
	}

	router := newConsoleTestRouter(t, tokenService, NewConsoleTicketService(time.Minute, "test-secret"), "Running", true)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vms/vm-a/console-ticket", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, rec.Code, rec.Body.String())
	}
}

func TestConsoleTicketRejectedWhenTampered(t *testing.T) {
	store := NewConsoleTicketService(time.Minute, "test-secret")
	token, _, err := store.Issue("alice", "alice-ns", "vm-a", time.Now().UTC())
	if err != nil {
		t.Fatalf("failed to issue ticket: %v", err)
	}

	replacement := "a"
	if strings.HasSuffix(token, replacement) {
		replacement = "b"
	}
	tampered := token[:len(token)-1] + replacement
	if _, err := store.Consume(tampered, "alice-ns", "vm-a", time.Now().UTC()); err == nil {
		t.Fatal("tampered ticket should fail")
	}
}

func TestConsoleTicketRejectedWhenExpired(t *testing.T) {
	store := NewConsoleTicketService(time.Minute, "test-secret")
	issuedAt := time.Date(2026, time.June, 19, 10, 0, 0, 0, time.UTC)
	token, _, err := store.Issue("alice", "alice-ns", "vm-a", issuedAt)
	if err != nil {
		t.Fatalf("failed to issue ticket: %v", err)
	}

	if _, err := store.Consume(token, "alice-ns", "vm-a", issuedAt.Add(2*time.Minute)); err == nil {
		t.Fatal("expired ticket should fail")
	}
}

func TestConsoleWebSocketUsesTicketWithoutBearerAuth(t *testing.T) {
	tokenService := newTestTokenService(t)
	tickets := NewConsoleTicketService(time.Minute, "test-secret")
	token, _, err := tickets.Issue("alice", "alice-ns", "vm-a", time.Now().UTC())
	if err != nil {
		t.Fatalf("failed to issue console ticket: %v", err)
	}

	router := newConsoleTestRouter(t, tokenService, tickets, "Running", false)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vms/vm-a/console?ticket="+token, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("console websocket ticket should bypass bearer auth middleware, got %d: %s", rec.Code, rec.Body.String())
	}
}

func newConsoleTestRouter(t *testing.T, tokenService *auth.TokenService, tickets *ConsoleTicketService, phase string, deleting bool) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		userTestGVR:               "KiteUserList",
		userTestVirtualMachineGVR: "KiteVirtualMachineList",
	}, newUserTestObject("alice", "alice", "alice-ns", auth.AccessLevelUser), newConsoleTestVM("vm-a", "alice-ns", phase, deleting))

	router := gin.Default()
	api := router.Group("/api")
	Register(api, Dependencies{
		Config: config.Config{
			PasswordSalt:   "test-salt",
			AccessTokenTTL: time.Hour,
		},
		TokenService:     tokenService,
		DynamicClient:    dynamicClient,
		ConsoleTickets:   tickets,
		ConsoleConnector: fakeConsoleConnector{},
	})

	return router
}

func newConsoleTestVM(name string, namespace string, phase string, deleting bool) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "hy3ons.github.io/v1",
			"kind":       "KiteVirtualMachine",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"cpu":        int64(2),
				"memory":     "4Gi",
				"image":      "ubuntu-22.04",
				"disk":       "20Gi",
				"powerState": "On",
				"sshId":      "alice",
				"delete":     deleting,
			},
			"status": map[string]any{
				"phase":             phase,
				"currentPowerState": "On",
			},
		},
	}
}

type fakeConsoleConnector struct{}

func (fakeConsoleConnector) Connect(_ context.Context, _ string, _ string) (ConsoleSocket, error) {
	return fakeConsoleSocket{}, nil
}

type fakeConsoleSocket struct{}

func (fakeConsoleSocket) ReadMessage() (int, []byte, error) {
	return 0, nil, nil
}

func (fakeConsoleSocket) WriteMessage(_ int, _ []byte) error {
	return nil
}

func (fakeConsoleSocket) Close() error {
	return nil
}
