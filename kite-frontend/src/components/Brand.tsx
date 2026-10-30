import Link from 'next/link';

type BrandProps = { readonly compact?: boolean; readonly href?: string; readonly className?: string; readonly prefetch?: boolean | null };

// Brand links to href with the original folded-kite symbol and optional wordmark.
// compact supplies an icon-only identity for narrow navigation and product previews.
export function Brand({ compact = false, href = '/', className = '', prefetch = null }: BrandProps) {
  return (
    <Link href={href} prefetch={prefetch} className={`kite-brand ${className}`} aria-label="Kite 홈">
      <svg viewBox="0 0 40 40" className="kite-mark" aria-hidden="true">
        <path d="M20 3.5c-1.2 0-2.4.5-3.3 1.4L5 16.6c-1.9 1.9-1.9 4.9 0 6.8l9 9c.9.9 2.4.3 2.4-1V16.4H31c1.3 0 1.9-1.5 1-2.4l-9-9a4.6 4.6 0 0 0-3-1.5Z" fill="currentColor" />
        <path d="M23.6 20.2H20v11.2c0 3.7 4.4 5.6 7.1 3l8-8c2.6-2.6.8-7.1-3-7.1h-3.3v3.2c0 3.1-2.5 5.6-5.6 5.6h-.4v-7.9Z" fill="currentColor" />
      </svg>
      {!compact && <span className="kite-wordmark">Kite<span className="brand-period">.</span></span>}
    </Link>
  );
}
