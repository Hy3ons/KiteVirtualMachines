import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
import { shouldClearSessionCookie } from './server/sessionCookie';

// proxy removes expired cookie input before page authorization and clears it in
// the browser response. request is the incoming page request; Go owns authorization.
export function proxy(request: NextRequest) {
  const cookie = request.cookies.get('accessToken');
  if (!cookie || !shouldClearSessionCookie(cookie.value, Date.now())) return NextResponse.next();
  request.cookies.delete('accessToken');
  const response = NextResponse.next({ request: { headers: request.headers } });
  response.cookies.delete('accessToken');
  return response;
}

export const config = { matcher: ['/((?!api|_next|images|favicon|apple-touch-icon|robots.txt|sitemap.xml).*)'] };
