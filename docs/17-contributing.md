# Contributing

Thanks for your interest in Tennon. This document covers how to get set up, the conventions
we follow, and how to propose changes.

## Repository layout

- `cmd/tennon` — Go: the **server** binary (controller (Kubebuilder), API, MCP proxy, and the
  embedded Vite/React web UI). One self-contained binary / container image.
- `cmd/tctl` — Go: the **client** CLI; also ships an embedded **MCP server**
  (`tctl mcp serve`) as the local stdio bridge for clients that launch a command.
- `web/` — Vite/React UI sources, embedded into the server binary at build time.
- `charts/tennon` — the Helm chart.
- `docs/` — the user guide; `docs/arch/` — the design/research notes.

## Development setup

- Go 1.24+ for the server and CLI.
- Node 20+ for the web UI build.
- For the Kubernetes backend: a local cluster (kind/minikube) and `kubectl`.
- For local iteration without a cluster: run the controller with `--store=sqlite`.

Build, run, and test commands live in the `Makefile` / package scripts; run `make help`.

## Conventions

- **Keep the planes separate.** Authorization, secret rendering, and routing live in the
  `tennon` server. The embedded UI and the CLI's MCP bridge are thin and hold no secrets.
- **CRDs are the source of truth.** UI and CLI actions must go through the API and result in
  CRD objects; do not add side channels that bypass reconciliation.
- **No YAML in the UI.** The interface is click-driven; it translates to resources.
- **Short-lived by default.** Prefer tokenless trust; never introduce long-lived shared
  secrets.
- **Secrets never leak.** No credential value in responses, status, logs, or UI.

## Proposing changes

1. Open an issue describing the problem and approach (link any relevant design note in
   `docs/`).
2. For non-trivial design, add or update a design note in `docs/` and reference it.
3. Keep PRs focused; include tests for behavior changes.
4. Update the relevant page in `docs/` when you change user-visible behavior.

## Commit & PR hygiene

- Clear, descriptive commits; reference issues.
- CI must pass (build, lint, type-check, tests) before review.

## Security disclosures

Please report vulnerabilities privately (see the repository's security policy) rather than
opening a public issue. We aim to acknowledge promptly and coordinate a fix and disclosure.

## License

By contributing you agree your contributions are licensed under the Apache License 2.0.
