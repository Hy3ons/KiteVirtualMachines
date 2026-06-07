#!/bin/sh
PRIVATE_KEY="/home/{{ .Username }}/.ssh/id_rsa"
VM_TARGET="{{ .VMUser }}@{{ .ServiceDNS }}"
RETRY_SECONDS="${KITE_PROXY_RETRY_SECONDS:-90}"
RETRY_INTERVAL_SECONDS="${KITE_PROXY_RETRY_INTERVAL_SECONDS:-2}"
STARTING_MESSAGE="${KITE_PROXY_STARTING_MESSAGE:-VirtualMachine is starting sshd server.}"

unset LC_ALL
export LANG=C.UTF-8

deadline="$(( $(date +%s) + RETRY_SECONDS ))"
ready="false"
message_printed="false"
while ! ssh -i "$PRIVATE_KEY" \
  -o BatchMode=yes \
  -o ConnectTimeout=3 \
  -o ConnectionAttempts=1 \
  -o StrictHostKeyChecking=no \
  -o UserKnownHostsFile=/dev/null \
  -o LogLevel=ERROR \
  "$VM_TARGET" true >/dev/null 2>&1; do
  if [ "$(date +%s)" -ge "$deadline" ]; then
    break
  fi
  if [ "$message_printed" != "true" ]; then
    printf '%s\n' "$STARTING_MESSAGE"
    message_printed="true"
  fi
  sleep "$RETRY_INTERVAL_SECONDS"
done

if ssh -i "$PRIVATE_KEY" \
  -o BatchMode=yes \
  -o ConnectTimeout=3 \
  -o ConnectionAttempts=1 \
  -o StrictHostKeyChecking=no \
  -o UserKnownHostsFile=/dev/null \
  -o LogLevel=ERROR \
  "$VM_TARGET" true >/dev/null 2>&1; then
  ready="true"
fi

if [ "$ready" != "true" ]; then
  if [ "$message_printed" != "true" ]; then
    printf '%s\n' "$STARTING_MESSAGE"
  fi
  exit 75
fi

exec ssh -i "$PRIVATE_KEY" \
  -o ConnectTimeout=10 \
  -o ConnectionAttempts=1 \
  -o StrictHostKeyChecking=no \
  -o UserKnownHostsFile=/dev/null \
  -o LogLevel=ERROR \
  "$VM_TARGET" "$@"
