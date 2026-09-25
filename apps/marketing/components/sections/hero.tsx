import { ArrowRight, Check, PlayCircle, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ProductMockup } from "@/components/product-mockup";
import { RotatingPhrase } from "@/components/rotating-phrase";
import { hero, whatsappLink, whatsappMessage } from "@/lib/site";

export function Hero() {
  return (
    <section id="topo" className="relative overflow-hidden pt-28 lg:pt-36">
      {/* Fundo decorativo */}
      <div className="pointer-events-none absolute inset-0 -z-10">
        <div className="absolute inset-0 bg-grid [mask-image:radial-gradient(70%_60%_at_50%_0%,#000_20%,transparent_100%)]" />
        <div className="absolute -top-32 left-1/2 h-[520px] w-[880px] -translate-x-1/2 rounded-full bg-sky-200/45 blur-3xl" />
        <div className="absolute right-[-10%] top-40 h-72 w-72 rounded-full bg-cyan-200/40 blur-3xl" />
      </div>

      <div className="container-x grid items-center gap-14 pb-16 lg:grid-cols-[1.05fr_1fr] lg:pb-24">
        <div className="animate-fade-up">
          <span className="inline-flex items-center gap-2 rounded-full border border-sky-200 bg-sky-50 px-3.5 py-1.5 text-xs font-semibold text-sky-700">
            <Sparkles className="h-3.5 w-3.5" />
            {hero.eyebrow}
          </span>

          <h1 className="mt-5 text-4xl font-bold leading-[1.08] tracking-tight text-slate-900 sm:text-5xl lg:text-[3.4rem]">
            {hero.titlePrefix}{" "}
            <RotatingPhrase phrases={hero.rotate} /> {hero.titleSuffix}
          </h1>

          <p className="mt-6 max-w-xl text-lg leading-relaxed text-slate-600">
            {hero.subtitle}
          </p>

          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <Button href="#contato" size="lg">
              {hero.primaryCta}
              <ArrowRight className="h-5 w-5" />
            </Button>
            <Button href="#recursos" variant="outline" size="lg">
              <PlayCircle className="h-5 w-5" />
              {hero.secondaryCta}
            </Button>
          </div>

          <ul className="mt-8 flex flex-wrap gap-x-6 gap-y-2">
            {hero.bullets.map((b) => (
              <li
                key={b}
                className="flex items-center gap-2 text-sm font-medium text-slate-600"
              >
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-emerald-100">
                  <Check className="h-3 w-3 text-emerald-600" />
                </span>
                {b}
              </li>
            ))}
          </ul>
        </div>

        <div className="relative animate-fade-up [animation-delay:120ms]">
          <ProductMockup />
          <div className="absolute -left-4 top-16 hidden animate-float-slow rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 shadow-lg lg:block">
            <p className="text-[10px] font-medium text-slate-400">
              Dízimo recebido
            </p>
            <p className="text-sm font-bold text-emerald-600">+ R$ 1.240,00</p>
          </div>
          <div className="absolute -right-4 bottom-14 hidden animate-float-slow rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 shadow-lg [animation-delay:1.5s] lg:block">
            <p className="text-[10px] font-medium text-slate-400">
              Escala confirmada
            </p>
            <p className="text-sm font-bold text-sky-600">Louvor · Domingo</p>
          </div>
        </div>
      </div>
    </section>
  );
}
