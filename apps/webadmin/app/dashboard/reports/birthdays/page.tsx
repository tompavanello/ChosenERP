"use client";

import { useCallback, useEffect, useState } from "react";
import { Cake, ChevronDown, Heart } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Select } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getBirthdays, type BirthdaysResult } from "@/lib/api";
import { ExportMenu } from "@/components/reports/export-menu";

const MONTHS = [
  "Janeiro", "Fevereiro", "Marco", "Abril", "Maio", "Junho",
  "Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro",
];

const pad = (n: number) => String(n).padStart(2, "0");

/** Data no formato dd/mm (o mes ja e o filtro; o ano nao e exibido). */
function dayMonth(day: number, month: number) {
  return `${pad(day)}/${pad(month)}`;
}

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

  const birthdays = data?.birthdays ?? [];
  const marriages = data?.marriages ?? [];

  return (
    <div className="page">
      <PageHeader
        title="Aniversariantes"
        description="Aniversarios de nascimento e de casamento"
      />

      <Card className="mb-4 flex flex-wrap items-end gap-3 p-3">
        <div>
          <label className="label">Mes</label>
          <Select className="h-8 w-44 text-sm" value={String(month)} onChange={(e) => setMonth(Number(e.target.value))}>
            {MONTHS.map((m, i) => <option key={i} value={i + 1}>{m}</option>)}
          </Select>
        </div>
        <p className="ml-auto text-xs text-zinc-400">
          Clique no titulo de cada quadro para imprimir ou exportar somente aquele tipo.
        </p>
      </Card>

      {loading ? (
        <div className="grid gap-4 lg:grid-cols-2">{[0, 1].map((i) => <Skeleton key={i} className="h-64" />)}</div>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          <Card className="overflow-hidden p-0">
            <div className="flex items-center justify-between gap-2 border-b border-zinc-100 px-2 py-1.5 dark:border-zinc-800">
              <ExportMenu
                path="/api/v1/reports/birthdays/export"
                params={{ month: String(month), part: "nascimento" }}
                filenameBase={`aniversariantes-nascimento-${pad(month)}`}
                trigger={
                  <Button variant="ghost" className="h-8 gap-2 px-2 text-sm font-semibold">
                    <Cake className="h-4 w-4 text-sky-600" />
                    Nascimento ({birthdays.length})
                    <ChevronDown className="h-3.5 w-3.5 text-zinc-400" />
                  </Button>
                }
              />
            </div>
            <Table>
              <THead><TRow><TH>Dia/Mês</TH><TH>Nome</TH><TH className="text-right">Idade</TH></TRow></THead>
              <TBody>
                {birthdays.map((b) => (
                  <TRow key={b.id}>
                    <TD className="font-medium tabular-nums">{dayMonth(b.day, month)}</TD>
                    <TD>{b.full_name}</TD>
                    <TD className="text-right">{b.age}</TD>
                  </TRow>
                ))}
                {birthdays.length === 0 && <TRow><TD colSpan={3} className="py-8 text-center text-zinc-400">Nenhum aniversariante neste mes.</TD></TRow>}
              </TBody>
            </Table>
          </Card>

          <Card className="overflow-hidden p-0">
            <div className="flex items-center justify-between gap-2 border-b border-zinc-100 px-2 py-1.5 dark:border-zinc-800">
              <ExportMenu
                path="/api/v1/reports/birthdays/export"
                params={{ month: String(month), part: "casamento" }}
                filenameBase={`aniversariantes-casamento-${pad(month)}`}
                trigger={
                  <Button variant="ghost" className="h-8 gap-2 px-2 text-sm font-semibold">
                    <Heart className="h-4 w-4 text-pink-500" />
                    Casamento ({marriages.length})
                    <ChevronDown className="h-3.5 w-3.5 text-zinc-400" />
                  </Button>
                }
              />
            </div>
            <Table>
              <THead><TRow><TH>Dia/Mês</TH><TH>Casal</TH><TH className="text-right">Anos</TH></TRow></THead>
              <TBody>
                {marriages.map((m) => (
                  <TRow key={m.id}>
                    <TD className="font-medium tabular-nums">{dayMonth(m.day, month)}</TD>
                    <TD>{m.full_name}{m.spouse_name ? ` & ${m.spouse_name}` : ""}</TD>
                    <TD className="text-right">{m.years}</TD>
                  </TRow>
                ))}
                {marriages.length === 0 && <TRow><TD colSpan={3} className="py-8 text-center text-zinc-400">Nenhum aniversario de casamento neste mes.</TD></TRow>}
              </TBody>
            </Table>
          </Card>
        </div>
      )}
    </div>
  );
}
