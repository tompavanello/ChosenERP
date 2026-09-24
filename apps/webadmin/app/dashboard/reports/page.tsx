import { redirect } from "next/navigation";

// O grupo "Relatorios" aponta para as paginas especificas; esta rota e so um
// atalho de compatibilidade (links antigos para /dashboard/reports).
export default function ReportsIndex() {
  redirect("/dashboard/reports/balance");
}
