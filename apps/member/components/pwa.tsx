"use client";

import { useEffect, useState } from "react";
import { Download, Share, WifiOff, X } from "lucide-react";

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
}

const DISMISS_KEY = "chosen_member_install_dismissed";

function isStandalone(): boolean {
  if (typeof window === "undefined") return false;
  const nav = window.navigator as Navigator & { standalone?: boolean };
  return window.matchMedia("(display-mode: standalone)").matches || nav.standalone === true;
}

function isIos(): boolean {
  if (typeof navigator === "undefined") return false;
  return /iphone|ipad|ipod/i.test(navigator.userAgent);
}

/** Registra o service worker, oferece a instalacao e avisa quando offline. */
export function Pwa() {
  const [deferred, setDeferred] = useState<BeforeInstallPromptEvent | null>(null);
  const [showInstall, setShowInstall] = useState(false);
  const [iosHint, setIosHint] = useState(false);
  const [offline, setOffline] = useState(false);

  useEffect(() => {
    if ("serviceWorker" in navigator) {
      const register = () => navigator.serviceWorker.register("/sw.js").catch(() => {});
      if (document.readyState === "complete") register();
      else window.addEventListener("load", register, { once: true });
    }

    const onPrompt = (e: Event) => {
      e.preventDefault();
      setDeferred(e as BeforeInstallPromptEvent);
      if (localStorage.getItem(DISMISS_KEY) !== "1" && !isStandalone()) setShowInstall(true);
    };
    window.addEventListener("beforeinstallprompt", onPrompt);

    const sync = () => setOffline(!navigator.onLine);
    sync();
    window.addEventListener("online", sync);
    window.addEventListener("offline", sync);

    // iOS nao dispara beforeinstallprompt: mostramos a dica manual.
    if (isIos() && !isStandalone() && localStorage.getItem(DISMISS_KEY) !== "1") {
      setIosHint(true);
    }

    return () => {
      window.removeEventListener("beforeinstallprompt", onPrompt);
      window.removeEventListener("online", sync);
      window.removeEventListener("offline", sync);
    };
  }, []);

  function dismiss() {
    localStorage.setItem(DISMISS_KEY, "1");
    setShowInstall(false);
    setIosHint(false);
  }

  async function install() {
    if (!deferred) return;
    await deferred.prompt();
    await deferred.userChoice;
    setDeferred(null);
    setShowInstall(false);
  }

  return (
    <>
      {offline && (
        <div className="fixed inset-x-0 top-0 z-50 flex items-center justify-center gap-1.5 bg-amber-500 px-4 py-1.5 text-center text-xs font-medium text-white">
          <WifiOff className="h-3.5 w-3.5" /> Sem conexão
        </div>
      )}

      {(showInstall || iosHint) && (
        <div className="fixed inset-x-0 bottom-0 z-50 mx-auto flex max-w-md items-start gap-3 border-t border-[var(--line)] bg-[var(--card)] p-4 shadow-lg">
          <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-[var(--brand-soft)] text-[var(--brand-strong)]">
            {iosHint ? <Share className="h-5 w-5" /> : <Download className="h-5 w-5" />}
          </span>
          <div className="flex-1">
            <p className="text-sm font-semibold">Instalar o app</p>
            {iosHint ? (
              <p className="mt-0.5 text-xs text-[var(--muted)]">
                Toque em <strong>Compartilhar</strong> e depois em <strong>Adicionar à Tela de Início</strong>.
              </p>
            ) : (
              <p className="mt-0.5 text-xs text-[var(--muted)]">
                Acesse a agenda, avisos e oração direto da tela inicial.
              </p>
            )}
            {!iosHint && (
              <button onClick={install} className="mt-2 text-sm font-medium text-[var(--brand)]">
                Instalar agora
              </button>
            )}
          </div>
          <button onClick={dismiss} aria-label="Agora não" className="text-[var(--muted)]">
            <X className="h-4 w-4" />
          </button>
        </div>
      )}
    </>
  );
}
