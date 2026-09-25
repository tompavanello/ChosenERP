"use client";

import { useState } from "react";
import { CheckCircle2, Loader2, Send } from "lucide-react";

type Status = "idle" | "sending" | "success" | "error";

const sizes = [
  "Até 100 membros",
  "100 a 300 membros",
  "300 a 1.000 membros",
  "Mais de 1.000 membros",
  "Múltiplas igrejas / convenção",
];

const inputClass =
  "w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2.5 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-4 focus:ring-sky-500/15 placeholder:text-slate-400";

export function LeadForm() {
  const [status, setStatus] = useState<Status>("idle");
  const [error, setError] = useState("");

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const data = Object.fromEntries(new FormData(form).entries());

    // Armadilha anti-bot: campo oculto que humanos não preenchem.
    if (String(data.company ?? "").trim() !== "") {
      setStatus("success");
      return;
    }

    setStatus("sending");
    setError("");
    try {
      const res = await fetch("/api/v1/public/leads", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: data.name,
          email: data.email,
          phone: data.phone,
          church_name: data.church_name,
          church_size: data.church_size,
          message: data.message,
          source: "site",
        }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body?.error ?? "Não foi possível enviar agora.");
      }
      form.reset();
      setStatus("success");
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível enviar agora. Tente novamente.",
      );
      setStatus("error");
    }
  }

  if (status === "success") {
    return (
      <div className="flex flex-col items-center justify-center rounded-2xl border border-emerald-200 bg-emerald-50 p-10 text-center">
        <CheckCircle2 className="h-12 w-12 text-emerald-500" />
        <h3 className="mt-4 text-xl font-semibold text-slate-900">
          Recebemos o seu contato!
        </h3>
        <p className="mt-2 max-w-sm text-sm text-slate-600">
          Um especialista do Chosen ERP vai falar com você em breve para agendar
          a demonstração.
        </p>
      </div>
    );
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4" noValidate>
      <div className="grid gap-4 sm:grid-cols-2">
        <div>
          <label htmlFor="name" className="mb-1.5 block text-sm font-medium text-slate-700">
            Seu nome *
          </label>
          <input
            id="name"
            name="name"
            required
            autoComplete="name"
            placeholder="Como podemos te chamar?"
            className={inputClass}
          />
        </div>
        <div>
          <label htmlFor="email" className="mb-1.5 block text-sm font-medium text-slate-700">
            E-mail *
          </label>
          <input
            id="email"
            name="email"
            type="email"
            required
            autoComplete="email"
            placeholder="voce@igreja.com"
            className={inputClass}
          />
        </div>
        <div>
          <label htmlFor="phone" className="mb-1.5 block text-sm font-medium text-slate-700">
            WhatsApp
          </label>
          <input
            id="phone"
            name="phone"
            type="tel"
            autoComplete="tel"
            placeholder="(00) 00000-0000"
            className={inputClass}
          />
        </div>
        <div>
          <label htmlFor="church_name" className="mb-1.5 block text-sm font-medium text-slate-700">
            Nome da igreja
          </label>
          <input
            id="church_name"
            name="church_name"
            autoComplete="organization"
            placeholder="Ex.: Igreja Videira"
            className={inputClass}
          />
        </div>
      </div>

      <div>
        <label htmlFor="church_size" className="mb-1.5 block text-sm font-medium text-slate-700">
          Tamanho da igreja
        </label>
        <select id="church_size" name="church_size" className={inputClass} defaultValue="">
          <option value="" disabled>
            Selecione uma opção
          </option>
          {sizes.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label htmlFor="message" className="mb-1.5 block text-sm font-medium text-slate-700">
          Como podemos ajudar?
        </label>
        <textarea
          id="message"
          name="message"
          rows={3}
          placeholder="Conte um pouco sobre a sua necessidade (opcional)."
          className={inputClass}
        />
      </div>

      {/* honeypot */}
      <input
        type="text"
        name="company"
        tabIndex={-1}
        autoComplete="off"
        aria-hidden="true"
        className="hidden"
      />

      {status === "error" && (
        <p className="rounded-lg bg-rose-50 px-3 py-2 text-sm text-rose-700">
          {error}
        </p>
      )}

      <button
        type="submit"
        disabled={status === "sending"}
        className="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-sky-600 px-6 py-3.5 text-base font-semibold text-white shadow-lg shadow-sky-600/25 transition-colors hover:bg-sky-700 disabled:opacity-70"
      >
        {status === "sending" ? (
          <>
            <Loader2 className="h-5 w-5 animate-spin" />
            Enviando...
          </>
        ) : (
          <>
            <Send className="h-5 w-5" />
            Quero uma demonstração
          </>
        )}
      </button>
      <p className="text-center text-xs text-slate-400">
        Seus dados são usados apenas para contato. Sem spam.
      </p>
    </form>
  );
}
