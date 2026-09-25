import { Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { SectionHeading } from "@/components/ui/section-heading";
import { pricing, whatsappLink } from "@/lib/site";

export function Pricing() {
  return (
    <section id="planos" className="scroll-mt-24 py-20 sm:py-24">
      <div className="container-x">
        <SectionHeading
          eyebrow="Planos"
          title="Um plano para cada momento da igreja"
          subtitle={pricing.note}
        />

        <div className="mt-14 grid items-stretch gap-6 lg:grid-cols-3">
          {pricing.tiers.map((tier) => (
            <div
              key={tier.name}
              className={
                "relative flex flex-col rounded-2xl border p-7 " +
                (tier.highlighted
                  ? "border-sky-600 bg-white shadow-product lg:-mt-4 lg:mb-4"
                  : "border-slate-200 bg-white")
              }
            >
              {tier.highlighted && (
                <span className="absolute -top-3 left-1/2 -translate-x-1/2 rounded-full bg-sky-600 px-3 py-1 text-[11px] font-semibold uppercase tracking-wide text-white">
                  Mais popular
                </span>
              )}
              <h3 className="text-lg font-bold text-slate-900">{tier.name}</h3>
              <p className="mt-1.5 text-sm text-slate-500">{tier.tagline}</p>

              <p className="mt-5 text-2xl font-bold text-slate-900">
                Sob consulta
              </p>
              <p className="text-xs text-slate-400">
                Proposta sob medida para a sua igreja
              </p>

              <ul className="mt-6 flex-1 space-y-3">
                {tier.features.map((f) => (
                  <li key={f} className="flex items-start gap-2.5">
                    <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-emerald-100">
                      <Check className="h-3 w-3 text-emerald-600" />
                    </span>
                    <span className="text-sm text-slate-700">{f}</span>
                  </li>
                ))}
              </ul>

              <Button
                href={whatsappLink(
                  `Olá! Tenho interesse no plano ${tier.name} do Chosen ERP.`,
                )}
                variant={tier.highlighted ? "primary" : "outline"}
                size="lg"
                className="mt-8 w-full"
              >
                Falar com especialista
              </Button>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
