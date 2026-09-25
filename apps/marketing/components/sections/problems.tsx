import { CheckCircle2, XCircle } from "lucide-react";
import { SectionHeading } from "@/components/ui/section-heading";
import { problems } from "@/lib/site";

export function Problems() {
  return (
    <section className="py-20 sm:py-24">
      <div className="container-x">
        <SectionHeading
          eyebrow="O problema"
          title="A gestão da igreja não pode viver em retalhos"
          subtitle="Quando cada área usa uma ferramenta diferente, o pastor perde tempo, o financeiro perde confiança e o membro se perde no caminho."
        />

        <div className="mt-14 grid gap-5 md:grid-cols-2">
          {problems.map((p) => (
            <div
              key={p.problem}
              className="rounded-2xl border border-slate-200 bg-white p-6 card-hover"
            >
              <div className="flex items-start gap-3">
                <XCircle className="mt-0.5 h-5 w-5 shrink-0 text-rose-500" />
                <p className="text-sm font-medium text-slate-500 line-through decoration-rose-300">
                  {p.problem}
                </p>
              </div>
              <div className="mt-4 flex items-start gap-3 border-t border-slate-100 pt-4">
                <CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0 text-emerald-500" />
                <p className="text-base font-semibold text-slate-800">
                  {p.solution}
                </p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
