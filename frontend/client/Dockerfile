# ---- Build the static site ----
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
# Optional: public site URL for canonical/share tags (storefront). Example: --build-arg VITE_SITE_URL=https://www.example.com.bd
ARG VITE_SITE_URL=
ENV VITE_SITE_URL=$VITE_SITE_URL
# VITE_API_URL stays empty: the browser calls /api on the same address and nginx forwards it to the backend.
RUN npm run build

# ---- Serve it ----
FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s CMD wget -qO- http://127.0.0.1/ >/dev/null || exit 1
