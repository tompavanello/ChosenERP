"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowDownRight, ArrowUpRight, Wallet } from "lucide-react";
import { getBalance, type Balance } from "@/lib/api";

export default function OverviewPage() {
  const router = useRouter();
  const [balance, setBalance] = useState<Balance | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const token = localStorage.getItem("chosen_token");
    if (!token) {
      router.replace("/");
      return;
    }
    getBalance(token).then(setBalance).catch((e) => setError(e instanceof Error ? e.message : "Erro"));
  }, [router]);

  const fmt = (n: number) =>
    n.toLocaleString("pt-BR", { style: "currency", currency: "BRL" });

  const cards = [
    { label: "Entradas", value: balance ? fmt(balance.income) : "...", icon: ArrowUpRight, tone: "text-emerald-600" },
    { label: "Saídas", value: balance ? fmt(balance.expense) : "...", icon: ArrowDownRight, tone: "text-red-600" },
    { label: "Saldo", value: balance ? fmt(balance.net) : "...", icon: Wallet, tone: "text-violet-700" },
  ];

  return (
    <div className="mx-auto max-w-5xl">
      <h2 className="mb-6 text-2xl font-semibold">Visão Geral</h2>
      {error && <p className="mb-4 text-sm text-red-600">{error}</p>}

      <div className="grid gap-4 md:grid-cols-3">
        {cards.map((c) => (
          <div key={c.label} className="card">
            <div className="flex items-center justify-between">
              <span className="text-sm text-zinc-500">{c.label}</span>
              <c.icon className={`h-5 w-5 ${c.tone}`} />
            </div>
            <p className="mt-3 text-2xl font-semibold">{c.value}</p>
          </div>
        ))}
      </div>

      {balance && balance.by_category.length > 0 && (
        <div className="card mt-6">
          <h3 className="mb-4 text-sm font-medium text-zinc-600">Por categoria</h3>
          <ul className="space-y-2">
            {balance.by_category.map((c) => (
              <li key={c.category_id} className="flex items-center justify-between text-sm">
                <span className="flex items-center gap-2">
                  <span className={`h-2 w-2 rounded-full ${c.type === "income" ? "bg-emerald-500" : "bg-red-500"}`} />
                  {c.category}
                </span>
                <span className="font-medium">{fmt(c.total)}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
