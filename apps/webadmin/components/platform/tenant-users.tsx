"use client";

import { useCallback, useEffect, useState } from "react";
import { KeyRound, Pencil, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Drawer } from "@/components/ui/modal";
import { Field, Input, Select } from "@/components/ui/input";
import { SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import {
  createAdminTenantUser, getAdminTenantBranches, getAdminTenantRoles, getAdminTenantUsers,
  resetAdminTenantUserPassword, updateAdminTenantUser,
  type AdminUser, type PlatformBranch, type RoleInfo,
} from "@/lib/api";
import { dateTimePt } from "@/lib/format";

const EMPTY = { email: "", full_name: "", role: "", branch_id: "", is_active: "true", password: "" };

/**
 * Acessos de uma igreja visto pelo console da plataforma (suporte): criar,
 * editar perfil/filial, ativar/desativar e redefinir senha.
 */
export function TenantUsers({ tenantId }: { tenantId: string }) {
  const { toast } = useToast();
  const [users, setUsers] = useState<AdminUser[] | null>(null);
  const [roles, setRoles] = useState<RoleInfo[]>([]);
  const [branches, setBranches] = useState<PlatformBranch[]>([]);

  const [drawer, setDrawer] = useState<{ open: boolean; editing?: AdminUser }>({ open: false });
  const [form, setForm] = useState({ ...EMPTY });
  const [saving, setSaving] = useState(false);

  const [pwUser, setPwUser] = useState<AdminUser | null>(null);
  const [newPw, setNewPw] = useState("");
  const [resetting, setResetting] = useState(false);

  const load = useCallback(async () => {
    try {
      const [u, r, b] = await Promise.all([
        getAdminTenantUsers(tenantId),
        getAdminTenantRoles(tenantId),
        getAdminTenantBranches(tenantId),
      ]);
      setUsers(u.users);
      setRoles(r.roles);
      setBranches(b.branches);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar usuarios", "error");
      setUsers([]);
    }
  }, [tenantId, toast]);

  useEffect(() => { load(); }, [load]);

  function openNew() {
    setForm({ ...EMPTY, role: roles[0]?.key ?? "" });
    setDrawer({ open: true });
  }
  function openEdit(u: AdminUser) {
    setForm({
      email: u.email, full_name: u.full_name, role: u.role,
      branch_id: u.branch_id ?? "", is_active: u.is_active ? "true" : "false", password: "",
    });
    setDrawer({ open: true, editing: u });
  }

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      if (drawer.editing) {
        await updateAdminTenantUser(tenantId, drawer.editing.id, {
          full_name: form.full_name,
          role: form.role,
          branch_id: form.branch_id, // "" limpa (Sede)
          is_active: form.is_active === "true",
        });
        toast("Acesso atualizado");
      } else {
        await createAdminTenantUser(tenantId, {
          email: form.email.trim(),
          full_name: form.full_name,
          role: form.role,
          branch_id: form.branch_id || undefined,
          is_active: form.is_active === "true",
          password: form.password,
        });
        toast("Acesso criado");
      }
      setDrawer({ open: false });
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar acesso", "error");
    } finally {
      setSaving(false);
    }
  }

  async function confirmReset() {
    if (!pwUser) return;
    if (newPw.length < 8) {
      toast("A senha deve ter ao menos 8 caracteres", "error");
      return;
    }
    setResetting(true);
    try {
      await resetAdminTenantUserPassword(tenantId, pwUser.id, newPw);
      toast("Senha redefinida");
      setPwUser(null);
      setNewPw("");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao redefinir senha", "error");
    } finally {
      setResetting(false);
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-zinc-500">Acessos da igreja</h3>
        <Button size="sm" onClick={openNew}><Plus className="h-3.5 w-3.5" /> Novo acesso</Button>
      </div>

      {users === null ? (
        <SkeletonRows rows={4} />
      ) : users.length === 0 ? (
        <Card><p className="text-sm text-zinc-400">Nenhum acesso cadastrado.</p></Card>
      ) : (
        <Card className="overflow-hidden p-0">
          <Table>
            <THead>
              <TRow>
                <TH>Nome</TH><TH>E-mail</TH><TH>Perfil</TH><TH>Filial</TH>
                <TH>Situacao</TH><TH>Ultimo acesso</TH><TH></TH>
              </TRow>
            </THead>
            <TBody>
              {users.map((u) => (
                <TRow key={u.id}>
                  <TD className="font-medium">{u.full_name}</TD>
                  <TD className="text-sm text-zinc-500">{u.email}</TD>
                  <TD><Badge tone="zinc">{u.role}</Badge></TD>
                  <TD className="text-xs text-zinc-500">
                    {u.branch_id ? (branches.find((b) => b.id === u.branch_id)?.name ?? "-") : "Sede"}
                  </TD>
                  <TD><Badge tone={u.is_active ? "green" : "zinc"}>{u.is_active ? "Ativo" : "Inativo"}</Badge></TD>
                  <TD className="text-xs text-zinc-400">{u.last_login_at ? dateTimePt(u.last_login_at) : "nunca"}</TD>
                  <TD className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button variant="outline" size="sm" onClick={() => openEdit(u)}>
                        <Pencil className="h-3.5 w-3.5" />
                      </Button>
                      <Button variant="outline" size="sm" onClick={() => { setPwUser(u); setNewPw(""); }} title="Redefinir senha">
                        <KeyRound className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        </Card>
      )}

      <Drawer open={drawer.open} onClose={() => setDrawer({ open: false })} title={drawer.editing ? "Editar acesso" : "Novo acesso"}>
        <form onSubmit={save} className="space-y-3">
          {!drawer.editing && (
            <Field label="E-mail *">
              <Input required type="email" className="h-8 text-sm" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
            </Field>
          )}
          <Field label="Nome completo *">
            <Input required className="h-8 text-sm" value={form.full_name} onChange={(e) => setForm({ ...form, full_name: e.target.value })} />
          </Field>
          <Field label="Perfil *" hint="super_admin/admin_sede definem administradores da igreja.">
            <Select required className="h-8 text-sm" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
              {roles.map((r) => <option key={r.key} value={r.key}>{r.name}</option>)}
            </Select>
          </Field>
          <Field label="Filial" hint="Vazio = Sede (todas as filiais).">
            <Select className="h-8 text-sm" value={form.branch_id} onChange={(e) => setForm({ ...form, branch_id: e.target.value })}>
              <option value="">Sede (todas as filiais)</option>
              {branches.map((b) => <option key={b.id} value={b.id}>{b.name}</option>)}
            </Select>
          </Field>
          {!drawer.editing && (
            <Field label="Senha inicial *" hint="Minimo 8 caracteres.">
              <Input required type="text" className="h-8 text-sm" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
            </Field>
          )}
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" className="h-4 w-4" checked={form.is_active === "true"} onChange={(e) => setForm({ ...form, is_active: e.target.checked ? "true" : "false" })} />
            Ativo
          </label>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setDrawer({ open: false })}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>

      <Drawer open={pwUser !== null} onClose={() => setPwUser(null)} title={pwUser ? `Redefinir senha - ${pwUser.full_name}` : "Redefinir senha"}>
        <div className="space-y-3">
          <Field label="Nova senha" hint="Minimo 8 caracteres.">
            <Input type="text" className="h-8 text-sm" value={newPw} onChange={(e) => setNewPw(e.target.value)} />
          </Field>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setPwUser(null)}>Cancelar</Button>
            <Button type="button" className="h-8 text-sm" disabled={resetting} onClick={confirmReset}>
              <KeyRound className="h-3.5 w-3.5" /> {resetting ? "Salvando..." : "Redefinir"}
            </Button>
          </div>
        </div>
      </Drawer>
    </div>
  );
}
