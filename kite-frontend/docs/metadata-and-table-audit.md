# Metadata and table audit — 2026-10-30

## Changes and evidence

The social preview previously reused a small favicon and omitted images when
`KITE_PUBLIC_URL` was unset. It now uses two optimized generated 1200×630 JPEGs:
the brand card (50,287 bytes) and the VM guide variant (32,340 bytes). Artwork and
the final generation prompt are recorded in [social-preview.md](./assets/social-preview.md).

Metadata now includes page-specific descriptions, matching OG/Twitter titles and
descriptions, site/locale, absolute image URLs, MIME type, dimensions and image
alt text. Public canonical/OG URLs omit tracking queries. Login/signup, console,
404 and unfinalized policy drafts are not indexed and do not claim the home
canonical. Sitemap discovery includes only the homepage and public guide.

`npm run audit:metadata` passed across 14 HTTP scenarios in development, covering
public pages, query variants, private mock screens and 404. It also passed against
the production server, where private routes were verified as login redirects.
Facebook/Twitter crawler user agents received metadata in the document head;
both social images returned public JPEG responses. Sharp confirmed 1200×630.

The request-origin production configuration passed simultaneous HTTP/HTTPS
forwarded-host checks without mixing origins. Explicit `KITE_PUBLIC_URL` was
verified to override forwarded host/protocol. No external fetch is made to a
host derived from those headers. Fixed-origin configuration and its static
prerender tradeoff are documented in the README.

Open Graph fields were checked against the [official protocol](https://ogp.me/),
and Next metadata handling against the installed framework guides/source.
No real social platform upload or cache-refresh test was performed.

## Resource table inventory and geometry

| Surface | Source | Previous issue | Shared correction |
| --- | --- | --- | --- |
| Workspace VMs | `UserDashboard.tsx` | 8px bottom/side inset; square bottom viewport | 24px inset (20px mobile), fully rounded scroll viewport |
| Assigned VMs | `UserVmOffersSection.tsx` | Same workspace wrapper behavior | Same inset and viewport contract |
| Administrator users | `AdminDashboard.tsx`, users pane | Ant's default upper corners only | Keeps existing panel inset; shares rounded viewport |
| Administrator VMs | `AdminDashboard.tsx`, vms pane | Same Ant table default | Same shared viewport contract |

Ant Design's installed `table/style/radius.js` supplies upper corner radii;
lower corners are not supplied for a table without a footer. The shared CSS
now frames the scroll viewport at 16px with a 15px inset curve inside its neutral
1px border. Last-row bottom rules no longer draw a straight line at the foot.
Hover fills, empty placeholders and loading-mask geometry use rounded bounds.
Horizontal scrolling remains on the native content viewport. No clipping is
added to the card or wrapper, and cell padding leaves room for focus outlines.
The SSH Gateway's tabular `Descriptions` already has a 16px clipped view inside
the settings-card inset; squircle support was added to that existing view too.

The design contract was updated before the table style edits. Type checking,
lint, 26 existing HTTP tests, CSS parsing, accent-stripe checks and the final
production build passed. These are source/build/HTTP checks, not pixel evidence.

## User QA

Per repository instructions, no browser/Playwright E2E, screenshots or Lighthouse
were run. The user should verify the four tables at desktop/mobile widths,
horizontal-scroll endpoints, last-row hover, empty/loading states and keyboard
focus/menu behavior. The inactive admin VM pane requires actual UI navigation;
HTTP SSR alone does not exercise that pane or hover states.
