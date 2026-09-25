import { SectionHeading } from "@/components/ui/section-heading";
import { steps } from "@/lib/site";

export function Steps() {
  return (
    <section className="py-20 sm:py-24">
      <div className="container-x">
        <SectionHeading
          eyebrow="Como funciona"
          title="Do primeiro contato à igreja organizada"
          subtitle="Sem projeto interminável: em poucos passos a sua equipe já está usando o sistema no dia a dia."
        />

        <div className="relative mt-14 grid gap-6 md:grid-cols-3">
          {steps.map((step, i) => (
            <div
              key={step.title}
              className="relative rounded-2xl border border-slate-200 bg-white p-7"
            >
              <span className="flex h-11 w-11 items-center justify-center rounded-xl bg-sky-600 text-lg font-bold text-white shadow-lg shadow-sky-600/25">
                {i + 1}
              </span>
              <h3 className="mt-4 text-lg font-semibold text-slate-900">
                {step.title}
              </h3>
              <p className="mt-2 text-sm leading-relaxed text-slate-600">
                {step.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
