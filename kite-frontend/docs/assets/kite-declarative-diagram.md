# Kubernetes and CRD architecture diagram

- Generated with the built-in `image_gen` tool on 2026-10-30.
- Delivered asset: `public/images/kite-kubernetes-crd-diagram.webp`, 1672 × 941.
- Original: `/Users/hhs2003/.codex/generated_images/01a18957-3b49-7471-9145-93c8fdbb7906/exec-e1ff595b-0d47-42ab-ae6b-36e26cd0af2e.png`.
- Encoded as WebP at quality 90 to preserve technical labels; original retained.
- Replaces the previous abstract 3D artwork in the landing architecture section.
- Illustrates CRD schema → custom resource, spec watch → reconcile → KubeVirt,
  runtime observation → status update. This is a conceptual control loop,
  not a literal Pod topology or a claim of direct VM execution by Kite.
- Kubernetes symbol identifies the technical platform. English labels and
  directional arrows were visually checked in the generated image; accessible
  Korean alt text and live captions explain the same relationship on the page.
- Field examples follow `kite/api/v1/types.go`. Reconciliation and status updates
  follow `kite/cmd/kite-controller/apps/machine-reconcile.go` and
  `kite/cmd/kite-controller/apps/kubevirt-status-reconcile.go`.

## Final generation prompt

```text
Use case: infographic-diagram
Asset type: production website technical architecture diagram for Kite, a Kubernetes / KubeVirt VM platform. Wide landscape 1536x864.
Primary request: a clear, accurate, flat technical concept diagram showing Kubernetes Custom Resource Definitions and the declarative reconciliation loop. The diagram must unmistakably look like Kubernetes architecture documentation, NOT an abstract product rendering.

Composition: generous 64px safe margins, white background. A large very pale blue rounded rectangular boundary, with an accurate blue Kubernetes seven-spoke helm logo at its upper-left followed by the label "Kubernetes cluster". Inside the boundary, a calm left-to-right diagram with three primary rectangular nodes of equal visual weight, aligned horizontally. Above or inside the left region identify the Kubernetes API server and its CRD-backed custom resource. Middle region the Kite controller. Right region KubeVirt runtime. Show three tiny VM instances inside the right node.

Exact labels, in clean highly legible sans serif typography:
Boundary: "Kubernetes cluster"
Left node top small label: "Kubernetes API"
Left node title: "KiteVirtualMachine"
Left node secondary label: "Custom resource"
Within the left node, two clearly separated small rows labelled "spec" and "status".
Under spec show a concise monospace example on three lines:
"cpu: 2"
"memory: 4Gi"
"powerState: On"
Within status show a short monospace example "phase: Running".
A small thin outlined schema document attached immediately above or beside the left node is labelled "CRD" and "Resource schema", with a short association line to the custom resource. This shows that the CRD defines the schema, whereas the custom resource holds the actual desired and observed state.
Middle node title: "Kite controller"
Middle node small subtitle: "Reconcile"
Right node title: "KubeVirt"
Its three VM instances each have the simple label "VM".

Connections with precise arrowheads: the custom resource's spec row points right into Kite controller, label "Watch spec". Kite controller points right into KubeVirt, label "Apply resources". A lower return arrow goes from KubeVirt back to Kite controller, label "Observe runtime". A separate parallel return arrow from the controller points left into the custom resource's status row, label "Update status". Route the forward connections above the return connections with ample separation, never crossing text or boxes. This is a feedback loop, NOT a one-way pipeline. Keep all arrows inside the Kubernetes boundary. No user, browser, API request or network ingress pipeline.

Style: professional developer documentation figure with restrained editorial polish, crisp vector-like geometry, 1px slate-blue connectors, blue Kubernetes symbol, near-black headings, slate body text, pale blue node headers, white node bodies, subtle neutral keylines, moderate 16px corners. No glossy materials. Large readable text. Exact English words only, no extra decorative labels or lorem ipsum.
Avoid: 3D objects, porcelain, machines, gears, glowing rings, glass boxes, physical pipes, clouds, gradients, heavy shadows, photographs, neon, cartoon mascots, decorative pseudo-code, watermarks. Do not add any other text.
```
