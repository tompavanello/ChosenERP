import { redirect } from "next/navigation";

// O grupo "Relatórios" aponta para as páginas específicas; esta rota é só um
// atalho de compatibilidade (links antigos para /dashboard/reports).
export default function ReportsIndex() {
  redirect("/dashboard/reports/balance");
}
