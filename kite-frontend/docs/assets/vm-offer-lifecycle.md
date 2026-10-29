# VM offer lifecycle illustration

- Created with the built-in `image_gen` tool on 2026-10-30.
- Project asset: `public/images/docs/vm-offer-lifecycle.webp`, 1536 × 768.
- Original preserved at `/Users/hhs2003/.codex/generated_images/01a18957-3b49-7471-9145-93c8fdbb7906/exec-7ee7e610-4657-47e9-a3ad-0ff7c99c431d.png`.
- Encoded as WebP, quality 86, with the original aspect ratio preserved.
- Used by `src/screens/docs/VmOfferGuide.tsx` through `next/image` with fixed
  intrinsic dimensions, responsive sizes, a descriptive alt and live HTML captions.
- Conceptual artwork, not a screenshot. Only an offer specification exists before
  claim; actual VM provisioning occurs after a successful user claim request.
- Source checks: `kite/internal/offer/service.go`,
  `kite/cmd/kite-api/apis/vm_offer.go`,
  `kite/cmd/kite-controller/apps/vm-offer-cleanup.go`,
  `src/screens/AdminVmOfferPanel.tsx` and `src/screens/UserVmOffersSection.tsx`.

## Final generation prompt

```text
Use case: scientific-educational
Asset type: explanatory illustration for the VM pre-allocation chapter of Kite's professional Korean documentation.
Primary request: show the exact conceptual sequence "administrator assigns a VM offer → user accepts the offer → an actual virtual machine is provisioned". The offer alone is a specification, not an already-running VM.
Style/medium: refined editorial 3D diagram, tactile matte porcelain and folded paper, precise minimal product documentation aesthetic. Light white background #ffffff with extremely pale sky blue #eaf3fc, soft lavender, subtle graphite #151719 details, tiny green ready indicator only in the final stage. Soft realistic contact shadows, crisp detailed shapes, restrained saturation.
Composition: wide landscape 1536 by 1024, three equally weighted clearly separated stages arranged left to right with thin gray directional arrows and abundant negative space. Keep every illustration and its number fully inside a generous 100px safe area. Stage 01: a thin upright specification card held above a small administrator desk/plinth; three embossed short horizontal specification rows, small CPU chip, memory and disk symbols, NO server. Stage 02: the same specification card beside a minimal user form represented by exactly four blank rows and a small check button, a small abstract person/avatar token; NO server yet. Stage 03: one compact isolated virtual machine represented by a single elegant pale blue desktop server cube with a tiny green light and a small terminal panel. Use a coherent physical scale. Number each scene with only the readable labels "01", "02", "03" at the bottom, aligned on a common baseline.
Constraints: image is a conceptual illustration, not a screenshot or imitation of actual Kite UI. Do not include any words, logos, branding, headings, captions or UI text beyond "01", "02", "03". No people photographs, no dark backgrounds, no complex racks, no dashboards, no cloud network clutter, no resource reservation claim, no extra stages. Avoid decorative gradients behind the diagram; background must remain white. The relationship and order must be immediately clear.
```
