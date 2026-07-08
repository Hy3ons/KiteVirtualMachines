package apis

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"k8s.io/client-go/rest"
)

const kubeVirtPlainStreamProtocol = "plain.kubevirt.io"

type ConsoleConnector interface {
	Connect(ctx context.Context, namespace string, name string) (ConsoleSocket, error)
}

type ConsoleSocket interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(messageType int, data []byte) error
	Close() error
}

type consoleMessageTypeMapper func(messageType int) int

type consoleCopyDirection struct {
	source            ConsoleSocket
	target            ConsoleSocket
	targetMessageType consoleMessageTypeMapper
}

type KubeVirtConsoleConnector struct {
	config *rest.Config
}

// NewKubeVirtConsoleConnector creates a KubeVirt console connector.
// config is the Kubernetes REST config used by the kite-api service account.
// The returned connector dials KubeVirt serial console subresources over WebSocket.
// This function is used by API startup when console dependencies are not injected by tests.
func NewKubeVirtConsoleConnector(config *rest.Config) *KubeVirtConsoleConnector {
	return &KubeVirtConsoleConnector{config: rest.CopyConfig(config)}
}

// Connect opens one KubeVirt serial console WebSocket.
// ctx controls the outbound dial to the Kubernetes API server.
// namespace and name identify the VirtualMachineInstance console subresource.
// The returned socket is bridged to the browser by vmConsoleHandler.
func (c *KubeVirtConsoleConnector) Connect(ctx context.Context, namespace string, name string) (ConsoleSocket, error) {
	endpoint, err := kubeVirtConsoleEndpoint(c.config, namespace, name)
	if err != nil {
		return nil, err
	}

	tlsConfig, err := rest.TLSConfigFor(c.config)
	if err != nil {
		return nil, fmt.Errorf("create KubeVirt console TLS config: %w", err)
	}

	header, err := kubeVirtConsoleHeader(c.config)
	if err != nil {
		return nil, err
	}

	dialer := websocket.Dialer{
		TLSClientConfig: tlsConfig,
		Subprotocols:    []string{kubeVirtPlainStreamProtocol},
	}
	socket, response, err := dialer.DialContext(ctx, endpoint, header)
	if err != nil {
		if response != nil {
			return nil, fmt.Errorf("connect KubeVirt console returned status %d: %w", response.StatusCode, err)
		}
		return nil, fmt.Errorf("connect KubeVirt console: %w", err)
	}

	return socket, nil
}

func kubeVirtConsoleEndpoint(config *rest.Config, namespace string, name string) (string, error) {
	if config == nil || strings.TrimSpace(config.Host) == "" {
		return "", fmt.Errorf("Kubernetes REST config host is required")
	}

	endpoint, err := url.Parse(config.Host)
	if err != nil {
		return "", fmt.Errorf("parse Kubernetes REST config host: %w", err)
	}
	switch endpoint.Scheme {
	case "https":
		endpoint.Scheme = "wss"
	case "http":
		endpoint.Scheme = "ws"
	default:
		return "", fmt.Errorf("unsupported Kubernetes REST config scheme %q", endpoint.Scheme)
	}

	endpoint.Path = "/apis/subresources.kubevirt.io/v1/namespaces/" +
		url.PathEscape(namespace) +
		"/virtualmachineinstances/" +
		url.PathEscape(name) +
		"/console"
	return endpoint.String(), nil
}

func kubeVirtConsoleHeader(config *rest.Config) (http.Header, error) {
	token := strings.TrimSpace(config.BearerToken)
	if token == "" && config.BearerTokenFile != "" {
		data, err := os.ReadFile(config.BearerTokenFile)
		if err != nil {
			return nil, fmt.Errorf("read service account token: %w", err)
		}
		token = strings.TrimSpace(string(data))
	}
	if token == "" {
		return nil, fmt.Errorf("Kubernetes bearer token is required for KubeVirt console")
	}

	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	return header, nil
}

func vmConsoleUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		Subprotocols: []string{kubeVirtPlainStreamProtocol},
		CheckOrigin: func(r *http.Request) bool {
			return consoleOriginAllowed(r)
		},
	}
}

func consoleOriginAllowed(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if originURL.Host == r.Host {
		return true
	}

	return origin == "http://localhost:5173" ||
		origin == "http://127.0.0.1:5173" ||
		strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:")
}

func bridgeConsole(ctx context.Context, browser ConsoleSocket, upstream ConsoleSocket) error {
	errs := make(chan error, 2)
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			_ = browser.Close()
			_ = upstream.Close()
		case <-done:
		}
	}()

	go copyConsoleMessages(ctx, consoleCopyDirection{
		source:            upstream,
		target:            browser,
		targetMessageType: preserveConsoleMessageType,
	}, errs)
	go copyConsoleMessages(ctx, consoleCopyDirection{
		source:            browser,
		target:            upstream,
		targetMessageType: kubeVirtInputMessageType,
	}, errs)

	err := <-errs
	_ = browser.Close()
	_ = upstream.Close()
	return err
}

// copyConsoleMessages copies one WebSocket direction until either peer closes.
// ctx cancels the copy loop when the HTTP request is gone.
// direction provides the source, target, and target frame type mapping.
// errs receives the first read, write, or cancellation error for bridgeConsole.
func copyConsoleMessages(ctx context.Context, direction consoleCopyDirection, errs chan<- error) {
	for {
		select {
		case <-ctx.Done():
			errs <- ctx.Err()
			return
		default:
		}

		messageType, data, err := direction.source.ReadMessage()
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				errs <- ctxErr
				return
			}
			errs <- err
			return
		}
		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			continue
		}
		if err := direction.target.WriteMessage(direction.targetMessageType(messageType), data); err != nil {
			errs <- err
			return
		}
	}
}

func preserveConsoleMessageType(messageType int) int {
	return messageType
}

// kubeVirtInputMessageType returns the frame type KubeVirt reads as serial input.
// Browser WebSocket string sends arrive as text frames, but KubeVirt's plain
// stream reader consumes binary frames for data written into virt-serial0.
func kubeVirtInputMessageType(_ int) int {
	return websocket.BinaryMessage
}

var _ ConsoleSocket = (*websocket.Conn)(nil)
