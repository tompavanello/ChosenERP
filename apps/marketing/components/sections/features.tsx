import {
  Baby,
  BarChart3,
  Building2,
  CalendarDays,
  DoorOpen,
  Gavel,
  HeartHandshake,
  MessageCircle,
  Network,
  QrCode,
  Receipt,
  ShieldCheck,
  Users,
  Wallet,
  type LucideIcon,
} from "lucide-react";
import { SectionHeading } from "@/components/ui/section-heading";
import { features, type FeatureIcon } from "@/lib/site";

const icons: Record<FeatureIcon, LucideIcon> = {
  users: Users,
  network: Network,
  wallet: Wallet,
  receipt: Receipt,
  calendar: CalendarDays,
  handshake: HeartHandshake,
  gavel: Gavel,
  message: MessageCircle,
  chart: BarChart3,
  building: Building2,
  baby: Baby,
  qr: QrCode,
  door: DoorOpen,
  shield: ShieldCheck,
};

export function Features() {
  return (
    <section id="recursos" className="scroll-mt-24 bg-slate-50/70 py-20 sm:py-24">
      <div className="container-x">
        <SectionHeading
          eyebrow="Recursos"
          title="Tudo o que a igreja precisa, em um único sistema"
          subtitle="Da secretaria ao financeiro, dos ministérios à governança. Módulos integrados que conversam entre si — sem retrabalho e sem planilha paralela."
        />

        <div className="mt-14 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {features.map((f) => {
            const Icon = icons[f.icon];
            return (
              <div
                key={f.title}
                className="group rounded-2xl border border-slate-200 bg-white p-6 card-hover"
              >
                <span className="inline-flex h-11 w-11 items-center justify-center rounded-xl bg-sky-50 text-sky-600 transition-colors group-hover:bg-sky-600 group-hover:text-white">
                  <Icon className="h-5 w-5" />
                </span>
                <h3 className="mt-4 text-base font-semibold text-slate-900">
                  {f.title}
                </h3>
                <p className="mt-2 text-sm leading-relaxed text-slate-600">
                  {f.description}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
