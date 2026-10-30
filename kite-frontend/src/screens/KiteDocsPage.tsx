'use client';

import Link from 'next/link';
import Image from 'next/image';
import { ArrowRightOutlined, ArrowUpOutlined } from '@ant-design/icons';
import { GlobalHeader } from '../components/GlobalHeader';
import { accessLevelDocs, architectureTimelineItems } from './KiteDocsData';
import { VmOfferGuide } from './docs/VmOfferGuide';

const guideSections = [
  { id: 'access-levels', title: '권한 레벨' },
  { id: 'resource-policy', title: '리소스 정책' },
  { id: 'vm-offers', title: 'VM 사전 할당' },
  { id: 'architecture', title: '아키텍처' },
  { id: 'ssh-gateway', title: 'SSH Gateway' },
] as const;

// KiteDocsPage presents the existing policies and architecture as an anchored reading guide.
// The shared data remains the source for all access levels and reconciliation stages.
export function KiteDocsPage() {
  return (
    <div className="public-page">
      <GlobalHeader />
      <main id="main-content" className="marketing-docs">
        <header className="marketing-docs-hero" id="guide-top">
          <p className="marketing-eyebrow">Documentation / Kite</p>
          <h1>Kite 사용 가이드</h1>
          <p>계정 권한, VM 할당과 접속, 플랫폼의 동작 구조를 안내합니다. 처음 사용하는 이용자와 서버를 관리하는 운영자를 위한 문서입니다.</p>
          <Link href="/dashboard" className="marketing-text-link">콘솔 열기 <ArrowRightOutlined /></Link>
        </header>

        <div className="marketing-docs-layout">
          <aside className="marketing-docs-aside">
            <nav aria-label="가이드 목차">
              <p className="marketing-eyebrow">이 가이드에서</p>
              {guideSections.map((section, index) => <a key={section.id} href={'#' + section.id}><span>0{index + 1}</span>{section.title}</a>)}
            </nav>
            <a className="marketing-docs-repository" href="https://github.com/Hy3ons/KiteVirtualMachines" target="_blank" rel="noreferrer">GitHub에서 소스 보기 <ArrowRightOutlined /></a>
          </aside>

          <div className="marketing-docs-body">
            <section id="access-levels" className="marketing-guide-section" aria-labelledby="access-title">
              <p className="marketing-eyebrow">01 / Access control</p>
              <h2 id="access-title">계정마다 알맞은 권한.</h2>
              <p>Kite는 숫자가 높을수록 더 넓은 작업을 허용하되 Level 2와 Level 3의 권한 범위를 분리합니다.</p>
              <div className="marketing-access-list">
                {accessLevelDocs.map((item) => (
                  <article className="marketing-access-row" key={item.level}>
                    <div><span className="marketing-level">Level {item.level}</span><span className="marketing-access-tone">{item.tone}</span></div>
                    <div><h3>{item.title}</h3><p>{item.summary}</p><ul>{item.rules.map((rule) => <li key={rule}>{rule}</li>)}</ul></div>
                  </article>
                ))}
              </div>
            </section>

            <section id="resource-policy" className="marketing-guide-section" aria-labelledby="policy-title">
              <p className="marketing-eyebrow">02 / Resource policy</p>
              <h2 id="policy-title">만들기 전에 확인하세요.</h2>
              <h3>Level 1 · 고정 스펙과 생성 한도</h3>
              <p>일반 사용자는 직접 VM을 만들 수 있지만 스펙과 생성 한도가 고정됩니다. Frontend와 API가 모두 2대 초과 생성을 막습니다.</p>
              <dl className="marketing-policy-specs"><div><dt>CPU</dt><dd>2</dd></div><div><dt>Memory</dt><dd>4Gi</dd></div><div><dt>Disk</dt><dd>25Gi</dd></div><div><dt>Quota</dt><dd>2 VMs</dd></div></dl>
              <div className="marketing-policy-note"><h3>Level 2 · 자기 VM은 자유 스펙으로</h3><p>자기 VM을 자유 스펙으로 만들 수 있지만 admin API에서는 사용자 권한 0과 1만 조정합니다. 전체 VM 제어, settings, 사용자 삭제, offer 생성은 Level 3 관리자 작업입니다.</p></div>
              <div className="marketing-policy-note"><h3>Level 0 · 로그인과 권한 확인</h3><p>로그인할 수 있지만 VM을 직접 생성하거나 제어하는 작업은 할 수 없습니다. Dashboard는 관리자 연락처를 보여줍니다. 관리자가 배정한 VM Offer는 명시적 배정 흐름을 따르며 직접 생성 제한과는 분리됩니다.</p></div>
            </section>

            <VmOfferGuide />

            <section id="architecture" className="marketing-guide-section" aria-labelledby="architecture-title">
              <p className="marketing-eyebrow">04 / Architecture</p>
              <h2 id="architecture-title">요청에서 실제 가상머신까지.</h2>
              <p>Kite는 Frontend가 KubeVirt를 직접 제어하지 않도록 역할을 나눕니다. API는 권한과 입력을 확인하고 controller는 CRD spec을 기준으로 클러스터 안의 실제 리소스를 조정합니다.</p>
              <figure className="docs-concept-figure">
                <a href="/images/docs/web-api-architecture.webp" target="_blank" rel="noreferrer" aria-label="웹 요청과 컨테이너 아키텍처 원본 보기 (새 탭)">
                  <Image src="/images/docs/web-api-architecture.webp" width={1512} height={1040} sizes="(max-width: 720px) calc(100vw - 72px), (max-width: 1024px) calc(100vw - 336px), 664px" alt="브라우저 요청을 Ingress가 Next.js 프런트엔드와 Go API 컨테이너로 나눠 전달합니다. Go API가 Kubernetes에 CRD spec을 기록하면 Controller 컨테이너가 이를 감시해 KubeVirt 리소스를 조정하고 관찰한 상태를 status에 기록합니다." />
                </a>
                <figcaption className="docs-figure-label">컨테이너 기반의 웹 요청·VM 관리 구조 · /api 요청은 Go API로 직접 전달됩니다. 이미지를 누르면 크게 볼 수 있습니다.</figcaption>
              </figure>
              <ol className="marketing-guide-steps marketing-architecture-steps">
                {architectureTimelineItems?.map((item, index) => (
                  <li key={index}><span>0{index + 1}</span><div>{item.content}</div></li>
                ))}
              </ol>
            </section>

            <section id="ssh-gateway" className="marketing-guide-section" aria-labelledby="gateway-title">
              <p className="marketing-eyebrow">05 / SSH gateway</p>
              <h2 id="gateway-title">하나의 SSH 주소로 VM에 접속하세요.</h2>
              <h3>도입한 이유</h3>
              <p>VM마다 외부 SSH 포트를 따로 열지 않기 위해 Kite Gateway가 하나의 SSH 진입점을 받습니다. Gateway는 SSH username을 <code>spec.sshId</code>와 매칭해 대상 VM을 찾습니다.</p>
              <figure className="docs-concept-figure">
                <a href="/images/docs/ssh-gateway-routing.webp" target="_blank" rel="noreferrer" aria-label="SSH Gateway 연결 구조 원본 보기 (새 탭)">
                  <Image src="/images/docs/ssh-gateway-routing.webp" width={1983} height={793} sizes="(max-width: 720px) calc(100vw - 72px), (max-width: 1024px) calc(100vw - 336px), 664px" alt="SSH 클라이언트가 라우터와 LoadBalancer Service를 거쳐 Kite SSH Gateway에 연결합니다. Gateway는 SSH ID와 초기 비밀번호를 확인하고 관리 SSH 키로 클러스터 내부 VM의 22번 포트에 연결합니다. 사용자 비밀번호는 VM에 전달하지 않습니다." />
                </a>
                <figcaption className="docs-figure-label">SSH Gateway 연결 구조 · 외부 22번 → Service 12311번은 포트 전달 예시입니다. 이미지를 누르면 크게 볼 수 있습니다.</figcaption>
              </figure>
              <h3>로그인 프록시 방식</h3>
              <p>사용자는 VM 생성 시 입력한 초기 비밀번호로 Gateway에 로그인합니다. Gateway는 이 비밀번호를 VM에 전달하지 않습니다. 인증 후 Kite 관리 SSH key로 VM sshd에 접속하고 세션만 중계합니다.</p>
              <div className="marketing-guide-callout"><strong>현재 지원 범위</strong><p>외부 사용자 public key 인증은 아직 지원하지 않습니다.</p></div>
              <h3>host SSH 포트</h3>
              <p>설치 스크립트는 host sshd 포트를 변경하지 않습니다. Level 3 관리자는 Admin Settings에서 Gateway Service 포트와 사용자 안내 포트를 따로 저장할 수 있습니다.</p>
              <p>예를 들어 외부 라우터가 22번을 Service 12311번으로 넘기면 UI는 사용자에게 22번 접속 명령을 보여줍니다.</p>
            </section>

            <div className="marketing-guide-end"><Link href="/login" className="marketing-button marketing-button--primary">Kite 시작하기 <ArrowRightOutlined /></Link><a href="#guide-top" className="marketing-text-link">맨 위로 <ArrowUpOutlined /></a></div>
          </div>
        </div>
      </main>
    </div>
  );
}
