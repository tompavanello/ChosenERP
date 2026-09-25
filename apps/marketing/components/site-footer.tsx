import { Mail, MapPin, MessageCircle, Phone } from "lucide-react";
import { LogoMark } from "@/components/brand";
import { nav, site, whatsappLink, whatsappMessage } from "@/lib/site";

export function SiteFooter() {
  const year = new Date().getFullYear();
  return (
    <footer className="border-t border-slate-800 bg-slate-950 text-slate-400">
      <div className="container-x grid gap-10 py-14 md:grid-cols-4">
        <div className="md:col-span-2">
          <div className="flex items-center gap-2.5">
            <LogoMark className="h-9 w-9" />
            <span className="text-lg font-bold text-white">Chosen ERP</span>
          </div>
          <p className="mt-4 max-w-sm text-sm leading-relaxed">
            Plataforma de gestão eclesiástica que une secretaria, financeiro,
            eventos, ministérios, governança e comunicação em um só lugar.
          </p>
          <a
            href={whatsappLink(whatsappMessage)}
            className="mt-5 inline-flex items-center gap-2 rounded-xl bg-sky-600 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-sky-500"
          >
            <MessageCircle className="h-4 w-4" />
            Falar com um especialista
          </a>
        </div>

        <div>
          <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-300">
            Navegação
          </h3>
          <ul className="mt-4 space-y-2.5 text-sm">
            {nav.map((item) => (
              <li key={item.href}>
                <a
                  href={item.href}
                  className="transition-colors hover:text-white"
                >
                  {item.label}
                </a>
              </li>
            ))}
            <li>
              <a
                href={site.appUrl}
                className="transition-colors hover:text-white"
              >
                Entrar no sistema
              </a>
            </li>
          </ul>
        </div>

        <div>
          <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-300">
            Contato
          </h3>
          <ul className="mt-4 space-y-3 text-sm">
            <li className="flex items-start gap-2.5">
              <Mail className="mt-0.5 h-4 w-4 shrink-0 text-sky-400" />
              <a
                href={`mailto:${site.email}`}
                className="transition-colors hover:text-white"
              >
                {site.email}
              </a>
            </li>
            <li className="flex items-start gap-2.5">
              <Phone className="mt-0.5 h-4 w-4 shrink-0 text-sky-400" />
              <span>{site.whatsappLabel}</span>
            </li>
            <li className="flex items-start gap-2.5">
              <MapPin className="mt-0.5 h-4 w-4 shrink-0 text-sky-400" />
              <span>{site.city}</span>
            </li>
          </ul>
        </div>
      </div>

      <div className="border-t border-slate-800/80">
        <div className="container-x flex flex-col items-center justify-between gap-3 py-6 text-xs sm:flex-row">
          <p>
            © {year} Chosen ERP. Todos os direitos reservados.
          </p>
          <p>
            {site.domain} · {site.slogan}
          </p>
        </div>
      </div>
    </footer>
  );
}
