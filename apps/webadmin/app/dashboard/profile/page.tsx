"use client";

import { useState } from "react";
import { KeyRound, Save, UserCog } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Field, Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { MfaCard } from "@/components/security/mfa-card";
import { updateProfile, changePassword } from "@/lib/api";

export default function ProfilePage() {
  const { toast } = useToast();
  const { user, refresh } = useAuth();

  const [profile, setProfile] = useState({ full_name: user?.full_name ?? "", email: user?.email ?? "" });
  const [savingProfile, setSavingProfile] = useState(false);

  const [pw, setPw] = useState({ current: "", next: "", confirm: "" });
  const [savingPw, setSavingPw] = useState(false);

  async function saveProfile(e: React.FormEvent) {
    e.preventDefault();
    setSavingProfile(true);
    try {
      await updateProfile({ full_name: profile.full_name, email: profile.email });
      await refresh();
      toast("Perfil atualizado.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar perfil", "error");
    } finally {
      setSavingProfile(false);
    }
  }

  async function savePassword(e: React.FormEvent) {
    e.preventDefault();
    if (pw.next !== pw.confirm) {
      toast("A confirmacao nao confere com a nova senha.", "error");
      return;
    }
    if (pw.next.length < 8) {
      toast("A nova senha deve ter ao menos 8 caracteres.", "error");
      return;
    }
    setSavingPw(true);
    try {
      await changePassword(pw.current, pw.next);
      setPw({ current: "", next: "", confirm: "" });
      toast("Senha alterada.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao alterar senha", "error");
    } finally {
      setSavingPw(false);
    }
  }

  return (
    <div className="page">
      <PageHeader title="Meu perfil" description="Seus dados de acesso e seguranca da conta" />

      <div className="grid gap-4 lg:grid-cols-2">
        <Card className="p-4">
          <div className="mb-4 flex items-center gap-2">
            <UserCog className="h-4 w-4 text-sky-600" />
            <h3 className="text-sm font-semibold">Dados do perfil</h3>
          </div>
          <form onSubmit={saveProfile} className="space-y-3">
            <Field label="Nome completo" required>
              <Input
                required className="h-8 text-sm"
                value={profile.full_name}
                onChange={(e) => setProfile({ ...profile, full_name: e.target.value })}
              />
            </Field>
            <Field label="E-mail" required>
              <Input
                required type="email" className="h-8 text-sm"
                value={profile.email}
                onChange={(e) => setProfile({ ...profile, email: e.target.value })}
              />
            </Field>
            <Field label="Perfil de acesso">
              <Input disabled className="h-8 text-sm" value={user?.role ?? ""} />
            </Field>
            <div className="flex justify-end pt-1">
              <Button type="submit" className="h-8 text-sm" disabled={savingProfile}>
                <Save className="h-4 w-4" /> {savingProfile ? "Salvando..." : "Salvar"}
              </Button>
            </div>
          </form>
        </Card>

        <Card className="p-4">
          <div className="mb-4 flex items-center gap-2">
            <KeyRound className="h-4 w-4 text-sky-600" />
            <h3 className="text-sm font-semibold">Alterar senha</h3>
          </div>
          <form onSubmit={savePassword} className="space-y-3">
            <Field label="Senha atual" required>
              <Input
                required type="password" className="h-8 text-sm"
                value={pw.current}
                onChange={(e) => setPw({ ...pw, current: e.target.value })}
              />
            </Field>
            <Field label="Nova senha" required hint="Minimo de 8 caracteres.">
              <Input
                required type="password" className="h-8 text-sm"
                value={pw.next}
                onChange={(e) => setPw({ ...pw, next: e.target.value })}
              />
            </Field>
            <Field label="Confirmar nova senha" required>
              <Input
                required type="password" className="h-8 text-sm"
                value={pw.confirm}
                onChange={(e) => setPw({ ...pw, confirm: e.target.value })}
              />
            </Field>
            <div className="flex justify-end pt-1">
              <Button type="submit" className="h-8 text-sm" disabled={savingPw}>
                <KeyRound className="h-4 w-4" /> {savingPw ? "Alterando..." : "Alterar senha"}
              </Button>
            </div>
          </form>
        </Card>
      </div>

      <div className="mt-4">
        <MfaCard />
      </div>
    </div>
  );
}
