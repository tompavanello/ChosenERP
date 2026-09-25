import { redirect } from "next/navigation";

// A conciliacao bancaria virou a aba "Bancaria" de /dashboard/finance/reconciliation.
// Mantem o link antigo funcionando.
export default function BankReconciliationRedirect() {
  redirect("/dashboard/finance/reconciliation?tab=bancaria");
}
