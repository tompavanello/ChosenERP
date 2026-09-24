import type { NextConfig } from "next";

// Destino do proxy local de /api. E configuracao de SERVIDOR, nao do bundle: o
// browser sempre chama caminho relativo (/api/v1/...) e resolve contra a origem
// que ele abriu.
//
// Em producao o rewrite quase nunca e exercido - o tunel Cloudflare manda
// /api/* direto para o Go antes de o Next ver o pedido. Ele existe para o caso
// local, onde webadmin (33000) e API (38080) sao portas diferentes: sem ele,
// /api/v1/... bateria na porta do Next e daria 404.
//
// Antes existia um `env: { API_URL }` aqui, gravado no bundle em build time.
// Aquilo amarrava o build a um endereco: o mesmo bundle nao servia local e
// remoto, e abrir o localhost mandava o browser para o dominio publico. Para
// apontar a API para outro lugar agora, mude API_ORIGIN - o bundle nao muda.
const API_ORIGIN = process.env.API_ORIGIN ?? "http://localhost:38080";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Gera .next/standalone/server.js: a imagem Docker do webadmin copia so o
  // runtime necessario (node_modules enxuto) em vez de reinstalar tudo.
  output: "standalone",
  // O `dev` e o `build` compartilham a mesma pasta e se corrompem quando
  // executados ao mesmo tempo. Para validar sem derrubar o dev server, use
  // uma pasta alternativa:  NEXT_DIST_DIR=.next-verify npm run build
  distDir: process.env.NEXT_DIST_DIR ?? ".next",
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${API_ORIGIN}/api/:path*` }];
  },
};

export default nextConfig;
