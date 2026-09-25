import {
  BarChart3,
  CalendarDays,
  Gavel,
  LayoutDashboard,
  Users,
  Wallet,
} from "lucide-react";

const menu = [
  { icon: LayoutDashboard, label: "Visão geral", active: true },
  { icon: Users, label: "Membros" },
  { icon: Wallet, label: "Financeiro" },
  { icon: CalendarDays, label: "Eventos" },
  { icon: Gavel, label: "Governança" },
  { icon: BarChart3, label: "Relatórios" },
];

const rows = [
  { name: "Dízimos e ofertas", value: "R$ 38.420,00", tone: "text-emerald-600" },
  { name: "Missões", value: "R$ 6.120,00", tone: "text-emerald-600" },
  { name: "Manutenção do templo", value: "R$ 4.870,00", tone: "text-rose-600" },
  { name: "Ação social", value: "R$ 2.310,00", tone: "text-rose-600" },
];

/** Previa do produto em HTML/CSS puro (sem imagem) — leve e sempre nitida. */
export function ProductMockup() {
  return (
    <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-product">
      {/* Barra do navegador */}
      <div className="flex items-center gap-3 border-b border-slate-100 bg-slate-50/80 px-4 py-2.5">
        <div className="flex gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full bg-rose-400" />
          <span className="h-2.5 w-2.5 rounded-full bg-amber-400" />
          <span className="h-2.5 w-2.5 rounded-full bg-emerald-400" />
        </div>
        <div className="mx-auto hidden rounded-md bg-white px-3 py-1 text-[11px] font-medium text-slate-400 ring-1 ring-slate-200 sm:block">
          app.erpchosen.com.br
        </div>
      </div>

      <div className="flex">
        {/* Sidebar */}
        <aside className="hidden w-44 shrink-0 border-r border-slate-100 bg-slate-50/60 p-3 sm:block">
          <div className="mb-3 flex items-center gap-2 px-1">
            <span className="flex h-6 w-6 items-center justify-center rounded-md bg-gradient-to-br from-sky-600 to-sky-400 text-[10px] font-bold text-white">
              C
            </span>
            <span className="text-xs font-semibold text-slate-700">
              Chosen ERP
            </span>
          </div>
          <nav className="space-y-0.5">
            {menu.map((m) => (
              <div
                key={m.label}
                className={
                  "flex items-center gap-2 rounded-md px-2 py-1.5 text-[11px] font-medium " +
                  (m.active
                    ? "bg-sky-100 text-sky-700"
                    : "text-slate-500")
                }
              >
                <m.icon className="h-3.5 w-3.5" />
                {m.label}
              </div>
            ))}
          </nav>
        </aside>

        {/* Conteudo */}
        <div className="min-w-0 flex-1 p-4">
          <div className="mb-3 flex items-end justify-between">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">
                Visão geral
              </p>
              <p className="text-sm font-semibold text-slate-800">
                Igreja Videira · Matriz
              </p>
            </div>
            <span className="hidden rounded-full bg-emerald-50 px-2 py-0.5 text-[10px] font-semibold text-emerald-700 sm:inline">
              Consolidado
            </span>
          </div>

          {/* Cards */}
          <div className="mb-3 grid grid-cols-3 gap-2">
            {[
              { k: "Membros", v: "1.248", t: "text-sky-600" },
              { k: "Entradas", v: "R$ 44,5k", t: "text-emerald-600" },
              { k: "Frequência", v: "812", t: "text-indigo-600" },
            ].map((c) => (
              <div
                key={c.k}
                className="rounded-lg border border-slate-100 bg-white p-2.5"
              >
                <p className="text-[10px] font-medium text-slate-400">{c.k}</p>
                <p className={"text-sm font-bold " + c.t}>{c.v}</p>
              </div>
            ))}
          </div>

          {/* Grafico */}
          <div className="mb-3 rounded-lg border border-slate-100 p-3">
            <svg viewBox="0 0 320 90" className="h-24 w-full" aria-hidden>
              <defs>
                <linearGradient id="mock-area" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#0ea5e9" stopOpacity="0.35" />
                  <stop offset="100%" stopColor="#0ea5e9" stopOpacity="0" />
                </linearGradient>
              </defs>
              <path
                d="M0 66 L40 58 L80 62 L120 44 L160 50 L200 30 L240 36 L280 18 L320 24 L320 90 L0 90 Z"
                fill="url(#mock-area)"
              />
              <path
                d="M0 66 L40 58 L80 62 L120 44 L160 50 L200 30 L240 36 L280 18 L320 24"
                fill="none"
                stroke="#0ea5e9"
                strokeWidth="2.5"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </div>

          {/* Tabela */}
          <div className="space-y-1.5">
            {rows.map((r) => (
              <div
                key={r.name}
                className="flex items-center justify-between rounded-md bg-slate-50 px-2.5 py-1.5"
              >
                <span className="text-[11px] text-slate-600">{r.name}</span>
                <span className={"text-[11px] font-semibold " + r.tone}>
                  {r.value}
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
