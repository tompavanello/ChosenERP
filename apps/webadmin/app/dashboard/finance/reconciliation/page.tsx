"use client";

import { useEffect, useState } from "react";
import { ClipboardCheck, Landmark } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Tabs } from "@/components/ui/tabs";
import { ReconciliationPanel } from "@/components/finance/reconciliation-panel";
import { BankReconciliationPanel } from "@/components/finance/bank-reconciliation-panel";

export default function ReconciliationPage() {
  const [tab, setTab] = useState("periodo");

  // Permite abrir a aba bancaria por link direto (?tab=bancaria), sem depender
  // de useSearchParams (que exigiria Suspense no App Router).
  useEffect(() => {
    const p = new URLSearchParams(window.location.search).get("tab");
    if (p === "bancaria" || p === "periodo") setTab(p);
  }, []);

  return (
    <div className="page">
      <PageHeader
        title="Conciliacao"
        description="Concilie o periodo e o extrato bancario. Uma pessoa lanca, outra concilia e outra audita."
      />
      <Tabs
        tabs={[
          { key: "periodo", label: "Periodo", icon: <ClipboardCheck className="h-4 w-4" /> },
          { key: "bancaria", label: "Bancaria", icon: <Landmark className="h-4 w-4" /> },
        ]}
        active={tab}
        onChange={setTab}
      />
      {tab === "periodo" ? <ReconciliationPanel /> : <BankReconciliationPanel />}
    </div>
  );
}
