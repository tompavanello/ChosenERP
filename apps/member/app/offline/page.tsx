"use client";

import { CloudOff, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

export default function OfflinePage() {
  return (
    <main className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-5 py-10">
      <header className="mb-6 flex flex-col items-center text-center">
        <span className="mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-[var(--brand-soft)] text-[var(--brand-strong)]">
          <CloudOff className="h-7 w-7" />
        </span>
        <h1 className="text-lg font-bold">Você está sem conexão</h1>
        <p className="mt-1 text-sm text-[var(--muted)]">
          Não foi possível falar com a sua igreja. Verifique a internet e tente novamente.
        </p>
      </header>

      <Card>
        <Button type="button" className="w-full" onClick={() => window.location.reload()}>
          <RefreshCw className="h-4 w-4" /> Tentar novamente
        </Button>
      </Card>

      <p className="mt-6 text-center text-xs text-[var(--muted)]">Chosen ERP · App do Membro</p>
    </main>
  );
}
