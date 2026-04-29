# `web/` — hekate web client (M3, off the critical path)

SvelteKit 2 + Svelte 5, `@sveltejs/adapter-node`. The web client is **not** the v0.1 reference client — the Go CLI is. This client is M3 in the roadmap and only pursued after M1 is shipping reads from external PDSes.

## Setup

```bash
cd web
pnpm install
pnpm dev          # http://localhost:5173
```

Requires Node 22+ and pnpm 9+. Both are installed via `corepack enable && corepack prepare pnpm@9.15.0 --activate` if you have Node already.

## Verify

```bash
pnpm check        # svelte-check + tsc
pnpm test         # vitest
pnpm build        # production build into ./build
pnpm preview      # serve the production build locally
```

## Configuration

- `PUBLIC_HEKATE_API_URL` — the hekate-server origin. Defaults to `http://localhost:8080` in dev.

## Why SvelteKit, not Next.js?

Per `docs/ROADMAP.md` and the plan-of-record, the workspace already has a Next.js project (`salt-mines/frontend/`). SvelteKit was chosen for hekate to keep web frameworks diverse across projects and to lean on Svelte 5's smaller-bundle / lower-onboarding-friction profile, which matches the OSS adoption pillar.

If you'd prefer a different framework, open an ADR proposing the swap; this is not a hill we will die on.
