// frontend/console/shell/src/features/list/application/listManagementService.tsx

import type { ReactNode } from "react";

import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../layout/List/List";
import { safeDateTimeLabelJa } from "../../../shared/util/dateJa";

import type { ListStatus } from "../domain/list";
import type { ListDTO } from "../infrastructure/dto/listDto";
import { fetchListsHTTP } from "../infrastructure/repository";

export type SortKey = "id" | "createdAt" | null;

export type ListManagementRowVM = {
  id: string;
  title: string;
  productName: string;
  tokenName: string;
  assigneeName: string;
  status: ListStatus;
  statusLabel: string;
  createdAt: string;
  createdAtRaw: string;
  statusBadgeText: string;
  statusBadgeClass: string;
};

type FilterOption = {
  value: string;
  label: string;
};

export type FilterOptions = {
  titleOptions: FilterOption[];
  productOptions: FilterOption[];
  tokenOptions: FilterOption[];
  managerOptions: FilterOption[];
  statusOptions: Array<{
    value: ListStatus;
    label: string;
  }>;
};

export type Filters = {
  titleFilter: string[];
  productFilter: string[];
  tokenFilter: string[];
  managerFilter: string[];
  statusFilter: string[];
};

const LIST_STATUS_PRESENTATION: Record<
  ListStatus,
  {
    label: string;
    badgeClassName: string;
  }
> = {
  listing: {
    label: "出品中",
    badgeClassName: "list-status-badge is-active",
  },
  suspended: {
    label: "保留中",
    badgeClassName: "list-status-badge is-paused",
  },
};

function mapListDTOToVMRow(
  dto: ListDTO,
): ListManagementRowVM {
  const status = dto.status ?? "suspended";
  const statusPresentation =
    LIST_STATUS_PRESENTATION[status];
  const createdAtRaw = dto.createdAt ?? "";

  return {
    id: dto.id,
    title: dto.title ?? "",
    productName: dto.productName ?? "",
    tokenName: dto.tokenName ?? "",
    assigneeName: dto.assigneeName || "未設定",
    status,
    statusLabel: statusPresentation.label,
    createdAt: safeDateTimeLabelJa(createdAtRaw, ""),
    createdAtRaw,
    statusBadgeText: statusPresentation.label,
    statusBadgeClass:
      statusPresentation.badgeClassName,
  };
}

export async function loadListManagementRows(): Promise<{
  rows: ListManagementRowVM[];
  error: string | null;
}> {
  try {
    const items = await fetchListsHTTP();

    return {
      rows: items.map(mapListDTOToVMRow),
      error: null,
    };
  } catch (error: unknown) {
    return {
      rows: [],
      error:
        error instanceof Error
          ? error.message
          : String(error ?? "unknown_error"),
    };
  }
}

function buildTextFilterOptions(
  values: string[],
): FilterOption[] {
  return Array.from(new Set(values))
    .filter((value) => value !== "")
    .map((value) => ({
      value,
      label: value,
    }));
}

export function buildFilterOptions(
  rows: ListManagementRowVM[],
): FilterOptions {
  const statuses = Array.from(
    new Set(rows.map((row) => row.status)),
  );

  return {
    titleOptions: buildTextFilterOptions(
      rows.map((row) => row.title),
    ),
    productOptions: buildTextFilterOptions(
      rows.map((row) => row.productName),
    ),
    tokenOptions: buildTextFilterOptions(
      rows.map((row) => row.tokenName),
    ),
    managerOptions: buildTextFilterOptions(
      rows.map((row) => row.assigneeName),
    ),
    statusOptions: statuses.map((status) => ({
      value: status,
      label: LIST_STATUS_PRESENTATION[status].label,
    })),
  };
}

export function applyFilters(
  rows: ListManagementRowVM[],
  filters: Filters,
): ListManagementRowVM[] {
  return rows.filter(
    (row) =>
      (filters.titleFilter.length === 0 ||
        filters.titleFilter.includes(row.title)) &&
      (filters.productFilter.length === 0 ||
        filters.productFilter.includes(
          row.productName,
        )) &&
      (filters.tokenFilter.length === 0 ||
        filters.tokenFilter.includes(row.tokenName)) &&
      (filters.managerFilter.length === 0 ||
        filters.managerFilter.includes(
          row.assigneeName,
        )) &&
      (filters.statusFilter.length === 0 ||
        filters.statusFilter.includes(row.status)),
  );
}

function toTimeMs(value: string): number {
  const time = new Date(value).getTime();

  return Number.isNaN(time) ? 0 : time;
}

export function applySort(
  rows: ListManagementRowVM[],
  activeKey: SortKey,
  direction: "asc" | "desc" | null,
): ListManagementRowVM[] {
  if (!activeKey || !direction) {
    return rows;
  }

  const sortedRows = [...rows];

  sortedRows.sort((a, b) => {
    if (activeKey === "createdAt") {
      const comparison =
        toTimeMs(a.createdAtRaw) -
        toTimeMs(b.createdAtRaw);

      return direction === "asc"
        ? comparison
        : -comparison;
    }

    const comparison = a.id.localeCompare(b.id);

    return direction === "asc"
      ? comparison
      : -comparison;
  });

  return sortedRows;
}

export function buildHeaders(args: {
  options: FilterOptions;
  selected: Filters;
  onChange: {
    setTitleFilter: (value: string[]) => void;
    setProductFilter: (value: string[]) => void;
    setTokenFilter: (value: string[]) => void;
    setManagerFilter: (value: string[]) => void;
    setStatusFilter: (value: string[]) => void;
  };
  sort: {
    activeKey: SortKey;
    direction: "asc" | "desc" | null;
    onChange: (
      key: SortKey,
      direction: "asc" | "desc" | null,
    ) => void;
  };
}): ReactNode[] {
  const { options, selected, onChange, sort } = args;

  const onChangeCreatedAt = (
    _key: string,
    nextDirection: "asc" | "desc",
  ) => {
    sort.onChange("createdAt", nextDirection);
  };

  return [
    <FilterableTableHeader
      key="title"
      label="タイトル"
      options={options.titleOptions}
      selected={selected.titleFilter}
      onChange={onChange.setTitleFilter}
    />,
    <FilterableTableHeader
      key="product"
      label="プロダクト名"
      options={options.productOptions}
      selected={selected.productFilter}
      onChange={onChange.setProductFilter}
    />,
    <FilterableTableHeader
      key="token"
      label="トークン名"
      options={options.tokenOptions}
      selected={selected.tokenFilter}
      onChange={onChange.setTokenFilter}
    />,
    <FilterableTableHeader
      key="manager"
      label="担当者"
      options={options.managerOptions}
      selected={selected.managerFilter}
      onChange={onChange.setManagerFilter}
    />,
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={options.statusOptions}
      selected={selected.statusFilter}
      onChange={onChange.setStatusFilter}
    />,
    <SortableTableHeader
      key="createdAt"
      label="作成日"
      sortKey="createdAt"
      activeKey={sort.activeKey ?? undefined}
      direction={sort.direction ?? undefined}
      onChange={onChangeCreatedAt}
    />,
  ];
}