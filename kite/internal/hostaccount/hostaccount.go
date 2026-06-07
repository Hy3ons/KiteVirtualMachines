package hostaccount

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	hostproxyshell "kite/internal/render/host-proxy-shell"
)

const (
	hostRootDefault   = "/host"
	HostAccountRoot   = "/var/lib/kite/accounts"
	maxSSHPasswordLen = 128
)

var usernamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)

type DesiredAccount struct {
	Username         string
	Password         string
	VMNamespace      string
	VMName           string
	NodeName         string
	SSHKeySecretName string
	PrivateKey       string
	ServiceName      string
	ServiceNamespace string
	VMUser           string
}

type OwnerMetadata struct {
	Username         string `json:"username"`
	VMNamespace      string `json:"vmNamespace"`
	VMName           string `json:"vmName"`
	NodeName         string `json:"nodeName"`
	SSHKeySecretName string `json:"sshKeySecretName"`
	ShellPath        string `json:"shellPath"`
	PrivateKeyPath   string `json:"privateKeyPath"`
}

type Manager struct {
	hostRoot string
}

// NewManager creates a host account manager.
// hostRoot is the container path where the host filesystem is mounted, usually /host.
// The returned manager is used by cmd/kite-host-agent to reconcile Linux users and proxy shells.
func NewManager(hostRoot string) *Manager {
	hostRoot = strings.TrimSpace(hostRoot)
	if hostRoot == "" {
		hostRoot = hostRootDefault
	}

	return &Manager{hostRoot: hostRoot}
}

// Ensure makes one host Linux account match the desired KiteVirtualMachine access state.
// ctx controls nsenter command execution.
// desired contains the VM sshId, sshPassword, private key, and fixed Kubernetes Service DNS target.
// A nil error means the host user, password, private key, shell script, and login shell were reconciled.
func (m *Manager) Ensure(ctx context.Context, desired DesiredAccount) error {
	if err := validateDesiredAccount(desired); err != nil {
		return err
	}

	shellPath := ShellPath(desired.Username)
	privateKeyPath := PrivateKeyPath(desired.Username)
	owner := OwnerMetadata{
		Username:         desired.Username,
		VMNamespace:      desired.VMNamespace,
		VMName:           desired.VMName,
		NodeName:         desired.NodeName,
		SSHKeySecretName: desired.SSHKeySecretName,
		ShellPath:        shellPath,
		PrivateKeyPath:   privateKeyPath,
	}

	if err := m.ensureOwnership(ctx, owner); err != nil {
		return err
	}
	if err := m.ensureUser(ctx, desired.Username, shellPath); err != nil {
		return err
	}
	if err := m.writePrivateKey(ctx, desired.Username, desired.PrivateKey); err != nil {
		return err
	}
	if err := m.writeHushLogin(ctx, desired.Username); err != nil {
		return err
	}
	if err := m.writeProxyShell(ctx, desired, shellPath); err != nil {
		return err
	}
	if err := m.setPassword(ctx, desired.Username, desired.Password); err != nil {
		return err
	}
	if err := m.setShell(ctx, desired.Username, shellPath); err != nil {
		return err
	}
	if err := m.writeOwner(owner); err != nil {
		return err
	}

	return nil
}

// Delete removes a Kite-managed host Linux account and its proxy shell.
// ctx controls nsenter command execution.
// username is spec.sshId from the deleted or delete-intent KiteVirtualMachine.
// vmNamespace and vmName identify the owner metadata that must match before deletion.
// This function refuses to delete non-Kite-owned host users.
func (m *Manager) Delete(ctx context.Context, username string, vmNamespace string, vmName string) error {
	username = strings.TrimSpace(username)
	if !usernamePattern.MatchString(username) {
		return fmt.Errorf("invalid sshId %q", username)
	}

	owner, found, err := m.readOwner(username)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	if owner.VMNamespace != vmNamespace || owner.VMName != vmName {
		return fmt.Errorf("host account %s is owned by %s/%s, not %s/%s", username, owner.VMNamespace, owner.VMName, vmNamespace, vmName)
	}

	if err := m.runHost(ctx, "userdel", "-r", username); err != nil && !isMissingUserError(err) {
		return err
	}

	_ = os.Remove(m.hostPath(owner.ShellPath))
	_ = os.Remove(m.ownerPath(username))
	return nil
}

// ShellPath returns the host-visible proxy shell path for one Kite SSH user.
// username is spec.sshId and maps to /home/<username>/custom-shell.sh.
// The returned path is used as the user's login shell on the host.
func ShellPath(username string) string {
	return filepath.Join("/home", username, "custom-shell.sh")
}

// PrivateKeyPath returns the host-visible private key path for one Kite SSH user.
// username is spec.sshId and maps to /home/<username>/.ssh/id_rsa.
// This path is referenced by the user's custom login shell.
func PrivateKeyPath(username string) string {
	return filepath.Join("/home", username, ".ssh", "id_rsa")
}

// HushLoginPath returns the host-visible path that suppresses login banners for one Kite SSH user.
// username is spec.sshId and maps to /home/<username>/.hushlogin.
// The returned path is reconciled by kite-host-agent so the host SSH hop stays quiet before proxying to the VM.
func HushLoginPath(username string) string {
	return filepath.Join("/home", username, ".hushlogin")
}

func validateDesiredAccount(desired DesiredAccount) error {
	if !usernamePattern.MatchString(strings.TrimSpace(desired.Username)) {
		return fmt.Errorf("sshId must match %s", usernamePattern.String())
	}
	if strings.TrimSpace(desired.Password) == "" {
		return errors.New("sshPassword is required")
	}
	if err := validateSSHPassword(desired.Password); err != nil {
		return err
	}
	if strings.TrimSpace(desired.VMNamespace) == "" || strings.TrimSpace(desired.VMName) == "" {
		return errors.New("VM namespace and name are required")
	}
	if strings.TrimSpace(desired.SSHKeySecretName) == "" {
		return errors.New("SSH key Secret name is required")
	}
	if strings.TrimSpace(desired.PrivateKey) == "" {
		return errors.New("SSH private key is required")
	}
	if strings.TrimSpace(desired.ServiceName) == "" || strings.TrimSpace(desired.ServiceNamespace) == "" {
		return errors.New("SSH Service name and namespace are required")
	}
	return nil
}

func validateSSHPassword(password string) error {
	if password != strings.TrimSpace(password) {
		return errors.New("sshPassword must not start or end with whitespace")
	}
	if len(password) > maxSSHPasswordLen {
		return errors.New("sshPassword must be at most 128 bytes")
	}
	if strings.Contains(password, ":") {
		return errors.New("sshPassword must not contain colon")
	}
	for _, r := range password {
		if r < 0x20 || r == 0x7f {
			return errors.New("sshPassword must not contain control characters")
		}
	}

	return nil
}

// EnsureClusterDNS configures the host resolver for Kubernetes Service DNS names.
// ctx controls nsenter command execution.
// dnsIP is the kube-system/kube-dns ClusterIP discovered by kite-host-agent.
// This function is used before writing proxy shells that target *.svc.cluster.local.
func (m *Manager) EnsureClusterDNS(ctx context.Context, dnsIP string) error {
	dnsIP = strings.TrimSpace(dnsIP)
	if dnsIP == "" {
		return errors.New("cluster DNS IP is required")
	}
	if err := m.runHost(ctx, "sh", "-c", "command -v resolvectl >/dev/null 2>&1"); err != nil {
		return fmt.Errorf("resolvectl is required to configure host cluster DNS: %w", err)
	}

	link, err := m.runHostOutput(ctx, "sh", "-c", "ip route show default 2>/dev/null | awk '{print $5; exit}'")
	if err != nil {
		return err
	}
	link = strings.TrimSpace(link)
	if link == "" {
		return errors.New("failed to detect host default network interface")
	}

	if err := m.runHost(ctx, "resolvectl", "dns", link, dnsIP); err != nil {
		return err
	}
	if err := m.runHost(ctx, "resolvectl", "domain", link, "~cluster.local", "~svc.cluster.local"); err != nil {
		return err
	}
	return m.runHost(ctx, "resolvectl", "default-route", link, "false")
}

func (m *Manager) ensureOwnership(ctx context.Context, owner OwnerMetadata) error {
	current, found, err := m.readOwner(owner.Username)
	if err != nil {
		return err
	}
	if found {
		if current.VMNamespace == owner.VMNamespace && current.VMName == owner.VMName {
			return nil
		}
		return fmt.Errorf("host account %s is already owned by %s/%s", owner.Username, current.VMNamespace, current.VMName)
	}

	if err := m.runHost(ctx, "getent", "passwd", owner.Username); err == nil {
		return fmt.Errorf("host account %s already exists and is not managed by Kite", owner.Username)
	} else if !isMissingUserError(err) {
		return err
	}

	return nil
}

func (m *Manager) ensureUser(ctx context.Context, username string, shellPath string) error {
	if err := m.runHost(ctx, "getent", "passwd", username); err == nil {
		return nil
	} else if !isMissingUserError(err) {
		return err
	}

	return m.runHost(ctx, "useradd", "-m", "-s", shellPath, username)
}

func (m *Manager) setPassword(ctx context.Context, username string, password string) error {
	if strings.HasPrefix(strings.TrimSpace(password), "$") {
		return m.runHostWithInput(ctx, username+":"+password+"\n", "chpasswd", "-e")
	}

	return m.runHostWithInput(ctx, username+":"+password+"\n", "chpasswd")
}

func (m *Manager) setShell(ctx context.Context, username string, shellPath string) error {
	return m.runHost(ctx, "usermod", "-s", shellPath, username)
}

func (m *Manager) writePrivateKey(ctx context.Context, username string, privateKey string) error {
	sshDir := filepath.Join("/home", username, ".ssh")
	if err := os.MkdirAll(m.hostPath(sshDir), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(m.hostPath(PrivateKeyPath(username)), []byte(privateKey), 0o600); err != nil {
		return err
	}
	return m.fixSSHFileOwnership(ctx, username)
}

func (m *Manager) fixSSHFileOwnership(ctx context.Context, username string) error {
	if err := os.Chmod(m.hostPath(filepath.Join("/home", username, ".ssh")), 0o700); err != nil {
		return err
	}
	if err := os.Chmod(m.hostPath(PrivateKeyPath(username)), 0o600); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return m.runHost(ctx, "chown", "-R", username+":"+username, filepath.Join("/home", username, ".ssh"))
}

// writeHushLogin creates the per-user marker that asks OpenSSH/PAM login helpers to skip banners.
// ctx controls the host chown command.
// username is the Kite-managed host account that proxies into one VM.
// This function is used by Ensure so recreated or modified accounts converge back to a quiet login hop.
func (m *Manager) writeHushLogin(ctx context.Context, username string) error {
	containerPath := m.hostPath(HushLoginPath(username))
	if err := os.MkdirAll(filepath.Dir(containerPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(containerPath, nil, 0o644); err != nil {
		return err
	}
	if err := os.Chmod(containerPath, 0o644); err != nil {
		return err
	}
	return m.runHost(ctx, "chown", username+":"+username, HushLoginPath(username))
}

func (m *Manager) writeProxyShell(ctx context.Context, desired DesiredAccount, shellPath string) error {
	vmUser := strings.TrimSpace(desired.VMUser)
	if vmUser == "" {
		vmUser = desired.Username
	}

	rendered, err := hostproxyshell.Render(hostproxyshell.ShellData{
		Username:   desired.Username,
		ServiceDNS: serviceDNS(desired.ServiceName, desired.ServiceNamespace),
		VMUser:     vmUser,
	})
	if err != nil {
		return err
	}

	containerPath := m.hostPath(shellPath)
	if err := os.MkdirAll(filepath.Dir(containerPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(containerPath, []byte(rendered), 0o755); err != nil {
		return err
	}
	if err := os.Chmod(containerPath, 0o755); err != nil {
		return err
	}
	return m.runHost(ctx, "chown", desired.Username+":"+desired.Username, shellPath)
}

func (m *Manager) writeOwner(owner OwnerMetadata) error {
	if err := os.MkdirAll(filepath.Dir(m.ownerPath(owner.Username)), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(owner, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(m.ownerPath(owner.Username), data, 0o600)
}

func (m *Manager) readOwner(username string) (OwnerMetadata, bool, error) {
	data, err := os.ReadFile(m.ownerPath(username))
	if errors.Is(err, os.ErrNotExist) {
		return OwnerMetadata{}, false, nil
	}
	if err != nil {
		return OwnerMetadata{}, false, err
	}

	var owner OwnerMetadata
	if err := json.Unmarshal(data, &owner); err != nil {
		return OwnerMetadata{}, false, err
	}
	return owner, true, nil
}

// ListOwners reads Kite-managed host account metadata files from the host filesystem.
// The returned slice is used by kite-host-agent GC to compare local accounts with cluster state.
func (m *Manager) ListOwners() ([]OwnerMetadata, error) {
	entries, err := os.ReadDir(m.hostPath(HostAccountRoot))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	owners := make([]OwnerMetadata, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(m.hostPath(HostAccountRoot), entry.Name()))
		if err != nil {
			return nil, err
		}

		var owner OwnerMetadata
		if err := json.Unmarshal(data, &owner); err != nil {
			return nil, err
		}
		owners = append(owners, owner)
	}

	return owners, nil
}

func (m *Manager) ownerPath(username string) string {
	return filepath.Join(m.hostPath(HostAccountRoot), username+".json")
}

func (m *Manager) hostPath(path string) string {
	return filepath.Join(m.hostRoot, strings.TrimPrefix(path, "/"))
}

func (m *Manager) runHost(ctx context.Context, name string, args ...string) error {
	return m.runHostWithInput(ctx, "", name, args...)
}

func (m *Manager) runHostWithInput(ctx context.Context, input string, name string, args ...string) error {
	nsenterArgs := append([]string{"-t", "1", "-m", "-u", "-i", "-n", "-p", "--", name}, args...)
	cmd := exec.CommandContext(ctx, "nsenter", nsenterArgs...)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("host command %s failed: %s", name, message)
	}
	return nil
}

func (m *Manager) runHostOutput(ctx context.Context, name string, args ...string) (string, error) {
	nsenterArgs := append([]string{"-t", "1", "-m", "-u", "-i", "-n", "-p", "--", name}, args...)
	cmd := exec.CommandContext(ctx, "nsenter", nsenterArgs...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("host command %s failed: %s", name, message)
	}
	return string(output), nil
}

func isMissingUserError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "not found") ||
		strings.Contains(message, "no such user") ||
		strings.Contains(message, "does not exist") ||
		strings.Contains(message, "exit status 2")
}

func safePathPart(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteByte('-')
	}
	if builder.Len() == 0 {
		return "unknown"
	}
	return builder.String()
}

func serviceDNS(serviceName string, namespace string) string {
	return strings.TrimSpace(serviceName) + "." + strings.TrimSpace(namespace) + ".svc.cluster.local"
}
