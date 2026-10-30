# Containerized web/API architecture

- Created with the built-in `image_gen` tool on 2026-10-30.
- Delivered asset: `public/images/docs/web-api-architecture.webp`,
  1512 × 1040, 81,574 bytes.
- Selected original: `/Users/hhs2003/.codex/generated_images/01a18957-3b49-7471-9145-93c8fdbb7906/exec-6e3287e7-6624-4ec9-bcc6-ad8d1be9ece1.png`.
- Generated base: `/Users/hhs2003/.codex/generated_images/01a18957-3b49-7471-9145-93c8fdbb7906/exec-16515431-0a13-4c10-9333-8bc38d52e96f.png`.
- Quality-90 WebP delivery; originals preserved. Korean alt text and a live
  caption accompany the figure, which links to its full-size version.
- Shows browser → Ingress → separate `/` and `/api` routes, Next.js session/SSR
  reads, Go API writes through the Kubernetes API, CRD watching, resource apply
  and observed-status updates. This is a conceptual web/VM control diagram;
  internal Services and the separately documented SSH transport are omitted.
- Three application containers run in the `kite` namespace. The Kubernetes
  API server is a separate cluster control-plane component, outside that
  namespace grouping. Container glyphs do not imply a Docker-engine dependency.
- Reviewed against `build/kite/frontend.yaml`, `api.yaml`, `controller.yaml`,
  `platform-ingress.yaml`, `kite-frontend/next.config.ts`, server API helpers
  and controller reconciliation code. The Go API does not invoke the
  controller through gRPC and does not directly reconcile KubeVirt resources.
- Visually checked route labels, port numbers (3000/8080), arrow direction,
  namespace boundary and separation of Kubernetes API/CRDs from Kite containers.

## Generation prompt

```text
Use case: infographic-diagram
Asset type: production cloud documentation architecture figure for Kite, a Kubernetes VM platform. Landscape approximately 1600x1100 with readable text. White background.
Primary request: show the actual containerized WEB/API and VM reconciliation architecture, with clear directional client request arrows. This is a precise engineering documentation figure, NOT a decorative illustration.

Composition: one large pale blue rounded "Kubernetes cluster" boundary covering most of the drawing, with a small accurate blue Kubernetes helm at its top-left. One "Browser client" icon is OUTSIDE the boundary on the far left. Inside, group THREE application nodes in a gently outlined sub-boundary called "kite namespace". Each application node has a small plain "Container" label and a modest shipping-container line glyph, indicating a container image workload, NOT a Docker daemon or nested container engine.
The three application nodes are:
1. "kite-frontend" with subtitle "Next.js :3000"
2. "kite-api" with subtitle "Go API :8080"
3. "kite-controller" with subtitle "Reconcile"
Also inside the Kubernetes cluster but OUTSIDE the namespace sub-boundary, show a "Kubernetes API server" node containing a small "Kite CRDs" document stack, and a separate "KubeVirt" runtime box containing two simple VM icons labelled "VM".

Arrange the diagram in two comfortably spaced rows:
TOP row is client access: Browser client → an "Ingress" router at the cluster edge, branching into kite-frontend and kite-api. Put kite-frontend above kite-api with both route labels readable. Label the Ingress-to-frontend branch "/" and the Ingress-to-API branch "/api". A shared connector from Browser client to Ingress is labelled "HTTPS". Both branches start at the Ingress; DO NOT route /api through Next.js. Show a subtle dashed internal frontend → Go API line labelled "Session / SSR reads", distinct from browser API routing.
MIDDLE/RIGHT region: Go API points to Kubernetes API server / Kite CRDs, labelled "Write spec". Desired state is stored through the Kubernetes API server, NOT delivered to the controller by RPC.
BOTTOM row: kite-controller watches Kite CRDs via a dashed arrow from Kubernetes API server to controller, labelled "Watch CRDs". The controller points toward KubeVirt using a clear arrow labelled "Apply resources". A return arrow from KubeVirt to controller is labelled "Observe state". A separate arrow from controller back to Kubernetes API server is labelled "Update status". Keep directions unambiguous; use elbow connectors on separate tracks, never through node bodies or text.
Use the space well: readable separated paths with no spaghetti. All nodes and connectors have ample 24px breathing room. The browser client stays outside the cluster. The Ingress stays outside the kite namespace subgroup but inside the cluster. Kubernetes API server is a cluster control-plane component, not a Kite application container. KubeVirt is a runtime, not a direct target of Browser or Go API. No arrow directly from Go API to Controller or KubeVirt. No gRPC, no database node, no nginx, no Vite. SSH Gateway is documented in a separate figure; this image covers web requests and VM control only.

Style: contemporary cloud architecture docs; precise flat vector-like icons, calm near-white surfaces, powder blue group boundaries, slate-gray keylines, dark charcoal headings, Kubernetes-blue primary HTTP connectors, thinner gray control-loop connectors. Rounded rectangles 12–16px corners, clean sans-serif labels with large readable type. NO 3D render, chrome, physical servers, clouds, glass, neon, gradients, heavy shadows, stock imagery or watermark. Do not use the Docker company logo or imply that Kubernetes requires the Docker engine.
Only render the exact English labels specified here; no decorative title or explanatory paragraphs. All port numbers and route strings must be exact.
```

## Final edit prompt

```text
Edit this technical architecture diagram with ONE precise correction. Preserve every existing label, icon, arrow, node, color, position and layout.
The inner rounded boundary labelled "kite namespace" incorrectly includes the "Kubernetes API server" node on the right. Narrow ONLY this namespace boundary so it surrounds ONLY the vertical column of three application containers: kite-frontend, kite-api, kite-controller. Its right boundary should be just to the right of those container boxes, approximately x=1020 in this 1513px-wide image, instead of at the far right. Keep its top/left/bottom edges where they are. Kubernetes API server and its Kite CRDs stack must clearly sit OUTSIDE the "kite namespace" boundary but INSIDE the larger "Kubernetes cluster" boundary. KubeVirt likewise stays outside kite namespace and inside Kubernetes cluster. Let the existing Write spec, Watch CRDs and Update status connectors cross the corrected inner namespace boundary naturally.
Do not move or redraw the nodes. Do not change typography, strings, port numbers, label spelling, arrow direction or anything else. In particular keep the exact separate Ingress "/" and "/api" branches, dashed Session / SSR reads, Watch CRDs, Apply resources, Observe state and Update status. No new text or decoration.
```
