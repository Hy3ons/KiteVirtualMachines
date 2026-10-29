import type { Metadata } from 'next';
import Link from 'next/link';
import { LegalPage } from '@/components/LegalPage';
import { createPageMetadata } from '@/lib/metadata';

// generateMetadata identifies the policy draft without indexing an unfinalized policy.
export function generateMetadata(): Promise<Metadata> {
  return createPageMetadata('개인정보처리방침', {
    path: '/privacy', noIndex: true,
    description: 'Kite의 계정, Google 로그인, VM 이용 정보와 개인정보 보호 정책 초안입니다.',
  });
}

// Privacy renders the public privacy draft without requiring a session or exposing account data.
export default function Privacy() {
  return (
    <LegalPage title="개인정보처리방침" document="privacy">
      <section aria-labelledby="privacy-operator">
        <h2 id="privacy-operator">1. 적용 범위와 개인정보처리자</h2>
        <p>이 방침은 해당 Kite 인스턴스를 운영하는 개인 또는 조직(이하 “운영자”)이 웹사이트, 계정 인증, 가상머신 할당 및 접속 서비스를 제공하며 처리하는 개인정보에 적용됩니다. Kite는 자체 구축형 오픈소스 소프트웨어이며, 소프트웨어 개발자와 실제 서버 운영자가 같다고 전제하지 않습니다.</p>
        <p>최종 방침에는 실제 운영자의 명칭, 개인정보 보호책임자 또는 담당 부서, 이메일·전화번호 등 문의처를 기재합니다. 현재 이 정보는 확정 전입니다.</p>
      </section>
      <section aria-labelledby="privacy-data">
        <h2 id="privacy-data">2. 처리하는 정보와 이용 목적</h2>
        <dl className="marketing-legal-definitions">
          <div><dt>이메일 계정</dt><dd>계정 이름, 이메일, 비밀번호 검증용 해시를 계정 생성, 본인 인증 및 계정 관리에 이용합니다. 비밀번호 원문을 계정 인증 정보로 보관하지 않습니다.</dd></div>
          <div><dt>Google 로그인 도입 시</dt><dd>Google 계정 식별자, 이메일과 이메일 인증 여부를 계정 식별과 로그인에 이용합니다. 이름·프로필 사진은 해당 권한을 요청하고 실제로 제공받는 경우에만 계정 표시에 이용합니다. Google 계정 비밀번호는 제공받지 않습니다.</dd></div>
          <div><dt>서비스 이용 정보</dt><dd>계정 권한, VM 이름·사양·상태·도메인, SSH 사용자 ID, 리소스 할당 및 작업 이력을 서비스 제공, 권한 제어와 장애 대응에 이용합니다. VM 초기 접속 정보와 디스크에는 이용자가 입력하거나 저장한 정보가 포함될 수 있습니다.</dd></div>
          <div><dt>접속·보안 정보</dt><dd>실제 운영 환경에서 수집하는 접속 IP, 요청 시각, 브라우저 정보, 인증 실패·관리 작업 기록은 부정 이용 방지와 장애 분석에 사용합니다. 최종 방침에는 실제 수집 항목과 보유기간을 확정해 공개합니다.</dd></div>
          <div><dt>문의 정보</dt><dd>이용자가 직접 제공한 연락처와 문의 내용을 요청 처리와 결과 안내에 이용합니다. 문의에 비밀번호, 인증 토큰 또는 불필요한 개인정보를 포함하지 마세요.</dd></div>
        </dl>
        <p>회원가입과 서비스 이용에 필요한 정보는 이용계약 체결·이행 등 해당 처리의 적법한 근거에 따라 처리합니다. 별도 동의가 필요한 처리는 목적과 항목을 안내하고 동의를 받은 뒤 진행합니다.</p>
      </section>
      <section aria-labelledby="privacy-google">
        <h2 id="privacy-google">3. Google 계정 정보의 이용</h2>
        <p>Google 로그인은 본인 확인에 필요한 최소 권한으로 도입할 예정입니다. 기본 인증·이메일 권한을 사용하며, 이름과 사진을 표시할 때에만 프로필 권한을 사용합니다. Gmail, Google Drive, 연락처의 접근 권한은 로그인 용도로 요청하지 않습니다.</p>
        <p>Google에서 받은 정보는 안내한 계정 인증·표시 목적에만 이용하며, 판매하거나 광고 타기팅에 이용하지 않습니다. 이용 목적이나 요청 권한이 달라지면 변경 내용을 먼저 안내하고 필요한 동의를 받습니다.</p>
        <p><a href="https://myaccount.google.com/connections" target="_blank" rel="noreferrer">Google 계정의 연결 관리</a>에서 Kite 연결을 해제할 수 있습니다. 연결 해제는 이후 Google 인증을 제한하지만, 이미 생성된 Kite 계정이나 VM을 자동 삭제하지는 않습니다. 계정 및 관련 정보 삭제는 운영자에게 별도로 요청할 수 있습니다.</p>
      </section>
      <section aria-labelledby="privacy-retention">
        <h2 id="privacy-retention">4. 보유기간과 파기</h2>
        <p>계정 및 Google 연동 정보는 계정 유지와 서비스 제공에 필요한 동안 보유하고, 탈퇴·계약 종료 또는 처리 목적 달성 시 지체 없이 파기하는 것을 원칙으로 합니다. VM 관련 정보는 해당 리소스 제공에 필요한 기간 동안 처리합니다.</p>
        <p>법령에 따른 보존 의무가 있으면 해당 정보, 근거와 기간을 안내하고 다른 용도로 사용하지 않도록 분리 보관합니다. 전자 파일은 복구가 어렵도록 삭제하고, 종이 문서는 분쇄 등 적절한 방법으로 파기합니다.</p>
        <p>접속·보안 로그와 백업의 구체적인 보유기간 및 삭제 주기는 아직 확정되지 않았습니다. 실제 저장·백업 설정을 확인한 뒤 최종 방침에 명시합니다. 계정 삭제와 VM 디스크 삭제의 범위도 요청 처리 시 안내합니다.</p>
      </section>
      <section aria-labelledby="privacy-sharing">
        <h2 id="privacy-sharing">5. 제공, 위탁과 국외이전</h2>
        <p>운영자는 개인정보를 제3자에게 판매하지 않습니다. 서비스 제공을 위한 처리와 별개로 제3자 제공이 필요하면 법령상 근거 또는 필요한 동의 절차를 확인하고 제공받는 자, 목적, 항목과 보유기간을 안내합니다.</p>
        <p>서버 운영, 메일 발송, 백업 등을 외부 업체에 맡기는 경우 수탁자와 업무를 공개합니다. 온프레미스 설치만으로 모든 개인정보가 국내에 머문다고 보장할 수는 없습니다.</p>
        <p>Google 인증 과정에는 Google 서비스와의 통신이 포함됩니다. 실제 로그인 방식과 계약 관계를 확인해 국외이전에 해당하는 경우 이전받는 자, 국가, 항목, 목적, 시기·방법, 보유기간, 거부 방법과 영향을 최종 방침 또는 별도 안내에 명시하고 필요한 절차를 이행합니다. 현재 위탁·이전 내역은 확정 전입니다.</p>
        <p>Google 자체의 정보 처리에는 <a href="https://policies.google.com/privacy" target="_blank" rel="noreferrer">Google 개인정보처리방침</a>도 적용됩니다.</p>
      </section>
      <section aria-labelledby="privacy-cookie">
        <h2 id="privacy-cookie">6. 쿠키와 안전성 확보</h2>
        <p>Kite는 로그인 상태 확인에 필요한 인증 쿠키를 사용합니다. 현재 이메일 로그인에서는 서버가 검증하는 HttpOnly 인증 쿠키를 이용하며, Google 로그인 도입 시 추가되는 인증 관련 쿠키도 최종 방침에 반영합니다. 브라우저 설정에서 쿠키를 차단하거나 삭제할 수 있으나 로그인 기능이 제한될 수 있습니다.</p>
        <p>운영자는 접근 권한 제한, 인증 정보 보호, 보안 통신과 관리 기록 점검 등 운영 환경에 필요한 보호조치를 적용해야 합니다. 사용자의 VM 데이터는 기술 지원이나 보안 대응 등에 필요한 범위에서 권한 있는 운영자가 접근할 수 있으며, VM 격리가 운영자의 관리 권한까지 제거하는 것은 아닙니다.</p>
      </section>
      <section aria-labelledby="privacy-rights">
        <h2 id="privacy-rights">7. 이용자의 권리와 문의</h2>
        <p>이용자와 적법한 대리인은 개인정보 열람·정정·삭제·처리정지, 동의 철회 및 계정 탈퇴를 운영자에게 요청할 수 있습니다. 운영자는 필요한 본인 확인 후 관련 법령에 따라 처리하고, 제한 사유가 있으면 그 이유를 안내합니다. 계정 관리나 삭제를 위한 별도 화면이 없다면 해당 인스턴스 관리자에게 요청할 수 있습니다.</p>
        <p>개인정보 처리에 관한 문의처와 보호책임자 정보는 운영자가 확정해 이 문서에 공개해야 합니다. 미성년자의 이용을 허용하고 법정대리인의 동의가 필요한 경우에는 해당 확인 절차를 마련한 뒤 서비스를 제공합니다.</p>
      </section>
      <section aria-labelledby="privacy-changes">
        <h2 id="privacy-changes">8. 방침의 공개와 변경</h2>
        <p>최종 방침의 시행일과 개정 내용은 이 페이지에 공개합니다. 처리 목적이나 제공 범위가 바뀌어 별도 동의가 필요한 경우에는 변경 방침 게시만으로 동의를 대신하지 않습니다.</p>
        <p>서비스 이용 조건은 <Link href="/terms">서비스 이용약관</Link>에서 확인할 수 있습니다.</p>
      </section>
    </LegalPage>
  );
}
