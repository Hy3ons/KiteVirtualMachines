# Kite frontend design convention

The implementation contract is [`DESIGN.md`](../DESIGN.md). The user's 2026-10-30
redesign instruction replaces the previous cream palette and zero-radius rules.

- Public pages: Aside and OpenAI Presence inspired bright composition, black
  original Kite identity, large Korean typography, pale sky product framing.
- Workspace: Google Cloud/Analytics inspired persistent navigation, current
  resource totals, functional filters and dense resource tables.
- Corners: iOS-style continuous squircles via `corner-shape: squircle` where
  supported; proportionate `border-radius` fallback elsewhere. Inputs, buttons,
  cards and sheets use distinct radii from the shared scale.
- Theme: `src/design/tokens.ts`, `src/theme.ts`, and `src/styles/tokens.css` mirror
  the palette and geometry in DESIGN.md. Change the contract before adding tokens.
- Shared components: Brand, ConsoleShell, MetricCard, StatusBadge, GlobalHeader
  and GlobalFooter. Keep account/session data scoped to the route AuthProvider.
- All VM, user, offer, SSH, console and settings operations stay connected to the
  existing Go API. No invented live metrics, uptime or historical charts.
- Verification: HTTP/fetch tests, types, lint and production build. The user owns
  visual and interaction QA; do not run Playwright or browser E2E.
