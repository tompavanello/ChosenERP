"use client";

import { useState, type ReactNode } from "react";
import { Field, Select } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Drawer } from "@/components/ui/modal";
import { PersonForm } from "@/components/people/person-form";
import { RELATION_KINDS } from "@/components/people/person-card";
import { RELATION_LABELS } from "@/lib/constants";
import type { Branch, Family, Person } from "@/lib/api";

export type EntityType = "member" | "visitor" | "benefactor";

export type EntityModal =
  | { kind: "create"; entityType: EntityType }
  | { kind: "edit"; entityType: EntityType; id: string; person: Person }
  | { kind: "convert-visitor"; visitor: Person }
  | { kind: "link-family"; family: Family }
  | { kind: "add-relationship"; member: Person }
  | null;

export interface UnifiedDrawerProps {
  modal: EntityModal;
  onClose: () => void;
  branches?: Branch[];
  membersCache?: Person[];
  onSubmitPerson: (type: EntityType, id: string | undefined, data: Record<string, unknown>) => Promise<void>;
  onSubmitFamily: (name: string) => Promise<void>;
  onAddRelationship: (memberId: string, relateMemberId: string, kind: string) => Promise<void>;
  onLinkFamilyMember: (familyId: string, memberId: string, relateId: string, relation: string) => Promise<void>;
  onConvertVisitor: (visitorId: string, data: Record<string, unknown>) => Promise<void>;
  membersForSelect?: Person[];
  saving?: boolean;
}

export function UnifiedDrawer({
  modal,
  onClose,
  branches = [],
  membersCache = [],
  onSubmitPerson,
  onSubmitFamily,
  onAddRelationship,
  onLinkFamilyMember,
  onConvertVisitor,
  membersForSelect = [],
  saving = false,
}: UnifiedDrawerProps) {
  if (!modal) return null;

  function renderContent(): ReactNode {
    if (!modal) return null;
    switch (modal.kind) {
      case "create":
      case "edit": {
        const type = modal.entityType;
        const title = modal.kind === "edit" ? `Editar ${labelFor(type)}` : `Novo ${labelFor(type)}`;
        return (
          <Drawer open={true} onClose={onClose} title={title}>
            <PersonForm
              entityType={type}
              initial={modal.kind === "edit" ? modal.person : null}
              saving={saving}
              submitLabel={modal.kind === "edit" ? "Salvar alteracoes" : "Salvar"}
              onSubmit={async (data) => {
                await onSubmitPerson(type, modal.kind === "edit" ? modal.id : undefined, data);
              }}
              onCancel={onClose}
            />
          </Drawer>
        );
      }
      case "convert-visitor": {
        const v = modal.visitor;
        return (
          <Drawer open={true} onClose={onClose} title={`Converter ${v.full_name} para membro`}>
            <PersonForm
              entityType="member"
              visitor={{
                full_name: v.full_name,
                email: v.email,
                phone: v.phone,
                whatsapp: v.whatsapp,
              }}
              saving={saving}
              submitLabel="Converter"
              onSubmit={async (data) => {
                await onConvertVisitor(v.id, data);
              }}
              onCancel={onClose}
            />
          </Drawer>
        );
      }
      case "link-family": {
        const f = modal.family;
        return (
          <FamilyMemberDrawer
            family={f}
            members={membersForSelect}
            saving={saving}
            onSubmit={async (memberId, relateId, relation) => {
              await onLinkFamilyMember(f.id, memberId, relateId, relation);
            }}
            onCancel={onClose}
          />
        );
      }
      case "add-relationship": {
        const m = modal.member;
        return (
          <AddRelationshipDrawer
            member={m}
            others={membersCache.filter((p) => p.id !== m.id)}
            saving={saving}
            onSubmit={async (relateId, kind) => {
              await onAddRelationship(m.id, relateId, kind);
            }}
            onCancel={onClose}
          />
        );
      }
      default:
        return null;
    }
  }

  return <>{renderContent()}</>;
}

function labelFor(type: EntityType) {
  return type === "member" ? "Membro" : type === "visitor" ? "Visitante" : "Benfeitor";
}

function FamilyMemberDrawer({
  family,
  members,
  saving,
  onSubmit,
  onCancel,
}: {
  family: Family;
  members: Person[];
  saving: boolean;
  onSubmit: (memberId: string, relateId: string, relation: string) => Promise<void>;
  onCancel: () => void;
}) {
  const [memberId, setMemberId] = useState("");
  const [relateId, setRelateId] = useState("");
  const [relation, setRelation] = useState("relative");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!memberId) return;
    await onSubmit(memberId, relateId, relation);
  }

  return (
    <Drawer open={true} onClose={onCancel} title={`Vincular a ${family.name}`}>
      <form onSubmit={submit} className="space-y-3">
        <Field label="Membro" required>
          <Select className="h-8 text-sm" value={memberId} onChange={(e) => setMemberId(e.target.value)} required>
            <option value="">Selecione...</option>
            {members.map((m) => (<option key={m.id} value={m.id}>{m.full_name}</option>))}
          </Select>
        </Field>
        <Field label="Vinculo com (responsavel, opcional)">
          <Select className="h-8 text-sm" value={relateId} onChange={(e) => setRelateId(e.target.value)}>
            <option value="">-</option>
            {members.filter((m) => m.id !== memberId).map((m) => (<option key={m.id} value={m.id}>{m.full_name}</option>))}
          </Select>
        </Field>
        <Field label="Relacao">
          <Select className="h-8 text-sm" value={relation} onChange={(e) => setRelation(e.target.value)}>
            {Object.entries(RELATION_LABELS).map(([k, v]) => (<option key={k} value={k}>{v}</option>))}
          </Select>
        </Field>
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="ghost" type="button" onClick={onCancel} disabled={saving}>Cancelar</Button>
          <Button type="submit" disabled={saving}>{saving ? " Salvando..." : "Vincular"}</Button>
        </div>
      </form>
    </Drawer>
  );
}

function AddRelationshipDrawer({
  member,
  others,
  saving,
  onSubmit,
  onCancel,
}: {
  member: Person;
  others: Person[];
  saving: boolean;
  onSubmit: (relateId: string, kind: string) => Promise<void>;
  onCancel: () => void;
}) {
  const [relateId, setRelateId] = useState("");
  const [kind, setKind] = useState("spouse");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!relateId) return;
    await onSubmit(relateId, kind);
  }

  return (
    <Drawer open={true} onClose={onCancel} title={`Adicionar vinculo a ${member.full_name}`}>
      <form onSubmit={submit} className="space-y-3">
        <Field label="Pessoa" required>
          <Select className="h-8 text-sm" value={relateId} onChange={(e) => setRelateId(e.target.value)} required>
            <option value="">Selecione...</option>
            {others.map((m) => (<option key={m.id} value={m.id}>{m.full_name}</option>))}
          </Select>
        </Field>
        <Field label="Relacionamento">
          <Select className="h-8 text-sm" value={kind} onChange={(e) => setKind(e.target.value)}>
            {RELATION_KINDS.map((k) => (<option key={k} value={k}>{RELATION_LABELS[k] ?? k}</option>))}
          </Select>
        </Field>
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="ghost" type="button" onClick={onCancel} disabled={saving}>Cancelar</Button>
          <Button type="submit" disabled={saving}>{saving ? " Salvando..." : "Vincular"}</Button>
        </div>
      </form>
    </Drawer>
  );
}
