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
	hostRootDefault = "/host"
	HostShellRoot   = "/var/lib/kite/bashs"
	HostAccountRoot = "/var/lib/kite/accounts"
)

var usernamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)

type DesiredAccount struct {
	Username    string
	Password    string
	VMNamespace string
	VMName      string
	ClusterIP   string
	Port        int64
}

type OwnerMetadata struct {
	Username    string `json:"username"`
	VMNamespace string `json:"vmNamespace"`
	VMName      string `json:"vmName"`
	ShellPath   string `json:"shellPath"`
}

type Manager struct {
	hostRoot string
}

// NewManager creates a host account manager.
// hostRoot is the container path where the host filesystem is mounted, usually /host.
// The returned manager is used by cmd/kite-account to reconcile Linux users and proxy shells.
func NewManager(hostRoot string) *Manager {
	hostRoot = strings.TrimSpace(hostRoot)
	if hostRoot == "" {
		hostRoot = hostRootDefault
	}

	return &Manager{hostRoot: hostRoot}
}

// Ensure makes one host Linux account match the desired KiteVirtualMachine access state.
// ctx controls nsenter command execution.
// desired contains the VM sshId, sshPassword, and ClusterIP Service target.
// A nil error means the host user, password, shell script, and login shell were reconciled.
func (m *Manager) Ensure(ctx context.Context, desired DesiredAccount) error {
	if err := validateDesiredAccount(desired); err != nil {
		return err
	}

	shellPath := ShellPath(desired.VMNamespace, desired.VMName)
	owner := OwnerMetadata{
		Username:    desired.Username,
		VMNamespace: desired.VMNamespace,
		VMName:      desired.VMName,
		ShellPath:   shellPath,
	}

	if err := m.ensureOwnership(ctx, owner); err != nil {
		return err
	}
	if err := m.writeProxyShell(desired, shellPath); err != nil {
		return err
	}
	if err := m.ensureUser(ctx, desired.Username, shellPath); err != nil {
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

// ShellPath returns the host-visible proxy shell path for one VM.
// vmNamespace and vmName identify the KiteVirtualMachine.
// The returned path is used as the user's login shell on the host.
func ShellPath(vmNamespace string, vmName string) string {
	return filepath.Join(HostShellRoot, safePathPart(vmNamespace)+"-"+safePathPart(vmName)+".sh")
}

func validateDesiredAccount(desired DesiredAccount) error {
	if !usernamePattern.MatchString(strings.TrimSpace(desired.Username)) {
		return fmt.Errorf("sshId must match %s", usernamePattern.String())
	}
	if strings.TrimSpace(desired.Password) == "" {
		return errors.New("sshPassword is required")
	}
	if strings.TrimSpace(desired.VMNamespace) == "" || strings.TrimSpace(desired.VMName) == "" {
		return errors.New("VM namespace and name are required")
	}
	if strings.TrimSpace(desired.ClusterIP) == "" || strings.TrimSpace(desired.ClusterIP) == "None" {
		return errors.New("SSH Service ClusterIP is required")
	}
	if desired.Port <= 0 {
		return errors.New("SSH Service port is required")
	}
	return nil
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
	return m.runHostWithInput(ctx, username+":"+password+"\n", "chpasswd")
}

func (m *Manager) setShell(ctx context.Context, username string, shellPath string) error {
	return m.runHost(ctx, "usermod", "-s", shellPath, username)
}

func (m *Manager) writeProxyShell(desired DesiredAccount, shellPath string) error {
	rendered, err := hostproxyshell.Render(hostproxyshell.ShellData{
		ClusterIP: desired.ClusterIP,
		Port:      desired.Port,
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
	return os.Chmod(containerPath, 0o755)
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
