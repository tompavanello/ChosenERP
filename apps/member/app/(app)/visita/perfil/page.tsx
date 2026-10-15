"use client";

import { useEffect, useState } from "react";
import { Save } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Field, Input } from "@/components/ui/input";
import { SkeletonRows } from "@/components/ui/skeleton";
import { getMeVisitor, updateMeVisitor, type VisitorProfile } from "@/lib/api";

export default function VisitorPerfilPage() {
  const [visitor, setVisitor] = useState<VisitorProfile | null>(null);
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [email, setEmail] = useState("");
  const [whatsapp, setWhatsapp] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  useEffect(() => {
    getMeVisitor()
      .then((v) => {
        setVisitor(v);
        setFirstName(v.first_name ?? "");
        setLastName(v.last_name ?? "");
        setEmail(v.email ?? "");
        setWhatsapp(v.whatsapp ?? "");
      })
      .catch(() => setVisitor(null))
      .finally(() => setLoading(false));
  }, []);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setMsg(null);
    try {
      const v = await updateMeVisitor({
        first_name: firstName,
        last_name: lastName,
        email: email || undefined,
        whatsapp: whatsapp || undefined,
      });
      setVisitor(v);
      setMsg("Contato atualizado.");
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Não foi possível salvar.");
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <SkeletonRows rows={4} />;
  if (!visitor) {
    return (
      <Card>
        <p className="text-sm text-[var(--muted)]">Não foi possível carregar seu contato.</p>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Meu contato</h1>

      <Card>
        <form onSubmit={save} className="space-y-4">
          <Field label="Nome">
            <Input value={firstName} onChange={(e) => setFirstName(e.target.value)} />
          </Field>
          <Field label="Sobrenome">
            <Input value={lastName} onChange={(e) => setLastName(e.target.value)} />
          </Field>
          <Field label="E-mail">
            <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </Field>
          <Field label="WhatsApp">
            <Input type="tel" value={whatsapp} onChange={(e) => setWhatsapp(e.target.value)} />
          </Field>
          <Field label="Telefone (login)">
            <Input value={visitor.phone ?? ""} disabled />
          </Field>
          {msg && <p className="text-sm text-[var(--brand)]">{msg}</p>}
          <Button type="submit" disabled={saving} className="w-full">
            <Save className="h-4 w-4" />
            {saving ? "Salvando..." : "Salvar"}
          </Button>
        </form>
      </Card>

      <p className="text-center text-xs text-[var(--muted)]">
        O telefone é o seu identificador de acesso e não pode ser alterado aqui.
      </p>
    </div>
  );
}
