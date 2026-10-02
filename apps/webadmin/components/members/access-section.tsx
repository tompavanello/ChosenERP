"use client";

import { useCallback, useEffect, useState } from "react";
import { KeyRound, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Field, Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import {
  createMemberAccess, getMemberAccess, resetMemberAccessPassword, updateMemberAccess,
  type Member, type MemberAccess,
} from "@/lib/api";
import { dateTimePt } from "@/lib/format";

/**
 * Acesso do membro ao app: identidade (e-mail/telefone) + senha provisoria.
 * Só a Sede (super_admin/admin_sede) gerencia este vinculo.
 */
export function AccessSection({
  member,
  canWrite,
}: {
  member: Member;
  canWrite: boolean;
}) {
  const { toast } = useToast();
  const [access, setAccess] = useState<MemberAccess | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);

  const [email, setEmail] = useState(member.email ?? "");
  const [phone, setPhone] = useState(member.whatsapp ?? member.phone ?? "");
  const [password, setPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const a = await getMemberAccess(member.id);
      setAccess(a);
      setEmail(a.email ?? member.email ?? "");
      setPhone(a.phone ?? member.whatsapp ?? member.phone ?? "");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar acesso", "error");
    } finally {
      setLoading(false);
    }
  }, [member.id, member.email, member.phone, member.whatsapp, toast]);

  useEffect(() => { load(); }, [load]);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    if (password.length < 8) {
      toast("A senha deve ter ao menos 8 caracteres", "error");
      return;
    }
    setBusy(true);
    try {
      await createMemberAccess(member.id, { email: email.trim(), phone: phone.trim(), password });
      toast("Acesso criado. O membro devera trocar a senha no primeiro acesso.");
      setPassword("");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao criar acesso", "error");
    } finally {
      setBusy(false);
    }
  }

  async function saveIdentifier(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await updateMemberAccess(member.id, { email: email.trim(), phone: phone.trim() });
      toast("Acesso atualizado");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao atualizar acesso", "error");
    } finally {
      setBusy(false);
    }
  }

  async function toggleActive() {
    if (!access) return;
    setBusy(true);
    try {
      await updateMemberAccess(member.id, { is_active: !access.is_active });
      toast(access.is_active ? "Acesso desativado" : "Acesso reativado");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao alterar situacao", "error");
    } finally {
      setBusy(false);
    }
  }

  async function resetPassword() {
    if (newPassword.length < 8) {
      toast("A senha deve ter ao menos 8 caracteres", "error");
      return;
    }
    setBusy(true);
    try {
      await resetMemberAccessPassword(member.id, newPassword);
      toast("Senha redefinida (provisoria)");
      setNewPassword("");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao redefinir senha", "error");
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <SkeletonRows rows={4} />;

  if (!access?.has_access) {
    return (
      <Card className="max-w-xl p-4">
        <div className="mb-3 flex items-center gap-2">
          <KeyRound className="h-4 w-4 text-sky-600" />
          <h3 className="text-sm font-semibold">Sem acesso ao app</h3>
        </div>
        <p className="mb-4 text-sm text-zinc-500">
          Crie a identidade do membro para ele entrar no app da igreja. O login sera pelo
          e-mail ou telefone informados, com uma senha provisoria.
        </p>
        <form onSubmit={create} className="space-y-3">
          <Field label="E-mail *">
            <Input required type="email" className="h-8 text-sm" value={email} onChange={(e) => setEmail(e.target.value)} />
          </Field>
          <Field label="Telefone" hint="Opcional. Apenas digitos; permite login por telefone.">
            <Input className="h-8 text-sm" value={phone} onChange={(e) => setPhone(e.target.value)} />
          </Field>
          <Field label="Senha provisoria *" hint="Minimo 8 caracteres. O membro troca no 1o acesso.">
            <Input required type="text" className="h-8 text-sm" value={password} onChange={(e) => setPassword(e.target.value)} />
          </Field>
          <div className="flex justify-end">
            <Button type="submit" className="h-8 text-sm" disabled={busy || !canWrite}>
              {busy ? "Criando..." : "Criar acesso"}
            </Button>
          </div>
        </form>
      </Card>
    );
  }

  return (
    <div className="max-w-xl space-y-4">
      <Card className="p-4">
        <div className="mb-3 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <KeyRound className="h-4 w-4 text-sky-600" />
            <h3 className="text-sm font-semibold">Acesso ao app</h3>
          </div>
          <Badge tone={access.is_active ? "green" : "zinc"}>{access.is_active ? "Ativo" : "Inativo"}</Badge>
        </div>
        <dl className="mb-4 grid grid-cols-2 gap-2 text-sm">
          <div className="flex justify-between gap-2">
            <dt className="text-zinc-500">Ultimo login</dt>
            <dd>{access.last_login_at ? dateTimePt(access.last_login_at) : "nunca"}</dd>
          </div>
          <div className="flex justify-between gap-2">
            <dt className="text-zinc-500">Senha provisoria</dt>
            <dd>{access.must_change_password ? "sim (troca no 1o acesso)" : "nao"}</dd>
          </div>
        </dl>
        <form onSubmit={saveIdentifier} className="space-y-3">
          <Field label="E-mail">
            <Input type="email" className="h-8 text-sm" value={email} onChange={(e) => setEmail(e.target.value)} />
          </Field>
          <Field label="Telefone" hint="Deixe vazio para remover o login por telefone.">
            <Input className="h-8 text-sm" value={phone} onChange={(e) => setPhone(e.target.value)} />
          </Field>
          <div className="flex justify-end gap-2">
            <Button variant="outline" type="button" className="h-8 text-sm" disabled={busy || !canWrite} onClick={toggleActive}>
              {access.is_active ? "Desativar acesso" : "Reativar acesso"}
            </Button>
            <Button type="submit" className="h-8 text-sm" disabled={busy || !canWrite}>
              Salvar identificador
            </Button>
          </div>
        </form>
      </Card>

      <Card className="p-4">
        <h3 className="mb-3 text-sm font-semibold">Redefinir senha</h3>
        <div className="flex items-end gap-2">
          <Field label="Nova senha provisoria" className="flex-1">
            <Input type="text" className="h-8 text-sm" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
          </Field>
          <Button variant="outline" type="button" className="h-8 text-sm" disabled={busy || !canWrite} onClick={resetPassword}>
            <RefreshCw className="h-3.5 w-3.5" /> Redefinir
          </Button>
        </div>
      </Card>
    </div>
  );
}
