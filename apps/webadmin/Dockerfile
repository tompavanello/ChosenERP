# syntax=docker/dockerfile:1

# Imagem de produção do Webadmin (Next.js 15 standalone).
# Build a partir da pasta do app:
#   docker build -t chosenerp/webadmin:local apps/webadmin

# ---- deps: instala dependências com o lockfile ----
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

# ---- builder: compila o Next (output: standalone) ----
FROM node:22-alpine AS builder
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
# API_ORIGIN é lido pelo `rewrites()` do next.config.ts em tempo de BUILD. No
# Compose é o nome do serviço da API na rede interna ("http://api:8080").
ARG API_ORIGIN=http://api:8080
ENV API_ORIGIN=$API_ORIGIN
# Dominio base do white-label (subdominio por igreja). Vai para o bundle do
# browser (NEXT_PUBLIC_*), por isso e ARG de build.
ARG NEXT_PUBLIC_BASE_DOMAIN=erpchosen.com.br
ENV NEXT_PUBLIC_BASE_DOMAIN=$NEXT_PUBLIC_BASE_DOMAIN
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

# ---- runner: imagem mínima, usuário não-root, só o standalone ----
FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=33000
ENV HOSTNAME=0.0.0.0
RUN addgroup -S -g 1001 nodejs && adduser -S -u 1001 -G nodejs nextjs
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
USER nextjs
EXPOSE 33000
CMD ["node", "server.js"]
