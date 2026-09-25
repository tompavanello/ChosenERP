import type { NextConfig } from "next";

// Destino do proxy local de /api. E configuracao de SERVIDOR. Em producao o
// tunel Cloudflare / nginx mandam /api/* direto para o Go antes de o Next ver o
// pedido; o rewrite existe so para o desenvolvimento local (marketing em 33000
// e API em 38080), onde sem ele /api/v1/... daria 404 na porta do Next.
const API_ORIGIN = process.env.API_ORIGIN ?? "http://localhost:38080";

// Cabeçalhos aplicados a todas as respostas do site institucional. O CSP e
// deliberadamente permissivo com inline (o Next injeta script/style inline sem
// nonce); o ganho real aqui e negar enquadramento (anti-clickjacking), travar o
// tipo de conteudo e restringir conexoes a propria origem.
const securityHeaders = [
  { key: "X-Content-Type-Options", value: "nosniff" },
  { key: "X-Frame-Options", value: "DENY" },
  { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
  { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=()" },
  {
    key: "Content-Security-Policy",
    value: [
      "default-src 'self'",
      "script-src 'self' 'unsafe-inline' 'unsafe-eval'",
      "style-src 'self' 'unsafe-inline'",
      "img-src 'self' data: blob:",
      "font-src 'self' data:",
      "connect-src 'self'",
      "frame-ancestors 'none'",
      "base-uri 'self'",
      "form-action 'self'",
    ].join("; "),
  },
];

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Gera .next/standalone/server.js: a imagem Docker copia so o runtime
  // necessario em vez de reinstalar tudo.
  output: "standalone",
  // O `dev` e o `build` compartilham a mesma pasta e se corrompem quando rodam
  // ao mesmo tempo. Para validar sem derrubar o dev: NEXT_DIST_DIR=.next-verify
  distDir: process.env.NEXT_DIST_DIR ?? ".next",
  poweredByHeader: false,
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${API_ORIGIN}/api/:path*` }];
  },
  async headers() {
    return [{ source: "/:path*", headers: securityHeaders }];
  },
};

export default nextConfig;
