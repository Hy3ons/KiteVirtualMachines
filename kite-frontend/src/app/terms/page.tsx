import type { Metadata } from 'next';
import Link from 'next/link';
import { LegalPage } from '@/components/LegalPage';
import { createPageMetadata } from '@/lib/metadata';

// generateMetadata identifies the terms draft without indexing an unfinalized policy.
export function generateMetadata(): Promise<Metadata> {
  return createPageMetadata('서비스 이용약관', {
    path: '/terms', noIndex: true,
    description: 'Kite 가상머신 서비스의 계정, 리소스 할당, 이용 범위와 운영 원칙에 관한 약관 초안입니다.',
  });
}

// Terms renders the public service terms draft independently of account authentication.
export default function Terms() {
  return (
    <LegalPage title="서비스 이용약관" document="terms">
      <section aria-labelledby="terms-purpose">
        <h2 id="terms-purpose">1. 목적과 적용 범위</h2>
        <p>이 약관은 해당 Kite 인스턴스 운영자(이하 “운영자”)와 이용자 사이의 계정, 가상머신 할당, 리소스 관리 및 접속 서비스 이용 조건을 정합니다. 실제 운영 주체와 연락처는 최종 약관에 명시합니다.</p>
        <p>Kite는 Kubernetes와 KubeVirt를 활용해 VM 작업 공간을 제공하는 소규모 팀용 플랫폼입니다. 한 대의 온프레미스 컴퓨터에서 단일 노드로 운영할 수 있으며, 제공 범위와 자원은 실제 설치 환경에 따라 달라집니다. 오픈소스 소프트웨어의 라이선스와 개별 인스턴스의 서비스 계약은 구분됩니다.</p>
      </section>
      <section aria-labelledby="terms-account">
        <h2 id="terms-account">2. 가입과 계정 인증</h2>
        <p>이용계약은 이용자가 공개된 약관과 필요한 개인정보 처리 안내를 확인하고 가입을 신청한 뒤 운영자가 이를 승인하면 성립합니다. 운영자는 조직 소속이나 초대 여부를 확인하거나 관리자가 허용한 이용자에게만 가입을 승인할 수 있습니다.</p>
        <p>이용자는 정확한 계정 정보를 제공하고 본인의 인증 수단을 안전하게 관리해야 합니다. 계정 도용이나 유출이 의심되면 즉시 운영자에게 알리고 비밀번호 변경, 세션 종료 등 가능한 조치를 취해야 합니다.</p>
        <p>Google 로그인이 도입되면 Google 계정 인증으로 Kite 계정을 생성하거나 연결할 수 있습니다. Google 계정 연결만으로 VM 사용 권한이 자동 부여되지는 않습니다. 계정 연결 해제와 Kite 탈퇴는 서로 다른 절차이며, Google 인증이 중단되면 다른 인증 수단이 없는 계정은 로그인에 제한을 받을 수 있습니다.</p>
      </section>
      <section aria-labelledby="terms-resources">
        <h2 id="terms-resources">3. 리소스 할당과 이용 범위</h2>
        <p>VM 생성·변경 권한, VM 개수, CPU·메모리·디스크 한도는 계정 권한과 운영 정책에 따릅니다. 관리자가 사전 배정한 VM은 배정된 사양, 수량과 유효기간 내에서 사용할 수 있습니다. 추가 자원이 필요하면 운영자에게 요청해야 합니다.</p>
        <p>이용자는 허용된 범위에서 VM을 생성·시작·중지하고, SSH Gateway나 웹 콘솔 등 제공되는 방법으로 접속할 수 있습니다. 네트워크, 외부 포트와 접속 경로는 운영 환경에 따라 제한되거나 변경될 수 있습니다.</p>
        <p>자원 할당은 물리 서버나 네트워크의 독점 사용을 의미하지 않습니다. 별도 계약이 없는 한 고가용성, 무중단 운영, 일정한 성능 또는 특정 가동률을 보장하지 않습니다. 유료 이용 조건을 도입하는 경우 가격, 과금 기준, 해지와 환불 조건을 사전에 별도로 안내합니다.</p>
      </section>
      <section aria-labelledby="terms-conduct">
        <h2 id="terms-conduct">4. 이용자가 지켜야 할 사항</h2>
        <ul>
          <li>타인의 계정을 도용하거나 승인 없이 접근 권한·인증 정보를 공유해서는 안 됩니다.</li>
          <li>호스트, 다른 VM 또는 타인의 네트워크에 무단 접근하거나 격리·권한·할당량을 우회해서는 안 됩니다.</li>
          <li>악성코드 유포, 스팸 발송, 서비스 거부 공격, 불법 자료 유통 등 법령이나 타인의 권리를 침해하는 행위를 해서는 안 됩니다.</li>
          <li>운영자가 허용하지 않은 채굴, 과도한 부하 발생 또는 제3자 재판매 등으로 공유 자원을 남용해서는 안 됩니다.</li>
          <li>이용자는 설치한 소프트웨어의 라이선스, VM 안에서 처리하는 데이터의 적법성, 공개한 서비스의 보안을 관리해야 합니다.</li>
        </ul>
      </section>
      <section aria-labelledby="terms-data">
        <h2 id="terms-data">5. 데이터, 초기 비밀번호와 백업</h2>
        <p>VM에 저장한 데이터와 이용자가 작성한 결과물의 권리는 이용자 또는 정당한 권리자에게 있습니다. 운영자는 서비스 제공, 장애 복구, 보안 대응과 법적 의무 이행에 필요한 범위에서만 해당 정보에 접근해야 합니다.</p>
        <p>VM 생성 시 입력한 초기 비밀번호는 이용자가 안전하게 보관해야 합니다. Kite 화면에서 해당 비밀번호를 변경하거나 복구하는 기능은 제공되지 않습니다. VM 내부의 인증 수단과 접근 권한은 이용자가 관리해야 합니다.</p>
        <p>별도로 백업을 제공한다고 명시하지 않은 경우 자동 백업·복구는 서비스에 포함되지 않습니다. 이용자는 중요한 데이터를 외부에 백업해야 합니다. VM 삭제, 배정 만료, 탈퇴 또는 서버 장애로 디스크 데이터가 복구되지 않을 수 있으므로, 운영자는 예정된 삭제 범위와 시점을 가능한 한 사전에 안내합니다.</p>
      </section>
      <section aria-labelledby="terms-operation">
        <h2 id="terms-operation">6. 운영 변경과 일시 중단</h2>
        <p>점검, 업데이트, 자원 부족, 장애 또는 보안 위협으로 서비스가 일시 중단되거나 일부 기능이 제한될 수 있습니다. 단일 노드 환경에서는 물리 서버의 장애가 여러 VM에 동시에 영향을 줄 수 있습니다.</p>
        <p>운영자는 예정된 중단이나 중요한 이용 조건 변경을 사전에 안내합니다. 긴급 보안 조치나 예기치 못한 장애로 사전 안내가 어려운 경우에는 조치 후 사유와 영향을 안내합니다. 서비스 종료 시에는 가능한 데이터 반출 기간과 종료 일정을 공지합니다.</p>
      </section>
      <section aria-labelledby="terms-restrictions">
        <h2 id="terms-restrictions">7. 이용 제한, 탈퇴와 삭제</h2>
        <p>법령 위반, 자원 남용, 보안 침해 또는 약관 위반이 확인되면 운영자는 필요한 범위에서 VM 중지, 외부 접속 차단이나 계정 이용 제한을 할 수 있습니다. 긴급 대응이 필요한 경우 우선 조치할 수 있으며, 이후 제한 사유와 이의 제기 방법을 안내합니다.</p>
        <p>이용자는 운영자에게 탈퇴나 이용 종료를 요청할 수 있습니다. 운영자는 필요한 본인 확인 후 계정, 할당 자원과 데이터의 처리 범위를 안내하고 요청을 처리합니다. 탈퇴 전에 필요한 데이터를 반출해야 하며, 개인정보 보유·파기는 <Link href="/privacy">개인정보처리방침</Link>에 따릅니다.</p>
      </section>
      <section aria-labelledby="terms-responsibility">
        <h2 id="terms-responsibility">8. 책임과 분쟁 처리</h2>
        <p>운영자와 이용자는 각자의 귀책사유로 상대방에게 발생시킨 손해에 대해 관련 법령에 따른 책임을 부담합니다. 이 약관은 운영자의 고의·중대한 과실에 따른 책임이나 법률상 배제할 수 없는 이용자의 권리를 제한하지 않습니다.</p>
        <p>이용자는 VM 격리가 모든 침해 가능성이나 데이터 손실을 없애는 보증은 아니라는 점을 이해하고, 서비스의 운영 수준에 맞는 데이터를 처리해야 합니다. 장애나 분쟁은 먼저 해당 인스턴스 운영자에게 문의하며, 해결되지 않은 사항에는 적용되는 법령과 적법한 분쟁 해결 절차를 따릅니다.</p>
      </section>
      <section aria-labelledby="terms-changes">
        <h2 id="terms-changes">9. 약관 변경과 시행</h2>
        <p>운영자는 최종 약관의 시행일, 운영 주체와 문의처를 이 페이지에 공개합니다. 약관을 변경할 때에는 내용, 사유와 적용 시점을 미리 안내하고, 이용자에게 중대한 불이익이 생기거나 별도 동의가 필요한 변경은 관련 법령에 따른 고지·동의 절차를 거칩니다.</p>
        <p>이 문서는 아직 시행되지 않은 초안이며, 단순히 게시하거나 페이지를 방문한 사실만으로 이용자의 약관 동의가 성립한다고 보지 않습니다.</p>
      </section>
    </LegalPage>
  );
}
