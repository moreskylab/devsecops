# ============================================================================
# Stage 1: Builder — install deps and build with Node
# ============================================================================
FROM node:22-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm ci

COPY . .
RUN npm run build

# ============================================================================
# Stage 2: Runtime — nginx-unprivileged (non-root, UID 101)
# ============================================================================
FROM nginxinc/nginx-unprivileged:1.27-alpine

# Copy built assets
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx config (use the secure variant by default)
COPY nginx-secure.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080

USER 101

CMD ["nginx", "-g", "daemon off;"]
