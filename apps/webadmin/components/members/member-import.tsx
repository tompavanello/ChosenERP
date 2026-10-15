"use client";

import { useState } from "react";
import { Download, Upload } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input, Select } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Modal } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import {
  previewMembers, importMembersFile,
  type ImportResult,
} from "@/lib/api";
import { downloadCSVTemplate } from "@/lib/template";

// Colunas do modelo (batem com os apelidos do mapeamento automatico) + exemplos.
const TEMPLATE_HEADERS = [
  "Nome", "Sobrenome", "E-mail", "Telefone", "WhatsApp", "Nascimento", "Sexo",
  "Estado civil", "CPF", "RG", "Profissao", "Nacionalidade", "Escolaridade",
  "Situacao", "CEP", "Logradouro", "Numero", "Complemento", "Bairro", "Cidade",
  "UF", "Observacoes",
];
const TEMPLATE_ROWS = [
  [
    "Joao", "da Silva", "joao@exemplo.com", "(11) 98888-7777", "(11) 98888-7777",
    "15/03/1990", "Masculino", "Casado", "123.456.789-09", "12.345.678-9",
    "Professor", "Brasileira", "Superior", "Ativo", "01310-100", "Av. Paulista",
    "1000", "Sala 5", "Bela Vista", "Sao Paulo", "SP", "Batizado em 2021",
  ],
  [
    "Maria", "Souza", "maria@exemplo.com", "", "", "", "Feminino", "Solteiro",
    "", "", "", "", "", "Membro", "", "", "", "", "", "", "", "",
  ],
];

const IMPORT_FIELDS: { key: string; label: string; required?: boolean; aliases: string[] }[] = [
  { key: "full_name", label: "Nome completo", aliases: ["nome completo", "nome_completo", "name"] },
  { key: "first_name", label: "Nome", aliases: ["nome", "first_name"] },
  { key: "last_name", label: "Sobrenome", aliases: ["sobrenome", "last_name"] },
  { key: "email", label: "E-mail", aliases: ["email", "e-mail"] },
  { key: "phone", label: "Telefone", aliases: ["telefone", "fone", "celular", "phone"] },
  { key: "whatsapp", label: "WhatsApp", aliases: ["whatsapp", "zap"] },
  { key: "birth_date", label: "Nascimento", aliases: ["nascimento", "data de nascimento", "birth"] },
  { key: "gender", label: "Sexo", aliases: ["sexo", "genero", "gender"] },
  { key: "marital_status", label: "Estado civil", aliases: ["estado civil", "civil"] },
  { key: "cpf", label: "CPF", aliases: ["cpf"] },
  { key: "rg", label: "RG", aliases: ["rg", "identidade"] },
  { key: "profession", label: "Profissao", aliases: ["profissao", "profissional"] },
  { key: "nationality", label: "Nacionalidade", aliases: ["nacionalidade"] },
  { key: "education", label: "Escolaridade", aliases: ["escolaridade", "instrucao"] },
  { key: "membership_status", label: "Situacao", aliases: ["situacao", "status", "membro"] },
  { key: "address_zip_code", label: "CEP", aliases: ["cep"] },
  { key: "address_street", label: "Logradouro", aliases: ["logradouro", "rua", "endereco"] },
  { key: "address_number", label: "Numero", aliases: ["numero", "num"] },
  { key: "address_complement", label: "Complemento", aliases: ["complemento"] },
  { key: "address_district", label: "Bairro", aliases: ["bairro"] },
  { key: "address_city", label: "Cidade", aliases: ["cidade", "municipio"] },
  { key: "address_state", label: "UF", aliases: ["uf", "estado"] },
  { key: "notes", label: "Observacoes", aliases: ["observac", "notas", "obs"] },
];

function colLetter(i: number): string {
  let s = "";
  i += 1;
  while (i > 0) {
    const r = (i - 1) % 26;
    s = String.fromCharCode(65 + r) + s;
    i = Math.floor((i - 1) / 26);
  }
  return s;
}

function fileToBase64(f: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => {
      const s = String(r.result);
      const i = s.indexOf(",");
      resolve(i >= 0 ? s.slice(i + 1) : s);
    };
    r.onerror = () => reject(new Error("Falha ao ler o arquivo"));
    r.readAsDataURL(f);
  });
}

function normHeader(s: string): string {
  return (s ?? "").toLowerCase().normalize("NFD").replace(/[\u0300-\u036f]/g, "").trim();
}

function guessMapping(rows: string[][]): Record<string, number> {
  if (rows.length === 0) return {};
  const header = rows[0].map(normHeader);
  const m: Record<string, number> = {};
  for (const f of IMPORT_FIELDS) {
    const idx = header.findIndex((h) => f.aliases.some((a) => h.includes(a)));
    if (idx >= 0) m[f.key] = idx;
  }
  return m;
}

export function MemberImport({
  open,
  onClose,
  onImported,
}: {
  open: boolean;
  onClose: () => void;
  onImported: () => void;
}) {
  const { toast } = useToast();
  const [file, setFile] = useState<{ data: string; filename: string } | null>(null);
  const [rows, setRows] = useState<string[][]>([]);
  const [startRow, setStartRow] = useState(1);
  const [map, setMap] = useState<Record<string, number>>({});
  const [importing, setImporting] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);

  function reset() {
    setFile(null);
    setRows([]);
    setStartRow(1);
    setMap({});
    setResult(null);
  }

  function close() {
    reset();
    onClose();
  }

  async function pick(f: File) {
    try {
      const data = await fileToBase64(f);
      setFile({ data, filename: f.name });
      setResult(null);
      const { rows } = await previewMembers(data, f.name);
      setRows(rows);
      setStartRow(rows.length > 1 ? 2 : 1);
      setMap(guessMapping(rows));
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao ler a planilha", "error");
    }
  }

  async function doImport() {
    if (!file) return;
    setImporting(true);
    try {
      const res = await importMembersFile(file.data, file.filename, startRow - 1, map);
      setResult(res);
      if (res.imported > 0) {
        toast(`${res.imported} membro(s) importado(s).`);
        onImported();
      } else {
        toast("Nenhum membro foi importado. Verifique os erros.", "error");
      }
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao importar", "error");
    } finally {
      setImporting(false);
    }
  }

  const maxCols = rows.length > 0 ? Math.max(...rows.map((r) => r.length), 0) : 0;

  return (
    <Modal open={open} onClose={close} title="Importar membros por planilha" size="xl">
      <div className="space-y-3 text-sm">
        <p className="text-zinc-500">
          Envie uma planilha <b>.xlsx</b> ou <b>.csv</b>. A primeira linha costuma ser o cabecalho;
          confira o mapeamento das colunas e a <b>linha inicial dos dados</b> antes de importar.
          O nome (ou nome completo) e obrigatorio; os demais campos sao opcionais.
        </p>

        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            type="button"
            onClick={() => downloadCSVTemplate("modelo-membros.csv", [TEMPLATE_HEADERS, ...TEMPLATE_ROWS])}
          >
            <Download className="h-4 w-4" /> Baixar modelo (CSV)
          </Button>
          <span className="text-xs text-zinc-400">Abra no Excel/Planilhas Google, preencha e salve como .csv ou .xlsx.</span>
        </div>

        <input
          type="file"
          accept=".csv,text/csv,.xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
          className="block w-full text-sm"
          onChange={async (e) => {
            const f = e.target.files?.[0];
            if (f) await pick(f);
            e.target.value = "";
          }}
        />

        {rows.length > 0 && (
          <>
            <div className="rounded border border-zinc-200 p-3 text-xs text-zinc-500 dark:border-zinc-700">
              Informe <b>Nome</b> e <b>Sobrenome</b>, ou apenas <b>Nome completo</b> (sera dividido).
              Datas: use <code>AAAA-MM-DD</code> ou <code>DD/MM/AAAA</code>. Sexo: masculino/feminino.
              Estado civil: solteiro/casado/divorciado/viuvo.
            </div>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              {IMPORT_FIELDS.map((f) => (
                <div key={f.key}>
                  <label className="label">{f.label}{f.required && " *"}</label>
                  <Select
                    className="h-8 text-sm"
                    value={map[f.key] ?? -1}
                    onChange={(e) => setMap({ ...map, [f.key]: Number(e.target.value) })}
                  >
                    <option value={-1}>- nao usar -</option>
                    {Array.from({ length: maxCols }, (_, i) => (
                      <option key={i} value={i}>
                        {colLetter(i)}{rows[0]?.[i] ? ` (${rows[0][i]})` : ""}
                      </option>
                    ))}
                  </Select>
                </div>
              ))}
              <div>
                <label className="label">Linha inicial (dados)</label>
                <Input
                  type="number" min="1" className="h-8 text-sm"
                  value={startRow}
                  onChange={(e) => setStartRow(Math.max(1, Number(e.target.value) || 1))}
                />
              </div>
            </div>

            <div className="max-h-64 overflow-auto rounded border border-zinc-200 dark:border-zinc-700">
              <table className="w-full text-xs">
                <tbody>
                  {rows.slice(0, 30).map((row, ri) => (
                    <tr key={ri} className={ri + 1 === startRow ? "bg-sky-50 dark:bg-sky-950/30" : ""}>
                      <td className="border-b border-zinc-100 px-2 py-1 text-right text-zinc-400 dark:border-zinc-800">{ri + 1}</td>
                      {row.map((cell, ci) => (
                        <td key={ci} className="whitespace-nowrap border-b border-zinc-100 px-2 py-1 dark:border-zinc-800">{cell}</td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}

        {result && (
          <div className="rounded border border-zinc-200 p-3 dark:border-zinc-700">
            <p className="font-medium">
              <Badge tone="green">{result.imported} importado(s)</Badge>{" "}
              {result.skipped > 0 && <Badge tone="amber">{result.skipped} ignorado(s)</Badge>}
            </p>
            {result.errors.length > 0 && (
              <ul className="mt-2 max-h-40 space-y-1 overflow-y-auto text-xs text-red-600">
                {result.errors.map((e, i) => (
                  <li key={i}>Linha {e.line}: {e.error}</li>
                ))}
              </ul>
            )}
          </div>
        )}

        <div className="flex justify-end gap-2 pt-2">
          <Button variant="ghost" type="button" onClick={close}>Fechar</Button>
          {rows.length > 0 && (
            <Button variant="outline" type="button" onClick={reset} disabled={importing}>
              Trocar arquivo
            </Button>
          )}
          <Button type="button" onClick={doImport} disabled={importing || !file}>
            {importing ? "Importando..." : (<><Upload className="h-4 w-4" /> Importar</>)}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
