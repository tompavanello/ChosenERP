"use client";

import { useEffect, useState } from "react";
import { ClipboardCheck, Landmark, ListChecks } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Tabs } from "@/components/ui/tabs";
import { ReconciliationPanel } from "@/components/finance/reconciliation-panel";
import { ReconciliationEntries } from "@/components/finance/reconciliation-entries";
import { BankReconciliationPanel } from "@/components/finance/bank-reconciliation-panel";

export default function ReconciliationPage() {
  const [tab, setTab] = useState("lancamentos");

  // Permite abrir uma aba por link direto (?tab=...), sem depender de
  // useSearchParams (que exigiria Suspense no App Router).
  useEffect(() => {
    const p = new URLSearchParams(window.location.search).get("tab");
    if (p === "bancaria" || p === "periodo" || p === "lancamentos") setTab(p);
  }, []);

  return (
    <div className="page">
      <PageHeader
        title="Conciliacao"
        description="Concilie lancamentos, encerre o periodo pelo proprio conciliador e importe o extrato bancario na aba Bancaria."
      />
      <Tabs
        tabs={[
          { key: "lancamentos", label: "Lancamentos", icon: <ListChecks className="h-4 w-4" /> },
          { key: "periodo", label: "Periodos", icon: <ClipboardCheck className="h-4 w-4" /> },
          { key: "bancaria", label: "Bancaria", icon: <Landmark className="h-4 w-4" /> },
        ]}
        active={tab}
        onChange={setTab}
      />
      {tab === "lancamentos" && <ReconciliationEntries />}
      {tab === "periodo" && <ReconciliationPanel />}
      {tab === "bancaria" && <BankReconciliationPanel />}
    </div>
  );
}
