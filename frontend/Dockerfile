# ---- build stage: compile TS + bundle with Vite ----
FROM node:24-alpine AS build
WORKDIR /src

# Lockfile-first for layer caching (same trick as go.mod in the API image).
COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# ---- final stage: nginx serves the static files and proxies /api ----
FROM nginx:1.29-alpine
COPY --from=build /src/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
