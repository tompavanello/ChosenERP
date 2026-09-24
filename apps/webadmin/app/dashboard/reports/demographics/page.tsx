"use client";

import { PageHeader } from "@/components/ui/page-header";
import { DemographicsSection } from "@/components/reports/demographics-section";
import { ExportButtons } from "@/components/reports/export-buttons";

export default function DemographicsReportPage() {
  return (
    <div className="page">
      <PageHeader
        title="Demográficos"
        description="Pirâmide etária, situação no Rol, estado civil e distribuição geográfica"
        actions={
          <ExportButtons
            path="/api/v1/reports/demographics/export"
            filenameBase={`demograficos-${new Date().getFullYear()}`}
          />
        }
      />
      <DemographicsSection />
    </div>
  );
}
