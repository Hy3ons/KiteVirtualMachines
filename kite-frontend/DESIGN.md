# Kite — design system

## 1. Direction and references

The public site is a quiet product launch: white canvas, black identity, generous
Korean typography, pale sky surfaces, and a dimensional, original console preview.
The workspace is an instrument: a persistent navigation rail, compact project
header, measured data cards, useful filters, and readable resource tables.

References inspected: https://aside.com/ (live page and screenshot, 2026-10-30),
https://openai.com/ko-KR/business/openai-presence/ (page content; live browser
initially challenged). Aside supplies bright framing and soft continuous corners;
Presence supplies restrained section hierarchy. Google Cloud/Analytics are user
references for navigation and information architecture, not copied assets.
No source logos, screenshots, marketing copy, or invented live metrics are reused.

The landing hero uses an original generated pastel cloudscape with a folded-paper
kite, saved as `public/images/kite-pastel-hero.webp`. It is decorative background
artwork with a quiet, light center for the live heading and actions. Its prompt
and provenance are recorded in `docs/assets/kite-pastel-hero.md`.

The previous cream/brown, Georgia-heading, zero-radius design is retired by the
user's explicit redesign instruction. Existing Go endpoints, authorization,
session ownership, SSR contracts and VM/console operations remain the authority.

## 2. Color roles

| CSS token | Value | Role |
| --- | --- | --- |
| --canvas | #ffffff | Public canvas, cards |
| --workspace | #f7f8fa | App canvas |
| --surface | #f1f3f5 | Controls and inset surfaces |
| --surface-hover | #e9edf2 | Hover surface |
| --ink | #151719 | Body and primary action |
| --ink-secondary | #5c626c | Secondary text |
| --ink-muted | #6b727c | Captions |
| --line | #e5e8ec | Neutral hairlines |
| --line-strong | #cbd1d9 | Control borders |
| --accent | #255bd8 | Workspace links, selected navigation |
| --accent-soft | #eaf0ff | Selected surface |
| --sky | #eaf3fc | Public product atmosphere |
| --sky-deep | #c5dcef | Atmosphere depth |
| --success | #217647 | Running state |
| --success-soft | #edf7f0 | Running wash |
| --warning | #986016 | Pending state |
| --warning-soft | #fff5e4 | Pending wash |
| --danger | #bd333b | Failure and destructive action |
| --danger-soft | #fff0f0 | Failure wash |
| --terminal | #171b22 | Actual serial terminal only |
| --terminal-ink | #e3e8ef | Terminal foreground |

Primary marketing actions are ink, workspace selections are blue washes. No accent
edge stripes. Status always has text or a glyph in addition to color.

## 3. Typography

Use the existing Pretendard Variable font for Korean and Latin UI with native
system fallbacks; no new font dependency or remote script. The prior Georgia and
brown decorative typography are removed. Display: 500, tightly tracked, fluid
44–80px, 1.12 line-height. Section claims: 32–48px. App title: 28–32px, 600.
Section title: 18–20px, 600. Body: 14–16px, 1.6. Small labels: 12px. Metric:
30–36px, 500, tabular figures. Technical identifiers: native ui-monospace, 12px.
Korean prose uses keep-all with overflow-wrap:anywhere; headings balance.

## 4. Spacing and layout

4px base; tokens --space-1 through --space-12: 4,8,12,16,20,24,32,40,48,64,80,96px.
Public content uses --public-content-width:1000px, centered with responsive side
gutters, including public navigation, footer, signup and guide content. Hero and
system section backgrounds keep their wider frames; an inner marketing-content
wrapper limits their actual content. Prose is 62ch; hero background max 1440px
with 16px outer inset.
The public footer starts with a neutral hairline and presents the brand, a short
single-node VM platform description and repository link. Its policy navigation
wraps on small screens. Privacy and terms pages share the public header, 1000px
outer content limit and a centered 760px reading column, with numbered sections,
44px document links and a last-revised date. Policy prose describes account/VM
storage, authentication hashes, request/gateway logs, console transport, the
external font CDN and Google authentication's permitted data scope. Service
terms distinguish software functionality, installation operations and VM-user
responsibilities, with announced idle-VM management and emergency intervention.
Deployment identity, retention and actual OAuth configuration require operational
verification; never replace unknown facts with invented providers or periods.
Documentation uses a compact page introduction, sticky chapter navigation and
task-oriented prose. The VM offer chapter pairs an original generated conceptual
illustration with live numbered captions, role prerequisites, actual menu paths,
field definitions and troubleshooting. The illustration is explicitly labelled
as a conceptual diagram rather than a product screenshot. Figures use a 16px
radius with 16px inset, fixed intrinsic dimensions and responsive width; captions
and three-column role summaries stack at <=720px. Native details/summary provides
keyboard-accessible optional operational notes and troubleshooting, with 44px
summary targets, neutral rules and no new motion or runtime dependencies.
Workspace topbar 64px, sidebar 224px, actual body content max 900px via
--console-content-width, centered within the right column with page padding 32px.
Page and resource-section headings have titles and actions without explanatory
subtitles. Metric labels identify their totals without redundant hint sentences.
Keep permission restrictions, actionable errors and configuration consequences.
The workspace overview starts with its access-level panel, followed by resource
metrics. It has no separate page title, overview eyebrow or personalized heading;
the breadcrumb identifies the workspace and the access-level panel supplies h1.
Public copy and personal VM screens use account/workspace language without
exposing Kubernetes namespace names. Namespace values stay in API contracts and
administrator resource tools where operators need them.
Console pages reserve the document scrollbar gutter so tab height changes do not
shift the workspace horizontally. Admin resource panels, tab panes, tables and
allocation forms fill the same bounded content width; wide tables scroll inside
their own region. Resource tabs use selected surface fills without a bottom rule.
The document owns page scroll; topbar and side navigation are sticky. Terminal
owns a bounded region, xterm owns its internal scroll. No competing body scroller.
Above 720px the appbar toggle switches the sidebar between a 224px labelled menu
and a 76px icon rail. At <=720px it becomes a disclosure drawer with labelled
actions, independent of the desktop collapsed state. Content gutters become
24px then 16px. Cards stack; tables alone may scroll horizontally.
All flex/grid children can shrink.
Desktop collapse state belongs to the root ConsoleNavigationProvider rather than
the per-page shell, so workspace/admin/settings navigation preserves the rail
width. It is an in-memory UI preference; account data stays in page-local stores.

## 5. Primitives and states

- Brand: original black folded-kite mark, wordmark, compact icon-only variant.
- GlobalHeader: public wordmark, docs/repository links, login/console action.
  Authenticated public and console headers use a single person-icon button.
  Its click popover has 24px inset, 24px outer radius and a 304px reading width,
  showing the verified name/email and freshly requested personal VM totals.
  Count all non-deleting VMs and show Running separately; loading/errors never
  imply zero. The existing /vms adapter runs only while this popover is open.
  Include dashboard and logout actions, Escape dismissal and keyboard focus.
  Public headers receive a request-scoped Go-verified session; no shared cache
  or browser storage supplies identity. Inert demo headers never fetch account data.
  Header text/icon actions share a gray squircle hover surface, using an isolated
  pseudo-element that rises 3px and scales .94 to 1 while fading in over 200ms.
  Text stays stationary. Keyboard focus/open state retain the surface; reduced
  motion switches immediately. Filled primary buttons retain their own style.
  Equal flexible side columns flank an auto-width navigation column, centering
  navigation on the page independently of the logo and account-action widths.
- ConsoleShell: sticky header, role-aware nav, account identity, logout,
  breadcrumbs, main landmark; a keyboard-accessible mobile disclosure menu.
  Account identity uses a neutral person icon; the popup shows username and
  registration email from Go's verified profile, wrapping long addresses.
- PageHeading: eyebrow, title, description, wrapping action cluster.
- MetricCard: truthful current resource totals, unit, descriptive footer; no
  historical trend or utilization without API data. Loading uses em dashes.
- StatusBadge: neutral/green/amber/red wash, dot and readable state text.
  Serial-console pages use the VM name as their page heading and keep VM power
  status in the heading's action cluster. The console panel has one connection
  badge such as "Serial console connected"; no blue connection-alert strip.
  Non-connected states retain their diagnostic text in the panel header, with
  error ink for failure. Live and marketing console frames share this behavior.
  Gateway alert headings place the label and badge as direct flex children with
  centered alignment and a 32px minimum row height to avoid inline baseline drift.
  SSH Gateway activation/settings opens a modal from the status heading. Only its
  two port fields are edited there, with question-icon help. A successful Go save
  closes the editor; validation or request failures keep input visible. Enabled
  gateways can be disabled in the same editor. Disabled gateways omit empty
  operational metadata, and no inline configuration form remains below status.
- Surface: neutral hairline, white fill, squircle corners; no hover lift unless
  the surface is actionable. Public preview alone uses depth shadow.
- Resource table: compact rows, quiet header, horizontal overflow region,
  actual search/status filters and existing VM operation controls.
  Personal VM rows show name, domain, SSH ID, status and a trailing vertical
  ellipsis button. Hover/focus on the name reveals CPU, memory and disk in a
  popover. The click menu uses text-only actions, preserves permission and power
  state restrictions, and opens a confirmation dialog before deletion.
  The action dropdown explicitly owns its nested geometry: outer radius 16px,
  menu inset 8px, item radius = outer radius minus inset (8px), 40px rows and
  8px/12px item padding. Scoped selectors outrank Ant's menu/item selectors so
  hover fills preserve that inset. Danger hover/focus uses a pale red wash and
  red text; it does not turn into a solid red tile. Focus outlines stay unclipped.
  The VM toolbar places search and status immediately before refresh and create,
  on the heading row at desktop widths. Small screens wrap controls in the same
  order. No filtered/total result counter is displayed.
  Assigned VM offers use four Korean-labelled columns: name/image, specs, expiry
  and claim action, with a 680px table minimum. The gift glyph is centered in a
  fixed 44px flex tile aligned with the section title.
- AuthPanel: labelled email/password, inline validation, submit busy state,
  current session actions. Login/signup continue to use authApi.
- Ant Design overlays: consistent modal/drawer/input/button tokens; existing
  validation, popconfirm, error notification and pending indicators preserved.

Default/hover/pressed/focus-visible are explicit for every custom control.
Tooltips use 12px text with 20px line-height, a 40px minimum height, 8px vertical
and 12px horizontal padding. Center the content vertically inside that minimum
height for both standard and shared tooltip containers; multiline text can grow.
Gateway port labels have focusable question icons. Their help allows up to 360px
within the viewport and distinguishes the Kubernetes Service port from the public
SSH port, with router forwarding examples; help never implies automatic router
or host SSH configuration.
Disabled and loading use native disabled/AntD loading semantics. Empty states
offer the permitted next action; errors describe failures without granting access.
Navigation uses real links and aria-current. Icon-only controls have names.

## 6. Motion

The landing product demo reuses the live ConsoleShell, account summary, VM table,
toolbar, creation form and console framing. The desktop canvas is 1920 × 920,
with a 224px rail and the actual 900px console content centered in the remaining
area. The whole canvas scales into a maximum 900px browser frame, giving a
431px-high viewport at full width and a deliberately distant product view.
Chrome follows the user's Safari compact-tabs screenshot: one 48px light bar,
with 8px red/yellow/green window controls (#ff5f57/#febc2e/#28c840), a floating
sidebar capsule and circular back control. A #ececed pill groups two quiet Kite
tabs and a white active address capsule, followed by share/new-tab/tab-overview
tools in another floating capsule. Native-style icons stay 14px, surfaces have
white hairlines and restrained neutral shadows, and the address shows the host
while its full route/title follows the scene. At narrow widths, background tabs
and sharing hide while the active address and main controls remain. No
visible product-demo label is shown. The pastel hero has zero bottom padding
and clips the browser's final 64px (32px on phones) at its rounded lower edge,
as in the user's reference. Playback controls precede the window so the crop
does not hide them. The hero's outer shadow carries its lower edge into white.
This changes the landing demo only.
The embedded application is inert and isolated from real APIs,
authentication, prefetch, WebSocket connections and persistent navigation state.
Modal/dropdown layers remain inside the scaled canvas rather than document.body.

GSAP is the user-selected motion engine and is loaded only as the demo approaches
the viewport. A 54s story clock at 1.4× speed (about 39s elapsed) loops through
login, VM creation, serial-console commands, dashboard return, VM deletion with
confirmation, and the landing page. The landing scene uses shared hero copy and
an inert public header without recursively embedding this demo. Its start button
leads into the next cycle. Console text demonstrates whoami, ls, mkdir, cd,
git init, touch, git status, uname and echo with matching prompts/output.
Deleting changes Running to Terminating, then removes the local row and resets
the resource counters. No live command, account or API delete is performed.
DOM-relative pointer targets use
the canvas scale, with power3.inOut moves (0.45–0.6s), a brief six-degree travel
tilt that settles before clicking, 0.10s click presses and readable typing.
The original ink arrow has a white edge and neutral shadow; its tip is the
coordinate origin. A counter-scaled appearance layer keeps the cursor 28 × 35px
on screen even as the product UI shrinks. Click feedback is one quiet neutral
halo, not a trail. Modal field gaps stay 16px; its body has a bounded scroll area
under a stationary title/close row. Console content and terminal output also own
their scroll areas. Timed native scrollTop transitions use a cubic ease, clamp to
each element's actual scroll range and position the next target before pointer
movement. Scroll state is derived from timeline time and measured DOM geometry,
including replay and chapter jumps; the landing document is never scrolled.
The console terminal is 400px high and follows new output inside its own frame.
Other animations use transform/opacity; story values change discretely.
The page camera is separate from browser chrome and the pointer. It smoothly
zooms/pans to the login form (1.75×), workspace creation (1.65×), each VM form
field (2×), the VM actions/menu (1.75–1.85×) and terminal (1.75×). Each task
returns to the full page before a scene change, including the final console.
Framing uses actual DOM geometry and the visible hero crop, with clamped pans
and cubic 0.23–0.55s story-time transitions. Scroll calculations account for
camera scale; pointer moves start after scroll/framing settles. Reduced motion
uses the full-page camera immediately, with no zoom or pan animation.
The timeline repeats indefinitely, pauses offscreen or in a hidden tab, supports explicit
pause/replay, and is reverted on unmount. Chapter controls are available only in
the explicit debug app mode; staging and production render no chapter navigation.
There is no default sample-data disclaimer below the demo. Reduced motion starts
paused with no animated cursor and disables playback; debug retains chapter
selection, while staging/production show the static console. No literal beui.dev cursor
demo matches this choreography; its loader reduced-motion/state-feedback pattern
was inspected, while GSAP's context/timeline docs supply the actual mechanism.

--motion-fast:150ms; --motion-normal:200ms; --ease:cubic-bezier(.2,.7,.2,1).
Only modest control color and opacity transitions; press transform scale(.98).
No scroll listeners, looping illustration, fabricated activity feed or shimmer.
Sidebar toggling uses the native View Transition API to animate snapshots of the
sidebar and main content for 200ms; the grid changes once instead of animating
layout each frame. The appbar stays still. Reduced motion or an unsupported API
switches immediately. A repeated toggle skips the previous transition.
Sidebar/main snapshots keep their intrinsic pixel size with object-fit:none and
top-left alignment inside a clipped transition group. Width interpolation reveals
or hides the snapshot rather than stretching icons and text. Live navigation
icons also have fixed 20px slots and 17px SVG dimensions.
Reduced motion removes transforms and smooth scrolling. Focus uses a visible
2px ring with 3px offset. No entrance animations hide SSR content.

## 7. Surface and squircle geometry

--radius-xs:6px (inline code), --radius-sm:10px (compact controls), --radius-md:16px (inputs/buttons/nav),
--radius-lg:24px (cards), --radius-xl:32px (public frames), --radius-full:999px
(status dots only). Prefer continuous corners with @supports(corner-shape:squircle)
and corner-shape:squircle; fallback is a proportionate native border-radius.
Never clip focus rings or overlay portals to fake the shape. Nested surfaces use
smaller radii. Neutral borders + tonal layers for the app; the public preview
adds --shadow-float:0 24px 80px rgba(35,55,80,.12),0 2px 8px rgba(35,55,80,.05).
--shadow-overlay:0 12px 40px rgba(21,23,25,.12) for modal/popover surfaces only.
The VM allocation dialog uses --shadow-dialog:0 24px 72px rgba(21,23,25,.20),
0 6px 20px rgba(21,23,25,.10) for clearer separation from the workspace.
VM creation password helper text has 8px inline padding and 8px top margin.

Rounded surface inset contract (reviewed across all eight stylesheets):

| Surface | Radius | Content inset and alignment |
| --- | --- | --- |
| Cards, metrics, settings, VM panels | 24px | --inset-panel:24px; 20px on small screens |
| Modal and drawer content | 24px | Same panel inset; modal title has close-button clearance; one 20px header gap |
| Inputs, selects, text buttons (normal/large) | 16px | --inset-control:16px for fields; text buttons 20px normal, 24px large |
| Popovers and dropdown surfaces | 16px | Popover --inset-popover:16px; menu outer 8px, rows 8px/12px |
| Status badges and tags | 10px | Explicit line-height and centered contents; badge 4px/12px |
| Fixed icon tiles | 10px or 16px | Flex/grid centering; fixed dimensions, no text padding |
| Public frames and signup story | 32px | Existing 32–64px desktop insets; at least 24px horizontal on mobile hero |
| Inline guide code | 6px | 2px vertical and 8px horizontal; not a 10px pill with 4px inset |
| Tables nested in resource cards | Outer card 24px, table viewport 16px | Card inset 24px desktop / 20px mobile on both sides and below; table cells retain 16px / 12px padding |

The panel inset is shared by headers, bodies and responsive overrides so their
text starts on the same vertical guide. Integrated terminal strips, table rules,
affixed input interiors and circular status dots are deliberate square/round
exceptions. Preserve focus rings and portal overflow rather than clipping them.
VM creation dialogs use a 40px flex-aligned header row. Their native 40px close
button starts at the same panel inset, giving title and close glyph one shared
vertical center. Scoped selectors outrank Ant Design's generated defaults.
All four resource tables (workspace VMs, assigned VMs, admin users and admin VMs)
share an independently rounded 16px scroll viewport inside their card's safe
inset. The viewport has a neutral 1px frame and a derived 15px inner curve; the
last row has no straight bottom rule. Its existing horizontal scroll area owns
corner clipping, including hover and empty fills. The card and table wrapper
stay unclipped, preserving notification/menu portals and outer focus treatment.
Table-edge focusable content keeps the cell inset rather than touching the crop.

## 8. Public metadata and social previews

Generated social artwork uses the existing original Kite mark and pastel cloud
direction, at 1200×630 JPEG: the general brand card and the guide's VM lifecycle
variant. These are share assets, not screenshots or private resource previews.
Every page has a Korean title/description, matching OG/Twitter text, absolute
image URLs, dimensions, MIME type and image alt text. Public canonical URLs drop
tracking queries; auth utilities, private pages, missing pages and policy pages
have no canonical and are not indexed. A fixed public origin can be configured;
otherwise request-scoped trusted proxy/Host information supplies the origin.

## 9. Accessibility and verification

Light mode is deliberate; browser color-scheme:light. Body text contrast targets
WCAG AA. Regular controls are 40px, compact secondary controls 32px and large
primary controls 48px. Labelled
forms, skip link, landmarks, focus treatment, no decorative emoji, useful loading
and error routes. No private SSR data or JWT is cached in shared browser state.

The user's standing QA instruction overrides browser/Playwright skill gates:
no local browser E2E, automated screenshots, Lighthouse or interaction harness.
Use typecheck, lint, fetch tests, production build and HTTP-only route smoke.
The user owns visual/responsive/keyboard interaction QA. No measured Lighthouse
score or pixel-fidelity claim is made. React inspection overlays/scripts are not
added to the product during this design change.
