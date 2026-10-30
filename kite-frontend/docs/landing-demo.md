# Landing product demonstration

The homepage uses GSAP 3.15 to demonstrate the existing product with local data.
It is a DOM composition, rather than a recording or a screenshot. The live
`ConsoleShell`, dashboard summary, toolbar, table columns, VM creation modal,
login form and serial-console frame supply the actual appearance. The logical
canvas is 1920 × 920: a 224px sidebar and the existing 900px content centered in
the remaining area. ResizeObserver scales this whole canvas into a maximum
900px browser frame. The full content viewport is approximately 431px high at
maximum width, and the original dashboard content appears 422px wide. The hero
has no bottom padding and clips the browser's last 64px (32px on phones) at its
rounded background edge, matching the user's supplied bottom-crop reference.
Playback controls sit above the browser so this crop does not hide them. The
user's Safari screenshot supplies the browser framing: a single 48px compact-tab
bar, traffic lights, sidebar/back capsules, two inactive Kite tabs and a white
active address pill, followed by share/new-tab/tab-overview tools. At phone
widths, background tabs and sharing hide; the address and main controls remain.
The sample address follows the existing login/dashboard/console route names.
The actual dashboard is unchanged.
The pointer's appearance is counter-scaled to keep a 28 × 35px screen size, with
the arrow tip at the measured target, restrained travel tilt and click feedback.

## Story and ownership

`src/components/marketing/demo/demoStory.ts` maps a time to a complete local
snapshot. A 54-second story clock loops at 1.4× speed (about 39 elapsed seconds).
It types a sample login, shows an empty workspace,
fills the VM creation form, changes Creating to Running, opens the row's Console
menu and logs into Ubuntu. It types `whoami`, `ls`, `mkdir kite-demo`, `cd kite-demo`,
`git init -b main`, `touch README.md`, `git status --short`, `uname -s` and an echo
command, with matching output and working-directory prompts. It then returns via
the workspace navigation, opens the VM menu and confirms deletion. The row changes
to Terminating, disappears and leaves the resource counters at zero. Finally, the
brand leads back to a miniature homepage built from the shared hero copy, without
recursively mounting another demonstration. Clicking its start button connects
to the next login cycle. Commands and deletion are display-only local states.
Passwords are masked in the forms and are never printed in the terminal output.

`demoTimeline.ts` animates the pointer, click feedback and progress with GSAP.
Target positions come from the mounted DOM and are converted into canvas units;
they are not fixed screen coordinates. Virtual clicks do not dispatch events.
`useDemoPlayback.ts` loads the engine on viewport entry, owns observers and
timeline disposal, and pauses when offscreen or when the document is hidden.
Replay and chapter jumps derive state from time, so previously typed fields and
created rows cannot accumulate across runs. Infinite repetition refreshes measured
pointer targets and restores the cursor/camera/scroll state at each cycle boundary.
`demoScroll.ts` derives native
scrollTop from the same clock and actual element geometry: it positions the
create button, scrolls through modal inputs to Submit, reveals the created VM
and then brings the console panel into view before pointer movement. The modal
title stays above its bounded body. Terminal output follows its own scroll area.
All scroll destinations are clamped to their local content range; neither
window.scrollTo nor scrollIntoView is used, so the landing document stays put.

`demoCamera.ts` derives a reversible page-only zoom/pan from that clock. Login,
workspace creation, VM fields, row actions and the console each get a measured
focus frame, followed by deletion confirmation and the landing start button.
Every task zooms out before its next scene. The browser chrome and fixed-size pointer remain
outside the camera. Geometry is inverted through the current camera to avoid
drift; scroll measurements use the element's effective scale, and framing uses
the actual visible hero edge so active targets do not land in the cropped area.
Reduced motion uses the overview immediately without camera animation.

The app canvas is inert and hidden from assistive navigation. Playback controls
sit outside it. Chapter selection is rendered only when
`NEXT_PUBLIC_KITE_MODE=debug`, including debug mode with mocks disabled. Stage
(`staging`) and production omit the chapter navigation; `NODE_ENV` is not used
because `npm run stage` also runs a Next development server. The default
sample-data disclaimer is omitted. Reduced motion disables playback and pointer
animation, opens a static console example and retains debug-only chapter selection.
The demonstration has its own auth/navigation providers, disables link prefetch,
keeps overlays within the canvas and supplies local callbacks instead of the
API or WebSocket adapters. A visitor's actual login/session is not used.

## Verification

- `npm run typecheck` and `npm run lint` pass.
- `npm test`: 50 cases, including nine story-clock cases, two real GSAP repeat/seek
  cases using plain-object targets, four scroll cases, six camera cases and
  29 HTTP contract cases. No browser engine is used.
- `npm run build:prod` passes with mock mode disabled.
- HTTP checks confirm the homepage renders the inert demo and its controls,
  with one main element and no server-rendering error.

Per the user's instructions, no Playwright, browser E2E or visual automation was
run. Manual QA remains with the user: watch the complete sequence, inspect cursor
alignment on the form/menu, zoom-out before navigation, hero-bottom clipping,
deletion confirmation, the empty workspace, home return and repeated cycle;
pause/replay and chapter jumps, resize the window, scroll the demo out of view
and test the OS reduced-motion preference. The
production login and console continue using the existing Go-backed flows.
