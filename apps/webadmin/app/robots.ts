import type { MetadataRoute } from "next";

/**
 * robots.txt do webadmin.
 *
 * /member/{token} é a carteirinha pública: a URL *é* a credencial (o token vem
 * do QR impresso no cartão), então ela nunca deve aparecer em resultado de
 * busca. O endpoint também responde X-Robots-Tag: noindex (internal/httpapi/
 * ratelimit.go), mas o robots.txt evita que o crawler chegue lá pela primeira
 * vez — o header sozinho só vale depois que a URL já foi visitada.
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
