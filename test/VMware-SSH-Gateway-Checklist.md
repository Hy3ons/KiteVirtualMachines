# VMware SSH Gateway Verification Checklist

이 문서는 `fix/safe-ssh-gateway-exposure` 작업을 실제 VMware/k3s 환경에서 검증할 때의 실행 순서다. 이번 구현 라운드에서는 실행하지 않고, 정적 검증만 수행한다.

## Fresh Install Safety

| 항목 | 확인 기준 |
| --- | --- |
| Host SSH 유지 | `ghcr-install.sh` 또는 `build-install.sh` 직후 host sshd 22번 접속이 그대로 유지된다. |
| Internal gateway only | `svc/kite-gateway`는 `ClusterIP`이고 `svc/kite-gateway-external`은 존재하지 않는다. |
| Runtime defaults | `kite-runtime-config`의 SSH gateway desired 값이 external/fallback disabled와 빈 포트로 남아 있다. |
| Status default | `kite-gateway-status`는 `Disabled` 또는 아직 미생성 상태이며, UI가 이를 안전하게 표시한다. |

## Admin Enabled External Gateway

| 항목 | 확인 기준 |
| --- | --- |
| External port 저장 | Admin Settings에서 external enabled와 custom port를 저장할 수 있다. |
| LoadBalancer 생성 | controller가 `kite-gateway-external` LoadBalancer를 지정 포트로 생성한다. |
| UI command | VM Dashboard, VM Detail, Connection Drawer가 custom port면 `ssh -p <port> <sshId>@<domain>`을 표시한다. |
| VM SSH | VM 생성 후 표시된 명령으로 VM SSH에 연결된다. |

## Host Fallback

| 항목 | 확인 기준 |
| --- | --- |
| Disabled default | fallback disabled 상태에서는 VM route가 없는 host username이 gateway로 우회되지 않는다. |
| Enabled with port | fallback enabled와 host sshd port를 저장하면 gateway Deployment env가 `$(KITE_NODE_IP):<hostPort>`로 바뀐다. |
| Unknown username | VM route가 없는 username은 host sshd로 전달된다. |
| Route priority | VM `sshId`가 존재하는 username은 host fallback으로 우회하지 않고 VM route가 우선된다. |

## Blocked and Cleanup

| 항목 | 확인 기준 |
| --- | --- |
| Missing external port | external enabled + empty port는 `Blocked`가 되고 external Service가 없다. |
| Missing host port | fallback enabled + empty host port는 `Blocked`가 되고 env가 비활성 상태로 유지된다. |
| Port conflict | external port와 host sshd port가 같으면 `Blocked`가 되고 external Service가 생성되지 않는다. |
| Uninstall | `uninstall.sh` 후 Kite 리소스만 삭제되고 host sshd 설정은 변경되지 않는다. |
