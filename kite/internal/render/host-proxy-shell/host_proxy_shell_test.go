package hostproxyshell

import (
	"strings"
	"testing"
)

// TestRenderIncludesQuietRetryingSSH verifies the host proxy shell stays quiet and waits for VM SSH readiness.
// The rendered shell is written by kite-host-agent as the managed Linux user's login shell.
func TestRenderIncludesQuietRetryingSSH(t *testing.T) {
	rendered, err := Render(ShellData{
		Username:   "asdf",
		ServiceDNS: "vps-access-asdf.kite-user.svc.cluster.local",
		VMUser:     "asdf",
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	required := []string{
		"#!/bin/sh",
		"unset LC_ALL",
		"export LANG=C.UTF-8",
		`RETRY_SECONDS="${KITE_PROXY_RETRY_SECONDS:-90}"`,
		`STARTING_MESSAGE="${KITE_PROXY_STARTING_MESSAGE:-VirtualMachine is starting sshd server.}"`,
		"-o BatchMode=yes",
		"-o ConnectTimeout=3",
		"-o ConnectionAttempts=1",
		"-o LogLevel=ERROR",
		`"$VM_TARGET" true >/dev/null 2>&1`,
		`printf '%s\n' "$STARTING_MESSAGE"`,
		"exit 75",
		`"$VM_TARGET" "$@"`,
	}
	for _, fragment := range required {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("rendered shell does not contain %q:\n%s", fragment, rendered)
		}
	}
}
