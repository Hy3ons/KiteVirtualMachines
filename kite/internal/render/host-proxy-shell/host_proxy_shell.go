package hostproxyshell

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed host-proxy-shell.sh
var hostProxyShellTemplate string

type ShellData struct {
	Username   string
	ServiceDNS string
	VMUser     string
}

// Render renders the host login shell used to proxy SSH into one VM Service.
// data provides the host username and fixed Kubernetes Service DNS read by kite-host-agent.
// The returned string is written to the Kite-managed user's home directory on the host filesystem.
// This renderer is used by cmd/kite-host-agent when reconciling Linux login shells.
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
