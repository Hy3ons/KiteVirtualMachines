# Declarative architecture illustration

- Created with the built-in `image_gen` tool on 2026-10-30.
- Delivered asset: `public/images/kite-declarative-architecture.webp`,
  1672 × 941, 40,876 bytes.
- Original: `/Users/hhs2003/.codex/generated_images/01a18957-3b49-7471-9145-93c8fdbb7906/exec-58199899-59e1-48eb-9afe-673494f7dce0.png`.
- Encoded as WebP at quality 84; original preserved without modification.
- Used in the landing architecture figure. Accessible headings and captions are
  live HTML, separate from the generated conceptual artwork. It is not a literal
  cluster topology, a product screenshot, or a guarantee of host-failure recovery.
- Copy follows the current controller: desired VM configuration in CRD `spec`,
  resource reconciliation, and observed runtime state written to CRD `status`.
  Relevant sources: `kite/cmd/kite-controller/apps/machine-reconcile.go` and
  `kite/cmd/kite-controller/apps/kubevirt-status-reconcile.go`.

## Final generation prompt

```text
Use case: stylized-concept
Asset type: production landing-page architecture illustration for Kite, an open-source Kubernetes and KubeVirt virtual-machine platform. Wide landscape 1536x864 composition.
Primary request: a precise, sophisticated editorial 3D systems illustration that visually expresses a declarative control loop: one desired-state document on the left, a reconciliation mechanism in the center, and three isolated virtual-machine modules on the right. A thin continuous rounded rectangular return path connects the runtime modules back toward the central mechanism, expressing observed state feedback. This is a conceptual architecture illustration, not a literal hardware or network topology.
Scene: one unified pale ice-blue studio plane, near-white backdrop, extremely subtle fine technical grid fading toward the edges, generous negative space and 10 percent breathing room around every object. All three stages sit on a shared low rectangular softly rounded base, coherent quiet horizontal composition viewed from an elevated orthographic angle.
Subjects: left a standing thin matte-white specification plate with a few precise recessed short neutral lines representing fields; center an elegant concentric circular precision assembly inset into a white low block, expressing ongoing reconciliation without gears; right three individual white translucent-edged cuboids with quiet powder-blue interiors, arranged neatly as separate VM instances. Connect groups with very fine machined channels in the base. Every object should feel clean and credible, no cartoon elements.
Style: premium technical editorial 3D still life, Apple-like material restraint, matte porcelain and very subtle frosted glass, physically plausible soft shadows, crisp geometry, careful small details, abundant space. Palette white, pale sky blue #eaf3fc, slate gray #5c626c, barely visible lavender; dark charcoal details used sparingly. Low saturation, high legibility on a white page. Soft diffuse top-left lighting, smooth clean studio finish.
Constraints: absolutely no text, numbers, letters, logos, watermarks, people, screenshots, UI cards, glossy chrome, neon glows, purple gradients, decorative clouds, padlocks, or brand symbols. No title embedded in the image. The live website will supply all headings, diagram labels and explanations in accessible HTML.
```
