package gateway

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

// TestNewServerSetsLoginBannerCallback verifies configured login banners reach SSH authentication.
// t is the Go test handle used for assertions.
// The test protects the operator-facing message shown before the password prompt.
func TestNewServerSetsLoginBannerCallback(t *testing.T) {
	server := newServerForBannerTest(t, "This server is Kite Gateway.\n")

	if server.sshConfig.BannerCallback == nil {
		t.Fatal("expected login banner callback")
	}
	if banner := server.sshConfig.BannerCallback(nil); banner != "This server is Kite Gateway.\n" {
		t.Fatalf("unexpected login banner %q", banner)
	}
}

// TestNewServerSkipsEmptyLoginBanner verifies empty banner config keeps SSH auth quiet.
// t is the Go test handle used for assertions.
// The test lets operators disable the pre-authentication message by clearing the env value.
func TestNewServerSkipsEmptyLoginBanner(t *testing.T) {
	server := newServerForBannerTest(t, "")

	if server.sshConfig.BannerCallback != nil {
		t.Fatal("expected empty login banner to skip callback")
	}
}

// TestServerSendsLoginBannerBeforeAuthentication verifies real SSH clients receive the banner.
// t is the Go test handle used for assertions.
// The test drives the gateway through an SSH handshake before password auth fails.
func TestServerSendsLoginBannerBeforeAuthentication(t *testing.T) {
	banner := "This server is Kite Gateway.\nUse your Kite VM sshId to connect to a VM.\n"
	address := freeLoopbackAddress(t)
	server := newServerWithConfigForBannerTest(t, ServerConfig{
		ListenAddress: address,
		LoginBanner:   banner,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe(ctx)
	}()

	received := make(chan string, 1)
	clientConfig := &ssh.ClientConfig{
		User:            "missing-user",
		Auth:            []ssh.AuthMethod{ssh.Password("bad-password")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         250 * time.Millisecond,
		BannerCallback: func(message string) error {
			select {
			case received <- message:
			default:
			}
			return nil
		},
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		client, err := ssh.Dial("tcp", address, clientConfig)
		if err == nil {
			_ = client.Close()
		}
		select {
		case got := <-received:
			if got != banner {
				t.Fatalf("unexpected login banner %q", got)
			}
			cancel()
			if listenErr := <-errCh; listenErr != nil {
				t.Fatalf("gateway listener stopped with error: %v", listenErr)
			}
			return
		case listenErr := <-errCh:
			t.Fatalf("gateway listener stopped before banner test completed: %v", listenErr)
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}
	t.Fatal("timed out waiting for SSH login banner")
}

// TestServerRejectsUnknownUserWithoutHostProxy verifies missing VM routes fail at the gateway.
// t is the Go test handle used for assertions.
// The test protects the policy that kite-gateway never proxies host Linux accounts.
func TestServerRejectsUnknownUserWithoutHostProxy(t *testing.T) {
	hostListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start host proxy probe listener: %v", err)
	}
	defer hostListener.Close()

	hostDialed := make(chan struct{}, 1)
	go func() {
		conn, acceptErr := hostListener.Accept()
		if acceptErr != nil {
			return
		}
		_ = conn.Close()
		hostDialed <- struct{}{}
	}()

	address := freeLoopbackAddress(t)
	server := newServerWithConfigForBannerTest(t, ServerConfig{ListenAddress: address})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe(ctx)
	}()

	clientConfig := &ssh.ClientConfig{
		User:            "host-linux-user",
		Auth:            []ssh.AuthMethod{ssh.Password("host-password")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         250 * time.Millisecond,
	}

	err = dialUntilAuthRejected(address, clientConfig, 3*time.Second)
	if err == nil {
		t.Fatal("expected unknown SSH username to be rejected")
	}
	if !strings.Contains(err.Error(), "unable to authenticate") {
		t.Fatalf("expected authentication failure, got %v", err)
	}

	select {
	case <-hostDialed:
		t.Fatal("gateway dialed a host listener for an unknown VM route")
	case <-time.After(150 * time.Millisecond):
	}

	cancel()
	if listenErr := <-errCh; listenErr != nil {
		t.Fatalf("gateway listener stopped with error: %v", listenErr)
	}
}

func newServerForBannerTest(t *testing.T, banner string) *Server {
	t.Helper()

	return newServerWithConfigForBannerTest(t, ServerConfig{LoginBanner: banner})
}

func newServerWithConfigForBannerTest(t *testing.T, config ServerConfig) *Server {
	t.Helper()

	server, err := NewServer(config, fake.NewSimpleDynamicClient(runtime.NewScheme()), NewRouteTable("banner-test-salt"))
	if err != nil {
		t.Fatalf("failed to create gateway server: %v", err)
	}
	return server
}

func dialUntilAuthRejected(address string, clientConfig *ssh.ClientConfig, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		client, err := ssh.Dial("tcp", address, clientConfig)
		if err == nil {
			_ = client.Close()
			return nil
		}
		lastErr = err
		if strings.Contains(err.Error(), "unable to authenticate") {
			return err
		}
		time.Sleep(20 * time.Millisecond)
	}
	return lastErr
}

func freeLoopbackAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate loopback address: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("failed to release loopback address: %v", err)
	}
	return address
}
