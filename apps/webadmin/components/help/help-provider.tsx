"use client";

import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { HelpDrawer } from "./help-drawer";
import { TourLibrary } from "./tour-library";

export type HelpTab = "guide" | "search" | "ai" | "tickets";
export type HelpArticleTarget = { slug: string; title: string } | null;

interface HelpContextValue {
  open: boolean;
  tab: HelpTab;
  article: HelpArticleTarget;
  tourOpen: boolean;
  openHelp: (tab?: HelpTab, article?: HelpArticleTarget) => void;
  closeHelp: () => void;
  setTab: (tab: HelpTab) => void;
  setArticle: (article: HelpArticleTarget) => void;
  openTours: () => void;
  closeTours: () => void;
}

const HelpContext = createContext<HelpContextValue | null>(null);

export function useHelp() {
  const ctx = useContext(HelpContext);
  if (!ctx) throw new Error("useHelp deve estar dentro de <HelpProvider>");
  return ctx;
}

export function HelpProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const [tab, setTab] = useState<HelpTab>("guide");
  const [article, setArticle] = useState<HelpArticleTarget>(null);
  const [tourOpen, setTourOpen] = useState(false);

  const openHelp = useCallback((nextTab: HelpTab = "guide", nextArticle: HelpArticleTarget = null) => {
    window.dispatchEvent(new CustomEvent("chosen:end-tour"));
    setTab(nextTab);
    setArticle(nextArticle);
    setOpen(true);
  }, []);
  const closeHelp = useCallback(() => {
    setOpen(false);
    setArticle(null);
  }, []);
  const openTours = useCallback(() => {
    // Evita sobreposicao: encerra um tour em andamento e fecha a gaveta.
    window.dispatchEvent(new CustomEvent("chosen:end-tour"));
    setOpen(false);
    setArticle(null);
    setTourOpen(true);
  }, []);
  const closeTours = useCallback(() => setTourOpen(false), []);

  const value = useMemo(
    () => ({ open, tab, article, tourOpen, openHelp, closeHelp, setTab, setArticle, openTours, closeTours }),
    [open, tab, article, tourOpen, openHelp, closeHelp, openTours, closeTours],
  );

  return (
    <HelpContext.Provider value={value}>
      {children}
      <HelpDrawer />
      <TourLibrary />
    </HelpContext.Provider>
  );
}
