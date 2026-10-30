import type { Metadata } from 'next';
import { getPublicOrigin } from '../server/publicOrigin';

export const HOME_TITLE = 'Kite | Kubernetes 기반 VM 플랫폼';
export const HOME_DESCRIPTION = 'Kite는 Kubernetes와 KubeVirt 기반의 작은 VM 플랫폼입니다. 소규모 팀이 단일 노드 온프레미스 환경에서 가상머신을 할당하고 관리할 수 있습니다.';

type PageMetadataOptions = {
  readonly description?: string;
  readonly noIndex?: boolean;
  readonly path?: string;
  readonly image?: 'brand' | 'guide';
  readonly home?: boolean;
};

// createPageMetadata gives route titles and social previews the same page identity.
// title omits the brand supplied by the root template; options chooses the public
// path, description, illustration and indexing policy without changing authorization.
export async function createPageMetadata(title: string, { description = HOME_DESCRIPTION, noIndex = false, path, image = 'brand', home = false }: PageMetadataOptions = {}): Promise<Metadata> {
  const origin = await getPublicOrigin();
  const socialTitle = home ? HOME_TITLE : `${title} | Kite`;
  const pageUrl = path ? new URL(path, origin) : undefined;
  const imageUrl = new URL(image === 'guide' ? '/images/social/kite-guide-social.jpg' : '/images/social/kite-social.jpg', origin);
  const alt = image === 'guide'
    ? '관리자 사전 할당, 사용자 수락, 가상머신 생성으로 이어지는 Kite VM 할당 흐름'
    : 'Kite 로고와 나만의 작은 클라우드 문구, 파스텔 구름 위 종이연과 가상머신';
  return {
    title: home ? { absolute: HOME_TITLE } : title,
    description,
    alternates: { canonical: !noIndex && pageUrl ? pageUrl : null },
    robots: noIndex ? { index: false, follow: false } : { index: true, follow: true },
    openGraph: {
      title: socialTitle, description, type: 'website', siteName: 'Kite', locale: 'ko_KR', url: pageUrl,
      images: [{ url: imageUrl, width: 1200, height: 630, type: 'image/jpeg', alt }],
    },
    twitter: { title: socialTitle, description, card: 'summary_large_image', images: [{ url: imageUrl, alt }] },
  };
}
