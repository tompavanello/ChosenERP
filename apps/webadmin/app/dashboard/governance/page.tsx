"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  CheckCircle2, FileSignature, Gavel, Plus, Scale, ScrollText, ShieldCheck, Trash2, Pencil, Vote as VoteIcon,
} from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Drawer } from "@/components/ui/modal";
import { Field, Input, Select, Textarea } from "@/components/ui/input";
import { Tabs } from "@/components/ui/tabs";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listMinutes, createMinute, updateMinute, deleteMinute, signMinute, listMinuteSignatures,
  listVotes, createVote, updateVote, deleteVote, openVote, closeVote, castBallot, getVoteResult,
  listLegalDocuments, createLegalDocument, updateLegalDocument, deleteLegalDocument,
  listMandates,
  type Minute, type MinuteSignature, type Vote, type VoteResult, type LegalDocument, type Mandate,
} from "@/lib/api";
import { datePt, dateTimePt } from "@/lib/format";

const MINUTE_KINDS = [
  { v: "assembleia", l: "Assembleia geral" },
  { v: "ordinaria", l: "Reuniao ordinaria" },
  { v: "extraordinaria", l: "Reuniao extraordinaria" },
  { v: "diretoria", l: "Reuniao de diretoria" },
  { v: "reuniao", l: "Reuniao" },
  { v: "outra", l: "Outra" },
];
const VOTE_KINDS = [
  { v: "assembleia", l: "Assembleia" },
  { v: "diretoria", l: "Diretoria" },
  { v: "orcamento", l: "Orcamento" },
  { v: "mocao", l: "Mocao" },
  { v: "eleicao", l: "Eleicao" },
  { v: "outra", l: "Outra" },
];
const LEGAL_KINDS = [
  { v: "escritura", l: "Escritura" },
  { v: "alvara", l: "Alvara" },
  { v: "contrato", l: "Contrato" },
  { v: "seguro", l: "Seguro" },
  { v: "convenio", l: "Convenio" },
  { v: "certidao", l: "Certidao" },
  { v: "outro", l: "Outro" },
];

const localNow = () => {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
};

const minuteStatusTone = (s: string) =>
  s === "assinada" ? "green" : s === "aprovada" ? "sky" : s === "cancelada" ? "red" : "zinc";
const voteStatusTone = (s: string) =>
  s === "aberta" ? "green" : s === "encerrada" ? "sky" : s === "cancelada" ? "red" : "zinc";

export default function GovernancePage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("governance.write");

  const [tab, setTab] = useState("atas");
  const [minutes, setMinutes] = useState<Minute[] | null>(null);
  const [votes, setVotes] = useState<Vote[] | null>(null);
  const [legal, setLegal] = useState<LegalDocument[] | null>(null);
  const [mandates, setMandates] = useState<Mandate[] | null>(null);

  const [minuteDrawer, setMinuteDrawer] = useState<{ open: boolean; editing?: Minute }>({ open: false });
  const [minuteForm, setMinuteForm] = useState({
    title: "", meeting_at: localNow(), kind: "assembleia", quorum_required: "0",
    attendance_count: "0", body: "",
  });
  const [signatures, setSignatures] = useState<MinuteSignature[] | null>(null);
  const [signatureFor, setSignatureFor] = useState<Minute | null>(null);
  const [saving, setSaving] = useState(false);

  const [voteDrawer, setVoteDrawer] = useState<{ open: boolean; editing?: Vote }>({ open: false });
  const [voteForm, setVoteForm] = useState({
    title: "", kind: "assembleia", minute_id: "", secret: "true",
    quorum_required: "0", min_attendance: "0", description: "", options: "Sim\nNao\nAbstencao",
  });

  const [legalDrawer, setLegalDrawer] = useState<{ open: boolean; editing?: LegalDocument }>({ open: false });
  const [legalForm, setLegalForm] = useState({
    kind: "convenio", title: "", reference: "", issued_at: "", expires_at: "", description: "",
  });

  const [resultFor, setResultFor] = useState<{ vote: Vote; result?: VoteResult } | null>(null);

  const load = useCallback(async () => {
    try {
      const [m, v, l, md] = await Promise.all([
        listMinutes(), listVotes(), listLegalDocuments(), listMandates(),
      ]);
      setMinutes(m.minutes);
      setVotes(v.votes);
      setLegal(l.documents);
      setMandates(md.mandates);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar governanca", "error");
      setMinutes([]); setVotes([]); setLegal([]); setMandates([]);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  // ---- Atas ----
  function openMinuteCreate() {
    setMinuteForm({ title: "", meeting_at: localNow(), kind: "assembleia", quorum_required: "0", attendance_count: "0", body: "" });
    setMinuteDrawer({ open: true });
  }
  function openMinuteEdit(m: Minute) {
    setMinuteForm({
      title: m.title,
      meeting_at: m.meeting_at.slice(0, 16),
      kind: m.kind,
      quorum_required: String(m.quorum_required),
      attendance_count: String(m.attendance_count),
      body: m.body ?? "",
    });
    setMinuteDrawer({ open: true, editing: m });
  }
  async function saveMinute(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const payload = {
        title: minuteForm.title,
        meeting_at: minuteForm.meeting_at,
        kind: minuteForm.kind,
        quorum_required: Number(minuteForm.quorum_required) || 0,
        attendance_count: Number(minuteForm.attendance_count) || 0,
        body: minuteForm.body,
      };
      if (minuteDrawer.editing) await updateMinute(minuteDrawer.editing.id, payload);
      else await createMinute(payload);
      toast("Ata salva.");
      setMinuteDrawer({ open: false });
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar ata", "error");
    } finally {
      setSaving(false);
    }
  }
  async function removeMinute(m: Minute) {
    if (!confirm(`Excluir a ata "${m.title}"?`)) return;
    try { await deleteMinute(m.id); toast("Ata excluida."); await load(); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function doSign(m: Minute) {
    if (!confirm(`Assinar internamente a ata "${m.title}"? Depois de assinada ela nao pode mais ser editada.`)) return;
    try { await signMinute(m.id); toast("Ata assinada."); await load(); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro ao assinar", "error"); }
  }
  async function viewSignatures(m: Minute) {
    setSignatureFor(m);
    setSignatures(null);
    try {
      const r = await listMinuteSignatures(m.id);
      setSignatures(r.signatures);
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); setSignatures([]); }
  }

  // ---- Votacoes ----
  function openVoteCreate() {
    setVoteForm({ title: "", kind: "assembleia", minute_id: "", secret: "true", quorum_required: "0", min_attendance: "0", description: "", options: "Sim\nNao\nAbstencao" });
    setVoteDrawer({ open: true });
  }
  function openVoteEdit(v: Vote) {
    setVoteForm({
      title: v.title, kind: v.kind, minute_id: v.minute_id ?? "", secret: v.secret ? "true" : "false",
      quorum_required: String(v.quorum_required), min_attendance: String(v.min_attendance),
      description: v.description ?? "",
      options: (v.options ?? []).map((o) => o.label).join("\n"),
    });
    setVoteDrawer({ open: true, editing: v });
  }
  async function saveVote(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const options = voteForm.options.split("\n").map((s) => s.trim()).filter(Boolean);
      const payload: Record<string, unknown> = {
        title: voteForm.title,
        kind: voteForm.kind,
        minute_id: voteForm.minute_id || undefined,
        secret: voteForm.secret === "true",
        quorum_required: Number(voteForm.quorum_required) || 0,
        min_attendance: Number(voteForm.min_attendance) || 0,
        description: voteForm.description || undefined,
        options,
      };
      if (voteDrawer.editing) await updateVote(voteDrawer.editing.id, payload);
      else await createVote(payload);
      toast("Votacao salva.");
      setVoteDrawer({ open: false });
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar votacao", "error");
    } finally {
      setSaving(false);
    }
  }
  async function doOpen(v: Vote) {
    try { await openVote(v.id); toast("Votacao aberta."); await load(); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function doClose(v: Vote) {
    if (!confirm(`Encerrar e apurar a votacao "${v.title}"?`)) return;
    try {
      const closed = await closeVote(v.id);
      toast("Votacao encerrada e apurada.");
      await load();
      setResultFor({ vote: closed, result: closed.result });
    } catch (err) { toast(err instanceof Error ? err.message : "Erro ao encerrar", "error"); }
  }
  async function doBallot(v: Vote, optionId: string) {
    try {
      const res = await castBallot(v.id, optionId);
      toast("Voto registrado em segredo.");
      await load();
      setResultFor({ vote: v, result: res });
    } catch (err) { toast(err instanceof Error ? err.message : "Erro ao votar", "error"); }
  }
  async function showResult(v: Vote) {
    try {
      const res = await getVoteResult(v.id);
      setResultFor({ vote: v, result: res });
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function removeVote(v: Vote) {
    if (!confirm(`Excluir a votacao "${v.title}"?`)) return;
    try { await deleteVote(v.id); toast("Votacao excluida."); await load(); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }

  // ---- Convenios ----
  function openLegalCreate() {
    setLegalForm({ kind: "convenio", title: "", reference: "", issued_at: "", expires_at: "", description: "" });
    setLegalDrawer({ open: true });
  }
  function openLegalEdit(d: LegalDocument) {
    setLegalForm({
      kind: d.kind, title: d.title, reference: d.reference ?? "",
      issued_at: d.issued_at ?? "", expires_at: d.expires_at ?? "", description: d.description ?? "",
    });
    setLegalDrawer({ open: true, editing: d });
  }
  async function saveLegal(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const payload: Record<string, unknown> = {
        kind: legalForm.kind, title: legalForm.title,
        reference: legalForm.reference || undefined,
        issued_at: legalForm.issued_at || undefined,
        expires_at: legalForm.expires_at || undefined,
        description: legalForm.description || undefined,
      };
      if (legalDrawer.editing) await updateLegalDocument(legalDrawer.editing.id, payload);
      else await createLegalDocument(payload);
      toast("Documento salvo.");
      setLegalDrawer({ open: false });
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar documento", "error");
    } finally {
      setSaving(false);
    }
  }
  async function removeLegal(d: LegalDocument) {
    if (!confirm(`Excluir o documento "${d.title}"?`)) return;
    try { await deleteLegalDocument(d.id); toast("Documento excluido."); await load(); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }

  const expiringMandates = useMemo(
    () => (mandates ?? []).filter((m) => m.status === "ativo" && m.expiring),
    [mandates],
  );
  const expiringLegal = useMemo(
    () => (legal ?? []).filter((d) => d.expires_at && (d.expired || (d.days_to_expiry ?? 999) <= 60)),
    [legal],
  );

  return (
    <div className="page">
      <PageHeader
        title="Governanca"
        description="Atas digitais, votacoes com quorum e voto secreto, mandatos e convenios"
        actions={canWrite ? (
          <Button onClick={tab === "atas" ? openMinuteCreate : tab === "votacoes" ? openVoteCreate : tab === "convenios" ? openLegalCreate : undefined}>
            <Plus className="h-4 w-4" /> Novo
          </Button>
        ) : undefined}
      />

      <Tabs
        tabs={[
          { key: "atas", label: "Atas", icon: <ScrollText className="h-4 w-4" /> },
          { key: "votacoes", label: "Votacoes", icon: <VoteIcon className="h-4 w-4" /> },
          { key: "mandatos", label: "Mandatos", icon: <Gavel className="h-4 w-4" /> },
          { key: "convenios", label: "Convenios", icon: <ShieldCheck className="h-4 w-4" /> },
        ]}
        active={tab}
        onChange={setTab}
      />

      {tab === "atas" && (
        <Card className="overflow-hidden p-0">
          {minutes === null ? (
            <div className="p-4"><SkeletonRows rows={5} /></div>
          ) : minutes.length === 0 ? (
            <EmptyState icon={<ScrollText className="h-10 w-10" />} title="Nenhuma ata" description="Registre a pauta e as deliberacoes das assembleias e reunioes." />
          ) : (
            <Table>
              <THead><TRow><TH>Data</TH><TH>Titulo</TH><TH>Tipo</TH><TH>Quorum</TH><TH>Assinaturas</TH><TH>Situacao</TH><TH className="text-right">Acoes</TH></TRow></THead>
              <TBody>
                {minutes.map((m) => (
                  <TRow key={m.id}>
                    <TD className="whitespace-nowrap text-sm">{dateTimePt(m.meeting_at)}</TD>
                    <TD className="font-medium">
                      {m.title}
                      {m.vote_count > 0 && <span className="ml-2 text-xs text-zinc-400">{m.vote_count} votacao(oes)</span>}
                    </TD>
                    <TD><Badge tone="zinc">{MINUTE_KINDS.find((k) => k.v === m.kind)?.l ?? m.kind}</Badge></TD>
                    <TD className="text-sm tabular-nums">{m.attendance_count}/{m.quorum_required || "-"}</TD>
                    <TD>
                      <button type="button" className="text-sm text-sky-600 hover:underline" onClick={() => viewSignatures(m)}>
                        {m.signature_count}
                      </button>
                    </TD>
                    <TD><Badge tone={minuteStatusTone(m.status)}>{m.status}</Badge></TD>
                    <TD>
                      <div className="flex justify-end gap-1">
                        {canWrite && m.status !== "assinada" && <Button variant="ghost" className="h-8 px-2" title="Assinar" onClick={() => doSign(m)}><FileSignature className="h-4 w-4" /></Button>}
                        {canWrite && m.status !== "assinada" && <Button variant="ghost" className="h-8 px-2" title="Editar" onClick={() => openMinuteEdit(m)}><Pencil className="h-4 w-4" /></Button>}
                        {canWrite && m.status !== "assinada" && <Button variant="ghost" className="h-8 px-2" title="Excluir" onClick={() => removeMinute(m)}><Trash2 className="h-4 w-4" /></Button>}
                      </div>
                    </TD>
                  </TRow>
                ))}
              </TBody>
            </Table>
          )}
        </Card>
      )}

      {tab === "votacoes" && (
        <Card className="overflow-hidden p-0">
          {votes === null ? (
            <div className="p-4"><SkeletonRows rows={5} /></div>
          ) : votes.length === 0 ? (
            <EmptyState icon={<VoteIcon className="h-10 w-10" />} title="Nenhuma votacao" description="Crie uma votacao com quorum e voto secreto para a assembleia." />
          ) : (
            <Table>
              <THead><TRow><TH>Titulo</TH><TH>Tipo</TH><TH>Ata</TH><TH>Quorum</TH><TH>Votos</TH><TH>Situacao</TH><TH className="text-right">Acoes</TH></TRow></THead>
              <TBody>
                {votes.map((v) => (
                  <TRow key={v.id}>
                    <TD className="font-medium">
                      {v.title}
                      {v.secret && <span title="Voto secreto"><ShieldCheck className="ml-1 inline h-3.5 w-3.5 text-emerald-600" /></span>}
                    </TD>
                    <TD><Badge tone="zinc">{VOTE_KINDS.find((k) => k.v === v.kind)?.l ?? v.kind}</Badge></TD>
                    <TD className="text-sm text-zinc-500">{v.minute_title ?? "-"}</TD>
                    <TD className={`text-sm tabular-nums ${v.quorum_met ? "text-emerald-600" : "text-amber-600"}`}>
                      {v.participant_count}/{v.quorum_required || "-"}
                    </TD>
                    <TD className="text-sm tabular-nums">{v.ballot_count}</TD>
                    <TD><Badge tone={voteStatusTone(v.status)}>{v.status}</Badge></TD>
                    <TD>
                      <div className="flex justify-end gap-1">
                        {canWrite && v.status === "rascunho" && <Button variant="ghost" className="h-8 px-2" title="Abrir" onClick={() => doOpen(v)}><CheckCircle2 className="h-4 w-4" /></Button>}
                        {v.status === "aberta" && <Button variant="ghost" className="h-8 px-2" title="Votar" onClick={() => openVoteEdit(v)}><VoteIcon className="h-4 w-4" /></Button>}
                        {canWrite && (v.status === "aberta" || v.status === "rascunho") && <Button variant="ghost" className="h-8 px-2" title="Encerrar e apurar" onClick={() => doClose(v)}><Scale className="h-4 w-4" /></Button>}
                        {v.status !== "rascunho" && <Button variant="ghost" className="h-8 px-2" title="Resultado" onClick={() => showResult(v)}><Scale className="h-4 w-4" /></Button>}
                        {canWrite && v.status === "rascunho" && <Button variant="ghost" className="h-8 px-2" title="Editar" onClick={() => openVoteEdit(v)}><Pencil className="h-4 w-4" /></Button>}
                        {canWrite && v.status === "rascunho" && <Button variant="ghost" className="h-8 px-2" title="Excluir" onClick={() => removeVote(v)}><Trash2 className="h-4 w-4" /></Button>}
                      </div>
                    </TD>
                  </TRow>
                ))}
              </TBody>
            </Table>
          )}
        </Card>
      )}

      {tab === "mandatos" && (
        <>
          {expiringMandates.length > 0 && (
            <Card className="mb-4 border-amber-300 bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-950/30 dark:text-amber-300">
              <Gavel className="mr-1 inline h-4 w-4" /> {expiringMandates.length} mandato(s) vencendo nos proximos 60 dias.
            </Card>
          )}
          <Card className="overflow-hidden p-0">
            {mandates === null ? (
              <div className="p-4"><SkeletonRows rows={5} /></div>
            ) : mandates.length === 0 ? (
              <EmptyState icon={<Gavel className="h-10 w-10" />} title="Nenhum mandato" description="Os mandatos vem dos cargos atribuidos aos membros." />
            ) : (
              <Table>
                <THead><TRow><TH>Membro</TH><TH>Cargo</TH><TH>Inicio</TH><TH>Vencimento</TH><TH>Dias p/ vencer</TH><TH>Situacao</TH></TRow></THead>
                <TBody>
                  {mandates.map((m) => (
                    <TRow key={m.id}>
                      <TD className="font-medium">{m.member_name}</TD>
                      <TD>{m.cargo_name}</TD>
                      <TD className="text-sm text-zinc-500">{m.started_at ? datePt(m.started_at) : "-"}</TD>
                      <TD className="text-sm text-zinc-500">{m.ends_at ? datePt(m.ends_at) : "-"}</TD>
                      <TD className="text-sm tabular-nums">
                        {m.days_to_expiry == null ? "-" : (
                          <span className={m.days_to_expiry < 0 ? "text-red-600" : m.expiring ? "text-amber-600" : ""}>
                            {m.days_to_expiry < 0 ? `${Math.abs(m.days_to_expiry)} atras` : m.days_to_expiry}
                          </span>
                        )}
                      </TD>
                      <TD><Badge tone={m.status === "ativo" ? (m.expiring ? "amber" : "green") : "zinc"}>{m.expiring && m.status === "ativo" ? "vencendo" : m.status}</Badge></TD>
                    </TRow>
                  ))}
                </TBody>
              </Table>
            )}
          </Card>
        </>
      )}

      {tab === "convenios" && (
        <>
          {expiringLegal.length > 0 && (
            <Card className="mb-4 border-amber-300 bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-950/30 dark:text-amber-300">
              <ShieldCheck className="mr-1 inline h-4 w-4" /> {expiringLegal.length} documento(s) vencido(s) ou vencendo nos proximos 60 dias.
            </Card>
          )}
          <Card className="overflow-hidden p-0">
            {legal === null ? (
              <div className="p-4"><SkeletonRows rows={5} /></div>
            ) : legal.length === 0 ? (
              <EmptyState icon={<ShieldCheck className="h-10 w-10" />} title="Nenhum documento" description="Cadastre escrituras, alvaras, contratos e seguros com alerta de vencimento." />
            ) : (
              <Table>
                <THead><TRow><TH>Titulo</TH><TH>Tipo</TH><TH>Referencia</TH><TH>Vencimento</TH><TH>Situacao</TH><TH className="text-right">Acoes</TH></TRow></THead>
                <TBody>
                  {legal.map((d) => (
                    <TRow key={d.id}>
                      <TD className="font-medium">{d.title}</TD>
                      <TD><Badge tone="zinc">{LEGAL_KINDS.find((k) => k.v === d.kind)?.l ?? d.kind}</Badge></TD>
                      <TD className="text-sm text-zinc-500">{d.reference ?? "-"}</TD>
                      <TD className="text-sm text-zinc-500">{d.expires_at ? datePt(d.expires_at) : "-"}</TD>
                      <TD>
                        {!d.expires_at ? <Badge tone="zinc">sem prazo</Badge>
                          : d.expired ? <Badge tone="red">vencido</Badge>
                          : (d.days_to_expiry ?? 999) <= 60 ? <Badge tone="amber">vence em {d.days_to_expiry}d</Badge>
                          : <Badge tone="green">valido</Badge>}
                      </TD>
                      <TD>
                        <div className="flex justify-end gap-1">
                          {canWrite && <Button variant="ghost" className="h-8 px-2" title="Editar" onClick={() => openLegalEdit(d)}><Pencil className="h-4 w-4" /></Button>}
                          {canWrite && <Button variant="ghost" className="h-8 px-2" title="Excluir" onClick={() => removeLegal(d)}><Trash2 className="h-4 w-4" /></Button>}
                        </div>
                      </TD>
                    </TRow>
                  ))}
                </TBody>
              </Table>
            )}
          </Card>
        </>
      )}

      {/* Drawer da ata */}
      <Drawer open={minuteDrawer.open} onClose={() => setMinuteDrawer({ open: false })} size="2xl" title={minuteDrawer.editing ? "Editar ata" : "Nova ata"}>
        <form onSubmit={saveMinute} className="space-y-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <Field label="Titulo" required className="sm:col-span-2">
              <Input required className="h-8 text-sm" value={minuteForm.title} onChange={(e) => setMinuteForm({ ...minuteForm, title: e.target.value })} />
            </Field>
            <Field label="Data e hora" required>
              <Input required type="datetime-local" className="h-8 text-sm" value={minuteForm.meeting_at} onChange={(e) => setMinuteForm({ ...minuteForm, meeting_at: e.target.value })} />
            </Field>
            <Field label="Tipo">
              <Select className="h-8 text-sm" value={minuteForm.kind} onChange={(e) => setMinuteForm({ ...minuteForm, kind: e.target.value })}>
                {MINUTE_KINDS.map((k) => <option key={k.v} value={k.v}>{k.l}</option>)}
              </Select>
            </Field>
            <Field label="Presentes" hint="Total declarado de presentes.">
              <Input type="number" min="0" className="h-8 text-sm" value={minuteForm.attendance_count} onChange={(e) => setMinuteForm({ ...minuteForm, attendance_count: e.target.value })} />
            </Field>
            <Field label="Quorum exigido" hint="0 = nao exigido.">
              <Input type="number" min="0" className="h-8 text-sm" value={minuteForm.quorum_required} onChange={(e) => setMinuteForm({ ...minuteForm, quorum_required: e.target.value })} />
            </Field>
            <Field label="Pauta e deliberacoes" className="sm:col-span-2">
              <Textarea rows={10} className="text-sm" value={minuteForm.body} onChange={(e) => setMinuteForm({ ...minuteForm, body: e.target.value })} />
            </Field>
          </div>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setMinuteDrawer({ open: false })}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>

      {/* Drawer de assinaturas */}
      <Drawer open={signatureFor !== null} onClose={() => setSignatureFor(null)} title={`Assinaturas - ${signatureFor?.title ?? ""}`}>
        {signatures === null ? <SkeletonRows rows={3} /> : signatures.length === 0 ? (
          <p className="text-sm text-zinc-500">Ainda nao ha assinaturas. A ata e congelada na primeira assinatura.</p>
        ) : (
          <ul className="space-y-3">
            {signatures.map((s) => (
              <li key={s.id} className="rounded border border-zinc-200 p-3 text-sm dark:border-zinc-800">
                <p className="font-medium">{s.signer_name} <span className="text-zinc-400">- {s.signer_role}</span></p>
                <p className="text-xs text-zinc-500">{dateTimePt(s.signed_at)}</p>
                <p className="mt-1 truncate font-mono text-[11px] text-zinc-400" title={s.hash}>hash: {s.hash}</p>
              </li>
            ))}
          </ul>
        )}
      </Drawer>

      {/* Drawer da votacao */}
      <Drawer open={voteDrawer.open} onClose={() => setVoteDrawer({ open: false })} size="2xl" title={voteDrawer.editing ? "Votacao" : "Nova votacao"}>
        {voteDrawer.editing && voteDrawer.editing.status === "aberta" && voteDrawer.editing.options && voteDrawer.editing.options.length > 0 ? (
          <div className="space-y-4">
            <p className="text-sm text-zinc-500">
              Quorum: {voteDrawer.editing.participant_count}/{voteDrawer.editing.quorum_required || "-"} - o voto e registrado em segredo.
            </p>
            <div className="space-y-2">
              {voteDrawer.editing.options.map((o) => (
                <Button key={o.id} variant="outline" className="w-full justify-start" onClick={() => doBallot(voteDrawer.editing!, o.id)}>
                  <VoteIcon className="h-4 w-4" /> {o.label}
                </Button>
              ))}
            </div>
          </div>
        ) : (
          <form onSubmit={saveVote} className="space-y-4">
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <Field label="Titulo" required className="sm:col-span-2">
                <Input required className="h-8 text-sm" value={voteForm.title} onChange={(e) => setVoteForm({ ...voteForm, title: e.target.value })} />
              </Field>
              <Field label="Tipo">
                <Select className="h-8 text-sm" value={voteForm.kind} onChange={(e) => setVoteForm({ ...voteForm, kind: e.target.value })}>
                  {VOTE_KINDS.map((k) => <option key={k.v} value={k.v}>{k.l}</option>)}
                </Select>
              </Field>
              <Field label="Ata vinculada">
                <Select className="h-8 text-sm" value={voteForm.minute_id} onChange={(e) => setVoteForm({ ...voteForm, minute_id: e.target.value })}>
                  <option value="">-</option>
                  {(minutes ?? []).map((m) => <option key={m.id} value={m.id}>{datePt(m.meeting_at)} - {m.title}</option>)}
                </Select>
              </Field>
              <Field label="Quorum obrigatorio" hint="Minimo de participantes para a votacao valer.">
                <Input type="number" min="0" className="h-8 text-sm" value={voteForm.quorum_required} onChange={(e) => setVoteForm({ ...voteForm, quorum_required: e.target.value })} />
              </Field>
              <Field label="Presenca minima">
                <Input type="number" min="0" className="h-8 text-sm" value={voteForm.min_attendance} onChange={(e) => setVoteForm({ ...voteForm, min_attendance: e.target.value })} />
              </Field>
              <Field label="Voto secreto">
                <Select className="h-8 text-sm" value={voteForm.secret} onChange={(e) => setVoteForm({ ...voteForm, secret: e.target.value })}>
                  <option value="true">Sim (recomendado)</option>
                  <option value="false">Nao</option>
                </Select>
              </Field>
              <Field label="Descricao" className="sm:col-span-2">
                <Textarea rows={2} className="text-sm" value={voteForm.description} onChange={(e) => setVoteForm({ ...voteForm, description: e.target.value })} />
              </Field>
              <Field label="Opcoes" hint="Uma por linha. So podem ser alteradas antes de haver votos." className="sm:col-span-2">
                <Textarea rows={4} className="text-sm" value={voteForm.options} onChange={(e) => setVoteForm({ ...voteForm, options: e.target.value })} />
              </Field>
            </div>
            <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
              <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setVoteDrawer({ open: false })}>Cancelar</Button>
              <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
            </div>
          </form>
        )}
      </Drawer>

      {/* Drawer de resultado */}
      <Drawer open={resultFor !== null} onClose={() => setResultFor(null)} title={`Apuracao - ${resultFor?.vote.title ?? ""}`}>
        {resultFor?.result ? (
          <div className="space-y-4">
            <div className={`rounded border p-3 text-sm ${resultFor.result.quorum_met ? "border-emerald-300 bg-emerald-50 text-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-300" : "border-amber-300 bg-amber-50 text-amber-800 dark:bg-amber-950/30 dark:text-amber-300"}`}>
              {resultFor.result.quorum_met ? "Quorum atingido." : "Quorum NAO atingido."} {resultFor.result.participants} participante(s) de {resultFor.result.quorum_required || "-"} exigido(s).
            </div>
            <Table>
              <THead><TRow><TH>Opcao</TH><TH className="text-right">Votos</TH></TRow></THead>
              <TBody>
                {resultFor.result.options.map((o) => (
                  <TRow key={o.id}>
                    <TD className={o.label === resultFor.result!.winner ? "font-semibold" : ""}>{o.label}</TD>
                    <TD className="text-right tabular-nums">{o.votes}</TD>
                  </TRow>
                ))}
              </TBody>
            </Table>
            <p className="text-xs text-zinc-400">Total de votos: {resultFor.result.total}. O resultado so traz contagens - o vinculo votoeleitor nao e armazenado.</p>
          </div>
        ) : <SkeletonRows rows={3} />}
      </Drawer>

      {/* Drawer de convenio */}
      <Drawer open={legalDrawer.open} onClose={() => setLegalDrawer({ open: false })} title={legalDrawer.editing ? "Editar documento" : "Novo documento legal"}>
        <form onSubmit={saveLegal} className="space-y-3">
          <Field label="Titulo" required>
            <Input required className="h-8 text-sm" value={legalForm.title} onChange={(e) => setLegalForm({ ...legalForm, title: e.target.value })} />
          </Field>
          <Field label="Tipo">
            <Select className="h-8 text-sm" value={legalForm.kind} onChange={(e) => setLegalForm({ ...legalForm, kind: e.target.value })}>
              {LEGAL_KINDS.map((k) => <option key={k.v} value={k.v}>{k.l}</option>)}
            </Select>
          </Field>
          <Field label="Referencia" hint="No do documento/processo.">
            <Input className="h-8 text-sm" value={legalForm.reference} onChange={(e) => setLegalForm({ ...legalForm, reference: e.target.value })} />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Emissao">
              <Input type="date" className="h-8 text-sm" value={legalForm.issued_at} onChange={(e) => setLegalForm({ ...legalForm, issued_at: e.target.value })} />
            </Field>
            <Field label="Vencimento" hint="Gera alerta na tela.">
              <Input type="date" className="h-8 text-sm" value={legalForm.expires_at} onChange={(e) => setLegalForm({ ...legalForm, expires_at: e.target.value })} />
            </Field>
          </div>
          <Field label="Descricao">
            <Textarea rows={3} className="text-sm" value={legalForm.description} onChange={(e) => setLegalForm({ ...legalForm, description: e.target.value })} />
          </Field>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setLegalDrawer({ open: false })}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
