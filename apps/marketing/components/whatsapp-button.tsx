import { MessageCircle } from "lucide-react";
import { whatsappLink, whatsappMessage } from "@/lib/site";

/** Botao flutuante de WhatsApp, sempre acessivel durante a rolagem. */
export function WhatsAppButton() {
  return (
    <a
      href={whatsappLink(whatsappMessage)}
      target="_blank"
      rel="noopener noreferrer"
      aria-label="Falar no WhatsApp"
      className="group fixed bottom-5 right-5 z-40 inline-flex items-center gap-2 rounded-full bg-emerald-500 py-3.5 pl-4 pr-5 text-sm font-semibold text-white shadow-xl shadow-emerald-600/30 transition-transform hover:scale-105"
    >
      <span className="relative flex h-5 w-5 items-center justify-center">
        <span className="absolute inline-flex h-full w-full rounded-full bg-white/60 animate-pulse-ring" />
        <MessageCircle className="relative h-5 w-5" />
      </span>
      <span className="hidden sm:inline">WhatsApp</span>
    </a>
  );
}
