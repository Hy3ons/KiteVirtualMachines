import 'server-only';

import { cache } from 'react';
import { headers } from 'next/headers';

class InvalidPublicOriginError extends Error { readonly name = 'InvalidPublicOriginError'; }

// getPublicOrigin resolves this installation's public origin for metadata URLs.
// An explicit KITE_PUBLIC_URL wins; otherwise trusted ingress headers or the local
// request host supply the origin. React cache shares it only within one render.
export const getPublicOrigin = cache(async (): Promise<URL> => {
  const configured = process.env.KITE_PUBLIC_URL?.trim();
  if (configured) {
    const url = new URL(configured);
    if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password || url.pathname !== '/' || url.search || url.hash) {
      throw new InvalidPublicOriginError('KITE_PUBLIC_URL must be an HTTP(S) origin without credentials, a path, query or fragment');
    }
    return url;
  }

  const incoming = await headers();
  const host = (incoming.get('x-forwarded-host') || incoming.get('host') || '').split(',').at(0)?.trim() ?? '';
  if (!host || /[\s/@\\?#]/.test(host)) throw new InvalidPublicOriginError('A valid public request host is required');
  const forwardedProtocol = incoming.get('x-forwarded-proto')?.split(',').at(0)?.trim();
  const local = /^(localhost|127\.0\.0\.1|\[::1\])(?::\d+)?$/.test(host);
  const protocol = forwardedProtocol || (local ? 'http' : 'https');
  if (protocol !== 'http' && protocol !== 'https') throw new InvalidPublicOriginError('Public metadata requires HTTP or HTTPS');
  const origin = new URL(`${protocol}://${host}`);
  if (origin.username || origin.password || origin.pathname !== '/' || origin.search || origin.hash) {
    throw new InvalidPublicOriginError('Invalid forwarded public origin');
  }
  return origin;
});
