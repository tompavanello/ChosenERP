"use client";

import { forwardRef, useMemo, type HTMLAttributes, type ReactNode } from "react";
import { ChevronUp, ChevronDown, ChevronLeft, ChevronRight, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { Skeleton, SkeletonRows, EmptyState } from "./skeleton";
import { Badge, type Tone } from "./badge";
import { Pagination } from "./pagination";

export interface Column<T> {
  key: string;
  label: string;
  sortable?: boolean;
  width?: string;
  align?: "left" | "center" | "right";
  render?: (row: T, index: number) => ReactNode;
  className?: string;
  headerClassName?: string;
  // Valor usado na ordenacao quando `key` nao existe na linha ou nao ordena bem
  // (ex.: idade derivada de birth_date). Sem ele a ordenacao usa row[key].
  sortValue?: (row: T) => string | number | null | undefined;
}

// Compara numeros como numeros e o resto como texto - `localeCompare` puro
// ordenava "10" antes de "9".
function compareValues(a: unknown, b: unknown): number {
  if (a == null && b == null) return 0;
  if (a == null) return -1;
  if (b == null) return 1;

  const na = typeof a === "number" ? a : Number(a);
  const nb = typeof b === "number" ? b : Number(b);
  if (!Number.isNaN(na) && !Number.isNaN(nb)) return na - nb;

  return String(a).localeCompare(String(b), "pt-BR", { numeric: true, sensitivity: "base" });
}

export interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  keyExtractor: (row: T) => string;
  loading?: boolean;
  emptyMessage?: string;
  emptyDescription?: string;
  emptyAction?: ReactNode;
  emptyIcon?: ReactNode;
  onRowClick?: (row: T) => void;
  pagination?: {
    page: number;
    pageSize: number;
    total: number;
    onPageChange: (page: number) => void;
    onPageSizeChange?: (size: number) => void;
    pageSizeOptions?: number[];
    serverSide?: boolean;
  };
  sort?: {
    column: string;
    direction: "asc" | "desc";
    onSort: (column: string) => void;
  };
  rowClassName?: (row: T, index: number) => string;
  hoverable?: boolean;
  striped?: boolean;
  compact?: boolean;
  showIndex?: boolean;
  selection?: {
    selectedKeys: Set<string>;
    onSelectionChange: (keys: Set<string>) => void;
    keyExtractor: (row: T) => string;
  };
  renderMobileCard?: (row: T, index: number) => ReactNode;
  className?: string;
}

// Devolve o CONTEUDO do <th> (texto ou botao de ordenacao), nunca um <th>:
// o elemento e criado pelo DataTable, e um <th> dentro de outro <th> e HTML
// invalido - o React acusa "In HTML, <th> cannot be a child of <th>" e quebra
// a hidratacao.
function SortableHeader<T>({
  column,
  sort,
  onSort
}: {
  column: Column<T>;
  sort?: DataTableProps<T>["sort"];
  onSort?: (col: string) => void;
}) {
  if (!column.sortable || !onSort) {
    return <>{column.label}</>;
  }

  const isActive = sort?.column === column.key;
  const direction = isActive ? sort.direction : null;

  return (
    <button
      type="button"
      onClick={() => onSort(column.key)}
      className="flex cursor-pointer items-center gap-1 font-semibold transition-colors hover:text-zinc-700 dark:hover:text-zinc-200"
    >
      {column.label}
      {isActive && (
        <span className="flex flex-col">
          {direction === "asc" && <ChevronUp className="h-3.5 w-3.5" />}
          {direction === "desc" && <ChevronDown className="h-3.5 w-3.5" />}
        </span>
      )}
    </button>
  );
}

export function DataTable<T>({  columns,
  data,
  keyExtractor,
  loading = false,
  emptyMessage = "Nenhum registro encontrado",
  emptyDescription,
  emptyAction,
  emptyIcon,
  onRowClick,
  pagination,
  sort,
  rowClassName,
  hoverable = true,
  striped = false,
  compact = false,
  showIndex = false,
  selection,
  renderMobileCard,
  className,
}: DataTableProps<T>) {
  const sortedData = useMemo(() => {
    if (!sort?.column) return data;
    const col = columns.find((c) => c.key === sort.column);
    return [...data].sort((a, b) => {
      const aVal = col?.sortValue ? col.sortValue(a) : a[sort.column as keyof T];
      const bVal = col?.sortValue ? col.sortValue(b) : b[sort.column as keyof T];
      const dir = sort.direction === "asc" ? 1 : -1;
      return compareValues(aVal, bVal) * dir;
    });
  }, [data, sort, columns]);

  if (loading) {
    return (
      <div className={cn("space-y-3", className)}>
        <SkeletonRows rows={5} />
        {pagination && (
          <div className="flex justify-center py-4">
            <Loader2 className="h-6 w-6 animate-spin text-sky-500" />
          </div>
        )}
      </div>
    );
  }

  if (sortedData.length === 0) {
    return (
      <div className={cn("py-12", className)}>
        <EmptyState
          icon={emptyIcon}
          title={emptyMessage}
          description={emptyDescription}
          action={emptyAction}
        />
      </div>
    );
  }

  const pageData = pagination
    ? pagination.serverSide
      ? sortedData
      : sortedData.slice((pagination.page - 1) * pagination.pageSize, pagination.page * pagination.pageSize)
    : sortedData;

  const handleSort = (columnKey: string) => {
    if (sort?.onSort && columns.find(c => c.key === columnKey)?.sortable) {
      sort.onSort(columnKey);
    }
  };

  return (
    <div
      className={cn(
        "overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950",
        className,
      )}
    >
      {/* Tabela. So e escondida no mobile quando existe card equivalente; sem
          renderMobileCard ela continua visivel e rola na horizontal, em vez de
          a pagina ficar em branco abaixo de 1024px. */}
      <div className={cn("overflow-x-auto", renderMobileCard && "hidden lg:block")}>
        <table className="w-full text-[13px]" role="grid">
          <thead className="sticky top-0 z-10 border-b border-zinc-200 bg-zinc-50/95 backdrop-blur dark:border-zinc-800 dark:bg-zinc-900/95">
            <tr>
              {selection && (
                <th className="w-10 px-2 py-2">
                  <input
                    type="checkbox"
                    checked={data.length > 0 && data.every(row => selection.selectedKeys.has(selection.keyExtractor(row)))}
                    onChange={() => {
                      if (data.length > 0 && data.every(row => selection.selectedKeys.has(selection.keyExtractor(row)))) {
                        selection.onSelectionChange(new Set());
                      } else {
                        selection.onSelectionChange(new Set(data.map(selection.keyExtractor)));
                      }
                    }}
                    className="rounded border-zinc-300 text-sky-600 focus:ring-sky-500"
                  />
                </th>
              )}
              {showIndex && <th className="w-8 px-2 py-2 text-left text-[10px] font-semibold uppercase tracking-wider text-zinc-400">#</th>}
              {columns.map((column) => (
                <th
                  key={column.key}
                  className={cn(
                    "whitespace-nowrap px-2 py-2 text-left text-[10px] font-semibold uppercase tracking-wider text-zinc-500",
                    column.align === "center" && "text-center",
                    column.align === "right" && "text-right",
                    column.width,
                    column.headerClassName
                  )}
                  scope="col"
                >
                  <SortableHeader column={column} sort={sort} onSort={handleSort} />
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800/70">
            {pageData.map((row, index) => {
              const rowKey = keyExtractor(row);
              const isSelected = selection?.selectedKeys.has(rowKey);

              return (
                <tr
                  key={rowKey}
                  className={cn(
                    "transition-colors",
                    hoverable && "hover:bg-zinc-50/80 dark:hover:bg-zinc-900/60",
                    striped && index % 2 === 1 && "bg-zinc-50/50 dark:bg-zinc-900/30",
                    isSelected && "bg-sky-50 dark:bg-sky-500/10",
                    rowClassName?.(row, index)
                  )}
                  onClick={() => onRowClick?.(row)}
                  style={{ cursor: onRowClick ? "pointer" : "default" }}
                >
                  {selection && (
                    <td className="px-2 py-1.5">
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={(e) => {
                          const newKeys = new Set(selection.selectedKeys);
                          if (e.target.checked) {
                            newKeys.add(rowKey);
                          } else {
                            newKeys.delete(rowKey);
                          }
                          selection.onSelectionChange(newKeys);
                        }}
                        onClick={(e) => e.stopPropagation()}
                        className="rounded border-zinc-300 text-sky-600 focus:ring-sky-500"
                      />
                    </td>
                  )}
                  {showIndex && (
                    <td className={cn("px-2 py-1.5 text-zinc-400", compact && "py-1")}>
                      {(pagination ? (pagination.page - 1) * pagination.pageSize : 0) + index + 1}
                    </td>
                  )}
                  {columns.map((column) => (
                    <td
                      key={column.key}
                      className={cn(
                        "px-2 py-1.5 align-middle",
                        compact && "py-1",
                        column.align === "center" && "text-center",
                        column.align === "right" && "text-right",
                        column.className
                      )}
                    >
                     {column.render
                       ? column.render(row, index)
                       : String(row[column.key as keyof T] ?? "-")}
                    </td>
                  ))}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {/* Mobile: cards */}
      {renderMobileCard && (
        <div className="lg:hidden">
          {pageData.map((row, index) => (
            <div key={keyExtractor(row)} className="border-b border-zinc-100 last:border-0 dark:border-zinc-800/70">
              {renderMobileCard(row, index)}
            </div>
          ))}
        </div>
      )}

      {pagination && (
        <Pagination
          page={pagination.page}
          total={pagination.total}
          perPage={pagination.pageSize}
          onChange={pagination.onPageChange}
        />
      )}
    </div>
  );
}

// Helper columns para uso comum
export function createStatusColumn<T>(key: string): Column<T> {
  return {
    key,
    label: "Status",
    width: "w-28",
    render: (row) => {
      const status = row[key as keyof T] as string;
      const statusMap: Record<string, { label: string; tone: Tone }> = {
        active: { label: "Ativo", tone: "green" },
        member: { label: "Membro", tone: "green" },
        inactive: { label: "Inativo", tone: "zinc" },
        visitor: { label: "Visitante", tone: "amber" },
        converted: { label: "Convertido", tone: "green" },
        welcome: { label: "Boas-vindas", tone: "zinc" },
        coffee_pastor: { label: "Cafe c/ Pastor", tone: "sky" },
        course: { label: "Curso", tone: "sky" },
        cell: { label: "Celula", tone: "indigo" },
        transferred: { label: "Transferido", tone: "sky" },
        deceased: { label: "Falecido", tone: "red" },
      };
      const config = statusMap[status] ?? { label: status || "-", tone: "zinc" as Tone };
      return <Badge tone={config.tone}>{config.label}</Badge>;
    },
  };
}

export function createPersonTypeColumn<T>(key: string): Column<T> {
  return {
    key,
    label: "Tipo",
    width: "w-28",
    render: (row) => {
      const type = row[key as keyof T] as string;
      const toneMap: Record<string, Tone> = {
        member: "sky",
        visitor: "amber",
        benefactor: "pink",
        family: "indigo",
      };
      const tone = toneMap[type] ?? "zinc" as Tone;
      return (
        <Badge tone={tone} variant="outline">
          {type.charAt(0).toUpperCase() + type.slice(1)}
        </Badge>
      );
    },
  };
}

export function createFinancialTypeColumn<T>(key: string): Column<T> {
  return {
    key,
    label: "Tipo",
    width: "w-24",
    render: (row) => {
      const type = row[key as keyof T] as string;
      const tone: Tone = type === "income" ? "green" : "red";
      return <Badge tone={tone}>{type === "income" ? "Entrada" : "Saida"}</Badge>;
    },
  };
}
