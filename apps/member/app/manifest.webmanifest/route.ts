import { headers } from "next/headers";

export const dynamic = "force-dynamic";

// Manifest PWA dinamico por igreja: resolve o tenant pelo subdominio (header
// X-Tenant-Slug repassado pelo nginx) e usa nome/cor/logo do branding da igreja.
export async function GET() {
  const h = await headers();
  const slug = h.get("x-tenant-slug") || "";
  const apiOrigin = process.env.API_ORIGIN ?? "http://localhost:38080";

  let name = "App do Membro";
  let color = "#0284c7";
  let logo: string | null = null;

  if (slug) {
    try {
      const res = await fetch(`${apiOrigin}/api/v1/public/tenant/${encodeURIComponent(slug)}`, {
        cache: "no-store",
      });
      if (res.ok) {
        const t = (await res.json()) as { name?: string; brand_color?: string; logo_url?: string };
        if (t.name) name = t.name;
        if (t.brand_color) color = t.brand_color;
        if (t.logo_url) logo = t.logo_url;
      }
    } catch {
      /* usa os padroes */
    }
  }

  // Icones estaticos garantem instalacao mesmo sem branding da igreja; o logo
  // da igreja (quando houver) e oferecido como icone adicional "any".
  const icons: { src: string; sizes: string; type?: string; purpose?: string }[] = [
    { src: "/icon-192.png", sizes: "192x192", type: "image/png", purpose: "any" },
    { src: "/icon-512.png", sizes: "512x512", type: "image/png", purpose: "any" },
    { src: "/maskable-512.png", sizes: "512x512", type: "image/png", purpose: "maskable" },
    { src: "/icon.svg", sizes: "any", type: "image/svg+xml", purpose: "any" },
  ];
  if (logo) icons.unshift({ src: logo, sizes: "any", purpose: "any" });

  const shortName = name.length > 12 ? `${name.slice(0, 12).trimEnd()}…` : name;

  return Response.json(
    {
      id: "/",
      name,
      short_name: shortName,
      description: "Agenda, avisos e pedidos de oração da sua igreja.",
      start_url: "/home",
      scope: "/",
      display: "standalone",
      display_override: ["standalone", "minimal-ui"],
      orientation: "portrait",
      background_color: "#ffffff",
      theme_color: color,
      lang: "pt-BR",
      dir: "ltr",
      categories: ["lifestyle", "social"],
      prefer_related_applications: false,
      icons,
      shortcuts: [
        { name: "Agenda", short_name: "Agenda", url: "/agenda" },
        { name: "Pedidos de oração", short_name: "Oração", url: "/oracao" },
        { name: "Avisos", short_name: "Avisos", url: "/avisos" },
      ],
    },
    { headers: { "Content-Type": "application/manifest+json" } },
  );
}
