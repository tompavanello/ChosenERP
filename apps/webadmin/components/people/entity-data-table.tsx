"use client";

import { useState, type ReactNode } from "react";
import { Search } from "lucide-react";
import { DataTable, type Column } from "@/components/ui/data-table";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { SkeletonRows } from "@/components/ui/skeleton";
import { StatCard } from "@/components/ui/stat-card";

export interface StatCardDef {
  label: string;
  value: string;
  icon?: React.ElementType;
  tone?: "sky" | "green" | "red" | "amber" | "zinc" | "pink" | "indigo";
  hint?: string;
}

export interface EntityDataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange?: (size: number) => void;
  pageSizeOptions?: number[];
  onSearch?: (q: string) => void;
  searchPlaceholder?: string;
  searchValue?: string;
  filters?: ReactNode;
  statCards?: StatCardDef[];
  mobileCard?: (row: T) => ReactNode;
  rowActions?: (row: T) => ReactNode;
  emptyTitle?: string;
  emptyDescription?: string;
  emptyAction?: ReactNode;
  loading?: boolean;
  emptyIcon?: ReactNode;
  pageSizeOptionsDefault?: number[];
}

const DEFAULT_PAGE_SIZES = [10, 15, 25, 50];

export function EntityDataTable<T>({
  columns,
  data,
  total,
  page,
  pageSize,
  onPageChange,
  onPageSizeChange,
  pageSizeOptions,
  onSearch,
  searchPlaceholder = "Buscar...",
  searchValue,
  filters,
  statCards,
  mobileCard,
  rowActions,
  emptyTitle = "Nenhum registro encontrado",
  emptyDescription = "Tente ajustar os filtros ou cadastre um novo registro.",
  emptyAction,
  loading,
  emptyIcon,
  pageSizeOptionsDefault = DEFAULT_PAGE_SIZES,
}: EntityDataTableProps<T>) {
  const [localSearch, setLocalSearch] = useState(searchValue ?? "");

  const effectiveSearch = onSearch ? localSearch : undefined;

  function handleSearch(e: React.ChangeEvent<HTMLInputElement>) {
    setLocalSearch(e.target.value);
    onSearch?.(e.target.value);
  }

  const handlePageSizeChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const size = Number(e.target.value);
    onPageSizeChange?.(size);
  };

  return (
    <div className="space-y-4">
      {statCards && statCards.length > 0 && (
        <div className="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {statCards.map((s) => (
            <StatCard
              key={s.label}
              label={s.label}
              value={s.value}
              icon={s.icon}
              tone={s.tone}
              hint={s.hint}
            />
          ))}
        </div>
      )}

      <Card className="p-3">
        <div className="flex flex-wrap items-center gap-2">
          {onSearch !== undefined && (
            <div className="relative flex-1 min-w-[200px]">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-zinc-400" />
              <Input
                className="h-8 pl-8 text-sm"
                placeholder={searchPlaceholder}
                value={localSearch}
                onChange={handleSearch}
              />
            </div>
          )}
          {filters}
          <select
            className="h-8 w-16 text-sm"
            value={pageSize}
            onChange={handlePageSizeChange}
          >
            {(pageSizeOptions ?? pageSizeOptionsDefault).map((sz) => (
              <option key={sz} value={sz}>{sz}</option>
            ))}
          </select>
        </div>
      </Card>

      {loading ? (
        <SkeletonRows rows={5} />
      ) : (
        <DataTable
          columns={columns}
          data={data}
          keyExtractor={(row) => (row as { id: string }).id}
          renderMobileCard={mobileCard ?? undefined}
          emptyMessage={emptyTitle}
          emptyDescription={emptyDescription}
          emptyAction={emptyAction}
          emptyIcon={emptyIcon ?? undefined}
          pagination={{
            page,
            pageSize,
            total,
            onPageChange,
            onPageSizeChange,
            pageSizeOptions: pageSizeOptions ?? pageSizeOptionsDefault,
            serverSide: true,
          }}
          hoverable
          striped
        />
      )}
    </div>
  );
}
