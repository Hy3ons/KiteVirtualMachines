//go:build ignore

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type options struct {
	mode            string
	host            string
	port            string
	user            string
	wantContains    string
	wantNotContains string
	timeout         time.Duration
}

func main() {
	opts := parseOptions()
	if err := run(opts); err != nil {
		log.Fatal(err)
	}
}

func parseOptions() options {
	timeout := flag.Duration("timeout", 15*time.Second, "maximum time for one SSH probe")
	opts := options{
		timeout: *timeout,
	}
	flag.StringVar(&opts.mode, "mode", "", "probe mode: banner or password")
	flag.StringVar(&opts.host, "host", "", "SSH host to probe")
	flag.StringVar(&opts.port, "port", "", "SSH port to probe")
	flag.StringVar(&opts.user, "user", "", "SSH username for password mode")
	flag.StringVar(&opts.wantContains, "want-contains", "", "substring required in banner mode")
	flag.StringVar(&opts.wantNotContains, "want-not-contains", "", "substring forbidden in banner mode")
	flag.Parse()
	opts.timeout = *timeout
	return opts
}

func run(opts options) error {
	if opts.host == "" || opts.port == "" {
		return fmt.Errorf("host and port are required")
	}

	switch opts.mode {
	case "banner":
		return checkBanner(opts)
	case "password":
		return checkPasswordLogin(opts)
	default:
		return fmt.Errorf("unknown mode %q; use banner or password", opts.mode)
	}
}

func checkBanner(opts options) error {
	address := net.JoinHostPort(opts.host, opts.port)
	conn, err := net.DialTimeout("tcp", address, opts.timeout)
	if err != nil {
		return fmt.Errorf("connect %s: %w", address, err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Printf("close %s: %v", address, closeErr)
		}
	}()

	if err := conn.SetDeadline(time.Now().Add(opts.timeout)); err != nil {
		return fmt.Errorf("set deadline %s: %w", address, err)
	}
	banner, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return fmt.Errorf("read SSH banner from %s: %w", address, err)
	}
	banner = strings.TrimSpace(banner)
	if !strings.HasPrefix(banner, "SSH-") {
		return fmt.Errorf("unexpected SSH banner from %s: %q", address, banner)
	}
	if opts.wantContains != "" && !strings.Contains(banner, opts.wantContains) {
		return fmt.Errorf("banner from %s is %q; expected it to contain %q", address, banner, opts.wantContains)
	}
	if opts.wantNotContains != "" && strings.Contains(banner, opts.wantNotContains) {
		return fmt.Errorf("banner from %s is %q; expected it not to contain %q", address, banner, opts.wantNotContains)
	}
	fmt.Println(banner)
	return nil
}

func checkPasswordLogin(opts options) error {
	if opts.user == "" {
		return fmt.Errorf("user is required in password mode")
	}
	password := os.Getenv("TEST_HOST_SSH_PASSWORD")
	if password == "" {
		return fmt.Errorf("TEST_HOST_SSH_PASSWORD is required in password mode")
	}

	address := net.JoinHostPort(opts.host, opts.port)
	config := &ssh.ClientConfig{
		User: opts.user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         opts.timeout,
	}
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return fmt.Errorf("password SSH login %s@%s: %w", opts.user, address, err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("close SSH client %s: %v", address, closeErr)
		}
	}()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("open SSH session %s@%s: %w", opts.user, address, err)
	}
	defer func() {
		if closeErr := session.Close(); closeErr != nil && closeErr.Error() != "EOF" {
			log.Printf("close SSH session %s: %v", address, closeErr)
		}
	}()

	if err := session.Run("true"); err != nil {
		return fmt.Errorf("run SSH smoke command %s@%s: %w", opts.user, address, err)
	}
	fmt.Printf("password login ok: %s@%s\n", opts.user, address)
	return nil
}
