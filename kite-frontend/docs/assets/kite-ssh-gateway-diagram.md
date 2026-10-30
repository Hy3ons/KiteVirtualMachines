# SSH Gateway routing diagram

- Generated with the built-in `image_gen` tool on 2026-10-30.
- Asset: `public/images/docs/ssh-gateway-routing.webp`, 1983 × 793.
- Original: `/Users/hhs2003/.codex/generated_images/01a18957-3b49-7471-9145-93c8fdbb7906/exec-3fc5c8eb-c9df-43bd-8c5c-ce5b0110c8ca.png`.
- Delivered as quality-90 WebP for readable labels; original preserved.
- Illustrates client → router → LoadBalancer Service → Kite Gateway → VM.
  The Kubernetes boundary contains the Service, Gateway and VM. The router
  remains external. The external 22 → Service 12311 mapping is an operator
  configuration example, not a default or a gateway-process listen address.
- The Gateway matches the SSH username to `spec.sshId`, verifies the initial
  password, then authenticates to the VM using a managed SSH key. The client
  password is not forwarded to the VM. Runtime behavior was checked against
  `kite/internal/gateway/routes.go` and `kite/internal/gateway/server.go`.
- Image inspected for labels, flow direction, cluster boundary and separation
  of password/key authentication. Korean alt text and caption accompany it.
  The document links to the full-size asset rather than adding a modal.

## Final generation prompt

```text
Use case: infographic-diagram
Asset type: professional cloud developer-documentation illustration for the SSH Gateway chapter of Kite, a Kubernetes VM platform. Wide landscape 1536x864, clean white background, generous 64px outer safe margins.
Primary request: a precise technical concept diagram explaining how an external user connects to a private VM through a router and Kite SSH Gateway, with separate user-password authentication and gateway-to-VM key authentication. It must look like an official cloud architecture documentation diagram, not marketing art.

Composition: one quiet left-to-right connection flow, four aligned nodes with plenty of room for labels: "SSH client" (small terminal icon), "Router" (simple network appliance icon), "Kite SSH Gateway" (simple proxy server icon), "Virtual machine" (simple VM/server icon).
The SSH client and Router are OUTSIDE the Kubernetes boundary. A large pale ice-blue rounded rectangular boundary contains only the Gateway and Virtual machine nodes, labelled "Kubernetes cluster", with a small blue Kubernetes helm symbol at the upper-left. A small plain service tag sits on the connector just INSIDE the Kubernetes boundary before the Gateway and reads "LoadBalancer Service" with ":12311" below it. This is the Service's port, NOT the Gateway process listen port.
From SSH client to Router draw a clear bidirectional SSH connector labelled "Public :22". From Router through the service tag to Gateway draw a clear connector. Above the router/service path add the discreet label "Port forwarding example". From Gateway to Virtual machine draw a bidirectional connector labelled "Kite-managed SSH key" and ":22". Keep arrowheads and paths clean, never crossing labels or boxes. Gateway-to-VM connection remains entirely INSIDE the cluster.

Inside or directly below the Gateway node, put two short well-spaced lines:
"Match username to spec.sshId"
"Verify initial password"
Under the client icon put "SSH ID + initial password".
At the bottom of the illustration put one understated note:
"User password is not forwarded to the VM."

The diagram should clearly communicate that the client password is verified at the gateway, while a different managed SSH key authenticates the gateway to the VM; do not draw a password arrow directly to the VM. Public 22 and Service 12311 are an EXAMPLE configured by an operator, not fixed port requirements. Do not suggest that the host sshd configuration is modified, or show individual public VM ports.
Style: crisp flat vector-like documentation drawing, accurate network diagram grammar, restrained slate-gray line icons, readable near-black English sans-serif labels, thin slate-blue connectors, near-white node bodies, subtle neutral keylines, pale blue cluster area, modest 12–16px corners. Match contemporary Google Cloud/Azure documentation quality. Clear hierarchy, no tiny microtext.
Text: only the exact English labels specified above; spell "spec.sshId", "LoadBalancer" and "Kite" correctly. No extra title, no legend table, no slogans.
Avoid: 3D render, clouds, porcelain, glass, neon, gradients, gears, decorative objects, complex topology, heavy shadows, watermarks, people, product screenshots.
```
