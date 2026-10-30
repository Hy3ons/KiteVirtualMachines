import type { MetadataRoute } from 'next';
import { getPublicOrigin } from '../server/publicOrigin';

// robots advertises public discovery while keeping crawlers out of API and console routes.
// Authentication remains enforced by Go and the Next session guards, not by this file.
export default async function robots(): Promise<MetadataRoute.Robots> {
  const origin = await getPublicOrigin();
  return {
    rules: { userAgent: '*', allow: '/', disallow: ['/api/', '/dashboard', '/admin'] },
    sitemap: new URL('/sitemap.xml', origin).href,
  };
}
