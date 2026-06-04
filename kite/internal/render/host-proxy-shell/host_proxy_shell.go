package hostproxyshell

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed host-proxy-shell.sh
var hostProxyShellTemplate string

type ShellData struct {
	ClusterIP string
	Port      int64
}

// Render renders the host login shell used to proxy SSH into one VM Service.
// data provides the Service ClusterIP and SSH port read by kite-account.
// The returned string is written to /var/lib/kite/bashs on the host filesystem.
// This renderer is used by cmd/kite-account when reconciling Linux login shells.
func Render(data ShellData) (string, error) {
	tmpl, err := template.New("host-proxy-shell.sh").Parse(hostProxyShellTemplate)
	if err != nil {
		return "", err
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return "", err
	}

	return output.String(), nil
}
