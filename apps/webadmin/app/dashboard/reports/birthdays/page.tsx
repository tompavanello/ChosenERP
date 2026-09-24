"use client";

import { useCallback, useEffect, useState } from "react";
import { Cake, Heart } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Select } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getBirthdays, type BirthdaysResult } from "@/lib/api";
import { datePt } from "@/lib/format";
import { ExportButtons } from "@/components/reports/export-buttons";

const MONTHS = [
  "Janeiro", "Fevereiro", "Marco", "Abril", "Maio", "Junho",
  "Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro",
];

export default function BirthdaysReportPage() {
  const { toast } = useToast();
  const [month, setMonth] = useState(new Date().getMonth() + 1);
  const [data, setData] = useState<BirthdaysResult | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async (m: number) => {
    setLoading(true);
    try {
      setData(await getBirthdays(m));
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    load(month);
  }, [load, month]);

  return (
    <div className="page">
      <PageHeader
        title="Aniversariantes"
        description="Aniversarios de nascimento e de casamento"
        actions={
          <div className="flex flex-col items-end gap-2">
            <div className="flex items-center gap-2">
              <span className="text-xs text-zinc-400">Nascimento</span>
              <ExportButtons
                path="/api/v1/reports/birthdays/export"
                params={{ month: String(month), part: "nascimento" }}
                filenameBase={`aniversariantes-nascimento-${new Date().getFullYear()}-${String(month).padStart(2, "0")}`}
              />
            </div>
            <div className="flex items-center gap-2">
              <span className="text-xs text-zinc-400">Casamento</span>
              <ExportButtons
                path="/api/v1/reports/birthdays/export"
                params={{ month: String(month), part: "casamento" }}
                filenameBase={`aniversariantes-casamento-${new Date().getFullYear()}-${String(month).padStart(2, "0")}`}
              />
            </div>
          </div>
        }
      />

      <Card className="mb-6 flex flex-wrap items-end gap-3">
        <div>
          <label className="label">Mes</label>
          <Select className="h-8 w-44 text-sm" value={String(month)} onChange={(e) => setMonth(Number(e.target.value))}>
            {MONTHS.map((m, i) => <option key={i} value={i + 1}>{m}</option>)}
          </Select>
        </div>
      </Card>

      {loading ? (
        <div className="grid gap-4 lg:grid-cols-2">{[0, 1].map((i) => <Skeleton key={i} className="h-64" />)}</div>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          <Card className="overflow-hidden p-0">
            <h3 className="flex items-center gap-2 border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700">
              <Cake className="h-4 w-4 text-sky-600" /> Nascimento ({data?.birthdays.length ?? 0})
            </h3>
            <Table>
              <THead><TRow><TH>Dia</TH><TH>Nome</TH><TH>Nascimento</TH><TH className="text-right">Idade</TH></TRow></THead>
              <TBody>
                {(data?.birthdays ?? []).map((b) => (
                  <TRow key={b.id}>
                    <TD className="font-medium">{b.day}</TD>
                    <TD>{b.full_name}</TD>
                    <TD>{datePt(b.birth_date)}</TD>
                    <TD className="text-right">{b.age}</TD>
                  </TRow>
                ))}
                {(data?.birthdays.length ?? 0) === 0 && <TRow><TD colSpan={4} className="py-8 text-center text-zinc-400">Nenhum aniversariante neste mes.</TD></TRow>}
              </TBody>
            </Table>
          </Card>

          <Card className="overflow-hidden p-0">
            <h3 className="flex items-center gap-2 border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700">
              <Heart className="h-4 w-4 text-pink-500" /> Casamento ({data?.marriages.length ?? 0})
            </h3>
            <Table>
              <THead><TRow><TH>Dia</TH><TH>Casal</TH><TH>Data</TH><TH className="text-right">Anos</TH></TRow></THead>
              <TBody>
                {(data?.marriages ?? []).map((m) => (
                  <TRow key={m.id}>
                    <TD className="font-medium">{m.day}</TD>
                    <TD>{m.full_name}{m.spouse_name ? ` & ${m.spouse_name}` : ""}</TD>
                    <TD>{datePt(m.marriage_date)}</TD>
                    <TD className="text-right">{m.years}</TD>
                  </TRow>
                ))}
                {(data?.marriages.length ?? 0) === 0 && <TRow><TD colSpan={4} className="py-8 text-center text-zinc-400">Nenhum aniversario de casamento neste mes.</TD></TRow>}
              </TBody>
            </Table>
          </Card>
        </div>
      )}
    </div>
  );
}
