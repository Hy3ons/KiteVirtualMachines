import type { Metadata } from 'next';
import { LandingPage } from '@/screens/LandingPage';
import { createPageMetadata } from '@/lib/metadata';
import { AuthProvider } from '@/store/useAuthStore';
import { getSession } from '@/server/api';

// generateMetadata publishes the homepage identity and installation-specific share URLs.
export function generateMetadata(): Promise<Metadata> {
  return createPageMetadata('Kite', { home: true, path: '/' });
}

// Page uses the request's verified cookie session for the header and primary action.
export default async function Page() {
  const user = await getSession();
  return <AuthProvider key={user ? user.username + ':' + user.access_level : 'anonymous'} user={user}><LandingPage startHref={user ? '/dashboard' : '/login'} /></AuthProvider>;
}
