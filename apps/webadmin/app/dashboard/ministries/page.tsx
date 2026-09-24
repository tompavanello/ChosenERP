"use client";

import { useEffect, useState } from "react";
import { Plus, Users, Church, Users2, CheckSquare, Trash2, Pencil } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field, Select } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { Drawer, Modal } from "@/components/ui/modal";
import { Tabs } from "@/components/ui/tabs";
import { Combobox } from "@/components/ui/combobox";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listMinistries, createMinistry, updateMinistry, deleteMinistry,
  listMinistryMembers, addMinistryMember, removeMinistryMember,
  listGroups, createGroup, updateGroup, deleteGroup,
  checkIn, listAttendance,
  listMembers, type Ministry, type SmallGroup, type MinistryMember, type AttendanceCheckin, type Member,
} from "@/lib/api";
import { dateTimePt } from "@/lib/format";

const GROUP_KIND: Record<string, string> = { cell: "Celula", ebd: "EBD", family: "Familia" };
const WEEKDAYS = ["Domingo", "Segunda", "Terca", "Quarta", "Quinta", "Sexta", "Sabado"];

const EMPTY_MINISTRY = { name: "", description: "", leader_id: "", is_active: "true" };
const EMPTY_GROUP = { name: "", kind: "cell", ministry_id: "", leader_id: "", address: "", max_members: "", weekday: "", meeting_time: "", is_active: "true" };

export default function MinistriesPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("ministries.write");
  const [tab, setTab] = useState("ministries");
  const [ministries, setMinistries] = useState<Ministry[] | null>(null);
  const [groups, setGroups] = useState<SmallGroup[] | null>(null);
  const [members, setMembers] = useState<Member[]>([]);

  const [ministryDrawer, setMinistryDrawer] = useState<{ open: boolean; editing?: Ministry }>({ open: false });
  const [mForm, setMForm] = useState({ ...EMPTY_MINISTRY });

  const [groupDrawer, setGroupDrawer] = useState<{ open: boolean; editing?: SmallGroup }>({ open: false });
  const [gForm, setGForm] = useState({ ...EMPTY_GROUP });

  const [volunteers, setVolunteers] = useState<Ministry | null>(null);
  const [volList, setVolList] = useState<MinistryMember[]>([]);
  const [selectedVolunteers, setSelectedVolunteers] = useState<string[]>([]);
  const [volForm, setVolForm] = useState({ role: "volunteer" });

  const [checkin, setCheckin] = useState<SmallGroup | null>(null);
  const [ciForm, setCiForm] = useState({ member_id: "", present: true });
  const [attendance, setAttendance] = useState<AttendanceCheckin[]>([]);
  const [attGroup, setAttGroup] = useState<SmallGroup | null>(null);

  const reload = async () => {
    setMinistries(await listMinistries().then((r) => r.ministries).catch(() => []));
    setGroups(await listGroups().then((r) => r.groups).catch(() => []));
  };

  useEffect(() => {
    reload();
    listMembers().then((r) => setMembers(r.members)).catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // ---- Ministerios ----
  function openMinistryCreate() { setMForm({ ...EMPTY_MINISTRY }); setMinistryDrawer({ open: true }); }
  function openMinistryEdit(m: Ministry) {
    setMForm({ name: m.name, description: m.description ?? "", leader_id: m.leader_id ?? "", is_active: m.is_active ? "true" : "false" });
    setMinistryDrawer({ open: true, editing: m });
  }
  async function saveMinistry(e: React.FormEvent) {
    e.preventDefault();
    try {
      const payload = { name: mForm.name, description: mForm.description, leader_id: mForm.leader_id, is_active: mForm.is_active === "true" };
      if (ministryDrawer.editing) await updateMinistry(ministryDrawer.editing.id, payload);
      else await createMinistry(payload);
      toast("Ministerio salvo.");
      setMinistryDrawer({ open: false });
      await reload();
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function removeMinistry(m: Ministry) {
    if (!confirm(`Excluir o ministerio "${m.name}"? Os vinculos de voluntarios serao removidos.`)) return;
    try {
      await deleteMinistry(m.id);
      toast("Ministerio excluido.");
      await reload();
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }

  // ---- Grupos ----
  function openGroupCreate() { setGForm({ ...EMPTY_GROUP }); setGroupDrawer({ open: true }); }
  function openGroupEdit(g: SmallGroup) {
    setGForm({
      name: g.name, kind: g.kind, ministry_id: g.ministry_id ?? "", leader_id: g.leader_id ?? "",
      address: g.address ?? "", max_members: g.max_members != null ? String(g.max_members) : "",
      weekday: g.weekday != null ? String(g.weekday) : "", meeting_time: g.meeting_time ?? "",
      is_active: g.is_active ? "true" : "false",
    });
    setGroupDrawer({ open: true, editing: g });
  }
  async function saveGroup(e: React.FormEvent) {
    e.preventDefault();
    try {
      const payload = {
        name: gForm.name, kind: gForm.kind,
        ministry_id: gForm.ministry_id, leader_id: gForm.leader_id, address: gForm.address,
        max_members: gForm.max_members === "" ? undefined : Number(gForm.max_members),
        weekday: gForm.weekday === "" ? undefined : Number(gForm.weekday),
        meeting_time: gForm.meeting_time, is_active: gForm.is_active === "true",
      };
      if (groupDrawer.editing) await updateGroup(groupDrawer.editing.id, payload);
      else await createGroup(payload);
      toast("Grupo salvo.");
      setGroupDrawer({ open: false });
      await reload();
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function removeGroup(g: SmallGroup) {
    if (!confirm(`Excluir o grupo "${g.name}"? O historico de frequencia sera removido.`)) return;
    try {
      await deleteGroup(g.id);
      toast("Grupo excluido.");
      await reload();
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }

  // ---- Voluntarios ----
  async function openVolunteers(m: Ministry) {
    setVolunteers(m);
    try { setVolList((await listMinistryMembers(m.id)).members); } catch { setVolList([]); }
  }
  async function addVol(e: React.FormEvent) {
    e.preventDefault();
    if (!volunteers || selectedVolunteers.length === 0) return;
    try {
      for (const memberId of selectedVolunteers) await addMinistryMember(volunteers.id, memberId, volForm.role);
      toast(`${selectedVolunteers.length} voluntario(s) adicionado(s).`);
      setSelectedVolunteers([]);
      setVolForm({ role: "volunteer" });
      setVolList((await listMinistryMembers(volunteers.id)).members);
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function removeVol(memberId: string) {
    if (!volunteers) return;
    try {
      await removeMinistryMember(volunteers.id, memberId);
      toast("Voluntario removido.");
      setVolList((await listMinistryMembers(volunteers.id)).members);
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }

  async function doCheckIn(e: React.FormEvent) {
    e.preventDefault();
    if (!checkin) return;
    const target = members.find((m) => m.id === ciForm.member_id);
    try {
      await checkIn(checkin.id, { member_id: ciForm.member_id || null, member_name: target?.full_name ?? "Visitante/Anonimo", present: ciForm.present });
      toast("Check-in registrado.");
      setCiForm({ member_id: "", present: true });
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }

  async function openAttendance(g: SmallGroup) {
    setAttGroup(g);
    try { setAttendance((await listAttendance(g.id)).attendance); } catch { setAttendance([]); }
  }

  const memberOptions = members.map((m) => ({ value: m.id, label: m.full_name }));

  return (
    <div className="page">
      <PageHeader title="Ministerios" description="Areas de servico, responsaveis e grupos/celulas" />

      <Tabs
        tabs={[
          { key: "ministries", label: "Ministerios", icon: <Church className="h-4 w-4" /> },
          { key: "groups", label: "Grupos / Celulas", icon: <Users2 className="h-4 w-4" /> },
        ]}
        active={tab}
        onChange={setTab}
      />

      {tab === "ministries" && (
        <>
          <div className="mb-4 flex justify-end">
            {canWrite && <Button onClick={openMinistryCreate}><Plus className="h-4 w-4" /> Novo Ministerio</Button>}
          </div>
          <Card className="overflow-hidden p-0">
            {ministries === null ? (
              <SkeletonRows />
            ) : ministries.length === 0 ? (
              <EmptyState icon={<Church className="h-10 w-10" />} title="Nenhum ministerio" description="Crie areas de servico e reuna voluntarios." />
            ) : (
              <Table>
                <THead><TRow><TH>Ministerio</TH><TH>Responsavel</TH><TH>Status</TH><TH className="text-right">Acoes</TH></TRow></THead>
                <TBody>
                  {ministries.map((m) => (
                    <TRow key={m.id}>
                      <TD>
                        <div className="font-medium">{m.name}</div>
                        <div className="text-xs text-zinc-500">{m.description}</div>
                      </TD>
                      <TD className="text-zinc-500">{m.leader_name ?? "-"}</TD>
                      <TD><Badge tone={m.is_active ? "green" : "zinc"}>{m.is_active ? "Ativo" : "Inativo"}</Badge></TD>
                      <TD className="text-right">
                        <Button variant="ghost" size="sm" onClick={() => openVolunteers(m)}><Users className="h-3.5 w-3.5" /> Voluntarios</Button>
                        {canWrite && <Button variant="ghost" size="sm" title="Editar" onClick={() => openMinistryEdit(m)}><Pencil className="h-3.5 w-3.5" /></Button>}
                        {canWrite && <Button variant="ghost" size="sm" title="Excluir" onClick={() => removeMinistry(m)}><Trash2 className="h-3.5 w-3.5 text-red-500" /></Button>}
                      </TD>
                    </TRow>
                  ))}
                </TBody>
              </Table>
            )}
          </Card>
        </>
      )}

      {tab === "groups" && (
        <>
          <div className="mb-4 flex justify-end">
            {canWrite && <Button onClick={openGroupCreate}><Plus className="h-4 w-4" /> Novo Grupo</Button>}
          </div>
          <Card className="overflow-hidden p-0">
            {groups === null ? (
              <SkeletonRows />
            ) : groups.length === 0 ? (
              <EmptyState icon={<Users2 className="h-10 w-10" />} title="Nenhum grupo" description="Crie celulas, EBDs ou grupos familiares." />
            ) : (
              <Table>
                <THead><TRow><TH>Grupo</TH><TH>Tipo</TH><TH>Lider</TH><TH>Encontro</TH><TH className="text-right">Acoes</TH></TRow></THead>
                <TBody>
                  {groups.map((g) => (
                    <TRow key={g.id}>
                      <TD className="font-medium">{g.name}{!g.is_active && <span className="ml-2 text-xs text-zinc-400">(inativo)</span>}</TD>
                      <TD><Badge tone="sky">{GROUP_KIND[g.kind] ?? g.kind}</Badge></TD>
                      <TD className="text-zinc-500">{g.leader_name ?? "-"}</TD>
                      <TD className="text-xs text-zinc-500">
                        {[g.weekday != null ? WEEKDAYS[g.weekday] : null, g.meeting_time].filter(Boolean).join(" ") || "-"}
                      </TD>
                      <TD className="text-right">
                        <Button variant="ghost" size="sm" onClick={() => setCheckin(g)}><CheckSquare className="h-3.5 w-3.5" /> Check-in</Button>
                        <Button variant="ghost" size="sm" onClick={() => openAttendance(g)}>Frequencia</Button>
                        {canWrite && <Button variant="ghost" size="sm" title="Editar" onClick={() => openGroupEdit(g)}><Pencil className="h-3.5 w-3.5" /></Button>}
                        {canWrite && <Button variant="ghost" size="sm" title="Excluir" onClick={() => removeGroup(g)}><Trash2 className="h-3.5 w-3.5 text-red-500" /></Button>}
                      </TD>
                    </TRow>
                  ))}
                </TBody>
              </Table>
            )}
          </Card>
        </>
      )}

      {/* Drawer ministerio */}
      <Drawer open={ministryDrawer.open} onClose={() => setMinistryDrawer({ open: false })} size="lg" title={ministryDrawer.editing ? "Editar ministerio" : "Novo ministerio"}>
        <form onSubmit={saveMinistry} className="space-y-3">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <Field label="Nome *" className="sm:col-span-2"><Input required className="h-8 text-sm" value={mForm.name} onChange={(e) => setMForm({ ...mForm, name: e.target.value })} /></Field>
            <Field label="Descricao" className="sm:col-span-2"><Input className="h-8 text-sm" value={mForm.description} onChange={(e) => setMForm({ ...mForm, description: e.target.value })} /></Field>
            <Field label="Responsavel">
              <Combobox value={mForm.leader_id} placeholder="Buscar membro..." searchPlaceholder="Buscar membro..." emptyMessage="Nenhum membro" options={memberOptions} onChange={(v) => setMForm({ ...mForm, leader_id: v })} />
            </Field>
            <Field label="Situacao">
              <Select className="h-8 text-sm" value={mForm.is_active} onChange={(e) => setMForm({ ...mForm, is_active: e.target.value })}>
                <option value="true">Ativo</option><option value="false">Inativo</option>
              </Select>
            </Field>
          </div>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setMinistryDrawer({ open: false })}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm">Salvar</Button>
          </div>
        </form>
      </Drawer>

      {/* Drawer grupo */}
      <Drawer open={groupDrawer.open} onClose={() => setGroupDrawer({ open: false })} size="lg" title={groupDrawer.editing ? "Editar grupo" : "Novo grupo"}>
        <form onSubmit={saveGroup} className="space-y-3">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <Field label="Nome *" className="sm:col-span-2"><Input required className="h-8 text-sm" value={gForm.name} onChange={(e) => setGForm({ ...gForm, name: e.target.value })} /></Field>
            <Field label="Tipo">
              <Select className="h-8 text-sm" value={gForm.kind} onChange={(e) => setGForm({ ...gForm, kind: e.target.value })}>
                <option value="cell">Celula</option><option value="ebd">EBD</option><option value="family">Familia</option>
              </Select>
            </Field>
            <Field label="Ministerio">
              <Select className="h-8 text-sm" value={gForm.ministry_id} onChange={(e) => setGForm({ ...gForm, ministry_id: e.target.value })}>
                <option value="">-</option>
                {(ministries ?? []).map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
              </Select>
            </Field>
            <Field label="Lider">
              <Combobox value={gForm.leader_id} placeholder="Buscar membro..." searchPlaceholder="Buscar membro..." emptyMessage="Nenhum membro" options={memberOptions} onChange={(v) => setGForm({ ...gForm, leader_id: v })} />
            </Field>
            <Field label="Dia da semana">
              <Select className="h-8 text-sm" value={gForm.weekday} onChange={(e) => setGForm({ ...gForm, weekday: e.target.value })}>
                <option value="">-</option>
                {WEEKDAYS.map((w, i) => <option key={i} value={i}>{w}</option>)}
              </Select>
            </Field>
            <Field label="Horario"><Input type="time" className="h-8 text-sm" value={gForm.meeting_time} onChange={(e) => setGForm({ ...gForm, meeting_time: e.target.value })} /></Field>
            <Field label="Maximo de membros"><Input type="number" min="0" className="h-8 text-sm" value={gForm.max_members} onChange={(e) => setGForm({ ...gForm, max_members: e.target.value })} /></Field>
            <Field label="Endereco" className="sm:col-span-2"><Input className="h-8 text-sm" value={gForm.address} onChange={(e) => setGForm({ ...gForm, address: e.target.value })} /></Field>
            <Field label="Situacao">
              <Select className="h-8 text-sm" value={gForm.is_active} onChange={(e) => setGForm({ ...gForm, is_active: e.target.value })}>
                <option value="true">Ativo</option><option value="false">Inativo</option>
              </Select>
            </Field>
          </div>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setGroupDrawer({ open: false })}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm">Salvar</Button>
          </div>
        </form>
      </Drawer>

      {/* Voluntarios */}
      <Modal open={!!volunteers} onClose={() => setVolunteers(null)} title={`Voluntarios - ${volunteers?.name ?? ""}`}>
        <form onSubmit={addVol} className="space-y-3">
          <Field label="Membros (selecao multipla)">
            <div className="space-y-1">
              {members.map((m) => {
                const isSelected = selectedVolunteers.includes(m.id);
                const alreadyAdded = volList.some((v) => v.member_id === m.id);
                return (
                  <label key={m.id} className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={isSelected}
                      disabled={alreadyAdded}
                      onChange={(e) => {
                        if (e.target.checked) setSelectedVolunteers([...selectedVolunteers, m.id]);
                        else setSelectedVolunteers(selectedVolunteers.filter((id) => id !== m.id));
                      }}
                      className="rounded border-zinc-300 text-sky-600 focus:ring-sky-500"
                    />
                    <span className={alreadyAdded ? "text-zinc-400" : ""}>{m.full_name}</span>
                    {alreadyAdded && <Badge tone="green" className="text-[10px]">Ativo</Badge>}
                  </label>
                );
              })}
            </div>
          </Field>
          <Field label="Funcao">
            <Select className="h-8 text-sm" value={volForm.role} onChange={(e) => setVolForm({ ...volForm, role: e.target.value })}>
              <option value="volunteer">Voluntario</option>
              <option value="coordinator">Coordenador</option>
              <option value="leader">Lider</option>
            </Select>
          </Field>
          <div className="flex justify-end gap-2">
            <Button type="submit" disabled={selectedVolunteers.length === 0} className="h-8 text-sm">
              Adicionar {selectedVolunteers.length > 0 && `(${selectedVolunteers.length})`}
            </Button>
          </div>
        </form>
        <div className="mt-4 border-t pt-3">
          {volList.length === 0 ? (
            <p className="py-4 text-center text-sm text-zinc-400">Nenhum voluntario ainda.</p>
          ) : (
            <ul className="space-y-2 text-sm">
              {volList.map((v) => (
                <li key={v.member_id} className="flex items-center justify-between">
                  <span className="font-medium">{v.member_name}</span>
                  <div className="flex items-center gap-2">
                    <Badge tone="sky">{v.role}</Badge>
                    <Button variant="ghost" size="sm" onClick={() => removeVol(v.member_id)} title="Remover"><Trash2 className="h-3.5 w-3.5 text-red-500" /></Button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </Modal>

      <Modal open={!!checkin} onClose={() => setCheckin(null)} title={`Check-in - ${checkin?.name ?? ""}`}>
        <form onSubmit={doCheckIn} className="space-y-3">
          <Field label="Membro (ou visitante anonimo)">
            <Combobox value={ciForm.member_id} placeholder="Selecione um membro..." searchPlaceholder="Buscar membro..." options={memberOptions} onChange={(val) => setCiForm({ ...ciForm, member_id: val })} />
          </Field>
          <Field label="Presenca">
            <Select value={String(ciForm.present)} onChange={(e) => setCiForm({ ...ciForm, present: e.target.value === "true" })}>
              <option value="true">Presente</option><option value="false">Faltou</option>
            </Select>
          </Field>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setCheckin(null)}>Cancelar</Button>
            <Button type="submit">Registrar</Button>
          </div>
        </form>
      </Modal>

      <Modal open={!!attGroup} onClose={() => setAttGroup(null)} title={`Frequencia - ${attGroup?.name ?? ""}`}>
        {attendance.length === 0 ? (
          <p className="py-6 text-center text-sm text-zinc-400">Sem registro de frequencia.</p>
        ) : (
          <ul className="max-h-80 space-y-2 overflow-y-auto text-sm">
            {attendance.map((a, i) => (
              <li key={i} className="flex items-center justify-between">
                <span className={a.member_id ? "font-medium" : "text-zinc-500"}>{a.member_name}</span>
                <span className="flex items-center gap-2 text-xs text-zinc-500">
                  <Badge tone={a.present ? "green" : "red"}>{a.present ? "Presente" : "Faltou"}</Badge>
                  {dateTimePt(a.attended_at)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </Modal>
    </div>
  );
}
