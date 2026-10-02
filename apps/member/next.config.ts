import type { NextConfig } from "next";

// Mesmo padrao do webadmin: o browser chama caminho relativo (/api/v1/...) e o
// rewrite resolve para a API em desenvolvimento. Em producao o nginx ja manda
// /api/* direto para o Go antes de o Next ver o pedido.
const API_ORIGIN = process.env.API_ORIGIN ?? "http://localhost:38080";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: "standalone",
  distDir: process.env.NEXT_DIST_DIR ?? ".next",
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${API_ORIGIN}/api/:path*` }];
  },
};

export default nextConfig;
