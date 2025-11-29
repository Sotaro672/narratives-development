// frontend/console/production/src/application/productionManagementService.tsx

import type {
  Production,
  ProductionStatus,
} from "../../../shell/src/shared/types/production";
import { listProductionsHTTP } from "../infrastructure/query/productionQuery";

/** ソートキー */
export type SortKey = "printedAt" | "createdAt" | "totalQuantity" | null;

/** 一覧表示用に totalQuantity を付与した行型（内部用） */
export type ProductionRow = Production & {
  totalQuantity: number;
};

/** 画面表示用の行型（ラベル済み） */
export type ProductionRowView = {
  id: string;
  productBlueprintId: string;
  assigneeId: string;
  status: ProductionStatus;
  totalQuantity: number;
  printedAtLabel: string;
  createdAtLabel: string;
};

/** ISO8601 → timestamp（不正 or 未設定は 0） */
const toTs = (iso?: string | null): number => {
  if (!iso) return 0;
  const t = Date.parse(iso);
  return Number.isNaN(t) ? 0 : t;
};

/** ISO8601 → YYYY/M/D（不正 or 未設定は "-"） */
const formatDate = (iso?: string | null): string => {
  if (!iso) return "-";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "-";
  const y = d.getFullYear();
  const m = d.getMonth() + 1;
  const day = d.getDate();
  return `${y}/${m}/${day}`;
};

/** backend から Production 一覧を取得して、totalQuantity を付与した行に変換 */
export async function loadProductionRows(): Promise<ProductionRow[]> {
  const items = await listProductionsHTTP();

  const rows: ProductionRow[] = items.map((p: any) => {
    const models = Array.isArray(p.models) ? p.models : [];
    const totalQuantity = models.reduce(
      (sum: number, m: any) => sum + (m?.quantity ?? 0),
      0,
    );

    const row: ProductionRow = {
      ...(p as Production),
      totalQuantity,
    };

    return row;
  });

  return rows;
}

/** 商品設計IDフィルタ用オプション生成 */
export function buildBlueprintOptions(baseRows: ProductionRow[]) {
  return Array.from(new Set(baseRows.map((p) => p.productBlueprintId))).map(
    (v) => ({ value: v, label: v }),
  );
}

/** 担当者IDフィルタ用オプション生成 */
export function buildAssigneeOptions(baseRows: ProductionRow[]) {
  return Array.from(new Set(baseRows.map((p) => p.assigneeId))).map((v) => ({
    value: v,
    label: v,
  }));
}

/** ステータスフィルタ用オプション生成 */
export function buildStatusOptions(baseRows: ProductionRow[]) {
  return Array.from(new Set(baseRows.map((p) => p.status))).map((v) => ({
    value: v,
    label: v,
  }));
}

/** フィルタ＋ソートを適用し、表示用行に変換 */
export function buildRowsView(params: {
  baseRows: ProductionRow[];
  blueprintFilter: string[];
  assigneeFilter: string[];
  statusFilter: ProductionStatus[];
  sortKey: SortKey;
  sortDir: "asc" | "desc" | null;
}): ProductionRowView[] {
  const {
    baseRows,
    blueprintFilter,
    assigneeFilter,
    statusFilter,
    sortKey,
    sortDir,
  } = params;

  let data = baseRows.filter((p) => {
    if (
      blueprintFilter.length > 0 &&
      !blueprintFilter.includes(p.productBlueprintId)
    ) {
      return false;
    }
    if (assigneeFilter.length > 0 && !assigneeFilter.includes(p.assigneeId)) {
      return false;
    }
    if (statusFilter.length > 0 && !statusFilter.includes(p.status)) {
      return false;
    }
    return true;
  });

  if (sortKey && sortDir) {
    data = [...data].sort((a, b) => {
      if (sortKey === "totalQuantity") {
        const av = a.totalQuantity;
        const bv = b.totalQuantity;
        return sortDir === "asc" ? av - bv : bv - av;
      }
      const av = toTs(a[sortKey]);
      const bv = toTs(b[sortKey]);
      return sortDir === "asc" ? av - bv : bv - av;
    });
  }

  return data.map<ProductionRowView>((p) => ({
    id: p.id,
    productBlueprintId: p.productBlueprintId,
    assigneeId: p.assigneeId,
    status: p.status,
    totalQuantity: p.totalQuantity,
    printedAtLabel: formatDate(p.printedAt),
    createdAtLabel: formatDate(p.createdAt),
  }));
}
