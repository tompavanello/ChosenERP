import type { MetadataRoute } from "next";

/**
 * robots.txt do webadmin.
 *
 * /member/{token} e a carteirinha publica: a URL *e* a credencial (o token vem
 * do QR impresso no cartao), entao ela nunca deve aparecer em resultado de
 * busca. O endpoint tambem responde X-Robots-Tag: noindex (internal/httpapi/
 * ratelimit.go), mas o robots.txt evita que o crawler chegue la pela primeira
 * vez - o header sozinho so vale depois que a URL ja foi visitada.
 */
export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: "/",
      disallow: ["/member/"],
    },
  };
}
