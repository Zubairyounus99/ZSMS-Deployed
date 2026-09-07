# ==============================================================================
# ZSMS Web Dashboard — Multi-Stage Dockerfile
# ==============================================================================

# Build Stage
FROM node:20-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache libc6-compat

COPY package.json package-lock.json* ./
RUN npm install

COPY . .

# Run Nuxt production build
RUN npm run build

# Runtime Stage
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
ENV PORT=3000
ENV HOST=0.0.0.0

# Security: Run as unprivileged node user
USER node

# Copy standalone Nitro output from builder
COPY --from=builder --chown=node:node /app/.output ./.output

EXPOSE 3000

HEALTHCHECK --interval=20s --timeout=3s --start-period=10s --retries=3 \
  CMD node -e "require('http').get('http://127.0.0.1:3000/', (res) => process.exit(res.statusCode >= 200 && res.statusCode < 400 ? 0 : 1)).on('error', () => process.exit(1))"

CMD ["node", ".output/server/index.mjs"]
