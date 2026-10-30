'use client';

import Link from 'next/link';
import { ArrowRightOutlined } from '@ant-design/icons';
import { GlobalHeader } from '../components/GlobalHeader';
import { ConsolePreview } from '../components/marketing/ConsolePreview';
import { LandingHeroCopy } from '../components/marketing/LandingHeroCopy';

// LandingPage introduces Kite and links account access to the dedicated login route.
// Its bounded product animation demonstrates the existing UI without using live account data.
export function LandingPage() {
  return (
    <div className="public-page">
      <GlobalHeader />
      <main id="main-content" className="marketing-main">
        <section className="marketing-hero" aria-labelledby="marketing-title">
          <div className="marketing-content">
            <LandingHeroCopy />
            <ConsolePreview />
          </div>
        </section>
        <div className="marketing-foundation" aria-label="기반 기술">
          <span>Built on open infrastructure</span>
          <div><strong>Kubernetes</strong><span aria-hidden="true">/</span><strong>KubeVirt</strong><span aria-hidden="true">/</span><strong>Open source</strong></div>
        </div>
        <section id="platform" className="marketing-section marketing-capabilities" aria-labelledby="capabilities-title">
          <div className="marketing-section-intro">
            <p className="marketing-eyebrow">A small, open cloud.</p>
            <h2 id="capabilities-title">하나의 노드에서 시작하는<br />나만의 VM 플랫폼.</h2>
            <p>Kite는 Kubernetes와 KubeVirt 기반의 작은 오픈소스 VM 할당 플랫폼입니다. 서버 한 대의 자원을 사용자별로 나누고, 가상머신 생성부터 외부 접속까지 한곳에서 관리합니다.</p>
            <Link href="/kite-docs" className="marketing-text-link">Kite 사용 가이드 <ArrowRightOutlined /></Link>
          </div>
          <ol className="marketing-capability-list">
            <li><span className="marketing-step-number">01</span><div><h3>단일 노드에서 시작하는 작은 클라우드</h3><p>작은 서버에서도 VM을 나눠 쓸 수 있도록, 사용자별 작업 공간을 구분하고 리소스 할당량을 관리합니다.</p></div></li>
            <li><span className="marketing-step-number">02</span><div><h3>Kubernetes가 관리하는 가상머신</h3><p>VM의 사양과 전원 상태를 Kubernetes 리소스로 기록합니다. Kite controller가 KubeVirt와 연동해 실제 VM을 요청한 상태로 맞춥니다.</p></div></li>
            <li><span className="marketing-step-number">03</span><div><h3>SSH Gateway로 외부에서도 접속</h3><p>클러스터 내부의 VM까지 SSH 연결을 중계하는 Gateway를 제공합니다. 대시보드에 표시된 명령으로 접속하고, 브라우저에서는 웹 콘솔을 사용할 수 있습니다.</p></div></li>
          </ol>
        </section>
        <section className="marketing-system" aria-labelledby="system-title">
          <div className="marketing-content marketing-system-content">
            <div>
              <p className="marketing-eyebrow">Open by design</p>
              <h2 id="system-title">단순한 화면 뒤,<br />분명한 구조.</h2>
              <p>요청은 API로, 원하는 상태는 CRD로.<br />Kite controller가 실제 리소스를 맞춥니다.</p>
              <a className="marketing-text-link" href="https://github.com/Hy3ons/KiteVirtualMachines" target="_blank" rel="noreferrer">소스 살펴보기 <ArrowRightOutlined /></a>
            </div>
            <ol className="marketing-system-flow" aria-label="Kite 리소스 생성 구조">
              <li><span>01 / Request</span><strong>Kite API</strong><p>세션과 권한 확인</p></li>
              <li><span>02 / Desired state</span><strong>Kubernetes CRD</strong><p>사용자가 요청한 상태 기록</p></li>
              <li><span>03 / Reconcile</span><strong>Kite controller</strong><p>KubeVirt 리소스 조정</p></li>
            </ol>
          </div>
        </section>
      </main>
    </div>
  );
}
