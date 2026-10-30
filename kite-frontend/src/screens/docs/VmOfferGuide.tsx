import Image from 'next/image';

// VmOfferGuide briefly introduces admin allocation and user acceptance in the public guide.
// The conceptual image illustrates the feature without repeating detailed operating instructions.
export function VmOfferGuide() {
  return (
    <section id="vm-offers" className="marketing-guide-section" aria-labelledby="offers-title">
      <p className="marketing-eyebrow">03 / VM offers</p>
      <h2 id="offers-title">VM 사전 할당</h2>
      <p>관리자가 사용자에게 VM 사양을 미리 배정할 수 있습니다. 사용자는 대시보드에서 배정 내용을 확인하고 이름과 접속 정보를 입력해 VM을 만듭니다. 직접 생성 한도보다 큰 VM이 필요할 때도 관리자에게 사전 할당을 요청할 수 있습니다.</p>
      <figure className="docs-offer-figure">
        <Image src="/images/docs/vm-offer-lifecycle.webp" alt="관리자가 사양을 배정하고 사용자가 수락하면 가상머신이 생성되는 흐름" width={1536} height={768} sizes="(max-width: 720px) calc(100vw - 72px), (max-width: 1024px) calc(100vw - 336px), 664px" />
        <figcaption className="docs-figure-label">관리자 배정 → 사용자 수락 → VM 생성 · 개념도</figcaption>
      </figure>
    </section>
  );
}
