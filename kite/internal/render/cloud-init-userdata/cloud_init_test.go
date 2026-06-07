package cloudinituserdata

import (
	"strings"
	"testing"
)

func TestUbuntu2204CloudInitUsesKeyOnlySSHAccess(t *testing.T) {
	obj, err := (&Ubuntu2204CloudInit{
		VmName:       "vm-a",
		Namespace:    "tenant-a",
		Id:           "asdf",
		SSHPublicKey: "ssh-rsa test",
	}).Render()
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	stringData, ok := obj.Object["stringData"].(map[string]any)
	if !ok {
		t.Fatalf("expected stringData in rendered cloud-init Secret")
	}
	userdata, ok := stringData["userdata"].(string)
	if !ok {
		t.Fatalf("expected stringData.userdata to be a string")
	}

	if strings.Contains(userdata, "chpasswd") || strings.Contains(userdata, "kite-set-password") {
		t.Fatalf("cloud-init userdata should not set VM passwords:\n%s", userdata)
	}
	if !strings.Contains(userdata, "PasswordAuthentication no") {
		t.Fatalf("cloud-init userdata should disable VM password SSH:\n%s", userdata)
	}
	if !strings.Contains(userdata, "ssh-rsa test") {
		t.Fatalf("cloud-init userdata should include Kite-managed SSH public key:\n%s", userdata)
	}
}
