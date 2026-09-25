import { Plus } from "lucide-react";
import { SectionHeading } from "@/components/ui/section-heading";
import { faq } from "@/lib/site";

export function Faq() {
  return (
    <section id="faq" className="scroll-mt-24 bg-slate-50/70 py-20 sm:py-24">
      <div className="container-x">
        <SectionHeading
          eyebrow="Dúvidas frequentes"
          title="Perguntas que todo pastor e tesoureiro faz"
          subtitle="Não encontrou a sua? Fale com um especialista pelo WhatsApp."
        />

        <div className="mx-auto mt-12 max-w-3xl divide-y divide-slate-200 rounded-2xl border border-slate-200 bg-white">
          {faq.map((item) => (
            <details key={item.q} className="group px-6">
              <summary className="flex cursor-pointer list-none items-center justify-between gap-4 py-5 text-left text-base font-semibold text-slate-800 [&::-webkit-details-marker]:hidden">
                {item.q}
                <Plus className="h-5 w-5 shrink-0 text-sky-600 transition-transform duration-300 group-open:rotate-45" />
              </summary>
              <p className="pb-5 pr-8 text-sm leading-relaxed text-slate-600">
                {item.a}
              </p>
            </details>
          ))}
        </div>
      </div>
    </section>
  );
}
