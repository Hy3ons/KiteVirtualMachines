'use client';

import Link from 'next/link';
import Image from 'next/image';
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
        <section id="architecture" className="marketing-system" aria-labelledby="system-title">
          <div className="marketing-content marketing-system-content">
            <header className="marketing-system-heading">
              <h2 id="system-title">Kubernetes와 CRD로,<br />상태를 설계합니다.</h2>
              <p>사용자의 요청을 CRD로 정의한 Kubernetes 리소스에 기록합니다. Kite Controller는 선언된 내용을 기준으로 VM과 관련 리소스를 지속적으로 조정하며 안정적인 운영을 제공합니다.</p>
            </header>
            <figure className="marketing-system-figure">
              <a href="/images/kite-kubernetes-crd-diagram.webp" target="_blank" rel="noreferrer" aria-label="Kubernetes와 CRD 개념도 원본 보기 (새 탭)">
                <Image className="marketing-system-art" src="/images/kite-kubernetes-crd-diagram.webp" width={1672} height={941} sizes="(max-width: 680px) calc(100vw - 112px), (max-width: 900px) 568px, 492px" alt="CRD가 KiteVirtualMachine의 형식을 정의하고, 컨트롤러가 spec을 읽어 KubeVirt VM을 조정한 뒤 status를 갱신하는 Kubernetes 제어 루프." />
              </a>
            </figure>
          </div>
        </section>
      </main>
    </div>
  );
}
