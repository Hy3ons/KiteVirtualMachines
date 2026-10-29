import type { Metadata } from 'next';
import { LandingPage } from '@/screens/LandingPage';
import { createPageMetadata } from '@/lib/metadata';

// generateMetadata publishes the homepage identity and installation-specific share URLs.
export function generateMetadata(): Promise<Metadata> {
  return createPageMetadata('Kite', { home: true, path: '/' });
}

// Page renders the public landing content without loading account-specific session data.
export default function Page() {
  return <LandingPage />;
}
