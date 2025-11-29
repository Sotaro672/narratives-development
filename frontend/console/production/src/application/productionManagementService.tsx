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

  console.log("[productionManagementService] fetched items:", items);

  const rows: ProductionRow[] = items.map((raw: any) => {
    // Models 配列（大文字 / 小文字両対応）
    const rawModels = Array.isArray(raw.models)
      ? raw.models
      : Array.isArray(raw.Models)
        ? raw.Models
        : [];

    // quantity / Quantity の両方に対応して総数計算
    const totalQuantity = rawModels.reduce(
      (sum: number, m: any) => sum + (m?.quantity ?? m?.Quantity ?? 0),
      0,
    );

    // Firestore からの大文字キーを camelCase に寄せる
    const row: ProductionRow = {
      ...(raw as Production),

      // ID
      id: raw.id ?? raw.ID ?? "",

      // 商品設計 ID
      productBlueprintId:
        raw.productBlueprintId ?? raw.ProductBlueprintID ?? "",

      // 担当者 ID
      assigneeId: raw.assigneeId ?? raw.AssigneeID ?? "",

      // ステータス（service の Status をそのまま使う）
      status: (raw.status ?? raw.Status ?? "") as ProductionStatus,

      // 日付系（ISO 文字列想定）
      printedAt: raw.printedAt ?? raw.PrintedAt ?? null,
      createdAt: raw.createdAt ?? raw.CreatedAt ?? null,
      updatedAt: raw.updatedAt ?? raw.UpdatedAt ?? null,

      // models も最低限そのまま渡す（型は Production 側に依存）
      models: rawModels,

      totalQuantity,
    };

    return row;
  });

  console.log(
    "[productionManagementService] rows with totalQuantity (normalized):",
    rows,
  );

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

  const view = data.map<ProductionRowView>((p) => ({
    id: p.id,
    productBlueprintId: p.productBlueprintId,
    assigneeId: p.assigneeId,
    status: p.status, // ★ service の status をそのまま渡す
    totalQuantity: p.totalQuantity,
    printedAtLabel: formatDate(p.printedAt),
    createdAtLabel: formatDate(p.createdAt),
  }));

  console.log("[productionManagementService] view rows:", view);

  return view;
}
