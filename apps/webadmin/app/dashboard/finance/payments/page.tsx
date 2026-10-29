"use client";

import { useCallback, useEffect, useState } from "react";
import { CreditCard, Plug, Plus, ScanLine, Trash2 } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Tabs } from "@/components/ui/tabs";
import { Drawer } from "@/components/ui/modal";
import { Field, Input, Select } from "@/components/ui/input";
import { CurrencyInput } from "@/components/ui/currency-input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { currency, datePt } from "@/lib/format";
import { ACCOUNT_TYPES } from "@/lib/constants";
import {
  listPaymentProducts, createPaymentProduct, updatePaymentProduct, deletePaymentProduct,
  listPaymentCharges, createPaymentCharge, refundPaymentCharge,
  listPaymentGateways, savePaymentGateway,
  listCategories, listAccounts,
  type PaymentProduct, type PaymentCharge, type PaymentGatewayConfig,
  type PaymentProductInput, type Category, type BankAccount,
} from "@/lib/api";

const STATUS_TONE: Record<string, "green" | "red" | "amber" | "zinc" | "sky"> = {
  paid: "green", pending: "amber", overdue: "red", canceled: "zinc", refunded: "sky", failed: "red",
};
const STATUS_LABEL: Record<string, string> = {
  paid: "Pago", pending: "Pendente", overdue: "Vencida", canceled: "Cancelada", refunded: "Estornada", failed: "Falhou",
};
const BILLING_TYPES = [
  { v: "PIX", l: "PIX" },
  { v: "BOLETO", l: "Boleto" },
  { v: "CREDIT_CARD", l: "Cartao" },
];

export default function PaymentMethodsPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("payments.write");
  const [tab, setTab] = useState("charges");

  const [products, setProducts] = useState<PaymentProduct[] | null>(null);
  const [charges, setCharges] = useState<PaymentCharge[] | null>(null);
  const [gateways, setGateways] = useState<PaymentGatewayConfig[] | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [status, setStatus] = useState("");

  const [chargeOpen, setChargeOpen] = useState(false);
  const [productOpen, setProductOpen] = useState(false);
  const [gatewayOpen, setGatewayOpen] = useState(false);
  const [editing, setEditing] = useState<PaymentProduct | null>(null);
  const [saving, setSaving] = useState(false);

  const [chargeForm, setChargeForm] = useState({
    product_id: "", billing_type: "PIX", amount: "", due_date: "",
    payer_name: "", payer_cpf_cnpj: "", payer_email: "", payer_phone: "", description: "",
  });
  const [productForm, setProductForm] = useState({
    name: "", description: "", amount: "", allowed: ["PIX", "BOLETO", "CREDIT_CARD"] as string[],
    category_id: "", account_id: "", is_recurring: false, is_active: true, sort_order: "0",
  });
  const [gwForm, setGwForm] = useState({
    provider: "asaas", environment: "sandbox", api_key: "", base_url: "", webhook_token: "",
    is_active: true, is_default: true,
  });

  const load = useCallback(async () => {
    try {
      const [p, c, g] = await Promise.all([
        listPaymentProducts(), listPaymentCharges(status), listPaymentGateways(),
      ]);
      setProducts(p.products);
      setCharges(c.charges);
      setGateways(g.gateways);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar meios de pagamento", "error");
      setProducts([]);
      setCharges([]);
      setGateways([]);
    }
    // Categorias/contas vem do financeiro (finance.read). Quem so tem
    // payments.read nao deve ter a pagina inteira quebrada por isso.
    if (hasPerm("finance.read")) {
      try {
        const [cat, acc] = await Promise.all([listCategories(), listAccounts()]);
        setCategories(cat.categories);
        setAccounts(acc.accounts);
      } catch {
        /* mantem vazio; produtos ainda podem ser editados sem trocar a categoria */
      }
    }
  }, [status, toast, hasPerm]);

  useEffect(() => { load(); }, [load]);

  const selectedProduct = products?.find((p) => p.id === chargeForm.product_id);

  async function submitCharge(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      await createPaymentCharge({
        product_id: chargeForm.product_id,
        billing_type: chargeForm.billing_type,
        amount_cents: chargeForm.amount ? Math.round(Number(chargeForm.amount) * 100) : undefined,
        due_date: chargeForm.due_date || undefined,
        payer_name: chargeForm.payer_name,
        payer_cpf_cnpj: chargeForm.payer_cpf_cnpj,
        payer_email: chargeForm.payer_email,
        payer_phone: chargeForm.payer_phone,
        description: chargeForm.description,
      });
      toast("Cobranca criada");
      setChargeOpen(false);
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao gerar cobranca", "error");
    } finally {
      setSaving(false);
    }
  }

  function openNewProduct() {
    setEditing(null);
    setProductForm({
      name: "", description: "", amount: "", allowed: ["PIX", "BOLETO", "CREDIT_CARD"],
      category_id: categories[0]?.id ?? "", account_id: accounts.find((a) => a.is_active)?.id ?? "",
      is_recurring: false, is_active: true, sort_order: "0",
    });
    setProductOpen(true);
  }
  function openEditProduct(p: PaymentProduct) {
    setEditing(p);
    setProductForm({
      name: p.name, description: p.description, amount: p.amount_cents ? (p.amount_cents / 100).toFixed(2) : "",
      allowed: p.allowed_billing_types, category_id: p.category_id, account_id: p.account_id ?? "",
      is_recurring: p.is_recurring, is_active: p.is_active, sort_order: String(p.sort_order),
    });
    setProductOpen(true);
  }

  async function submitProduct(e: React.FormEvent) {
    e.preventDefault();
    if (!productForm.account_id) {
      toast("Selecione a conta bancaria do produto.", "error");
      return;
    }
    setSaving(true);
    try {
      const payload: PaymentProductInput = {
        name: productForm.name,
        description: productForm.description,
        amount_cents: productForm.amount ? Math.round(Number(productForm.amount) * 100) : null,
        allowed_billing_types: productForm.allowed,
        category_id: productForm.category_id,
        account_id: productForm.account_id,
        is_recurring: productForm.is_recurring,
        is_active: productForm.is_active,
        sort_order: Number(productForm.sort_order || 0),
      };
      if (editing) await updatePaymentProduct(editing.id, payload);
      else await createPaymentProduct(payload);
      toast(editing ? "Produto atualizado" : "Produto criado");
      setProductOpen(false);
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar produto", "error");
    } finally {
      setSaving(false);
    }
  }

  async function removeProduct(p: PaymentProduct) {
    if (!confirm(`Excluir o produto "${p.name}"?`)) return;
    try {
      await deletePaymentProduct(p.id);
      toast("Produto excluido");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao excluir", "error");
    }
  }

  async function submitGateway(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      await savePaymentGateway(gwForm.provider, {
        environment: gwForm.environment,
        api_key: gwForm.api_key || undefined,
        base_url: gwForm.base_url || undefined,
        webhook_token: gwForm.webhook_token || undefined,
        is_active: gwForm.is_active,
        is_default: gwForm.is_default,
      });
      toast("Gateway salvo");
      setGatewayOpen(false);
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar gateway", "error");
    } finally {
      setSaving(false);
    }
  }

  async function refund(c: PaymentCharge) {
    if (!confirm("Estornar esta cobranca no gateway?")) return;
    try {
      await refundPaymentCharge(c.id);
      toast("Cobranca estornada");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao estornar", "error");
    }
  }

  return (
    <div className="page space-y-4">
      <PageHeader
        title="Meios de pagamento"
        description="Cobranca dos membros via gateway (Asaas). Cada produto define a categoria contabil do lancamento."
        actions={
          canWrite ? (
            tab === "charges" ? (
              <Button onClick={() => { setChargeForm({ product_id: products?.[0]?.id ?? "", billing_type: "PIX", amount: "", due_date: "", payer_name: "", payer_cpf_cnpj: "", payer_email: "", payer_phone: "", description: "" }); setChargeOpen(true); }}>
                <Plus className="h-4 w-4" /> Nova cobranca
              </Button>
            ) : tab === "products" ? (
              <Button onClick={openNewProduct}><Plus className="h-4 w-4" /> Novo produto</Button>
            ) : (
              <Button onClick={() => setGatewayOpen(true)}><Plug className="h-4 w-4" /> Configurar gateway</Button>
            )
          ) : undefined
        }
      />

      <Tabs
        tabs={[
          { key: "charges", label: "Cobrancas", icon: <ScanLine className="h-4 w-4" /> },
          { key: "products", label: "Produtos", icon: <CreditCard className="h-4 w-4" /> },
          { key: "gateway", label: "Gateway", icon: <Plug className="h-4 w-4" /> },
        ]}
        active={tab}
        onChange={setTab}
      />

      {tab === "charges" && (
        <Card className="overflow-hidden p-0">
          <CardHeader className="flex flex-row items-center justify-between px-4 pt-4">
            <CardTitle>Cobrancas</CardTitle>
            <select className="select h-8 w-40 text-sm" value={status} onChange={(e) => setStatus(e.target.value)}>
              <option value="">Todas</option>
              <option value="pending">Pendentes</option>
              <option value="paid">Pagas</option>
              <option value="overdue">Vencidas</option>
            </select>
          </CardHeader>
          {charges === null ? <div className="p-4"><SkeletonRows rows={4} /></div>
            : charges.length === 0 ? <EmptyState icon={<ScanLine className="h-10 w-10" />} title="Sem cobrancas" description="Gere uma cobranca para um membro." />
            : (
            <Table>
              <THead><TRow><TH>Pagador</TH><TH>Status</TH><TH className="text-right">Valor</TH><TH>Forma</TH><TH>Vencimento</TH><TH>Pago em</TH><TH></TH></TRow></THead>
              <TBody>
                {charges.map((c) => (
                  <TRow key={c.id}>
                    <TD className="font-medium">{c.payer_name || "-"}</TD>
                    <TD><Badge tone={STATUS_TONE[c.status] ?? "zinc"}>{STATUS_LABEL[c.status] ?? c.status}</Badge></TD>
                    <TD className="text-right tabular-nums">{currency(c.amount_cents / 100)}</TD>
                    <TD>{c.billing_type}</TD>
                    <TD>{c.due_date ? datePt(c.due_date) : "-"}</TD>
                    <TD>{c.paid_at ? datePt(c.paid_at) : "-"}</TD>
                    <TD className="text-right">
                      <div className="flex justify-end gap-1">
                        {c.invoice_url && <a className="btn-base btn-outline h-7 px-2 text-xs" href={c.invoice_url} target="_blank" rel="noreferrer">Ver</a>}
                        {canWrite && c.status === "paid" && !c.financial_transaction_id && (
                          <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={() => refund(c)}>Estornar</Button>
                        )}
                      </div>
                    </TD>
                  </TRow>
                ))}
              </TBody>
            </Table>
          )}
        </Card>
      )}

      {tab === "products" && (
        <Card className="overflow-hidden p-0">
          {products === null ? <div className="p-4"><SkeletonRows rows={4} /></div>
            : products.length === 0 ? <EmptyState icon={<CreditCard className="h-10 w-10" />} title="Sem produtos" description="Crie produtos como Dizimo, Oferta, Evento." />
            : (
            <Table>
              <THead><TRow><TH>Produto</TH><TH>Categoria</TH><TH className="text-right">Valor</TH><TH>Formas</TH><TH>Recorrente</TH><TH>Situacao</TH><TH></TH></TRow></THead>
              <TBody>
                {products.map((p) => (
                  <TRow key={p.id}>
                    <TD className="font-medium">{p.name}</TD>
                    <TD>{p.category_name || "-"}</TD>
                    <TD className="text-right tabular-nums">{p.amount_cents ? currency(p.amount_cents / 100) : "Aberto"}</TD>
                    <TD className="text-xs text-zinc-500">{p.allowed_billing_types.join(", ")}</TD>
                    <TD>{p.is_recurring ? "Sim" : "Nao"}</TD>
                    <TD><Badge tone={p.is_active ? "green" : "zinc"}>{p.is_active ? "Ativo" : "Inativo"}</Badge></TD>
                    <TD className="text-right">
                      {canWrite && (
                        <div className="flex justify-end gap-1">
                          <Button variant="outline" size="sm" className="h-7 px-2 text-xs" onClick={() => openEditProduct(p)}>Editar</Button>
                          <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={() => removeProduct(p)}><Trash2 className="h-3.5 w-3.5" /></Button>
                        </div>
                      )}
                    </TD>
                  </TRow>
                ))}
              </TBody>
            </Table>
          )}
        </Card>
      )}

      {tab === "gateway" && (
        <Card className="overflow-hidden p-0">
          {gateways === null ? <div className="p-4"><SkeletonRows rows={3} /></div>
            : gateways.length === 0 ? <EmptyState icon={<Plug className="h-10 w-10" />} title="Nenhum gateway" description="Configure o Asaas para receber pagamentos." />
            : (
            <Table>
              <THead><TRow><TH>Provider</TH><TH>Ambiente</TH><TH>Credenciais</TH><TH>Padrao</TH><TH>Situacao</TH><TH></TH></TRow></THead>
              <TBody>
                {gateways.map((g) => (
                  <TRow key={g.id}>
                    <TD className="font-medium capitalize">{g.provider}</TD>
                    <TD>{g.environment}</TD>
                    <TD>{g.has_credentials ? "Configuradas" : "-"}</TD>
                    <TD>{g.is_default ? <Badge tone="sky">Padrao</Badge> : "-"}</TD>
                    <TD><Badge tone={g.is_active ? "green" : "zinc"}>{g.is_active ? "Ativo" : "Inativo"}</Badge></TD>
                    <TD className="text-right">
                      {canWrite && (
                        <Button
                          variant="outline" size="sm" className="h-7 px-2 text-xs"
                          onClick={() => {
                            setGwForm({
                              provider: g.provider, environment: g.environment, api_key: "", base_url: "",
                              webhook_token: "", is_active: g.is_active, is_default: g.is_default,
                            });
                            setGatewayOpen(true);
                          }}
                        >
                          Editar
                        </Button>
                      )}
                    </TD>
                  </TRow>
                ))}
              </TBody>
            </Table>
          )}
        </Card>
      )}

      {/* Drawer: nova cobranca */}
      <Drawer open={chargeOpen} onClose={() => setChargeOpen(false)} title="Nova cobranca">
        <form onSubmit={submitCharge} className="space-y-3">
          <Field label="Produto *">
            <Select className="h-8 text-sm" value={chargeForm.product_id} onChange={(e) => setChargeForm({ ...chargeForm, product_id: e.target.value, billing_type: products?.find((p) => p.id === e.target.value)?.allowed_billing_types?.[0] ?? "PIX" })}>
              <option value="">Selecione...</option>
              {products?.map((p) => <option key={p.id} value={p.id}>{p.name}{p.amount_cents ? ` - ${currency(p.amount_cents / 100)}` : ""}</option>)}
            </Select>
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Forma *">
              <Select className="h-8 text-sm" value={chargeForm.billing_type} onChange={(e) => setChargeForm({ ...chargeForm, billing_type: e.target.value })}>
                {(selectedProduct?.allowed_billing_types ?? ["PIX", "BOLETO", "CREDIT_CARD"]).map((b) => (
                  <option key={b} value={b}>{BILLING_TYPES.find((x) => x.v === b)?.l ?? b}</option>
                ))}
              </Select>
            </Field>
            <Field label="Valor" hint={selectedProduct?.amount_cents ? "Vazio usa o do produto" : "Obrigatorio"}>
              <CurrencyInput className="h-8 text-sm" value={chargeForm.amount} onChange={(v) => setChargeForm({ ...chargeForm, amount: v })} />
            </Field>
            <Field label="Vencimento">
              <Input type="date" className="h-8 text-sm" value={chargeForm.due_date} onChange={(e) => setChargeForm({ ...chargeForm, due_date: e.target.value })} />
            </Field>
            <Field label="Descricao">
              <Input className="h-8 text-sm" value={chargeForm.description} onChange={(e) => setChargeForm({ ...chargeForm, description: e.target.value })} />
            </Field>
            <Field label="Pagador">
              <Input className="h-8 text-sm" value={chargeForm.payer_name} onChange={(e) => setChargeForm({ ...chargeForm, payer_name: e.target.value })} />
            </Field>
            <Field label="CPF/CNPJ">
              <Input className="h-8 text-sm" value={chargeForm.payer_cpf_cnpj} onChange={(e) => setChargeForm({ ...chargeForm, payer_cpf_cnpj: e.target.value })} />
            </Field>
            <Field label="E-mail">
              <Input className="h-8 text-sm" value={chargeForm.payer_email} onChange={(e) => setChargeForm({ ...chargeForm, payer_email: e.target.value })} />
            </Field>
            <Field label="Telefone">
              <Input className="h-8 text-sm" value={chargeForm.payer_phone} onChange={(e) => setChargeForm({ ...chargeForm, payer_phone: e.target.value })} />
            </Field>
          </div>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setChargeOpen(false)}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving || !chargeForm.product_id}>{saving ? "Gerando..." : "Gerar cobranca"}</Button>
          </div>
        </form>
      </Drawer>

      {/* Drawer: produto */}
      <Drawer open={productOpen} onClose={() => setProductOpen(false)} title={editing ? "Editar produto" : "Novo produto"}>
        <form onSubmit={submitProduct} className="space-y-3">
          <Field label="Nome *">
            <Input required className="h-8 text-sm" value={productForm.name} onChange={(e) => setProductForm({ ...productForm, name: e.target.value })} />
          </Field>
          <Field label="Descricao">
            <Input className="h-8 text-sm" value={productForm.description} onChange={(e) => setProductForm({ ...productForm, description: e.target.value })} />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Valor" hint="Vazio = valor aberto">
              <CurrencyInput className="h-8 text-sm" value={productForm.amount} onChange={(v) => setProductForm({ ...productForm, amount: v })} />
            </Field>
            <Field label="Categoria contabil *" hint="Categoria do lancamento ao receber">
              <Select required className="h-8 text-sm" value={productForm.category_id} onChange={(e) => setProductForm({ ...productForm, category_id: e.target.value })}>
                <option value="">Selecione...</option>
                {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </Select>
            </Field>
            <Field label="Conta bancaria *" hint="Conta que recebe os valores do produto">
              <Select
                required
                className="h-8 text-sm"
                value={productForm.account_id}
                onChange={(e) => setProductForm({ ...productForm, account_id: e.target.value })}
              >
                <option value="" disabled>Selecione...</option>
                {accounts.filter((a) => a.is_active || a.id === productForm.account_id).map((a) => (
                  <option key={a.id} value={a.id}>{a.name} - {ACCOUNT_TYPES[a.account_type] ?? a.account_type}{a.is_active ? "" : " (inativa)"}</option>
                ))}
              </Select>
              {accounts.filter((a) => a.is_active).length === 0 && (
                <p className="mt-1 text-xs text-red-600">
                  Cadastre uma conta bancaria ativa em Contas Bancarias.
                </p>
              )}
            </Field>
            <Field label="Ordem">
              <Input type="number" className="h-8 text-sm" value={productForm.sort_order} onChange={(e) => setProductForm({ ...productForm, sort_order: e.target.value })} />
            </Field>
          </div>
          <div className="border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <p className="mb-2 text-[11px] font-semibold uppercase tracking-wide text-zinc-400">Formas aceitas</p>
            <div className="flex gap-4">
              {BILLING_TYPES.map((b) => (
                <label key={b.v} className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    className="h-4 w-4"
                    checked={productForm.allowed.includes(b.v)}
                    onChange={(e) => setProductForm({
                      ...productForm,
                      allowed: e.target.checked ? [...productForm.allowed, b.v] : productForm.allowed.filter((x) => x !== b.v),
                    })}
                  />
                  {b.l}
                </label>
              ))}
            </div>
          </div>
          <div className="flex gap-4">
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" className="h-4 w-4" checked={productForm.is_recurring} onChange={(e) => setProductForm({ ...productForm, is_recurring: e.target.checked })} />
              Permite recorrencia
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" className="h-4 w-4" checked={productForm.is_active} onChange={(e) => setProductForm({ ...productForm, is_active: e.target.checked })} />
              Ativo
            </label>
          </div>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setProductOpen(false)}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving || !productForm.account_id}>
              {saving ? "Salvando..." : "Salvar"}
            </Button>
          </div>
        </form>
      </Drawer>

      {/* Drawer: gateway */}
      <Drawer open={gatewayOpen} onClose={() => setGatewayOpen(false)} title="Configurar gateway">
        <form onSubmit={submitGateway} className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <Field label="Provider *">
              <Select className="h-8 text-sm" value={gwForm.provider} onChange={(e) => setGwForm({ ...gwForm, provider: e.target.value })}>
                <option value="asaas">Asaas</option>
                <option value="mercadopago" disabled>Mercado Pago (em breve)</option>
                <option value="stone" disabled>Stone (em breve)</option>
                <option value="picpay" disabled>PicPay (em breve)</option>
              </Select>
            </Field>
            <Field label="Ambiente">
              <Select className="h-8 text-sm" value={gwForm.environment} onChange={(e) => setGwForm({ ...gwForm, environment: e.target.value })}>
                <option value="sandbox">Sandbox</option>
                <option value="production">Producao</option>
              </Select>
            </Field>
          </div>
          <Field label="API key" hint="Deixe vazio para manter a atual">
            <Input type="password" className="h-8 text-sm" value={gwForm.api_key} onChange={(e) => setGwForm({ ...gwForm, api_key: e.target.value })} />
          </Field>
          <Field label="Base URL" hint="Ex.: https://api.asaas.com/v3">
            <Input className="h-8 text-sm" value={gwForm.base_url} onChange={(e) => setGwForm({ ...gwForm, base_url: e.target.value })} />
          </Field>
          <Field label="Webhook token" hint="Deixe vazio para manter o atual">
            <Input className="h-8 text-sm" value={gwForm.webhook_token} onChange={(e) => setGwForm({ ...gwForm, webhook_token: e.target.value })} />
          </Field>
          <div className="flex gap-4">
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" className="h-4 w-4" checked={gwForm.is_active} onChange={(e) => setGwForm({ ...gwForm, is_active: e.target.checked })} />
              Ativo
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" className="h-4 w-4" checked={gwForm.is_default} onChange={(e) => setGwForm({ ...gwForm, is_default: e.target.checked })} />
              Padrao da igreja
            </label>
          </div>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setGatewayOpen(false)}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}


