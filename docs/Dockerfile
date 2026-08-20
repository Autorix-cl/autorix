# Stage 1: Build VitePress static documentation site
FROM node:22-alpine AS builder

WORKDIR /app

COPY docs/package*.json ./
RUN npm ci

COPY docs/ ./
RUN npm run build

# Stage 2: Serve static production assets with Nginx
FROM nginx:alpine-slim

COPY docs/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/.vitepress/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:80/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
