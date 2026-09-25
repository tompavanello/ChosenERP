import { Fingerprint, Lock, ScrollText, ShieldCheck } from "lucide-react";
import { SectionHeading } from "@/components/ui/section-heading";
import { security } from "@/lib/site";

const icons = [Lock, ScrollText, Fingerprint, ShieldCheck];

export function Security() {
  return (
    <section className="bg-slate-50/70 py-20 sm:py-24">
      <div className="container-x">
        <SectionHeading
          eyebrow="Segurança e privacidade"
          title="Proteção de nível corporativo para os dados da igreja"
          subtitle="A mesma tecnologia usada por sistemas financeiros sérios, aplicada a membros, dízimos e documentos institucionais."
        />

        <div className="mt-14 grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
          {security.map((item, i) => {
            const Icon = icons[i] ?? ShieldCheck;
            return (
              <div
                key={item}
                className="rounded-2xl border border-slate-200 bg-white p-6 text-center card-hover"
              >
                <span className="mx-auto inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-sky-50 text-sky-600">
                  <Icon className="h-6 w-6" />
                </span>
                <p className="mt-4 text-sm font-medium leading-relaxed text-slate-700">
                  {item}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
