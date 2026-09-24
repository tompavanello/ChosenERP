"use client";

import { PageHeader } from "@/components/ui/page-header";
import { DemographicsSection } from "@/components/reports/demographics-section";
import { ExportButtons } from "@/components/reports/export-buttons";

export default function DemographicsReportPage() {
  return (
    <div className="page">
      <PageHeader
        title="Demograficos"
        description="Piramide etaria, situacao no Rol, estado civil e distribuicao geografica"
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
