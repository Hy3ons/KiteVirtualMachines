import { Brand } from '../../Brand';
import { AccountMenu } from '../../AccountMenu';

// DemoPublicHeader mirrors the public navigation without prefetch or account requests.
// authenticated selects the static person icon after the example account signs in.
export function DemoPublicHeader({ authenticated = false }: { readonly authenticated?: boolean }) {
  return <div className="public-header">
    <Brand prefetch={false} />
    <div className="public-nav"><span className="appbar-action">플랫폼</span><span className="appbar-action">가이드</span><span className="appbar-action">GitHub</span></div>
    <div className="public-header-actions">{authenticated ? <AccountMenu preview /> : <><span className="public-login-link appbar-action">로그인</span><span className="button button--ink button--compact">콘솔 열기</span></>}</div>
  </div>;
}
