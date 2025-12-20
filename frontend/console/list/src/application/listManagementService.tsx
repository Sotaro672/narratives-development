// frontend/console/list/src/application/listManagementService.tsx

import React from "react";
import { FilterableTableHeader } from "../../../shell/src/layout/List/List";

import type { ListStatus } from "../../../shell/src/shared/types/list";

// 既存 mock（バックエンド未実装/障害時のフォールバック用）
import {
  LISTINGS,
  getListStatusLabel as getListStatusLabelMock,
  type ListingRow,
} from "../infrastructure/mockdata/mockdata";

// ✅ HTTP は repository へ移譲
import { fetchListsHTTP } from "../infrastructure/http/listRepositoryHTTP";

export type SortKey = "id" | null;

export type ListManagementRowVM = {
  id: string;

  // ✅ 画面表示（左から）
  productName: string;
  tokenName: string;
  assigneeName: string;

  status: ListStatus;
  statusLabel: string;

  // view-only (page keeps className only)
  statusBadgeText: string;
  statusBadgeClass: string;
};

export type FilterOptions = {
  productOptions: Array<{ value: string; label: string }>;
  tokenOptions: Array<{ value: string; label: string }>;
  managerOptions: Array<{ value: string; label: string }>;
  statusOptions: Array<{ value: ListStatus; label: string }>;
};

export type Filters = {
  productFilter: string[];
  tokenFilter: string[];
  managerFilter: string[];
  statusFilter: string[]; // ListStatus as string
};

export function normalizeStatus(raw: unknown): ListStatus {
  const s = String(raw ?? "").trim().toLowerCase();

  // Firestore: "list" / domain: "listing"
  if (s === "list" || s === "listing") return "listing";

  // Firestore: "hold" / domain: "suspended"
  if (s === "hold" || s === "suspended") return "suspended";

  if (s === "deleted") return "deleted";

  // unknown -> suspended（安全側）
  return "suspended";
}

export function getStatusLabelJP(status: ListStatus): string {
  // 表示要件: 出品中｜保留中
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

function buildFallbackRowsFromMock(): ListManagementRowVM[] {
  return (LISTINGS as ListingRow[]).map((r) => {
    const badge = buildStatusBadge(r.status);
    return {
      id: r.id,
      productName: r.productName,
      tokenName: r.tokenName,
      assigneeName: r.assigneeName,
      status: r.status,
      statusLabel: getListStatusLabelMock(r.status),
      statusBadgeText: badge.text,
      statusBadgeClass: badge.className,
    };
  });
}

/**
 * DTO -> ViewModel（best-effort）
 * ※ backend の DTO が enrich 済みなら productName/tokenName/assigneeName を優先
 * ※ 未 enrich の場合は title を productName として表示（最低限の可視化）
 */
export function mapAnyToVMRow(x: any): ListManagementRowVM {
  const id = String(x?.id ?? x?.ID ?? "").trim();

  const productName =
    String(x?.productName ?? x?.product_name ?? x?.title ?? x?.Title ?? "").trim();

  const tokenName = String(x?.tokenName ?? x?.token_name ?? "").trim();

  const assigneeName =
    String(x?.assigneeName ?? x?.assignee_name ?? "").trim() || "未設定";

  const st = normalizeStatus(x?.status ?? x?.Status);

  const badge = buildStatusBadge(st);

  return {
    id: id || "(missing id)",
    productName: productName || "",
    tokenName: tokenName || "",
    assigneeName,

    status: st,
    statusLabel: getStatusLabelJP(st),

    statusBadgeText: badge.text,
    statusBadgeClass: badge.className,
  };
}

/**
 * ✅ 一覧ロード（バックエンド → 失敗時は mock）
 * - HTTP は listRepositoryHTTP.tsx 側に寄せた
 */
export async function loadListManagementRows(): Promise<{
  rows: ListManagementRowVM[];
  error: string | null;
  usedFallback: boolean;
}> {
  try {
    const items = await fetchListsHTTP(); // ✅ repository 経由
    const mapped = items.map(mapAnyToVMRow).filter((r) => r.id !== "(missing id)");

    // eslint-disable-next-line no-console
    console.log("[list/listManagementService] mapped rows", {
      count: mapped.length,
      sample: mapped.slice(0, 3),
    });

    return { rows: mapped, error: null, usedFallback: false };
  } catch (e: any) {
    // eslint-disable-next-line no-console
    console.log("[list/listManagementService] fetch lists failed -> fallback to mock", {
      error: String(e?.message ?? e),
    });

    return {
      rows: buildFallbackRowsFromMock(),
      error: String(e?.message ?? e),
      usedFallback: true,
    };
  }
}

/**
 * ✅ Filter options（現在の rows から生成）
 */
export function buildFilterOptions(rows: ListManagementRowVM[]): FilterOptions {
  const productOptions = Array.from(new Set(rows.map((r) => r.productName)))
    .filter((v) => String(v ?? "").trim() !== "")
    .map((v) => ({ value: v, label: v }));

  const tokenOptions = Array.from(new Set(rows.map((r) => r.tokenName)))
    .filter((v) => String(v ?? "").trim() !== "")
    .map((v) => ({ value: v, label: v }));

  const managerOptions = Array.from(new Set(rows.map((r) => r.assigneeName)))
    .filter((v) => String(v ?? "").trim() !== "")
    .map((v) => ({ value: v, label: v }));

  const uniqStatus = Array.from(new Set<ListStatus>(rows.map((r) => r.status)));
  const statusOptions = uniqStatus.map((status) => ({
    value: status,
    label: getStatusLabelJP(status),
  }));

  return { productOptions, tokenOptions, managerOptions, statusOptions };
}

/**
 * ✅ フィルタ適用（4列）
 */
export function applyFilters(
  rows: ListManagementRowVM[],
  f: Filters,
): ListManagementRowVM[] {
  return rows.filter(
    (r) =>
      (f.productFilter.length === 0 || f.productFilter.includes(r.productName)) &&
      (f.tokenFilter.length === 0 || f.tokenFilter.includes(r.tokenName)) &&
      (f.managerFilter.length === 0 || f.managerFilter.includes(r.assigneeName)) &&
      (f.statusFilter.length === 0 || f.statusFilter.includes(r.status)),
  );
}

/**
 * ✅ ソート（必要最低限：id）
 */
export function applySort(
  rows: ListManagementRowVM[],
  activeKey: SortKey,
  direction: "asc" | "desc" | null,
): ListManagementRowVM[] {
  if (!activeKey || !direction) return rows;

  const data = [...rows];
  data.sort((a, b) => {
    const cmp = a.id.localeCompare(b.id);
    return direction === "asc" ? cmp : -cmp;
  });
  return data;
}

/**
 * ✅ ヘッダ生成（4列）
 */
export function buildHeaders(args: {
  options: FilterOptions;
  selected: Filters;
  onChange: {
    setProductFilter: (v: string[]) => void;
    setTokenFilter: (v: string[]) => void;
    setManagerFilter: (v: string[]) => void;
    setStatusFilter: (v: string[]) => void;
  };
}): React.ReactNode[] {
  const { options, selected, onChange } = args;

  return [
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
  ];
}
