"use client";

import { useCallback, useEffect, useState } from "react";
import { Award, FileText, Send } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Field, Input, Select, Textarea } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { useToast } from "@/components/ui/toast";
import { listMemberDocuments, issueMemberDocument, openDocument, type MemberDocument } from "@/lib/api";
import { dateTimePt } from "@/lib/format";

const CERT_TYPES = [
  { v: "baptism", l: "Batismo" },
  { v: "marriage", l: "Casamento" },
  { v: "presentation", l: "Apresentação de criança" },
];
const LETTER_TYPES = [
  { v: "transfer", l: "Transferência" },
  { v: "recommendation", l: "Recomendação" },
];

export function SecretariaDocs({ memberId, canWrite }: { memberId: string; canWrite: boolean }) {
  const { toast } = useToast();
  const [docs, setDocs] = useState<MemberDocument[]>([]);
  const [kind, setKind] = useState("certificate");
  const [type, setType] = useState("baptism");
  const [notes, setNotes] = useState("");
  const [city, setCity] = useState("");
  const [spouse, setSpouse] = useState("");
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      setDocs((await listMemberDocuments(memberId)).documents);
    } catch {
      setDocs([]);
    }
  }, [memberId]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    setType(kind === "certificate" ? "baptism" : "transfer");
  }, [kind]);

  const types = kind === "certificate" ? CERT_TYPES : LETTER_TYPES;

  async function emit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const doc = await issueMemberDocument(memberId, { kind, type, notes, city, spouse });
      toast("Documento emitido.");
      setNotes("");
      setSpouse("");
      await load();
      await openDocument(doc.id);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao emitir", "error");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card className="p-3">
      <div className="mb-3 flex items-center gap-2">
        <Award className="h-4 w-4 text-sky-600" />
        <h3 className="text-sm font-semibold">Certidões e cartas</h3>
      </div>

      {canWrite && (
        <form onSubmit={emit} className="mb-4 grid grid-cols-1 gap-3 rounded-lg border border-dashed border-zinc-300 p-3 sm:grid-cols-2 dark:border-zinc-700">
          <Field label="Tipo de documento">
            <Select className="h-8 text-sm" value={kind} onChange={(e) => setKind(e.target.value)}>
              <option value="certificate">Certificado</option>
              <option value="letter">Carta</option>
            </Select>
          </Field>
          <Field label="Modelo">
            <Select className="h-8 text-sm" value={type} onChange={(e) => setType(e.target.value)}>
              {types.map((t) => <option key={t.v} value={t.v}>{t.l}</option>)}
            </Select>
          </Field>
          <Field label="Cidade">
            <Input className="h-8 text-sm" value={city} onChange={(e) => setCity(e.target.value)} />
          </Field>
          {kind === "certificate" && type === "marriage" && (
            <Field label="Cônjuge">
              <Input className="h-8 text-sm" value={spouse} onChange={(e) => setSpouse(e.target.value)} />
            </Field>
          )}
          <Field label="Texto (opcional, deixa em branco para o padrão)" className="sm:col-span-2">
            <Textarea rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} />
          </Field>
          <div className="flex justify-end sm:col-span-2">
            <Button type="submit" className="h-8 text-sm" disabled={saving}>
              <Send className="h-3.5 w-3.5" /> {saving ? "Emitindo..." : "Emitir e abrir"}
            </Button>
          </div>
        </form>
      )}

      {docs.length === 0 ? (
        <p className="py-3 text-center text-sm text-zinc-400">Nenhum documento emitido.</p>
      ) : (
        <ul className="space-y-2 text-sm">
          {docs.map((d) => (
            <li key={d.id} className="flex items-center justify-between gap-2">
              <span className="flex items-center gap-2">
                <FileText className="h-3.5 w-3.5 text-zinc-400" /> {d.title}
                <Badge tone="zinc" className="text-[10px]">{d.document_ref}</Badge>
              </span>
              <span className="flex items-center gap-2">
                <span className="text-xs text-zinc-400">{dateTimePt(d.created_at)}</span>
                <Button variant="ghost" size="sm" onClick={() => openDocument(d.id)}>Abrir</Button>
              </span>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
