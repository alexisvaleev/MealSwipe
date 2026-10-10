# ---------- build ----------
# node:22-slim (glibc) — the musl resolver in alpine fails hard on hosts whose
# DNS refuses AAAA queries (EAI_AGAIN bursts during `npm ci`).
FROM node:25-slim AS build
WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
ARG VITE_APP_NAME=MealSwipe
ENV VITE_APP_NAME=$VITE_APP_NAME
RUN npm run build

# ---------- run ----------
FROM nginxinc/nginx-unprivileged:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget -qO- http://127.0.0.1:8080/ >/dev/null 2>&1 || exit 1
