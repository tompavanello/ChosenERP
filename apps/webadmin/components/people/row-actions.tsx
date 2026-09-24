"use client";

import { MoreHorizontal } from "lucide-react";
import { Dropdown, type DropdownItem } from "@/components/ui/dropdown";
import { Button, type ButtonProps } from "@/components/ui/button";

export interface RowActionItem {
  label: string;
  icon?: React.ReactNode;
  iconSrc?: React.ElementType;
  danger?: boolean;
  onClick: () => void;
  disabled?: boolean;
}

export function RowActions({
  items,
}: {
  items: RowActionItem[];
}) {
  const dropdownItems: DropdownItem[] = items.map((it) => ({
    label: it.label,
    icon: it.icon,
    danger: it.danger,
    disabled: it.disabled,
    onClick: it.onClick,
  }));
  return (
    <Dropdown
      triggerAsChild
      trigger={
        // Visivel sempre (nao so no hover da linha): o gatilho escondido por
        // `opacity-0 group-hover:*` dependia de um `group` que a linha nao tem -
        // as acoes ficavam invisiveis, e em telas de toque nao ha hover.
        <Button type="button" variant="ghost" size="sm" className="h-7 w-7 px-1 text-zinc-400 opacity-70 hover:opacity-100">
          <MoreHorizontal className="h-4 w-4" />
          <span className="sr-only">Acoes do membro</span>
        </Button>
      }
      items={dropdownItems}
    />
  );
}

export function ActionButton({ icon: Icon, label, onClick, tooltip, variant = "ghost", size = "sm" }: {
  icon: React.ElementType;
  label?: string;
  onClick: () => void;
  tooltip?: string;
  variant?: ButtonProps["variant"];
  size?: ButtonProps["size"];
}) {
  return (
    <Button variant={variant} size={size} onClick={onClick} title={tooltip} className="h-8 px-2">
      <Icon className="h-3.5 w-3.5" />
      {label && <span className="ml-1 text-xs">{label}</span>}
    </Button>
  );
}
