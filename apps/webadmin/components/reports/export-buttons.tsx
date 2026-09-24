"use client";

import { Download, FileSpreadsheet, FileText } from "lucide-react";
import { Button } from "@/components/ui/button";
import { downloadExport, openPrint } from "@/lib/api";
import { useToast } from "@/components/ui/toast";

/**
 * Botoes de exportacao de um relatorio nos tres formatos.
 *
 * - CSV e Excel (XLSX) baixam o arquivo.
 * - PDF abre a versao de impressao numa aba nova (o navegador salva em PDF),
 *   porque o endpoint exige auth e uma navegacao nova nao envia o Bearer.
 */
export function ExportButtons({
  path,
  params,
  filenameBase,
}: {
  path: string;
  params?: Record<string, string>;
  filenameBase: string;
}) {
  const { toast } = useToast();

  const url = (format: string) => {
    const q = new URLSearchParams(params ?? {});
    q.set("format", format);
    return `${path}?${q.toString()}`;
  };

  async function go(kind: "csv" | "xlsx" | "pdf") {
    try {
      if (kind === "pdf") {
        await openPrint(url("pdf"));
      } else {
        await downloadExport(url(kind), `${filenameBase}.${kind}`);
      }
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao exportar", "error");
    }
  }

  return (
    <div className="flex flex-wrap gap-2">
      <Button variant="outline" onClick={() => go("csv")}>
        <Download className="h-4 w-4" /> CSV
      </Button>
      <Button variant="outline" onClick={() => go("xlsx")}>
        <FileSpreadsheet className="h-4 w-4" /> Excel
      </Button>
      <Button variant="outline" onClick={() => go("pdf")}>
        <FileText className="h-4 w-4" /> PDF
      </Button>
    </div>
  );
}
