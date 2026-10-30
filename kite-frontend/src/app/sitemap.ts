import type { MetadataRoute } from 'next';
import { getPublicOrigin } from '../server/publicOrigin';

// sitemap lists indexable public content only; auth utilities, draft policies and
// private resources stay out. No fabricated last-modified date is published.
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const origin = await getPublicOrigin();
  return ['/', '/kite-docs'].map((path) => ({ url: new URL(path, origin).href }));
}
