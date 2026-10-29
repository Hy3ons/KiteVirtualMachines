import type { NextConfig } from 'next';

class InvalidMockModeError extends Error {
  readonly name = 'InvalidMockModeError';
}

const nextConfig: NextConfig = {
  output: 'standalone',
  // Production Ingress sends /api directly to Go. Only local next dev uses this proxy.
  async rewrites() {
    if (process.env.NODE_ENV !== 'development') return [];
    return [{ source: '/api/v1/:path*', destination: `${process.env.KITE_API_INTERNAL_URL || 'http://127.0.0.1:8080'}/api/v1/:path*` }];
  },
};

if (process.env.NEXT_PUBLIC_USE_MOCK === 'true' && process.env.NEXT_PUBLIC_KITE_MODE !== 'debug') {
  throw new InvalidMockModeError('Mock API is only allowed in explicit debug builds');
}

export default nextConfig;
