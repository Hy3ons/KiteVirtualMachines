package cloudinituserdata

import (
	"strings"
	"testing"
)

func TestUbuntu2204CloudInitSetsPasswordThroughBase64Script(t *testing.T) {
	obj, err := (&Ubuntu2204CloudInit{
		VmName:       "vm-a",
		Namespace:    "tenant-a",
		Id:           "asdf",
		Password:     "asdf",
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

	if strings.Contains(userdata, "asdf: asdf") {
		t.Fatalf("cloud-init userdata should not render chpasswd with a space after colon:\n%s", userdata)
	}
	if !strings.Contains(userdata, "password=\"$(printf '%s' 'YXNkZg==' | base64 -d)\"") {
		t.Fatalf("cloud-init userdata should decode password from base64:\n%s", userdata)
	}
	if !strings.Contains(userdata, "printf 'root:%s\\nasdf:%s\\n'") {
		t.Fatalf("cloud-init userdata should set root and VM user passwords through chpasswd:\n%s", userdata)
	}
}
