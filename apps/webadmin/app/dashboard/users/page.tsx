"use client";

import { useCallback, useEffect, useState } from "react";
import { KeyRound, Pencil, Plus, ShieldCheck, ShieldOff, UserCog } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Drawer, Modal } from "@/components/ui/modal";
import { Field, Input, Select } from "@/components/ui/input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { useBranches } from "@/lib/swr-hooks";
import { MfaCard } from "@/components/security/mfa-card";
import {
  listUsers, createUser, updateUser, resetUserPassword, listRoles,
  type AdminUser, type RoleInfo,
} from "@/lib/api";
import { dateTimePt } from "@/lib/format";

const EMPTY_FORM = {
  email: "", full_name: "", password: "", role: "", branch_id: "", is_active: "true",
};

export default function UsersPage() {
  const { toast } = useToast();
  const { user, hasPerm } = useAuth();
  const { data: branchData } = useBranches();
  const canManage = hasPerm("users.write");
  const [users, setUsers] = useState<AdminUser[] | null>(null);
  const [roles, setRoles] = useState<RoleInfo[]>([]);
  const [drawer, setDrawer] = useState<{ open: boolean; editing?: AdminUser }>({ open: false });
  const [form, setForm] = useState({ ...EMPTY_FORM });
  const [saving, setSaving] = useState(false);
  const [resetTarget, setResetTarget] = useState<AdminUser | null>(null);
  const [newPassword, setNewPassword] = useState("");

  const load = useCallback(async () => {
    try {
      const [u, r] = await Promise.all([listUsers(), listRoles()]);
      setUsers(u.users);
      setRoles(r.roles);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao carregar usuarios", "error");
      setUsers([]);
    }
  }, [toast]);

  useEffect(() => {
    load();
  }, [load]);

  const roleName = (key: string) => roles.find((r) => r.key === key)?.name ?? key;
  const branchName = (id?: string) =>
    !id ? "Sede" : branchData?.branches.find((b) => b.id === id)?.name ?? "-";

  function openCreate() {
    setForm({ ...EMPTY_FORM, role: roles[0]?.key ?? "" });
    setDrawer({ open: true });
  }
  function openEdit(u: AdminUser) {
    setForm({
      email: u.email, full_name: u.full_name, password: "",
      role: u.role, branch_id: u.branch_id ?? "", is_active: u.is_active ? "true" : "false",
    });
    setDrawer({ open: true, editing: u });
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      if (drawer.editing) {
        await updateUser(drawer.editing.id, {
          full_name: form.full_name,
          role: form.role,
          branch_id: form.branch_id,
          is_active: form.is_active === "true",
        });
        toast("Usuario atualizado.");
      } else {
        await createUser({
          email: form.email,
          full_name: form.full_name,
          password: form.password,
          role: form.role,
          branch_id: form.branch_id,
          is_active: form.is_active === "true",
        });
        toast("Usuario criado.");
      }
      setDrawer({ open: false });
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar", "error");
    } finally {
      setSaving(false);
    }
  }

  async function toggleActive(u: AdminUser) {
    try {
      await updateUser(u.id, { is_active: !u.is_active });
      await load();
      toast(u.is_active ? "Usuario desativado." : "Usuario ativado.");
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    }
  }

  async function doResetPassword(e: React.FormEvent) {
    e.preventDefault();
    if (!resetTarget) return;
    try {
      await resetUserPassword(resetTarget.id, newPassword);
      toast("Senha redefinida.");
      setResetTarget(null);
      setNewPassword("");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  if (!hasPerm("users.read")) {
    return (
      <div className="page">
        <EmptyState icon={<UserCog className="h-10 w-10" />} title="Acesso restrito" description="Somente administradores podem gerir usuarios." />
      </div>
    );
  }

  return (
    <div className="page">
      <PageHeader
        title="Usuarios e Acessos"
        description="Crie usuarios, defina perfis e gerencie a seguranca"
        actions={canManage ? <Button onClick={openCreate}><Plus className="h-4 w-4" /> Novo usuario</Button> : undefined}
      />

      <MfaCard />

      <Card className="overflow-hidden p-0">
        {users === null ? (
          <div className="p-4"><SkeletonRows rows={5} /></div>
        ) : users.length === 0 ? (
          <EmptyState icon={<UserCog className="h-10 w-10" />} title="Nenhum usuario" description="Cadastre o primeiro usuario da equipe." />
        ) : (
          <Table>
            <THead>
              <TRow>
                <TH>Usuario</TH><TH>Perfil</TH><TH>Filial</TH><TH>Status</TH><TH>MFA</TH><TH>Asltimo acesso</TH><TH className="text-right">Acoes</TH>
              </TRow>
            </THead>
            <TBody>
              {users.map((u) => (
                <TRow key={u.id}>
                  <TD>
                    <p className="font-medium">{u.full_name}{u.id === user?.id ? " (voce)" : ""}</p>
                    <p className="text-xs text-zinc-400">{u.email}</p>
                  </TD>
                  <TD><Badge tone="sky">{roleName(u.role)}</Badge></TD>
                  <TD className="text-sm">{branchName(u.branch_id)}</TD>
                  <TD><Badge tone={u.is_active ? "green" : "zinc"}>{u.is_active ? "Ativo" : "Inativo"}</Badge></TD>
                  <TD>{u.mfa_enabled ? <Badge tone="green">Ativo</Badge> : <span className="text-xs text-zinc-400">-</span>}</TD>
                  <TD className="text-xs text-zinc-400">{u.last_login_at ? dateTimePt(u.last_login_at) : "nunca"}</TD>
                  <TD>
                    {canManage && (
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" className="h-8 px-2" title="Editar" onClick={() => openEdit(u)}>
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" className="h-8 px-2" title="Redefinir senha" onClick={() => setResetTarget(u)}>
                          <KeyRound className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" className="h-8 px-2" title={u.is_active ? "Desativar" : "Ativar"} onClick={() => toggleActive(u)}>
                          {u.is_active ? <ShieldOff className="h-4 w-4" /> : <ShieldCheck className="h-4 w-4" />}
                        </Button>
                      </div>
                    )}
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Drawer
        open={drawer.open}
        onClose={() => setDrawer({ open: false })}
        title={drawer.editing ? `Editar - ${drawer.editing.full_name}` : "Novo usuario"}
        size="lg"
      >
        <form onSubmit={submit} className="space-y-3">
          <Field label="Nome completo *">
            <Input required className="h-8 text-sm" value={form.full_name} onChange={(e) => setForm({ ...form, full_name: e.target.value })} />
          </Field>
          {!drawer.editing && (
            <>
              <Field label="E-mail *">
                <Input required type="email" className="h-8 text-sm" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
              </Field>
              <Field label="Senha *" hint="Minimo de 8 caracteres.">
                <Input required type="password" className="h-8 text-sm" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
              </Field>
            </>
          )}
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <Field label="Perfil">
              <Select className="h-8 text-sm" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
                {roles.map((r) => <option key={r.key} value={r.key}>{r.name}</option>)}
              </Select>
            </Field>
            <Field label="Filial">
              <Select className="h-8 text-sm" value={form.branch_id} onChange={(e) => setForm({ ...form, branch_id: e.target.value })}>
                <option value="">Sede (todas as filiais)</option>
                {(branchData?.branches ?? []).map((b) => <option key={b.id} value={b.id}>{b.name}</option>)}
              </Select>
            </Field>
          </div>
          <Field label="Situacao">
            <Select className="h-8 text-sm" value={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.value })}>
              <option value="true">Ativo</option>
              <option value="false">Inativo</option>
            </Select>
          </Field>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setDrawer({ open: false })}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>

      <Modal open={!!resetTarget} onClose={() => setResetTarget(null)} title={`Redefinir senha - ${resetTarget?.full_name ?? ""}`}>
        <form onSubmit={doResetPassword} className="space-y-3">
          <Field label="Nova senha" hint="Minimo de 8 caracteres.">
            <Input required type="password" className="h-8 text-sm" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
          </Field>
          <div className="flex justify-end gap-2">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setResetTarget(null)}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm">Redefinir</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
