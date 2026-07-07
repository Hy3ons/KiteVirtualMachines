# VMware SSH Gateway Verification Checklist

이 문서는 `fix/safe-ssh-gateway-exposure` 작업을 실제 VMware/k3s 환경에서 검증할 때의 실행 순서다. 이번 구현 라운드에서는 실행하지 않고, 정적 검증만 수행한다.

## Fresh Install Safety

| 항목 | 확인 기준 |
| --- | --- |
| Host SSH 유지 | `ghcr-install.sh` 또는 `build-install.sh` 직후 host sshd 22번 접속이 그대로 유지된다. |
| Internal gateway only | `svc/kite-gateway`는 `ClusterIP`이고 `svc/kite-gateway-external`은 존재하지 않는다. |
| Runtime defaults | `kite-runtime-config`의 SSH gateway desired 값이 external disabled와 빈 포트로 남아 있고 legacy host proxy 키를 사용하지 않는다. |
| Status default | `kite-gateway-status`는 `Disabled` 또는 아직 미생성 상태이며, UI가 이를 안전하게 표시한다. |

## Admin Enabled External Gateway

| 항목 | 확인 기준 |
| --- | --- |
| Service port 저장 | Admin Settings에서 external enabled와 Gateway Service port를 저장할 수 있다. |
| User-facing port 저장 | Admin Settings에서 사용자 안내 port를 Service port와 다르게 저장할 수 있다. 예: Service `12311`, 사용자 안내 `22`. |
| LoadBalancer 생성 | controller가 `kite-gateway-external` LoadBalancer를 Service port로 생성한다. |
| UI command | VM Dashboard, VM Detail, Connection Drawer는 user-facing port가 `22`이면 `ssh <sshId>@<domain>`, custom이면 `ssh -p <port> <sshId>@<domain>`을 표시한다. |
| VM SSH | VM 생성 후 표시된 명령으로 VM SSH에 연결된다. |

## No Host Proxy

| 항목 | 확인 기준 |
| --- | --- |
| Unknown username | VM route가 없는 username은 host sshd로 전달되지 않고 인증 실패한다. |
| Host SSH 유지 | Gateway external port 설정 전후에도 host sshd listen port가 변경되지 않는다. |

## Blocked and Cleanup

| 항목 | 확인 기준 |
| --- | --- |
| Missing external port | external enabled + empty port는 `Blocked`가 되고 external Service가 없다. |
| Uninstall | `uninstall.sh` 후 Kite 리소스만 삭제되고 host sshd 설정은 변경되지 않는다. |
