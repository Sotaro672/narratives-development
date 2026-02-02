// frontend/console/list/src/application/listManagementService.tsx

import React from "react";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";

import type { ListStatus } from "../../../shell/src/shared/types/list";

// ✅ 分割後のHTTP入口（index.ts 経由）
import { fetchListsHTTP } from "../infrastructure/http/list";

export type SortKey = "id" | "createdAt" | null;

export type ListManagementRowVM = {
  id: string;

  // ✅ 画面表示（左から）
  title: string; // ✅ タイトル列（最左）
  productName: string;
  tokenName: string;
  assigneeName: string;

  status: ListStatus;
  statusLabel: string;

  // ✅ NEW: 作成日（ステータスの右隣）
  createdAt: string;

  // ✅ NEW: sort 用（ISO/RFC3339 を保持）
  createdAtRaw: string;

  // view-only (page keeps className only)
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

function pad2(n: number): string {
  return String(n).padStart(2, "0");
}

// ✅ yyyy/mm/dd hh:mm 形式（入力が不正ならそのまま返す）
function formatYMDHM(v: unknown): string {
  const raw = String(v ?? "").trim();
  if (!raw) return "";

  const d = new Date(raw);
  if (!Number.isFinite(d.getTime())) return raw;

  const yyyy = d.getFullYear();
  const mm = pad2(d.getMonth() + 1);
  const dd = pad2(d.getDate());
  const hh = pad2(d.getHours());
  const mi = pad2(d.getMinutes());

  return `${yyyy}/${mm}/${dd} ${hh}:${mi}`;
}

/**
 * DTO -> ViewModel（best-effort）
 * ※ backend の DTO が enrich 済みなら title/productName/tokenName/assigneeName を優先
 * ※ 未 enrich の場合でも title は表示できるようにする（最低限の可視化）
 */
export function mapAnyToVMRow(x: any): ListManagementRowVM {
  const id = String(x?.id ?? x?.ID ?? "").trim();

  // ✅ title の名揺れを吸収（一覧DTO側が listingTitle で返ってくる可能性がある）
  const title = String(
    x?.title ??
      x?.Title ??
      x?.listingTitle ??
      x?.ListingTitle ??
      x?.listing_title ??
      "",
  ).trim();

  const productName = String(x?.productName ?? x?.product_name ?? "").trim();

  const tokenName = String(x?.tokenName ?? x?.token_name ?? "").trim();

  const assigneeName =
    String(x?.assigneeName ?? x?.assignee_name ?? "").trim() || "未設定";

  const st = normalizeStatus(x?.status ?? x?.Status);

  const badge = buildStatusBadge(st);

  // ✅ NEW: createdAt（名揺れ吸収 + 表示フォーマット）
  const createdAtRaw = String(
    x?.createdAt ?? x?.CreatedAt ?? x?.created_at ?? "",
  ).trim();
  const createdAt = formatYMDHM(createdAtRaw);

  return {
    id: id || "(missing id)",

    title: title || "",
    productName: productName || "",
    tokenName: tokenName || "",
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
 * ✅ 一覧ロード（バックエンドのみ）
 * - mockdata フォールバックは削除
 */
export async function loadListManagementRows(): Promise<{
  rows: ListManagementRowVM[];
  error: string | null;
}> {
  try {
    // fetchListsHTTP の返りは ListDTO[] だが、ここでは DTO 形が揺れる前提で unknown[] 扱いにする
    const items = (await fetchListsHTTP()) as unknown[];

    const mapped = items
      .map((x) => mapAnyToVMRow(x as any))
      // ✅ r が暗黙anyにならないよう型を明示
      .filter((r: ListManagementRowVM) => r.id !== "(missing id)");

    // eslint-disable-next-line no-console
    console.log("[list/listManagementService] mapped rows", {
      count: mapped.length,
      sample: mapped.slice(0, 3),
    });

    return { rows: mapped, error: null };
  } catch (e: unknown) {
    const errMsg =
      e instanceof Error ? String(e.message) : String(e ?? "unknown_error");

    // eslint-disable-next-line no-console
    console.log("[list/listManagementService] fetch lists failed", {
      error: errMsg,
    });

    return {
      rows: [],
      error: errMsg,
    };
  }
}

/**
 * ✅ Filter options（現在の rows から生成）
 */
export function buildFilterOptions(rows: ListManagementRowVM[]): FilterOptions {
  const titleOptions = Array.from(new Set(rows.map((r) => r.title)))
    .filter((v) => String(v ?? "").trim() !== "")
    .map((v) => ({ value: v, label: v }));

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

  return {
    titleOptions,
    productOptions,
    tokenOptions,
    managerOptions,
    statusOptions,
  };
}

/**
 * ✅ フィルタ適用（5列）
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
  const d = new Date(String(v ?? "").trim());
  const t = d.getTime();
  return Number.isFinite(t) ? t : 0;
}

/**
 * ✅ ソート（id / createdAt）
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

    // default: id
    const cmp = a.id.localeCompare(b.id);
    return direction === "asc" ? cmp : -cmp;
  });
  return data;
}

/**
 * ✅ ヘッダ生成（6列）
 * - createdAt は「ステータスの右隣」+ SortableTableHeader
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

  // ✅ NEW: SortableTableHeader 用（hook から渡す）
  sort: {
    activeKey: SortKey;
    direction: "asc" | "desc" | null;
    onChange: (key: SortKey, dir: "asc" | "desc" | null) => void;
  };
}): React.ReactNode[] {
  const { options, selected, onChange, sort } = args;

  // ✅ SortableTableHeader の onChange 型に合わせる（key は string で来る）
  const onChangeCreatedAt = (key: string, nextDirection: "asc" | "desc") => {
    // key は "createdAt" 固定運用（念のためガード）
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

    // ✅ NEW: ステータスの右隣に作成日列（ソートあり / フィルタなし）
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
