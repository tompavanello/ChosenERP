import { Mail, MessageCircle, Phone } from "lucide-react";
import { LeadForm } from "@/components/lead-form";
import { finalCta, site, whatsappLink, whatsappMessage } from "@/lib/site";

export function FinalCta() {
  return (
    <section id="contato" className="scroll-mt-24 py-20 sm:py-24">
      <div className="container-x grid items-start gap-14 lg:grid-cols-2">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.18em] text-sky-600">
            Demonstração
          </p>
          <h2 className="mt-3 text-3xl font-bold leading-tight text-slate-900 sm:text-4xl">
            {finalCta.title}
          </h2>
          <p className="mt-4 max-w-lg text-lg leading-relaxed text-slate-600">
            {finalCta.subtitle}
          </p>

          <div className="mt-8 space-y-3">
            <a
              href={whatsappLink(whatsappMessage)}
              className="flex items-center gap-3 rounded-2xl border border-slate-200 bg-white p-4 transition-colors hover:border-emerald-300 hover:bg-emerald-50"
            >
              <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-100 text-emerald-600">
                <MessageCircle className="h-5 w-5" />
              </span>
              <span>
                <span className="block text-sm font-semibold text-slate-900">
                  WhatsApp
                </span>
                <span className="block text-sm text-slate-500">
                  {site.whatsappLabel}
                </span>
              </span>
            </a>
            <a
              href={`mailto:${site.email}`}
              className="flex items-center gap-3 rounded-2xl border border-slate-200 bg-white p-4 transition-colors hover:border-sky-300 hover:bg-sky-50"
            >
              <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-sky-100 text-sky-600">
                <Mail className="h-5 w-5" />
              </span>
              <span>
                <span className="block text-sm font-semibold text-slate-900">
                  E-mail
                </span>
                <span className="block text-sm text-slate-500">
                  {site.email}
                </span>
              </span>
            </a>
            <div className="flex items-center gap-3 rounded-2xl border border-slate-200 bg-white p-4">
              <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-slate-100 text-slate-600">
                <Phone className="h-5 w-5" />
              </span>
              <span>
                <span className="block text-sm font-semibold text-slate-900">
                  Atendimento
                </span>
                <span className="block text-sm text-slate-500">
                  {site.city}
                </span>
              </span>
            </div>
          </div>
        </div>

        <div className="rounded-3xl border border-slate-200 bg-white p-7 shadow-product sm:p-9">
          <LeadForm />
        </div>
      </div>
    </section>
  );
}
