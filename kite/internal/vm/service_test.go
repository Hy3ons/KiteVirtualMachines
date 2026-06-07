package vm

import "testing"

func TestCreateRecordRejectsUnsafeSSHPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{name: "leading space", password: " password"},
		{name: "trailing space", password: "password "},
		{name: "newline", password: "pass\nword"},
		{name: "carriage return", password: "pass\rword"},
		{name: "colon", password: "pass:word"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := createRecord(CreateRequest{
				Namespace:   "tenant-a",
				Disk:        "20Gi",
				SSHID:       "asdf",
				SSHPassword: tt.password,
			})
			if err == nil {
				t.Fatalf("expected unsafe sshPassword %q to be rejected", tt.password)
			}
		})
	}
}

func TestCreateRecordAcceptsSafeSSHPassword(t *testing.T) {
	record, err := createRecord(CreateRequest{
		Namespace:   "tenant-a",
		Disk:        "20Gi",
		SSHID:       "asdf",
		SSHPassword: "pass word_123!",
	})
	if err != nil {
		t.Fatalf("expected safe sshPassword to be accepted, got %v", err)
	}
	if record.Spec.SSHPassword != "pass word_123!" {
		t.Fatalf("unexpected sshPassword %q", record.Spec.SSHPassword)
	}
}
