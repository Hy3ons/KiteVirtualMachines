package gateway

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"k8s.io/client-go/dynamic"
)

const (
	defaultListenAddress   = ":2222"
	defaultBackendTimeout  = 90 * time.Second
	defaultBackendInterval = 2 * time.Second
	defaultHandshakeWait   = 15 * time.Second
	defaultHostSSHWait     = 5 * time.Second
	channelCloseGrace      = 2 * time.Second
	channelRejectGrace     = 1 * time.Second
	backendTypeVM          = "vm"
	backendTypeHost        = "host"
	startingMessage        = "VirtualMachine is starting sshd server.\n"
)

// ServerConfig contains runtime settings for the Kite gateway server.
// ListenAddress is the TCP address the SSH server binds to, for example ":2222".
// HostKeyPath optionally points to a PEM private key used as the SSH server host key.
// BackendTimeout controls how long one authenticated connection waits for the VM sshd.
// BackendRetryInterval controls the wait between VM sshd dial attempts.
// HandshakeTimeout limits how long a raw TCP client may take to complete SSH handshake.
// LoginBanner is shown by SSH clients before authentication when it is not empty.
// HostFallbackEnabled allows non-Kite SSH users to be proxied to the host sshd.
// HostFallbackAddress is the host sshd address, usually the node IP on port 2222.
// HostFallbackTimeout controls password auth timeout when checking the host sshd.
type ServerConfig struct {
	ListenAddress        string
	HostKeyPath          string
	BackendTimeout       time.Duration
	BackendRetryInterval time.Duration
	HandshakeTimeout     time.Duration
	LoginBanner          string
	HostFallbackEnabled  bool
	HostFallbackAddress  string
	HostFallbackTimeout  time.Duration
}

type pendingHostBackend struct {
	client *ssh.Client
	timer  *time.Timer
}

// Server terminates external SSH connections and proxies them to Kite VM SSH Services.
// It reads routes from RouteTable and Kubernetes Secret/Service state through dynamicClient.
type Server struct {
	config              ServerConfig
	dynamicClient       dynamic.Interface
	routes              *RouteTable
	sshConfig           *ssh.ServerConfig
	pendingHostBackends map[string]pendingHostBackend
	hostBackendMu       sync.Mutex
}

// NewServer creates a Kite gateway server.
// config defines listener and SSH host-key behavior.
// dynamicClient reads VM-owned Service and Secret resources.
// routes resolves external SSH usernames to KiteVirtualMachine targets.
func NewServer(config ServerConfig, dynamicClient dynamic.Interface, routes *RouteTable) (*Server, error) {
	if dynamicClient == nil {
		return nil, errors.New("dynamic client is required")
	}
	if routes == nil {
		return nil, errors.New("route table is required")
	}
	if config.ListenAddress == "" {
		config.ListenAddress = defaultListenAddress
	}
	if config.BackendTimeout <= 0 {
		config.BackendTimeout = defaultBackendTimeout
	}
	if config.BackendRetryInterval <= 0 {
		config.BackendRetryInterval = defaultBackendInterval
	}
	if config.HandshakeTimeout <= 0 {
		config.HandshakeTimeout = defaultHandshakeWait
	}
	if config.HostFallbackTimeout <= 0 {
		config.HostFallbackTimeout = defaultHostSSHWait
	}
	config.HostFallbackAddress = strings.TrimSpace(config.HostFallbackAddress)
	if config.HostFallbackEnabled && config.HostFallbackAddress == "" {
		return nil, errors.New("host fallback address is required when host fallback is enabled")
	}

	signer, err := loadOrGenerateHostSigner(config.HostKeyPath)
	if err != nil {
		return nil, err
	}

	server := &Server{
		config:              config,
		dynamicClient:       dynamicClient,
		routes:              routes,
		pendingHostBackends: make(map[string]pendingHostBackend),
	}
	sshConfig := &ssh.ServerConfig{
		NoClientAuth: false,
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			route, err := routes.AuthenticatePassword(conn.User(), password)
			if err == nil {
				return &ssh.Permissions{
					Extensions: map[string]string{
						"backend":     backendTypeVM,
						"username":    route.Username,
						"vmNamespace": route.VMNamespace,
						"vmName":      route.VMName,
					},
				}, nil
			}

			if errors.Is(err, ErrRouteNotFound) && server.config.HostFallbackEnabled {
				hostBackend, hostErr := dialPasswordSSH(server.config.HostFallbackAddress, conn.User(), password, server.config.HostFallbackTimeout)
				if hostErr == nil {
					server.storePendingHostBackend(conn.SessionID(), hostBackend)
					return &ssh.Permissions{
						Extensions: map[string]string{
							"backend":  backendTypeHost,
							"username": conn.User(),
						},
					}, nil
				}
				log.Printf("host SSH fallback rejected for user=%s remote=%s: %v", conn.User(), conn.RemoteAddr(), hostErr)
			}

			log.Printf("SSH auth rejected for user=%s remote=%s: %v", conn.User(), conn.RemoteAddr(), err)
			return nil, err
		},
		ServerVersion: "SSH-2.0-kite-gateway",
	}
	if config.LoginBanner != "" {
		sshConfig.BannerCallback = func(_ ssh.ConnMetadata) string {
			return config.LoginBanner
		}
	}
	sshConfig.AddHostKey(signer)
	server.sshConfig = sshConfig
	return server, nil
}

// ListenAndServe starts accepting SSH connections until ctx is cancelled.
// ctx is used to close the listener during shutdown.
// A nil error means the server stopped because ctx was cancelled.
func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.config.ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.ListenAddress, err)
	}
	defer listener.Close()

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	log.Printf("kite-gateway listening on %s", s.config.ListenAddress)
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("failed to accept SSH connection: %v", err)
			continue
		}
		go s.handleConnection(ctx, conn)
	}
}

// handleConnection completes one external SSH connection and proxies it to a VM.
// ctx controls backend dial cancellation.
// tcpConn is the raw client TCP connection accepted by ListenAndServe.
// This method is used per accepted SSH client connection.
func (s *Server) handleConnection(ctx context.Context, tcpConn net.Conn) {
	defer tcpConn.Close()
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = tcpConn.Close()
		case <-done:
		}
	}()
	defer close(done)

	_ = tcpConn.SetDeadline(time.Now().Add(s.config.HandshakeTimeout))
	serverConn, chans, reqs, err := ssh.NewServerConn(tcpConn, s.sshConfig)
	if err != nil {
		log.Printf("failed SSH handshake from %s: %v", tcpConn.RemoteAddr(), err)
		return
	}
	_ = tcpConn.SetDeadline(time.Time{})
	defer serverConn.Close()
	go ssh.DiscardRequests(reqs)

	if serverConn.Permissions != nil && serverConn.Permissions.Extensions["backend"] == backendTypeHost {
		backend, ok := s.takePendingHostBackend(serverConn.SessionID())
		if !ok {
			log.Printf("authenticated host fallback user has no pending backend user=%s remote=%s", serverConn.User(), serverConn.RemoteAddr())
			rejectAllChannels(chans, "Host SSH session is not available.\n")
			return
		}
		defer backend.Close()
		s.proxyBackendChannels(chans, backend, "host", serverConn.User(), "", "")
		return
	}

	route, err := s.routes.Get(serverConn.User())
	if err != nil {
		log.Printf("authenticated user has no current route user=%s remote=%s: %v", serverConn.User(), serverConn.RemoteAddr(), err)
		rejectAllChannels(chans, "KiteVirtualMachine route is not available.\n")
		return
	}

	backend, err := s.dialBackend(ctx, route)
	if err != nil {
		log.Printf("failed to dial backend for user=%s vm=%s/%s: %v", route.Username, route.VMNamespace, route.VMName, err)
		rejectAllChannels(chans, startingMessage)
		return
	}
	defer backend.Close()

	s.proxyBackendChannels(chans, backend, "vm", route.Username, route.VMNamespace, route.VMName)
}

// proxyBackendChannels forwards all SSH channels from one authenticated frontend connection.
// chans is the frontend channel stream returned by ssh.NewServerConn.
// backend is an already-authenticated SSH client, either a Kite VM or host sshd fallback.
// backendKind is used only for logs and should be "vm" or "host".
// username, namespace, and name identify the route for debugging.
// This method waits until the frontend channel stream closes and all accepted channel proxies return.
func (s *Server) proxyBackendChannels(chans <-chan ssh.NewChannel, backend *ssh.Client, backendKind string, username string, namespace string, name string) {
	var wg sync.WaitGroup
	for newChannel := range chans {
		wg.Add(1)
		go func(ch ssh.NewChannel) {
			defer wg.Done()
			if err := proxyChannel(ch, backend); err != nil {
				log.Printf("failed to proxy SSH channel backend=%s user=%s target=%s/%s: %v", backendKind, username, namespace, name, err)
			}
		}(newChannel)
	}
	wg.Wait()
}

// dialBackend waits until the target VM SSH Service, key Secret, and sshd are reachable.
// ctx controls Kubernetes API calls and retry cancellation.
// route identifies the VM namespace, Service, Secret, and internal SSH username.
// The returned client is an authenticated SSH client connected to the VM sshd.
func (s *Server) dialBackend(ctx context.Context, route Route) (*ssh.Client, error) {
	deadline := time.Now().Add(s.config.BackendTimeout)
	var lastErr error
	for {
		if err := EnsureServiceExists(ctx, s.dynamicClient, route); err != nil {
			lastErr = err
		} else {
			privateKey, err := ReadPrivateKey(ctx, s.dynamicClient, route)
			if err != nil {
				lastErr = err
			} else if client, err := dialSSH(route, privateKey); err == nil {
				return client, nil
			} else {
				lastErr = err
			}
		}

		if time.Now().After(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(s.config.BackendRetryInterval):
		}
	}
	return nil, lastErr
}

// dialSSH opens an SSH client connection to the VM access Service.
// route provides the target Service DNS name and VM Linux username.
// privateKey is the Kite-managed private key matching the VM cloud-init public key.
// The returned client is used to open backend channels for external SSH sessions.
func dialSSH(route Route, privateKey string) (*ssh.Client, error) {
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse VM SSH private key: %w", err)
	}
	config := &ssh.ClientConfig{
		User: route.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	client, err := ssh.Dial("tcp", route.TargetAddress(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect VM SSH target %s: %w", route.TargetAddress(), err)
	}
	return client, nil
}

// dialPasswordSSH opens an SSH client connection with username/password auth.
// address is the target SSH server address, for example "10.0.0.1:2222".
// username and password come from the external SSH password callback.
// timeout bounds host fallback authentication.
// This function is used only for host sshd fallback when a username is not a Kite VM route.
func dialPasswordSSH(address string, username string, password []byte, timeout time.Duration) (*ssh.Client, error) {
	if strings.TrimSpace(address) == "" {
		return nil, errors.New("host SSH fallback address is not configured")
	}
	config := &ssh.ClientConfig{
		User: strings.TrimSpace(username),
		Auth: []ssh.AuthMethod{
			ssh.Password(string(password)),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate host sshd %s: %w", address, err)
	}
	return client, nil
}

// storePendingHostBackend keeps one authenticated host sshd backend until SSH handshake completes.
// sessionID identifies the frontend SSH session that authenticated through host fallback.
// client is closed automatically if the frontend disconnects before handleConnection takes it.
// This method is used by PasswordCallback because the password is not available after handshake.
func (s *Server) storePendingHostBackend(sessionID []byte, client *ssh.Client) {
	key := sessionKey(sessionID)
	timer := time.AfterFunc(s.config.HandshakeTimeout, func() {
		s.hostBackendMu.Lock()
		pending, ok := s.pendingHostBackends[key]
		if ok {
			delete(s.pendingHostBackends, key)
		}
		s.hostBackendMu.Unlock()
		if ok {
			_ = pending.client.Close()
		}
	})

	s.hostBackendMu.Lock()
	if previous, ok := s.pendingHostBackends[key]; ok {
		previous.timer.Stop()
		_ = previous.client.Close()
	}
	s.pendingHostBackends[key] = pendingHostBackend{
		client: client,
		timer:  timer,
	}
	s.hostBackendMu.Unlock()
}

// takePendingHostBackend returns and removes one authenticated host sshd backend.
// sessionID must match the SSH session that passed PasswordCallback.
// The boolean return is false when the backend expired or was already consumed.
// This method is used by handleConnection immediately after ssh.NewServerConn succeeds.
func (s *Server) takePendingHostBackend(sessionID []byte) (*ssh.Client, bool) {
	key := sessionKey(sessionID)

	s.hostBackendMu.Lock()
	pending, ok := s.pendingHostBackends[key]
	if ok {
		delete(s.pendingHostBackends, key)
	}
	s.hostBackendMu.Unlock()
	if !ok {
		return nil, false
	}
	pending.timer.Stop()
	return pending.client, true
}

// sessionKey converts an SSH session id into a stable map key.
// sessionID comes from ssh.ConnMetadata or ssh.ServerConn.
// The returned hex string avoids using raw binary bytes as a map key in logs or debugging.
func sessionKey(sessionID []byte) string {
	return hex.EncodeToString(sessionID)
}

// proxyChannel mirrors one frontend SSH channel to a backend VM SSH channel.
// frontendNewChannel is the channel requested by the external SSH client.
// backend is the authenticated SSH client connected to the VM.
// The returned error describes channel setup failures; stream copy failures are intentionally best-effort.
func proxyChannel(frontendNewChannel ssh.NewChannel, backend *ssh.Client) error {
	frontend, frontendRequests, err := frontendNewChannel.Accept()
	if err != nil {
		return fmt.Errorf("failed to accept frontend channel: %w", err)
	}
	defer frontend.Close()

	backendChannel, backendRequests, err := backend.OpenChannel(frontendNewChannel.ChannelType(), frontendNewChannel.ExtraData())
	if err != nil {
		_, _ = frontend.Write([]byte("VirtualMachine SSH channel is not ready.\n"))
		return fmt.Errorf("failed to open backend channel: %w", err)
	}
	defer backendChannel.Close()

	backendRequestsDone := make(chan struct{})

	go func() {
		_, _ = io.Copy(backendChannel, frontend)
		_ = backendChannel.CloseWrite()
	}()

	go func() {
		forwardRequests(frontendRequests, backendChannel)
	}()

	go func() {
		defer close(backendRequestsDone)
		forwardRequests(backendRequests, frontend)
	}()

	_, _ = io.Copy(frontend, backendChannel)
	_ = frontend.CloseWrite()
	waitForChannelRequests(backendRequestsDone, channelCloseGrace)
	return nil
}

// waitForChannelRequests gives backend requests such as exit-status a short time to reach the client.
// done is closed by the backend request forwarding goroutine when the backend channel is finished.
// timeout bounds session cleanup so a finished VM shell cannot leave the frontend SSH client hanging.
// This function is used by proxyChannel during session teardown.
func waitForChannelRequests(done <-chan struct{}, timeout time.Duration) {
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

// forwardRequests forwards SSH channel requests from one side to the other.
// requests is the source request stream.
// target is the channel that receives equivalent requests.
// This function preserves request acknowledgements when the source requested a reply.
func forwardRequests(requests <-chan *ssh.Request, target ssh.Channel) {
	for req := range requests {
		ok, err := target.SendRequest(req.Type, req.WantReply, req.Payload)
		if req.WantReply {
			if err != nil {
				_ = req.Reply(false, nil)
			} else {
				_ = req.Reply(ok, nil)
			}
		}
	}
}

// rejectAllChannels accepts and closes pending frontend channels with a user-facing message.
// chans is the stream of client-requested channels after authentication.
// message explains why no VM backend is available.
// This function is used when the VM route disappears or sshd is still starting.
func rejectAllChannels(chans <-chan ssh.NewChannel, message string) {
	timer := time.NewTimer(channelRejectGrace)
	defer timer.Stop()

	for {
		select {
		case newChannel, ok := <-chans:
			if !ok {
				return
			}
			rejectChannel(newChannel, message)
		case <-timer.C:
			return
		}
	}
}

// rejectChannel accepts one SSH channel long enough to write a user-facing error.
// newChannel is a client-requested channel that cannot be backed by a VM session.
// message explains why the gateway is closing the channel.
// This function is used when a route is missing or a VM sshd is still starting.
func rejectChannel(newChannel ssh.NewChannel, message string) {
	channel, requests, err := newChannel.Accept()
	if err != nil {
		_ = newChannel.Reject(ssh.ConnectionFailed, message)
		return
	}
	go ssh.DiscardRequests(requests)
	_, _ = channel.Write([]byte(message))
	_ = channel.Close()
}

// loadOrGenerateHostSigner returns the SSH server host key signer.
// path optionally points to a PEM private key on disk.
// The returned signer is generated ephemerally when no path is configured.
// This function is used during server startup before accepting SSH clients.
func loadOrGenerateHostSigner(path string) (ssh.Signer, error) {
	if path != "" {
		keyData, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("SSH host key %s is not mounted; generating ephemeral host key", path)
				return generateEphemeralHostSigner()
			}
			return nil, fmt.Errorf("failed to read SSH host key %s: %w", path, err)
		}
		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH host key %s: %w", path, err)
		}
		return signer, nil
	}

	return generateEphemeralHostSigner()
}

// generateEphemeralHostSigner creates a temporary SSH server host key.
// It is used when no persistent Secret-backed host key is configured or mounted.
// The returned signer changes after gateway restart, so install scripts should create the Secret for stable clients.
func generateEphemeralHostSigner() (ssh.Signer, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral SSH host key: %w", err)
	}
	keyData := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	return ssh.ParsePrivateKey(keyData)
}
