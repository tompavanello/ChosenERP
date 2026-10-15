"use client";

import { PageHeader } from "@/components/ui/page-header";
import { MaterialsManager } from "@/components/materials/materials-manager";

export default function MaterialsPage() {
  return (
    <div className="page">
      <PageHeader title="Materiais de estudo" description="Arquivos e links para a igreja ou para um grupo/celula (app do membro)" />
      <MaterialsManager />
    </div>
  );
}
