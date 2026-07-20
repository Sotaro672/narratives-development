// frontend/console/list/src/application/listManagementService.tsx
import React from "react";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../shell/src/layout/List/List";
import type { ListStatus } from "../../shell/src/shared/types/list";
import { fetchListsHTTP } from "../infrastructure/repository";
import { safeDateTimeLabelJa } from "../../shell/src/shared/util/dateJa";

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

export type FilterOptions = {
  titleOptions: Array<{ value: string; label: string }>;
  productOptions: Array<{ value: string; label: string }>;
  tokenOptions: Array<{ value: string; label: string }>;
  managerOptions: Array<{ value: string; label: string }>;
  statusOptions: Array<{ value: ListStatus; label: string }>;
};

export type Filters = {
  titleFilter: string[];
  productFilter: string[];
  tokenFilter: string[];
  managerFilter: string[];
  statusFilter: string[];
};

export function normalizeStatus(raw: unknown): ListStatus {
  const status = String(raw ?? "").toLowerCase();

  if (status === "listing") return "listing";
  if (status === "suspended") return "suspended";

  return "suspended";
}

export function getStatusLabelJP(status: ListStatus): string {
  if (status === "listing") return "出品中";
  return "保留中";
}

export function buildStatusBadge(
  status: ListStatus,
): { text: string; className: string } {
  if (status === "listing") {
    return {
      text: "出品中",
      className: "list-status-badge is-active",
    };
  }

  return {
    text: "保留中",
    className: "list-status-badge is-paused",
  };
}

/**
 * DTO -> ViewModel
 * レスポンス仕様を正として使用する。
 */
export function mapAnyToVMRow(x: any): ListManagementRowVM {
  const id = String(x?.id ?? x?.ID ?? "");
  const title = String(x?.title ?? "");
  const productName = String(x?.productName ?? "");
  const tokenName = String(x?.tokenName ?? "");
  const assigneeName = String(x?.assigneeName ?? "") || "未設定";
  const status = normalizeStatus(x?.status);
  const badge = buildStatusBadge(status);
  const createdAtRaw = String(x?.createdAt ?? "");
  const createdAt = safeDateTimeLabelJa(createdAtRaw, "");

  return {
    id: id || "(missing id)",
    title,
    productName,
    tokenName,
    assigneeName,
    status,
    statusLabel: getStatusLabelJP(status),
    createdAt,
    createdAtRaw,
    statusBadgeText: badge.text,
    statusBadgeClass: badge.className,
  };
}

/**
 * 一覧ロード
 * fetchListsHTTP() は items 配列そのものを返す。
 */
export async function loadListManagementRows(): Promise<{
  rows: ListManagementRowVM[];
  error: string | null;
}> {
  try {
    const response = await fetchListsHTTP();
    const items = Array.isArray(response) ? response : [];

    const mapped = items
      .map((x) => mapAnyToVMRow(x as any))
      .filter((row: ListManagementRowVM) => row.id !== "(missing id)");

    return {
      rows: mapped,
      error: null,
    };
  } catch (e: unknown) {
    const error =
      e instanceof Error
        ? String(e.message)
        : String(e ?? "unknown_error");

    return {
      rows: [],
      error,
    };
  }
}

/**
 * Filter options
 */
export function buildFilterOptions(
  rows: ListManagementRowVM[],
): FilterOptions {
  const titleOptions = Array.from(
    new Set(rows.map((row) => row.title)),
  )
    .filter((value) => String(value ?? "") !== "")
    .map((value) => ({
      value,
      label: value,
    }));

  const productOptions = Array.from(
    new Set(rows.map((row) => row.productName)),
  )
    .filter((value) => String(value ?? "") !== "")
    .map((value) => ({
      value,
      label: value,
    }));

  const tokenOptions = Array.from(
    new Set(rows.map((row) => row.tokenName)),
  )
    .filter((value) => String(value ?? "") !== "")
    .map((value) => ({
      value,
      label: value,
    }));

  const managerOptions = Array.from(
    new Set(rows.map((row) => row.assigneeName)),
  )
    .filter((value) => String(value ?? "") !== "")
    .map((value) => ({
      value,
      label: value,
    }));

  const uniqueStatuses = Array.from(
    new Set<ListStatus>(rows.map((row) => row.status)),
  );

  const statusOptions = uniqueStatuses.map((status) => ({
    value: status,
    label: getStatusLabelJP(status),
  }));

  return {
    titleOptions,
    productOptions,
    tokenOptions,
    managerOptions,
    statusOptions,
  };
}

/**
 * フィルタ適用
 */
export function applyFilters(
  rows: ListManagementRowVM[],
  filters: Filters,
): ListManagementRowVM[] {
  return rows.filter(
    (row) =>
      (filters.titleFilter.length === 0 ||
        filters.titleFilter.includes(row.title)) &&
      (filters.productFilter.length === 0 ||
        filters.productFilter.includes(row.productName)) &&
      (filters.tokenFilter.length === 0 ||
        filters.tokenFilter.includes(row.tokenName)) &&
      (filters.managerFilter.length === 0 ||
        filters.managerFilter.includes(row.assigneeName)) &&
      (filters.statusFilter.length === 0 ||
        filters.statusFilter.includes(row.status)),
  );
}

function toTimeMs(value: string): number {
  const date = new Date(String(value ?? ""));
  const time = date.getTime();

  return Number.isFinite(time) ? time : 0;
}

/**
 * ソート（id / createdAt）
 */
export function applySort(
  rows: ListManagementRowVM[],
  activeKey: SortKey,
  direction: "asc" | "desc" | null,
): ListManagementRowVM[] {
  if (!activeKey || !direction) return rows;

  const data = [...rows];

  data.sort((a, b) => {
    if (activeKey === "createdAt") {
      const timeA = toTimeMs(a.createdAtRaw);
      const timeB = toTimeMs(b.createdAtRaw);
      const comparison = timeA - timeB;

      return direction === "asc" ? comparison : -comparison;
    }

    const comparison = a.id.localeCompare(b.id);

    return direction === "asc" ? comparison : -comparison;
  });

  return data;
}

/**
 * ヘッダ生成
 */
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
}): React.ReactNode[] {
  const { options, selected, onChange, sort } = args;

  const onChangeCreatedAt = (
    key: string,
    nextDirection: "asc" | "desc",
  ) => {
    const sortKey: SortKey =
      key === "createdAt" ? "createdAt" : null;

    sort.onChange(sortKey, nextDirection);
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