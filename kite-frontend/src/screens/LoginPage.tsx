'use client';

import { GlobalHeader } from '../components/GlobalHeader';
import { AuthPanel } from '../components/marketing/AuthPanel';

// LoginPage places account access in a focused public page, separate from the landing content.
export function LoginPage() {
  return (
    <div className="public-page">
      <GlobalHeader />
      <main id="main-content" className="marketing-login-page" aria-label="로그인">
        <AuthPanel />
      </main>
    </div>
  );
}
