"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { KeyRound } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Field, Input } from "@/components/ui/input";
import { changePassword, clearSession, fetchMe, getToken } from "@/lib/api";

export default function TrocarSenhaPage() {
  const router = useRouter();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!getToken()) router.replace("/");
  }, [router]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (next.length < 8) {
      setError("A nova senha deve ter ao menos 8 caracteres.");
      return;
    }
    if (next !== confirm) {
      setError("As senhas nao conferem.");
      return;
    }
    setSaving(true);
    try {
      await changePassword(current, next);
      await fetchMe();
      router.replace("/home");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Nao foi possivel trocar a senha.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-5 py-10">
      <header className="mb-6 flex flex-col items-center text-center">
        <span className="mb-3 flex h-12 w-12 items-center justify-center rounded-2xl bg-[var(--brand)] text-white">
          <KeyRound className="h-6 w-6" />
        </span>
        <h1 className="text-lg font-bold">Defina sua nova senha</h1>
        <p className="mt-1 text-sm text-[var(--muted)]">
          Sua senha atual e provisoria. Crie uma nova para continuar.
        </p>
      </header>

      <Card>
        <form onSubmit={submit} className="space-y-3">
          <Field label="Senha provisoria (atual)">
            <Input type="password" autoComplete="current-password" required value={current} onChange={(e) => setCurrent(e.target.value)} />
          </Field>
          <Field label="Nova senha">
            <Input type="password" autoComplete="new-password" required value={next} onChange={(e) => setNext(e.target.value)} />
          </Field>
          <Field label="Confirmar nova senha">
            <Input type="password" autoComplete="new-password" required value={confirm} onChange={(e) => setConfirm(e.target.value)} />
          </Field>
          {error && <p className="text-sm text-red-600">{error}</p>}
          <Button type="submit" disabled={saving} className="w-full">
            {saving ? "Salvando..." : "Salvar nova senha"}
          </Button>
        </form>
      </Card>

      <button
        type="button"
        onClick={() => { clearSession(); router.replace("/"); }}
        className="mt-4 text-center text-xs text-[var(--muted)]"
      >
        Sair
      </button>
    </main>
  );
}
