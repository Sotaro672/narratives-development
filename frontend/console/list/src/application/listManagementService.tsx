// frontend/console/list/src/application/listManagementService.tsx

import React from "react";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";

import type { ListStatus } from "../../../shell/src/shared/types/list";

// 分割後のHTTP入口（index.ts 経由）
import { fetchListsHTTP } from "../infrastructure/http/list";
import { safeDateTimeLabelJa } from "../../../shell/src/shared/util/dateJa";

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

  if (status === "list" || status === "listing") return "listing";
  if (status === "hold" || status === "suspended") return "suspended";
  if (status === "deleted") return "deleted";

  return "suspended";
}

export function getStatusLabelJP(status: ListStatus): string {
  if (status === "listing") return "出品中";
  if (status === "suspended") return "保留中";
  return "削除済み";
}

export function buildStatusBadge(
  status: ListStatus,
): { text: string; className: string } {
  if (status === "listing") {
    return { text: "出品中", className: "list-status-badge is-active" };
  }
  if (status === "suspended") {
    return { text: "保留中", className: "list-status-badge is-paused" };
  }
  return { text: "削除済み", className: "list-status-badge is-paused" };
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

  const st = normalizeStatus(x?.status);
  const badge = buildStatusBadge(st);

  const createdAtRaw = String(x?.createdAt ?? "");
  const createdAt = safeDateTimeLabelJa(createdAtRaw, "");

  return {
    id: id || "(missing id)",
    title,
    productName,
    tokenName,
    assigneeName,
    status: st,
    statusLabel: getStatusLabelJP(st),
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
      .filter((r: ListManagementRowVM) => r.id !== "(missing id)");

    return { rows: mapped, error: null };
  } catch (e: unknown) {
    const errMsg =
      e instanceof Error ? String(e.message) : String(e ?? "unknown_error");

    return {
      rows: [],
      error: errMsg,
    };
  }
}

/**
 * Filter options
 */
export function buildFilterOptions(rows: ListManagementRowVM[]): FilterOptions {
  const titleOptions = Array.from(new Set(rows.map((r) => r.title)))
    .filter((v) => String(v ?? "") !== "")
    .map((v) => ({ value: v, label: v }));

  const productOptions = Array.from(new Set(rows.map((r) => r.productName)))
    .filter((v) => String(v ?? "") !== "")
    .map((v) => ({ value: v, label: v }));

  const tokenOptions = Array.from(new Set(rows.map((r) => r.tokenName)))
    .filter((v) => String(v ?? "") !== "")
    .map((v) => ({ value: v, label: v }));

  const managerOptions = Array.from(new Set(rows.map((r) => r.assigneeName)))
    .filter((v) => String(v ?? "") !== "")
    .map((v) => ({ value: v, label: v }));

  const uniqStatus = Array.from(new Set<ListStatus>(rows.map((r) => r.status)));
  const statusOptions = uniqStatus.map((status) => ({
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
  f: Filters,
): ListManagementRowVM[] {
  return rows.filter(
    (r) =>
      (f.titleFilter.length === 0 || f.titleFilter.includes(r.title)) &&
      (f.productFilter.length === 0 || f.productFilter.includes(r.productName)) &&
      (f.tokenFilter.length === 0 || f.tokenFilter.includes(r.tokenName)) &&
      (f.managerFilter.length === 0 || f.managerFilter.includes(r.assigneeName)) &&
      (f.statusFilter.length === 0 || f.statusFilter.includes(r.status)),
  );
}

function toTimeMs(v: string): number {
  const d = new Date(String(v ?? ""));
  const t = d.getTime();
  return Number.isFinite(t) ? t : 0;
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
      const ta = toTimeMs(a.createdAtRaw);
      const tb = toTimeMs(b.createdAtRaw);
      const cmp = ta - tb;
      return direction === "asc" ? cmp : -cmp;
    }

    const cmp = a.id.localeCompare(b.id);
    return direction === "asc" ? cmp : -cmp;
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
    setTitleFilter: (v: string[]) => void;
    setProductFilter: (v: string[]) => void;
    setTokenFilter: (v: string[]) => void;
    setManagerFilter: (v: string[]) => void;
    setStatusFilter: (v: string[]) => void;
  };
  sort: {
    activeKey: SortKey;
    direction: "asc" | "desc" | null;
    onChange: (key: SortKey, dir: "asc" | "desc" | null) => void;
  };
}): React.ReactNode[] {
  const { options, selected, onChange, sort } = args;

  const onChangeCreatedAt = (key: string, nextDirection: "asc" | "desc") => {
    const k: SortKey = key === "createdAt" ? "createdAt" : null;
    sort.onChange(k, nextDirection);
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