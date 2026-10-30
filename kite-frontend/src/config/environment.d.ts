// Declare only the environment keys this frontend reads. They remain optional:
// deployment settings may be absent, and callers must supply or validate defaults.
// Named NEXT_PUBLIC accesses also preserve Next.js build-time substitution.
declare namespace NodeJS {
  interface ProcessEnv {
    readonly KITE_API_INTERNAL_URL?: string;
    readonly KITE_PUBLIC_URL?: string;
    readonly NEXT_PUBLIC_API_BASE_URL?: string;
    readonly NEXT_PUBLIC_KITE_MODE?: string;
    readonly NEXT_PUBLIC_USE_MOCK?: string;
    readonly NEXT_PUBLIC_DEBUG_DIRECT_ROUTES?: string;
  }
}
