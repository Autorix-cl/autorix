# 1. Base stage
FROM node:20-alpine AS base
WORKDIR /app

# 2. Dependencies stage
FROM base AS deps
COPY package.json package-lock.json* ./
RUN npm ci --prefer-offline || npm install

# 3. Build stage
FROM base AS builder
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# 4. Production runner stage
FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Non-root user for Zero Trust security
RUN addgroup -S autorix && adduser -S autorix -G autorix

COPY --from=builder /app/public ./public
COPY --from=builder --chown=autorix:autorix /app/.next/standalone ./
COPY --from=builder --chown=autorix:autorix /app/.next/static ./.next/static

USER autorix:autorix
EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
