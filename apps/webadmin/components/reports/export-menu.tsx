"use client";

import type { ReactNode } from "react";
import { Download, FileSpreadsheet, FileText } from "lucide-react";
import { Dropdown, type DropdownItem } from "@/components/ui/dropdown";
import { downloadExport, openPrint } from "@/lib/api";
import { useToast } from "@/components/ui/toast";

/**
 * Menu compacto de exportacao (PDF/Excel/CSV). Use quando a tela teria varios
 * botoes visiveis: o gatilho pode ser o proprio titulo do card ou um botao
 * "Exportar".
 */
export function ExportMenu({
  path,
  params,
  filenameBase,
  trigger,
}: {
  path: string;
  params?: Record<string, string>;
  filenameBase: string;
  trigger: ReactNode;
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

  const items: DropdownItem[] = [
    { label: "Imprimir / PDF", icon: <FileText className="h-4 w-4" />, onClick: () => go("pdf") },
    { label: "Excel (XLSX)", icon: <FileSpreadsheet className="h-4 w-4" />, onClick: () => go("xlsx") },
    { label: "CSV", icon: <Download className="h-4 w-4" />, onClick: () => go("csv") },
  ];

  return <Dropdown trigger={trigger} triggerAsChild items={items} />;
}
