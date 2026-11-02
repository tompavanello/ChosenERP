"use client";

import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { HelpSheet } from "./help-sheet";

export type HelpTab = "guide" | "search" | "ai" | "tickets";

interface HelpContextValue {
  open: boolean;
  tab: HelpTab;
  openHelp: (tab?: HelpTab) => void;
  closeHelp: () => void;
  setTab: (tab: HelpTab) => void;
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
  const openHelp = useCallback((t: HelpTab = "guide") => { setTab(t); setOpen(true); }, []);
  const closeHelp = useCallback(() => setOpen(false), []);
  const value = useMemo(() => ({ open, tab, openHelp, closeHelp, setTab }), [open, tab, openHelp, closeHelp]);
  return (
    <HelpContext.Provider value={value}>
      {children}
      <HelpSheet />
    </HelpContext.Provider>
  );
}
