import { CheckCircle2, Gavel } from "lucide-react";
import { SectionHeading } from "@/components/ui/section-heading";
import { governance } from "@/lib/site";

export function Governance() {
  return (
    <section
      id="governanca"
      className="relative scroll-mt-24 overflow-hidden bg-slate-950 py-20 sm:py-24"
    >
      <div className="pointer-events-none absolute inset-0 bg-grid-dark [mask-image:radial-gradient(80%_70%_at_20%_0%,#000_10%,transparent_100%)]" />
      <div className="pointer-events-none absolute -right-20 top-10 h-80 w-80 rounded-full bg-sky-600/20 blur-3xl" />

      <div className="container-x relative grid items-center gap-12 lg:grid-cols-2">
        <div>
          <span className="inline-flex items-center gap-2 rounded-full border border-sky-500/30 bg-sky-500/10 px-3.5 py-1.5 text-xs font-semibold text-sky-300">
            <Gavel className="h-3.5 w-3.5" />
            Diferencial exclusivo
          </span>
          <SectionHeading
            eyebrow="Governança e compliance"
            title={governance.title}
            subtitle={governance.description}
            align="left"
            tone="dark"
            className="mt-5 max-w-none"
          />
        </div>

        <ul className="space-y-3">
          {governance.items.map((item) => (
            <li
              key={item}
              className="flex items-start gap-3 rounded-2xl border border-slate-800 bg-slate-900/70 p-5 backdrop-blur"
            >
              <CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0 text-sky-400" />
              <span className="text-sm leading-relaxed text-slate-200">
                {item}
              </span>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
