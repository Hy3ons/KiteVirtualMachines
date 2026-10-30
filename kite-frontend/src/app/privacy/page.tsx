import type { Metadata } from 'next';
import Link from 'next/link';
import { LegalPage } from '@/components/LegalPage';
import { createPageMetadata } from '@/lib/metadata';

// generateMetadata describes the public privacy policy using the shared page metadata.
export function generateMetadata(): Promise<Metadata> {
  return createPageMetadata('개인정보처리방침', {
    path: '/privacy', noIndex: true,
    description: 'Kite의 계정·VM 정보, Google 인증 정보의 처리 범위, 접속 로그, 외부 서비스와 개인정보 보호 원칙을 안내합니다.',
  });
}

// Privacy documents repository-backed data flows and the permitted scope of Google authentication.
// Deployment-specific operator identity and retention settings must match the actual installation.
export default function Privacy() {
  return (
    <LegalPage title="개인정보처리방침" document="privacy">
      <section aria-labelledby="privacy-operator">
        <h2 id="privacy-operator">1. 적용 범위와 개인정보처리자</h2>
        <p>이 방침은 Kite 웹사이트의 계정 인증, 가상머신 할당·관리와 SSH Gateway·웹 콘솔 접속 과정에서 처리하는 개인정보에 적용됩니다. 개인정보처리자는 이용 중인 Kite 설치 환경을 직접 운영하는 개인 또는 조직(이하 “운영자”)입니다.</p>
        <p>Kite는 자체 구축형 오픈소스 소프트웨어입니다. 계정과 VM 정보는 해당 운영자가 관리하는 서버·클러스터에 저장되며, 소프트웨어 개발자에게 이용자 정보를 자동으로 전송하는 기능은 없습니다. 설치 운영자와 소프트웨어 개발자가 다른 경우 개인정보 처리 요청은 실제 운영자에게 해야 합니다.</p>
      </section>
      <section aria-labelledby="privacy-data">
        <h2 id="privacy-data">2. 처리하는 정보와 이용 목적</h2>
        <dl className="marketing-legal-definitions">
          <div><dt>계정 정보</dt><dd>회원가입 시 입력한 사용자 이름·이메일, 내부 계정 식별자, 비밀번호 검증용 해시와 계정 권한을 가입·로그인, 이용자 식별과 접근 제어에 사용합니다. 비밀번호는 원문으로 저장하지 않으며, 프로필 사진은 저장하지 않습니다.</dd></div>
          <div><dt>VM·자원 배정 정보</dt><dd>VM 이름, 소유자, CPU·메모리·디스크 사양, 이미지, 전원·생성 상태, 도메인, SSH 사용자 ID와 비밀번호 검증용 해시를 자원 생성·조회·접속 제어에 사용합니다. 사전 배정 내역에는 배정·수락한 계정과 유효기간도 포함됩니다.</dd></div>
          <div><dt>VM 초기 로그인 정보</dt><dd>생성 시 입력한 초기 비밀번호를 검증용 해시와 게스트 OS 로그인용 해시로 변환해 저장하고, 플랫폼이 생성한 VM 접속용 SSH 키를 관리합니다. 초기 비밀번호 원문을 다시 조회·복구하는 기능은 제공하지 않습니다. VM 디스크에는 이용자가 저장한 파일과 프로그램의 정보가 포함됩니다.</dd></div>
          <div><dt>웹 접속 로그</dt><dd>API 요청 시각, 접속 IP, 요청 방식·경로, 응답 상태, 처리 시간과 오류 정보를 장애 분석과 부정 이용 대응에 사용합니다. 요청 경로에는 VM 이름 등 이용자가 지정한 값이 포함될 수 있습니다.</dd></div>
          <div><dt>SSH 접속·오류 정보</dt><dd>SSH Gateway의 인증 거부·연결 오류 처리 과정에서 접속 IP·포트, SSH 사용자명, 대상 VM 식별 정보와 오류 사유를 기록합니다. 이 정보는 접속 장애 분석과 보안 대응에 사용합니다.</dd></div>
          <div><dt>콘솔·SSH 통신 내용</dt><dd>웹 콘솔과 SSH Gateway는 VM 접속에 필요한 입력·출력을 중계합니다. 기본 구현에는 명령어와 화면 전체를 녹화해 별도 보관하는 기능이 없습니다. 다만 VM 내부의 운영체제·프로그램이나 운영자가 별도로 구성한 시스템에 기록이 남을 수 있습니다.</dd></div>
          <div><dt>문의 정보</dt><dd>운영자에게 직접 전달한 연락처와 문의 내용을 요청 처리와 결과 안내에 사용합니다. 문의에 비밀번호·인증 토큰이나 불필요한 개인정보를 포함하지 마세요.</dd></div>
        </dl>
        <p>계정·VM 제공에 필요한 정보는 이용계약의 체결·이행을 위해 처리합니다. 접속·보안 기록은 서비스의 안전한 운영과 사고 대응에 필요한 범위에서 적법한 근거에 따라 처리하며, 별도 동의가 필요한 처리는 목적과 항목을 안내하고 동의를 받은 뒤 진행합니다.</p>
      </section>
      <section aria-labelledby="privacy-google">
        <h2 id="privacy-google">3. Google 로그인과 계정 정보 보호</h2>
        <p>Google 로그인의 개인정보 처리 범위는 계정 인증에 필요한 Google 계정 고유 식별자, 이메일 주소와 이메일 인증 여부로 한정합니다. 인증 응답은 로그인한 계정의 진위와 유효성을 확인하는 데 사용하며, Google 계정의 비밀번호를 제공받거나 저장하지 않습니다.</p>
        <p>Google 계정 정보는 계정 식별과 로그인 처리에만 사용합니다. Gmail·Google Drive·연락처·캘린더 접근 권한을 요청하거나, Google 정보를 판매·광고·행동 분석 또는 AI 모델 학습에 사용하지 않습니다. 계정 정보와 VM 내용·콘솔 명령어를 Google에 전송하는 용도로 사용하지 않습니다.</p>
        <p>Google 계정 정보에 대한 접근 철회는 <a href="https://myaccount.google.com/connections" target="_blank" rel="noreferrer">Google 계정의 연결 관리</a>에서 할 수 있습니다. 보관 중인 계정 정보의 열람·삭제와 동의 철회는 운영자에게 요청할 수 있으며, 처리 목적이 끝나면 관련 정보를 파기합니다. Google 자체의 인증·접속 정보 처리에는 <a href="https://policies.google.com/privacy" target="_blank" rel="noreferrer">Google 개인정보처리방침</a>이 적용됩니다.</p>
      </section>
      <section aria-labelledby="privacy-retention">
        <h2 id="privacy-retention">4. 보유기간과 파기</h2>
        <p>계정 정보는 계정 유지 기간 동안, VM 설정·접속용 인증 정보와 디스크는 해당 자원을 제공하는 동안 보유합니다. 탈퇴·이용 종료 또는 처리 목적 달성 시 지체 없이 파기합니다. 계정 삭제 시 연결된 VM도 삭제 대상이 되므로, 필요한 파일은 미리 반출해야 합니다.</p>
        <p>접속·보안 로그와 문의 내용은 장애·보안 조사 또는 문의 처리에 필요한 범위에서 보관하고, 처리 목적이 끝나거나 보유기간이 지나면 파기합니다. 법령에 따른 보존 의무가 있으면 대상 항목과 법적 근거·기간을 안내하고 다른 목적으로 사용하지 않도록 분리 보관합니다.</p>
        <p>전자 정보는 계정·VM 관리 리소스와 연결된 디스크 볼륨을 삭제하는 방식으로 파기합니다. 서비스에는 자동 백업이나 백업의 일괄 보관기간이 기본 제공되지 않습니다. 운영자가 별도로 만든 백업·스토리지 사본에는 해당 운영자의 보관·파기 정책을 적용하며, 운영자는 삭제 요청을 처리할 때 이 사본과 저장 매체의 처리 범위도 확인해야 합니다.</p>
      </section>
      <section aria-labelledby="privacy-sharing">
        <h2 id="privacy-sharing">5. 외부 서비스와 정보 제공</h2>
        <dl className="marketing-legal-definitions">
          <div><dt>Google 계정 인증</dt><dd>이용자가 선택한 Google 계정의 인증을 처리하는 외부 서비스입니다. Google과의 인증 통신 범위는 위 3항에 한정하며, VM·디스크를 보관하는 서비스로 사용하지 않습니다.</dd></div>
          <div><dt>jsDelivr 글꼴 CDN</dt><dd>웹 화면의 Pretendard 글꼴과 스타일 파일을 불러오기 위해 브라우저가 cdn.jsdelivr.net에 요청합니다. 이 과정에서 접속 IP, 요청한 파일 주소, 브라우저가 보내는 접속 정보가 CDN에 전달됩니다. Kite 계정 정보나 VM 내용을 전달하기 위한 연동이 아닙니다. 외부 요청의 처리에는 <a href="https://www.jsdelivr.com/terms/privacy-policy" target="_blank" rel="noreferrer">jsDelivr 개인정보처리방침</a>이 적용됩니다.</dd></div>
          <div><dt>GitHub 외부 링크</dt><dd>소스 코드·문서 링크를 직접 누르면 GitHub로 이동하며, 이후 접속 정보는 GitHub의 정책에 따라 처리됩니다. Kite의 계정 정보나 VM 내용을 GitHub로 자동 전송하지 않습니다.</dd></div>
        </dl>
        <p>기본 구성에는 광고 추적, Google Analytics 등 사용자 행동 분석, 외부 오류 수집 또는 메일 발송 서비스 연동이 없습니다. Kubernetes·KubeVirt와 스토리지는 운영자가 관리하는 설치 환경의 구성 요소이며, 이 소프트웨어를 사용한다는 이유로 해당 프로젝트 개발자에게 계정·VM 정보를 보내지 않습니다.</p>
        <p>개인정보를 제3자에게 판매하지 않습니다. 법령상 의무나 별도 동의 등 적법한 근거가 있는 경우에만 필요한 범위에서 제공하며, 별도의 외부 처리업체를 이용하는 운영자는 수탁자·업무와 처리 항목을 공개해야 합니다.</p>
        <p>Google 인증과 jsDelivr 파일 요청은 외부 사업자의 서버와 통신하므로, 접속 정보 등이 국내 밖에서 처리될 수 있습니다. 실제 서비스 운영자는 적용되는 국외이전 요건에 따라 이전받는 자·국가·항목·목적·방법·보유기간과 거부 방법을 안내하고 필요한 절차를 이행해야 합니다. 자체 설치만으로 모든 처리가 국내에서 이루어진다고 보장하지 않습니다.</p>
      </section>
      <section aria-labelledby="privacy-cookie">
        <h2 id="privacy-cookie">6. 쿠키와 안전성 확보</h2>
        <p>로그인 상태 확인에 필요한 accessToken 인증 쿠키를 사용합니다. 웹 스크립트가 읽을 수 없는 HttpOnly 속성과 SameSite=Lax 속성을 사용하며, HTTPS 접속에서는 Secure 속성을 적용하고 브라우저 보관기간은 1시간입니다. 로그아웃 시 쿠키를 삭제하며, 토큰의 실제 유효성은 서버에서 확인합니다. 브라우저 설정에서 쿠키를 차단·삭제할 수 있으나 로그인 기능이 제한될 수 있습니다.</p>
        <p>광고·분석용 쿠키를 사용하지 않으며, 로그인 토큰과 계정 정보를 브라우저의 localStorage에 저장하지 않습니다. 서버는 계정 권한을 확인해 접근을 제한하고, 비밀번호는 검증용 해시로 처리합니다. 운영자는 HTTPS, 서버·클러스터 접근 권한, 저장 매체와 백업의 보호조치를 관리해야 합니다.</p>
        <p>권한 있는 운영자는 장애 복구·기술 지원·보안 대응이나 법적 의무 이행에 필요한 최소 범위에서 VM 정보와 데이터에 접근할 수 있습니다. 자원 확인과 VM 정지는 운영 목적에 한정하며, VM 격리가 운영자의 시스템 관리 권한까지 없애는 것은 아닙니다.</p>
      </section>
      <section aria-labelledby="privacy-rights">
        <h2 id="privacy-rights">7. 이용자의 권리와 문의</h2>
        <p>이용자와 적법한 대리인은 개인정보 열람·정정·삭제·처리정지, 동의 철회와 탈퇴를 해당 서비스 운영자에게 요청할 수 있습니다. 계정 관리·삭제 화면이 없는 경우에도 관리자에게 요청할 수 있으며, 운영자는 필요한 본인 확인 후 법정 절차에 따라 처리하고 제한 사유가 있으면 그 이유를 안내합니다.</p>
        <p>접수처는 해당 설치 환경의 운영자가 제공하는 서비스 관리자 연락처입니다. 운영자는 실제 개인정보처리자 명칭, 개인정보 보호책임자와 연락처를 이용자가 확인할 수 있도록 공개해야 합니다. 법정대리인의 동의가 필요한 아동의 개인정보는 필요한 확인·동의 절차 없이 처리하지 않습니다.</p>
      </section>
      <section aria-labelledby="privacy-changes">
        <h2 id="privacy-changes">8. 방침의 공개와 변경</h2>
        <p>개정일은 이 페이지 상단에 표시합니다. 개인정보의 처리 목적·항목·외부 제공 범위 또는 보관 정책을 변경하면 변경 내용과 적용 시점을 안내하며, 별도 동의가 필요한 경우에는 변경 방침 게시만으로 동의를 대신하지 않습니다.</p>
        <p>서비스 이용 조건은 <Link href="/terms">서비스 이용약관</Link>에서 확인할 수 있습니다.</p>
      </section>
    </LegalPage>
  );
}
