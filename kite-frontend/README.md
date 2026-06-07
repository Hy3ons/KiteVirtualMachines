# Kite Frontend

Kite Frontend is a React + TypeScript + Vite application served by the
`kite-frontend` nginx image.

## Environment

This frontend does not load `.env.*` files. `vite.config.ts` points Vite's
`envDir` at an unused directory, so configuration must come from the current
shell session or Docker build args.

Local development defaults:

```sh
npm run dev
```

Production build defaults:

```sh
npm run build:prod
```

Override values in the shell session when needed:

```sh
VITE_API_BASE_URL=http://localhost:8080/api/v1 VITE_USE_MOCK=false npm run build:stage
```

Docker builds accept the same values as build args:

```sh
docker build \
  --build-arg VITE_BUILD_MODE=production \
  --build-arg VITE_API_BASE_URL=/api/v1 \
  --build-arg VITE_USE_MOCK=false \
  -t anacnu.com/kite-frontend:latest .
```
