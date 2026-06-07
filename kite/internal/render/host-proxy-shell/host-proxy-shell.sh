#!/bin/bash
PRIVATE_KEY="/home/{{ .Username }}/.ssh/id_rsa"
VM_TARGET="{{ .VMUser }}@{{ .ServiceDNS }}"

unset LC_ALL
export LANG=C.UTF-8

exec ssh -i "$PRIVATE_KEY" \
  -o StrictHostKeyChecking=no \
  -o UserKnownHostsFile=/dev/null \
  -o LogLevel=ERROR \
  "$VM_TARGET" "$@"
