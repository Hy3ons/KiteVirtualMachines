FROM node:22-alpine AS dependencies
WORKDIR /app
COPY package*.json ./
RUN npm ci --no-audit --no-fund

FROM node:22-alpine AS build
WORKDIR /app
COPY --from=dependencies /app/node_modules ./node_modules
COPY . .

ARG NEXT_PUBLIC_KITE_MODE=production
ARG NEXT_PUBLIC_API_BASE_URL=/api/v1
ARG NEXT_PUBLIC_USE_MOCK=false
ENV NEXT_PUBLIC_KITE_MODE=${NEXT_PUBLIC_KITE_MODE} \
    NEXT_PUBLIC_API_BASE_URL=${NEXT_PUBLIC_API_BASE_URL} \
    NEXT_PUBLIC_USE_MOCK=${NEXT_PUBLIC_USE_MOCK} \
    NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:22-alpine AS runtime
WORKDIR /app
LABEL org.opencontainers.image.source="https://github.com/Hy3ons/KiteVirtualMachines" \
      org.opencontainers.image.description="Kite Next.js frontend" \
      org.opencontainers.image.licenses="MIT"
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0 \
    KITE_API_INTERNAL_URL=http://kite-api:8080
COPY --from=build --chown=node:node /app/.next/standalone ./
COPY --from=build --chown=node:node /app/.next/static ./.next/static
COPY --from=build --chown=node:node /app/public ./public
USER node
EXPOSE 3000
CMD ["node", "server.js"]
