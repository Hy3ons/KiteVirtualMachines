import Image from 'next/image';
import Link from 'next/link';

// VmOfferGuide explains the shipped admin allocation and user claim flows.
// Generated artwork is conceptual; labels, instructions and API constraints remain accessible HTML.
export function VmOfferGuide() {
  return (
    <section id="vm-offers" className="marketing-guide-section" aria-labelledby="offers-title">
      <p className="marketing-eyebrow">03 / VM offers</p>
      <h2 id="offers-title">VM 사전 할당</h2>
      <p>관리자가 사용할 사양을 먼저 정해 두고, 사용자가 이름과 접속 정보를 입력하면 가상머신을 만드는 기능입니다. 사전 할당만으로 VM이 실행되거나 물리 자원이 예약되지는 않습니다.</p>

      <figure className="docs-offer-figure">
        <Image src="/images/docs/vm-offer-lifecycle.webp" alt="관리자가 사양을 배정하고, 사용자가 접속 정보를 입력해 수락하면 실제 가상머신이 생성되는 세 단계 흐름" width={1536} height={768} sizes="(max-width: 720px) calc(100vw - 72px), (max-width: 1024px) calc(100vw - 336px), 664px" />
        <figcaption>
          <span className="docs-figure-label">VM 사전 할당 흐름 · 개념도</span>
          <ol className="docs-figure-stages">
            <li><span>01</span><div><strong>관리자 사전 할당</strong><p>사용자와 사양 선택</p></div></li>
            <li><span>02</span><div><strong>사용자 수락</strong><p>이름과 접속 정보 입력</p></div></li>
            <li><span>03</span><div><strong>가상머신 생성</strong><p>리소스 준비와 부팅</p></div></li>
          </ol>
        </figcaption>
      </figure>

      <h3>시작하기 전에</h3>
      <dl className="docs-offer-prerequisites">
        <div><dt>배정하는 계정</dt><dd>Level 3 전체 관리자</dd></div>
        <div><dt>할당받는 계정</dt><dd>배정 대상 사용자 본인</dd></div>
        <div><dt>기본 유효기간</dt><dd>배정 후 24시간</dd></div>
      </dl>
      <p>일반 계정의 직접 생성 사양보다 큰 VM이 필요하다면 관리자에게 사전 할당을 요청하세요. 할당받을 때는 관리자가 정한 CPU·메모리·디스크·이미지를 그대로 사용합니다.</p>

      <h3>관리자: 사용자에게 VM 배정하기</h3>
      <ol className="marketing-guide-steps docs-offer-steps">
        <li><span>01</span><div><h4>VM 사전 할당 열기</h4><p><Link href="/admin/dashboard">사용자 및 리소스</Link>에서 <strong>VM 사전 할당</strong> 탭을 선택합니다. 이 작업은 Level 3 관리자에게만 허용됩니다.</p></div></li>
        <li><span>02</span><div><h4>대상과 사양 지정하기</h4><p>대상 사용자를 고르고 CPU, 메모리, 디스크, 이미지를 입력합니다. 예를 들어 <code>4 CPU / 8Gi / 30Gi</code>, 이미지 <code>ubuntu-22.04</code>를 배정할 수 있습니다. 예시는 고정 제한값이 아닙니다.</p></div></li>
        <li><span>03</span><div><h4>사전 할당 생성하기</h4><p><strong>사전 할당 생성</strong>을 누릅니다. 이 단계에서는 배정 정보만 저장됩니다. 사용자에게 대시보드에서 만료일을 확인하고 할당받도록 안내하세요.</p></div></li>
      </ol>

      <h3>사용자: 배정된 가상머신 만들기</h3>
      <ol className="marketing-guide-steps docs-offer-steps">
        <li><span>01</span><div><h4>배정 내용 확인하기</h4><p><Link href="/dashboard#offers">워크스페이스의 배정된 가상머신</Link>에서 사양, 이미지와 만료일을 확인하고 <strong>할당받기</strong>를 누릅니다. 본인에게 배정된 항목만 표시됩니다.</p></div></li>
        <li><span>02</span><div><h4>이름과 접속 정보 입력하기</h4><p><strong>배정된 가상머신 만들기</strong> 창에서 아래 네 항목을 입력합니다. 배정받은 사양은 이 창에서 변경할 수 없습니다.</p></div></li>
      </ol>
      <dl className="docs-offer-fields">
        <div><dt>가상머신 이름</dt><dd>대시보드에서 구분할 이름입니다. 예: <code>project-runner</code></dd></div>
        <div><dt>도메인 접두사</dt><dd>VM에 사용할 도메인의 앞부분입니다. 소문자 영문·숫자·하이픈을 사용하며, 이미 사용 중이거나 예약된 값은 사용할 수 없습니다. 예: <code>project-runner</code></dd></div>
        <div><dt>SSH 사용자 ID</dt><dd>SSH Gateway가 VM을 찾는 접속 ID입니다. 다른 VM과 중복될 수 없습니다. 예: <code>project-runner</code></dd></div>
        <div><dt>초기 비밀번호</dt><dd>VM 접속에 사용할 비밀번호를 직접 설정합니다. Kite 로그인 비밀번호와는 별개입니다.</dd></div>
      </dl>
      <aside className="marketing-guide-callout docs-offer-note" aria-label="초기 비밀번호 안내"><strong>초기 비밀번호를 안전하게 보관하세요.</strong><p>VM 생성 시 정한 초기 비밀번호입니다. Kite에서 변경하거나 복구할 수 없습니다.</p></aside>

      <h3>할당받은 다음에는</h3>
      <p>정보를 확인하고 <strong>할당받기</strong>를 누릅니다. 요청이 성공하면 사용한 배정은 목록에서 사라지고, 생성된 VM은 <Link href="/dashboard#instances">가상머신 목록</Link>에서 확인할 수 있습니다. 같은 배정으로 여러 VM을 만들 수는 없습니다.</p>
      <p>요청 성공과 VM의 부팅 완료는 별개입니다. 이미지와 디스크 준비, 부팅이 끝날 때까지 상태를 확인하세요. 실행 상태와 접속 준비가 확인되면 상세 화면에서 SSH 연결 정보나 웹 콘솔을 이용할 수 있습니다. 외부 SSH 접속에는 관리자의 <a href="#ssh-gateway">SSH Gateway 설정</a>이 필요합니다.</p>

      <h3>문제가 생겼다면</h3>
      <div className="docs-offer-troubleshooting">
        <details><summary>배정된 가상머신이 보이지 않아요.</summary><p>배정받은 계정으로 로그인했는지 확인하세요. 만료되었거나 이미 사용한 배정은 목록에 나타나지 않습니다. 관리자가 다른 계정에 배정했거나 생성 요청 실패 뒤 배정이 보이지 않는 경우에는 관리자에게 확인을 요청하세요.</p></details>
        <details><summary>이름이나 접속 정보가 중복되었다고 나와요.</summary><p>다른 VM에서 사용하지 않는 이름·도메인 접두사·SSH ID를 준비하고 표시된 오류를 확인하세요. 실패 후 배정 상태가 바뀌었다면 반복 제출하지 말고 관리자에게 배정과 VM 생성 여부를 확인해 달라고 요청하세요.</p></details>
        <details><summary>24시간이 지나면 만들어진 VM도 삭제되나요?</summary><p>24시간은 아직 사용하지 않은 배정의 기본 유효기간입니다. 만료된 배정은 사용할 수 없고 컨트롤러가 정리합니다. 이미 생성된 VM은 이 배정의 만료를 이유로 삭제되지 않습니다.</p></details>
      </div>
      <details className="docs-offer-technical"><summary>운영자를 위한 처리 구조</summary><p>관리자 요청은 <code>KiteVirtualMachineOffer</code>에 사양과 만료 시각을 기록합니다. 사용자의 수락 요청을 Go API가 검증한 뒤 배정 사양과 입력 정보를 합쳐 <code>KiteVirtualMachine</code>을 생성하고, 성공한 배정은 제거합니다. 이후 컨트롤러가 실제 KubeVirt 리소스를 조정합니다.</p><p>기본 TTL은 24시간이며, 만료된 항목은 API 목록에서 제외되고 컨트롤러가 시작 시점과 이후 매시간 정리합니다. 이 흐름은 VM 직접 생성과 별도로 처리됩니다.</p><a href="#architecture" className="marketing-text-link">전체 아키텍처 읽기</a></details>
    </section>
  );
}
