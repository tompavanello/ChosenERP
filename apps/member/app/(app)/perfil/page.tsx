"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Activity, ChevronRight, Save } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Field, Input } from "@/components/ui/input";
import { SkeletonRows } from "@/components/ui/skeleton";
import {
  datePt,
  getMeFamily,
  getMeMember,
  updateMeMember,
  type Family,
  type Member,
} from "@/lib/api";

const STATUS_LABELS: Record<string, string> = {
  active: "Ativo Professo",
  member: "Ativo Não Professo",
  inactive: "Inativo",
};

export default function PerfilPage() {
  const [member, setMember] = useState<Member | null>(null);
  const [families, setFamilies] = useState<Family[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const [form, setForm] = useState({ email: "", phone: "", whatsapp: "" });

  useEffect(() => {
    Promise.all([getMeMember(), getMeFamily().catch(() => [])])
      .then(([m, f]) => {
        setMember(m);
        setFamilies(f);
        setForm({ email: m.email ?? "", phone: m.phone ?? "", whatsapp: m.whatsapp ?? "" });
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Erro ao carregar."))
      .finally(() => setLoading(false));
  }, []);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setMsg(null);
    try {
      const updated = await updateMeMember({
        email: form.email || undefined,
        phone: form.phone || undefined,
        whatsapp: form.whatsapp || undefined,
      });
      setMember(updated);
      setEditing(false);
      setMsg("Dados atualizados.");
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Não foi possível salvar.");
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <SkeletonRows rows={5} />;
  if (error || !member) {
    return (
      <Card>
        <p className="text-sm text-[var(--muted)]">
          {error ?? "Sua conta ainda não está vinculada a um cadastro de membro. Procure a secretaria."}
        </p>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Meu perfil</h1>

      <Card>
        <div className="flex items-center gap-3">
          {member.photo_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={member.photo_url} alt={member.full_name} className="h-14 w-14 rounded-full object-cover" />
          ) : (
            <span className="flex h-14 w-14 items-center justify-center rounded-full bg-[var(--brand-soft)] text-lg font-semibold text-[var(--brand-strong)]">
              {member.full_name?.charAt(0) ?? "?"}
            </span>
          )}
          <div>
            <p className="font-semibold">{member.full_name}</p>
            <p className="text-sm text-[var(--muted)]">
              {STATUS_LABELS[member.membership_status] ?? member.membership_status}
            </p>
            {member.card_ref && (
              <p className="tnum font-mono text-xs text-[var(--muted)]">{member.card_ref}</p>
            )}
          </div>
        </div>

        <dl className="mt-4 space-y-2 text-sm">
          <Row label="Nascimento" value={datePt(member.birth_date)} />
          <Row label="Batismo" value={datePt(member.baptism_date)} />
          <Row label="Membro desde" value={datePt(member.joined_at)} />
        </dl>
      </Card>

      <Card>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-[var(--muted)]">Contato</h2>
          {!editing && (
            <button onClick={() => setEditing(true)} className="text-sm text-[var(--brand)]">
              Editar
            </button>
          )}
        </div>
        {editing ? (
          <form onSubmit={save} className="space-y-3">
            <Field label="E-mail">
              <Input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
            </Field>
            <Field label="Telefone">
              <Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} />
            </Field>
            <Field label="WhatsApp">
              <Input value={form.whatsapp} onChange={(e) => setForm({ ...form, whatsapp: e.target.value })} />
            </Field>
            {msg && <p className="text-sm text-[var(--brand)]">{msg}</p>}
            <div className="flex gap-2">
              <Button type="submit" disabled={saving} className="flex-1">
                <Save className="h-4 w-4" /> {saving ? "Salvando..." : "Salvar"}
              </Button>
              <Button type="button" variant="outline" onClick={() => setEditing(false)}>
                Cancelar
              </Button>
            </div>
          </form>
        ) : (
          <dl className="space-y-2 text-sm">
            <Row label="E-mail" value={member.email ?? "-"} />
            <Row label="Telefone" value={member.phone ?? "-"} />
            <Row label="WhatsApp" value={member.whatsapp ?? "-"} />
          </dl>
        )}
      </Card>

      <Card>
        <h2 className="mb-2 text-sm font-semibold text-[var(--muted)]">Minha família</h2>
        {families.length === 0 ? (
          <p className="text-sm text-[var(--muted)]">Nenhuma família vinculada.</p>
        ) : (
          <ul className="space-y-1 text-sm">
            {families.map((f) => (
              <li key={f.id} className="flex items-center justify-between">
                <span>{f.name}</span>
                {f.code && <span className="tnum font-mono text-xs text-[var(--muted)]">{f.code}</span>}
              </li>
            ))}
          </ul>
        )}
      </Card>

      <Link href="/frequencia" className="block">
        <Card className="flex items-center justify-between">
          <span className="flex items-center gap-2 text-sm font-medium">
            <Activity className="h-4 w-4 text-[var(--brand)]" /> Minha frequência
          </span>
          <ChevronRight className="h-4 w-4 text-[var(--muted)]" />
        </Card>
      </Link>
    </div>
  );
}

function Row({ label, value }: { label: string; value?: string | null }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <dt className="text-[var(--muted)]">{label}</dt>
      <dd>{value || "-"}</dd>
    </div>
  );
}
