#!/bin/bash
PRIVATE_KEY="/home/{{ .Username }}/.ssh/id_rsa"
VM_TARGET="{{ .VMUser }}@{{ .ServiceDNS }}"

exec ssh -i "$PRIVATE_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "$VM_TARGET" "$@"
