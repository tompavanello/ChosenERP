import { segments, stats } from "@/lib/site";

export function Trust() {
  return (
    <section className="border-y border-slate-100 bg-slate-50/70">
      <div className="container-x py-14">
        <p className="text-center text-xs font-semibold uppercase tracking-[0.18em] text-slate-400">
          Feito para igrejas de todo porte
        </p>
        <div className="mt-5 flex flex-wrap items-center justify-center gap-x-8 gap-y-3">
          {segments.map((s) => (
            <span
              key={s}
              className="text-sm font-semibold text-slate-500"
            >
              {s}
            </span>
          ))}
        </div>

        <dl className="mt-12 grid grid-cols-2 gap-6 lg:grid-cols-4">
          {stats.map((s) => (
            <div key={s.label} className="text-center">
              <dt className="text-3xl font-bold tracking-tight text-sky-600 sm:text-4xl">
                {s.value}
              </dt>
              <dd className="mx-auto mt-2 max-w-[16rem] text-sm text-slate-600">
                {s.label}
              </dd>
            </div>
          ))}
        </dl>
      </div>
    </section>
  );
}
