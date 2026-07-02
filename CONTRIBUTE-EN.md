# Kite Contribution Guide

Thank you for your interest in Kite. Contributions are welcome, from small
UI/UX improvements and documentation fixes to tests, deployment scripts, API,
controller, gateway, and frontend changes. We try to accept most contributions
as long as they fit the project's design philosophy.

The Korean guide, [CONTRIBUTE-KO.md](CONTRIBUTE-KO.md), is the source of truth.
If this English guide differs in meaning, follow the Korean document.

## Open an issue first

Every contribution starts with an issue.

Kite is under active development. There may be unmerged work on `stage`, or
ongoing validation work on `dev`. Before opening a PR, please open an issue and
share:

- the problem or improvement you want to work on,
- whether you want to implement it yourself,
- the expected impact area,
- which test environments you can run: k3s, minikube, generic k8s, or none.

The maintainer will review the current development state and answer with:

- whether the work is ready to start,
- whether the PR should target `stage` or `main`,
- the design direction to follow,
- the required test scope,
- any E2E tests the maintainer will run instead.

## Branch policy

- `main`: production publishing branch. Only fully integrated and E2E-verified
  changes land here.
- `stage`: integration branch before main. A checkout from stage should build
  and install without immediate breakage.
- `dev`: fast integration and remote validation branch. Experiments and messy
  intermediate work are allowed here.
- task branches: use `feature/*`, `fix/*`, `docs/*`, `chore/*`, or `refactor/*`.

See [docs/branch-policy.md](docs/branch-policy.md) for the full branch policy.

## Design philosophy

Kite is built around Kubernetes CRDs and controller reconciliation.

- The API server writes desired state into CRDs such as `KiteUser` and
  `KiteVirtualMachine`.
- The controller watches CRDs and reconciles real Kubernetes, KubeVirt, and CDI
  resources.
- Observed runtime state is written back to CRD `status`.
- The old protobuf/gRPC API-to-controller command path is not the current design.
- Reuse existing helpers, renderers, stores, shell prompts, and test harnesses
  before adding new structure.
- Avoid broad global state.
- Avoid large amounts of duplicated code.
- Frontend mock data is useful for development, but it is not production proof.

If a proposed change does not fit this philosophy, we will use the issue to
reshape it instead of rejecting it silently.

## What to include in a PR

Please include:

```text
Purpose
- What changed and why.

Changes
- A short summary of the main changes.

Impact
- Affected areas: API, CRD, controller, frontend, gateway, deploy/test scripts.

Tests
- Commands you ran and their results.
- Tests you could not run, with the reason and required environment.
```

Commit messages should follow [docs/commit-convention.md](docs/commit-convention.md).

## Test Contract

You do not need to own every cluster environment to contribute. The issue or PR
must make the test responsibility clear.

When needed, the maintainer will provide a Test Contract:

```text
Test Contract
- Base branch:
- Change type:
- Required local checks:
- Required cluster E2E:
- Optional checks:
- Maintainer-run checks:
- Tests the contributor could not run:
```

Run the tests you can run, and clearly list anything you could not run. If
k3s, minikube, or generic k8s E2E is required as a merge gate and you cannot run
it, the maintainer will run it before merging.

Use [test/Test-Specification.md](test/Test-Specification.md) for the minimum
test expectations by change type. Release-like changes follow the E2E gate in
[test/Readme.md](test/Readme.md).

## Examples of welcome contributions

- UI/UX improvements for login, VM creation, and admin screens
- Documentation improvements and typo fixes
- Additional test cases
- Better shell script prompts and failure messages
- CRD schema and controller reconcile stability
- Gateway SSH authentication, fallback, and banner improvements
- More reliable k3s, minikube, and generic k8s install/uninstall flows

Small contributions are welcome. Start with an issue, and we will help choose
the right branch and test path for the current project state.
