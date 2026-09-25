import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { SiteHeader } from "@/components/site-header";
import { SiteFooter } from "@/components/site-footer";
import { WhatsAppButton } from "@/components/whatsapp-button";
import { site } from "@/lib/site";

const inter = Inter({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-inter",
});

const baseUrl = `https://${site.domain}`;

export const metadata: Metadata = {
  metadataBase: new URL(baseUrl),
  title: {
    default: "Chosen ERP · A escolha inteligente para a igreja",
    template: "%s · Chosen ERP",
  },
  description: site.description,
  keywords: [
    "gestão de igreja",
    "sistema para igreja",
    "software eclesiástico",
    "ERP para igreja",
    "secretaria de membros",
    "financeiro de igreja",
    "multi-filial",
  ],
  authors: [{ name: "Chosen ERP" }],
  alternates: { canonical: "/" },
  openGraph: {
    type: "website",
    locale: "pt_BR",
    url: baseUrl,
    siteName: "Chosen ERP",
    title: "Chosen ERP · A escolha inteligente para a igreja",
    description: site.description,
  },
  twitter: {
    card: "summary_large_image",
    title: "Chosen ERP · A escolha inteligente para a igreja",
    description: site.description,
  },
  robots: { index: true, follow: true },
};

const jsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "Organization",
      name: "Chosen ERP",
      url: baseUrl,
      logo: `${baseUrl}/logo-mark.svg`,
      email: site.email,
      areaServed: "BR",
      description: site.description,
    },
    {
      "@type": "SoftwareApplication",
      name: "Chosen ERP",
      applicationCategory: "BusinessApplication",
      operatingSystem: "Web",
      description: site.description,
      offers: {
        "@type": "Offer",
        price: "0",
        priceCurrency: "BRL",
        description: "Demonstração e proposta sob medida.",
      },
    },
  ],
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="pt-BR" className={inter.variable}>
      <body className="min-h-screen antialiased">
        <a
          href="#topo"
          className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[60] focus:rounded-lg focus:bg-sky-600 focus:px-4 focus:py-2 focus:text-sm focus:font-semibold focus:text-white"
        >
          Ir para o conteúdo
        </a>
        <SiteHeader />
        <main>{children}</main>
        <SiteFooter />
        <WhatsAppButton />
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
        />
      </body>
    </html>
  );
}
