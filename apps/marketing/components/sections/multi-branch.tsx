import { Building2, Check, ChevronDown } from "lucide-react";
import { SectionHeading } from "@/components/ui/section-heading";
import { multiBranch } from "@/lib/site";

function Node({
  label,
  sub,
  tone = "default",
}: {
  label: string;
  sub: string;
  tone?: "default" | "brand";
}) {
  return (
    <div
      className={
        "rounded-xl border px-3.5 py-2.5 text-center shadow-sm " +
        (tone === "brand"
          ? "border-sky-600 bg-sky-600 text-white"
          : "border-slate-200 bg-white text-slate-700")
      }
    >
      <p className="text-xs font-semibold">{label}</p>
      <p
        className={
          "text-[10px] " + (tone === "brand" ? "text-sky-100" : "text-slate-400")
        }
      >
        {sub}
      </p>
    </div>
  );
}

export function MultiBranch() {
  return (
    <section id="multi-filial" className="scroll-mt-24 py-20 sm:py-24">
      <div className="container-x grid items-center gap-14 lg:grid-cols-2">
        <div>
          <span className="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-white px-3.5 py-1.5 text-xs font-semibold text-slate-600">
            <Building2 className="h-3.5 w-3.5 text-sky-600" />
            Multi-filial
          </span>
          <SectionHeading
            eyebrow="Estrutura de governo"
            title={multiBranch.title}
            subtitle={multiBranch.description}
            align="left"
            className="mt-5 max-w-none"
          />
          <ul className="mt-8 space-y-3">
            {multiBranch.items.map((item) => (
              <li key={item} className="flex items-start gap-3">
                <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-sky-100">
                  <Check className="h-3 w-3 text-sky-700" />
                </span>
                <span className="text-sm leading-relaxed text-slate-700">
                  {item}
                </span>
              </li>
            ))}
          </ul>
        </div>

        {/* Diagrama da hierarquia */}
        <div className="rounded-2xl border border-slate-200 bg-slate-50/70 p-8">
          <div className="flex flex-col items-center">
            <Node label="Matriz" sub="Sede administrativa" tone="brand" />
            <ChevronDown className="my-1 h-4 w-4 text-slate-300" />
            <div className="grid w-full grid-cols-2 gap-4">
              <div className="flex flex-col items-center">
                <Node label="Filial Norte" sub="Igreja local" />
                <ChevronDown className="my-1 h-4 w-4 text-slate-300" />
                <Node label="PAE" sub="Ponto de evangelização" />
              </div>
              <div className="flex flex-col items-center">
                <Node label="Filial Sul" sub="Igreja local" />
                <ChevronDown className="my-1 h-4 w-4 text-slate-300" />
                <Node label="PAE" sub="Ponto de evangelização" />
              </div>
            </div>
          </div>
          <p className="mt-6 text-center text-xs text-slate-500">
            A sede enxerga o consolidado; cada unidade grava apenas no seu
            escopo.
          </p>
        </div>
      </div>
    </section>
  );
}
