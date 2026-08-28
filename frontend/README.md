# ArmadaCMS admin frontend

This directory contains the React-Admin SPA used for the ArmadaCMS admin interface.

For full-project setup, Docker-based local development, environment variables, and backend integration, use the repository root guide: [`../README.md`](../README.md).

## Use this README when

Use this document only if you want to run or validate the frontend directly from `frontend/` against an already running API.

## Install dependencies

```sh
pnpm install --frozen-lockfile
```

## Run locally

```sh
pnpm run dev
```

## Validate changes

```sh
pnpm run build
pnpm run lint:check
pnpm run type-check
pnpm run format:check
pnpm run test
```
